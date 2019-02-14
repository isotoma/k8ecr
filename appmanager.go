package main

import (
	"os"
	"path/filepath"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type Container struct {
	Name    string
	Current Image
	Latest  string
}

// Resource is something that has an image
type Resource struct {
	Name       string
	App        string
	Containers []Container
}

// App is a grouping of deployments and cronjobs with the same app label
// in the same namespace
type App struct {
	Name        string
	Deployments []Resource
	Cronjobs    []Resource
}

// AppManager finds and updates applications
// and their deployments and cronjobs
type AppManager struct {
	clientset *kubernetes.Clientset
	Namespace string
	Apps      map[string]App
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

// Init initialises the application manager
func (a *AppManager) Init(namespace string) error {
	clientset, err := getClientSet()
	if err != nil {
		return err
	}
	a.clientset = clientset
	a.Namespace = namespace
	return a.Scan()
}

func makeContainerList(spec []corev1.Container) []Container {
	containers := make([]Container, len(spec))
	for _, c := range spec {
		container := Container{
			Name:    c.Name,
			Current: newImage(c.Image),
		}
		containers = append(containers, container)
	}
	return containers
}

func (a *AppManager) scanDeployments() ([]Resource, error) {
	resources := make([]Resource, 0)
	client := a.clientset.AppsV1().Deployments(a.Namespace)
	response, err := client.List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for _, item := range response.Items {
		if item.Name != "" {
			resource := Resource{
				Name:       item.Name,
				App:        item.ObjectMeta.Labels["App"],
				Containers: makeContainerList(item.Spec.Template.Spec.Containers),
			}
			resources = append(resources, resource)
		}
	}
	return resources, nil
}

func (a *AppManager) scanCronjobs() ([]Resource, error) {
	resources := make([]Resource, 0)
	client := a.clientset.BatchV1beta1().CronJobs(a.Namespace)
	response, err := client.List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for _, item := range response.Items {
		if item.Name != "" {
			resource := Resource{
				Name:       item.Name,
				App:        item.ObjectMeta.Labels["App"],
				Containers: makeContainerList(item.Spec.JobTemplate.Spec.Template.Spec.Containers),
			}
			resources = append(resources, resource)
		}
	}
	return resources, nil
}

func groupResources(deployments []Resource, cronjobs []Resource) map[string]App {
	apps := make(map[string]App)
	for _, item := range deployments {
		app, ok := apps[item.App]
		if !ok {
			app = App{
				Deployments: make([]Resource, 0),
				Cronjobs:    make([]Resource, 0),
			}
		}
		app.Deployments = append(app.Deployments, item)
		apps[item.App] = app
	}
	for _, item := range cronjobs {
		app, ok := apps[item.App]
		if !ok {
			app = App{
				Deployments: make([]Resource, 0),
				Cronjobs:    make([]Resource, 0),
			}
		}
		app.Cronjobs = append(app.Cronjobs, item)
		apps[item.App] = app
	}
	return apps
}

// Scan the specified namespace in the cluster and find
// all the deployments and cronjobs
// then create applications
func (a *AppManager) Scan() error {
	deployments, err := a.scanDeployments()
	if err != nil {
		return err
	}
	cronjobs, err := a.scanCronjobs()
	if err != nil {
		return err
	}
	a.Apps = groupResources(deployments, cronjobs)
	return nil
}
