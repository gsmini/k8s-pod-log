package main

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/gsmini/k8s-pod-log/pkg/kubernetes"
	myWss "github.com/gsmini/k8s-pod-log/pkg/websocket"
	"go.uber.org/zap"
)

func wsHandler(ctx *gin.Context) {
	namespace := ctx.DefaultQuery("namespace", "")
	podName := ctx.DefaultQuery("pod", "")
	containerName := ctx.DefaultQuery("container", "")
	//1- 获取回话id
	sessionId := ctx.DefaultQuery("session", "")
	if len(namespace) == 0 || len(podName) == 0 || len(containerName) == 0 || len(sessionId) == 0 {
		ctx.JSON(403, map[string]string{"message": "参数错误"})
	}

	//服务升级，对于来到的http连接进行服务升级，升级到ws
	conn, err := myWss.Upgrade.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		panic(err)
	}
	//2- 动态创建对应的k8s client 链接
	clientSet, err := kubernetes.NewClientSetFromKubeConfig()
	if err != nil {
		panic(err.Error())
	}

	//3- websocket读写数据实现pod的日志实时查看的功能
	ctx2, cancel := context.WithCancel(context.Background())
	wsClient := &myWss.Client{
		SessionId:        sessionId,
		Socket:           conn,
		KeepAliveTimeout: 60,
		K8sClientSet:     clientSet,
		Ctx:              ctx2,
		Cancel:           cancel,
	}
	//其实就是每次websocket链接去go一个go程 对这个conn去读写数据
	go wsClient.Write(ctx2, namespace, podName, containerName)
	go wsClient.KeepAlive(ctx2)

}
func main() {
	// 使用gin框架，和普通的http协议的服务器没什么不一样
	//ws://127.0.0.1:8090/connect?namespace=xxx&pod=xxx&container=xxx
	srv := gin.Default()

	srv.GET("/ws", wsHandler)
	zap.L().Info("Start Server ...")
	srv.Run("127.0.0.1:8282")

}
