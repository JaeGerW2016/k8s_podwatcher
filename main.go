package main

import (
	"flag"
	"fmt"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
	config2 "k8s_podwatcher-master/pkg/config"
	"k8s_podwatcher-master/pkg/controller"
	"k8s_podwatcher-master/pkg/handlers/email"
	"log"
	"os"
	"os/signal"
	"syscall"
)
var namespace string
func init() {
	flag.StringVar(&namespace,"namespace","default","namespace")
	flag.Parse()
}
func main() {
	config, err := config2.GetConfig()
	if err != nil {
		log.Panic(fmt.Sprintf("get config failed: %v", err))
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		klog.Fatalf("Failed to create clientset: %v", err)
	}
	versionInfo, err := clientset.Discovery().ServerVersion()
	if err != nil {
		klog.Fatalf("Failed to discover server version: %v", err)
	}
	klog.Infof("Server version: %v", versionInfo)

	stopCh := make(chan struct{})
	sigsCh := make(chan os.Signal,1)
	signal.Notify(sigsCh,syscall.SIGINT,syscall.SIGTERM)
	go func() {
		sig := <-sigsCh
		klog.Infof("got signal: %s",sig)
		close(stopCh)
	}()

	controller.NewController(clientset,email.NewHaddler(),namespace).Run(stopCh)
}
