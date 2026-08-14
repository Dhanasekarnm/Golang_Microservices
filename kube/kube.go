package kube

import (
	"log"
	"os"
	"pythonrpa/operations"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func ReadConfig() (*kubernetes.Clientset, error) {
	var error error
	error = nil
	kubeconfigbase64 := os.Getenv("KUBECONFIGBASE64")
	kubeconfig := os.Getenv("KUBECONFIG")

	log.Println("B64:", kubeconfigbase64)
	log.Println("config:", kubeconfig)

	err := operations.DecodeBase64toFile(kubeconfigbase64, kubeconfig)
	if err != nil {
		log.Println(err)
		error = err
	}

	stat, err := os.Stat(kubeconfig)
	if err != nil {
		log.Println("Config file available", stat)
	} else {
		log.Println("Config file not available")
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		log.Println(err)
		error = err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Println(err)
		error = err
	}
	return clientset, error
}
