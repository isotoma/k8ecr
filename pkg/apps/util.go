package apps

import (
	"os"
	"path/filepath"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func getClusterConfig(kubeconfig string) (*rest.Config, error) {
	if kubeconfig == "" {
		// We are running in-cluster
		config, err := rest.InClusterConfig()
		if err != nil {
			return nil, err
		}
		return config, nil
	} else {
		config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, err
		}
		return config, nil
	}
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}

func getClientSet() (*kubernetes.Clientset, error) {
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		home := homeDir()
		defpath := filepath.Join(home, ".kube", "config")
		if _, err := os.Stat(defpath); err == nil {
			kubeconfig = defpath
		}
	}
	config, err := getClusterConfig(kubeconfig)
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(config)
}
