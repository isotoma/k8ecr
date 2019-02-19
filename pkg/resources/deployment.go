package resources

import (
	"fmt"

	"github.com/isotoma/k8ecr/pkg/apps"
	appsv1beta1 "k8s.io/api/apps/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var deploymentResource = &apps.ResourceManager{
	Kind: "Deployment",
	Resources: func(mgr *apps.AppManager) ([]interface{}, error) {
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
	Generator: func(item interface{}) []apps.Container {
		var d appsv1beta1.Deployment
		d = item.(appsv1beta1.Deployment)
		allResources := make([]apps.Container, 0)
		for _, r := range resources(d.Name, d.ObjectMeta, d.Spec.Template.Spec.Containers) {
			allResources = append(allResources, r)
		}
		return allResources
	},
	Upgrade: func(mgr *apps.AppManager, image *apps.ChangeSet, resource apps.Container) error {
		client := mgr.ClientSet.AppsV1beta1().Deployments(mgr.Namespace)
		item, err := client.Get(resource.ContainerID.Resource, metav1.GetOptions{})
		if err != nil {
			return err
		}
		for i, container := range item.Spec.Template.Spec.Containers {
			if container.Name == resource.ContainerID.Container {
				fmt.Printf("        %s/%s image -> %s\n", resource.ContainerID.Resource, resource.ContainerID.Container, image.RegistryPath())
				item.Spec.Template.Spec.Containers[i].Image = image.RegistryPath()
			}
		}
		_, err = client.Update(item)
		return err
	},
}
