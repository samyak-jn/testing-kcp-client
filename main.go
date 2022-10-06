package main

import (
	"context"
	"flag"
	"fmt"
	"path/filepath"

	kuberneteskcp "github.com/kcp-dev/client-go/clients/clientset/versioned"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

func main() {
	fmt.Println("Exploring KCP Client-Go library!")

	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, "cps-kubeconfig"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	cfg, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		logrus.WithError(err).Fatal("could not get config")
	}

	clientset, err := kuberneteskcp.NewForConfig(cfg)
	if err != nil {
		panic(err.Error())
	}

	appsclusterInterface := clientset.AppsV1()
	statefulappsclusterInterface, err := appsclusterInterface.StatefulSets().List(context.Background(), metav1.ListOptions{})

	if err != nil {
		panic(err.Error())
	}
	fmt.Println(statefulappsclusterInterface)

}
