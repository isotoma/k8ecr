package resources

import (
	"fmt"
	"strings"

	"github.com/isotoma/k8ecr/pkg/apps"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func parse(url string) (*apps.ImageIdentifier, apps.Version) {
	p1 := strings.Split(url, "/")
	registry := p1[0]
	if strings.HasSuffix(registry, "amazonaws.com") {
		fmt.Printf("Registry %s\n", registry)
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
		return &apps.ImageIdentifier{
			Registry: registry,
			Repo:     repo,
		}, apps.Version(version)
	}
	return nil, ""
}

func resources(name string, meta metav1.ObjectMeta, spec []corev1.Container) []apps.Container {
	res := make([]apps.Container, 0)
	for _, c := range spec {
		id, version := parse(c.Image)
		if id != nil {
			r := apps.Container{
				ContainerID: apps.ContainerIdentifier{
					Resource:  name,
					Container: c.Name,
				},
				ImageID: *id,
				App:     meta.Labels["app"],
				Current: version,
			}
			res = append(res, r)
		}
	}
	return res
}

func Register() {
	apps.RegisterResource(deploymentResource)
	apps.RegisterResource(cronjobResource)
}
