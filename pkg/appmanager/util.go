package apps

import (
	"os"
	"path/filepath"

	"k8s.io/client-go/kubernetes"
)

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
