package websocket

import (
	"bufio"
	"context"
	"github.com/gorilla/websocket"
	"io"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	"net/http"
	"time"
)

//websocket的升级配置
var Upgrade = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024 * 1024 * 10,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// Client 表示websocket链接
type Client struct {
	SessionId        string
	Socket           *websocket.Conn
	PodLog           chan []byte
	LastPingTime     time.Time // 最近的探活时间
	KeepAliveTimeout int64
	Ctx              context.Context
	Cancel           context.CancelFunc
	K8sClientSet     *kubernetes.Clientset //k8s客户端
}

func (c *Client) KeepAlive(ctx context.Context) {
	defer c.Socket.Close()
	tick := time.NewTicker(time.Second * time.Duration(1+c.KeepAliveTimeout))
	defer tick.Stop()
	for {
		select {
		case <-tick.C:
			if time.Now().Sub(c.LastPingTime) > time.Second*time.Duration(c.KeepAliveTimeout) {
				klog.Info("Proxy KeepAlive timeout:lastPingTime:%v", c.LastPingTime)
				return
			}
		}
	}
}
func (c *Client) Read() {
	defer func() {
		c.Socket.Close()
	}()

	for {
		c.Socket.PongHandler()
		_, _, err := c.Socket.ReadMessage()
		if err != nil {
			c.Socket.Close()
			break
		}
	}
}

func (c *Client) Write(ctx context.Context, namespace, podName, container string) {
	defer func() {
		c.Socket.Close()
	}()

	logOptions := &apiv1.PodLogOptions{
		Container: container,
		Follow:    true,                           //实时流
		Previous:  false,                          //以前的不要
		SinceTime: &metav1.Time{Time: time.Now()}, //取现在开始的日志 以前的不要,不然会把历史log全部读出来导致broken pipe
	}
	//真正获取pod的日志了
	stream, err := c.K8sClientSet.CoreV1().Pods(namespace).GetLogs(podName, logOptions).Stream(context.TODO())

	if err != nil {
		//输出内容
		klog.Info("sessionid 为 %s 的pod %s消息读取错误 %s", c.SessionId, podName, err.Error())
		c.Socket.WriteMessage(websocket.TextMessage, []byte("读取消息出错，请查看日,se志"))
		return

	}
	defer stream.Close()

	for {
		buffer := bufio.NewReader(stream)
		for {
			message, err := buffer.ReadString('\n') // 读到一个换行就结束
			if err == io.EOF {                      // io.EOF表示文件的末尾
				break
			}
			//输出内容
			//log.Printf("发送到到客户端的信息:%s", message)

			err = c.Socket.WriteMessage(websocket.TextMessage, []byte(message))
			if err != nil {
				klog.Errorf("sessionid 为 %s 的pod %s消息发送出错 %s", c.SessionId, podName, err.Error())
			}

		}

	}

}
