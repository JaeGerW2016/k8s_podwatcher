package config

import (
	"flag"
	"fmt"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"
	"os"
	"os/user"
	"path/filepath"
)

var (
	kubeconfigPath, apiServerURL string
)

func init() {
	flag.StringVar(&kubeconfigPath, "kubeconfigPath", "", "Paths to a kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&apiServerURL, "master", "", "(Deprecated: switch to `--kubeconfig`) The address of the Kubernetes API server. Overrides any value in kubeconfig. "+
		"Only required if out-of-cluster.")
}

func GetConfig() (*rest.Config, error) {
	if len(kubeconfigPath) > 0 {
		return clientcmd.BuildConfigFromFlags(apiServerURL, kubeconfigPath)
	}
	if len(os.Getenv("KUBECONFIG")) > 0 {
		return clientcmd.BuildConfigFromFlags(apiServerURL, os.Getenv("KUBECONFIG"))
	}
	if kubeconfig, err := rest.InClusterConfig(); err == nil {
		return kubeconfig, nil
	}
	if usr, err := user.Current(); err == nil {
		if kubeconfig, err := clientcmd.BuildConfigFromFlags("", filepath.Join(usr.HomeDir, ".kube", "config")); err == nil {
			return kubeconfig, nil
		}
	}
	return nil, fmt.Errorf("could not locate a kubeconfig")
}

func GetConfigOrDie() (*rest.Config, error){
	config, err := GetConfig()
	if err != nil {
		klog.Error(err, "unable to get kubeconfig")
		os.Exit(1)
	}
	return config, nil
}
