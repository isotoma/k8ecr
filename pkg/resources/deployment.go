package resources

import (
	"fmt"
	"strings"

	apps "github.com/isotoma/k8ecr/pkg/imagemanager"
	appsv1beta1 "k8s.io/api/apps/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func parse(url string) (apps.ImageIdentifier, apps.Version) {
	p1 := strings.Split(url, "/")
	registry := p1[0]
	var p2 []string
	switch {
	case len(p1) == 1:
		p2 = strings.Split(p1[0], ":")
	case len(p1) == 2:
		p2 = strings.Split(p1[1], ":")
	default:
		panic(fmt.Errorf("Unexpected number of / in image"))
	}
	repo := p2[0]
	version := "latest"
	if len(p2) == 2 {
		version = p2[1]
	}
	return apps.ImageIdentifier{
		Registry: registry,
		Repo:     repo,
	}, apps.Version(version)
}

func resources(name string, meta metav1.ObjectMeta, spec []corev1.Container) []apps.Resource {
	res := make([]apps.Resource, 0)
	for _, c := range spec {
		id, version := parse(c.Image)
		r := apps.Resource{
			ContainerID: apps.ContainerIdentifier{
				Resource:  name,
				Container: c.Name,
			},
			ImageID: id,
			App:     meta.Labels["app"],
			Current: version,
		}
		res = append(res, r)
	}
	return res
}

var deploymentResource = &apps.ResourceManager{
	Kind: "Deployment",
	Fetcher: func(mgr *apps.ImageManager) ([]interface{}, error) {
		client := mgr.ClientSet.AppsV1beta1().Deployments(mgr.Namespace)
		response, err := client.List(metav1.ListOptions{})
		if err != nil {
			return nil, err
		}
		empty := make([]interface{}, len(response.Items))
		for i, item := range response.Items {
			empty[i] = item
		}
		return empty, nil
	},
	Generator: func(item interface{}) []apps.Resource {
		var d appsv1beta1.Deployment
		d = item.(appsv1beta1.Deployment)
		allResources := make([]apps.Resource, 0)
		for _, r := range resources(d.Name, d.ObjectMeta, d.Spec.Template.Spec.Containers) {
			allResources = append(allResources, r)
		}
		return allResources
	},
}

func init() {
	fmt.Println("Regsitered")
	apps.RegisterResource(deploymentResource)
}
