package resources

import "github.com/isotoma/k8ecr/pkg/imagemanager"

func resources(name string, meta metav1.ObjectMeta, spec []corev1.Container) []Resource {
	res := make([]Resource, 0)
	for _, c := range spec {
		id, version := parse(c.Image)
		r := Resource{
			ContainerID: ContainerIdentifier{
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

var deploymentResource = &imagemanager.ResourceManager{
	Kind: "Deployment",
	Fetcher: func(mgr *ImageManager) ([]Resource, error) {
		client := mgr.clientset.AppsV1beta1().Deployments(mgr.Namespace)
		response, err := client.List(metav1.ListOptions{})
		if err != nil {
			return nil, err
		}
		return response.Items, nil
	},
	Generator: func(item error) []Resource {
		allResources := make([]Resource, 0)
		for _, r := range resources(item.Name, item.ObjectMeta, item.Spec.Template.Spec.Containers) {
			allResources = append(allResources, r)
		}
		return allResources
	},
}

func init() {
	imagemanager.registerResource(deploymentResource)
}
