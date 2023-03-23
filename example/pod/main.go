package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"path/filepath"
)

func main() {
	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		fmt.Println(home)
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")

	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	var container = "kubernetes-dashboard"
	var namespace = "kubernetes-dashboard"
	var podName = "kubernetes-dashboard-798cb8bdb-pkvb7"
	logOptions := &apiv1.PodLogOptions{
		Container: container,
		Follow:    true,
	}

	//真正获取pod的日志了
	stream, err := clientset.CoreV1().Pods(namespace).GetLogs(podName, logOptions).Stream(context.TODO())
	if err != nil {
		fmt.Println(err.Error())
		panic("xxx")
	}
	defer stream.Close()
	for {
		buffer := bufio.NewReader(stream)
		for {
			_, err := buffer.ReadString('\n') // 读到一个换行就结束
			if err == io.EOF {                // io.EOF表示文件的末尾
				break
			}
			//输出内容
			//fmt.Printf(str)
		}

	}
	select {}
}
