package main

import (
	"os"
	"path/filepath"
	"sort"

	"github.com/Masterminds/semver"
	v1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

type DeployCommand struct{}

var deployCommand DeployCommand

func latestVersion(versions []string) (string, error) {
	vs := make([]*semver.Version, len(versions))
	for i, r := range versions {
		v, err := semver.NewVersion(r)
		if err != nil {
			return "", err
		}
		vs[i] = v
	}
	sort.Sort(semver.Collection(vs))
	return vs[len(vs)-1].Original(), nil
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}

func getDeployments(namespace string) (*v1.DeploymentList, error) {
	home := homeDir()
	kubeconfig := filepath.Join(home, ".kube", "config")
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, err
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	deployments, err := clientset.AppsV1().Deployments(namespace).List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return deployments, nil
}

func init() {
	parser.AddCommand(
		"deploy",
		"Deploy",
		"Deploy an image to your cluster",
		&deployCommand)
}
