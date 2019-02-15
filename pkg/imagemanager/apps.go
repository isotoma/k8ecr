package imagemanager

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// Version is a version number expressed as a string
type Version string

// ImageIdentifier images are identified by their repo and registry
type ImageIdentifier struct {
	Repo     string
	Registry string
}

// ContainerIdentifier is a unique identifier for a container
type ContainerIdentifier struct {
	Resource  string
	Container string
}

// Resource is something that has an image
type Resource struct {
	ContainerID ContainerIdentifier
	ImageID     ImageIdentifier
	Current     Version
}

// ImageMap is a mapping of containers
type ImageMap struct {
	ImageID     ImageIdentifier
	NeedsUpdate bool
	UpdateTo    Version
	Deployments []Resource
	Cronjobs    []Resource
}

// Versions returns all versions in use for the image
func (i *ImageMap) Versions() []string {
	versions := make(map[Version]bool)
	for _, d := range i.Deployments {
		versions[d.Current] = true
	}
	for _, c := range i.Cronjobs {
		versions[c.Current] = true
	}
	rv := make([]string, 0)
	for v := range versions {
		rv = append(rv, string(v))
	}
	return rv
}

// ImageManager finds and updates Imagelications
// and their deployments and cronjobs
type ImageManager struct {
	clientset *kubernetes.Clientset
	Namespace string
	Images    map[ImageIdentifier]ImageMap
}

// NewImageManager creates a new Image manager
func NewImageManager(namespace string) (*ImageManager, error) {
	clientset, err := getClientSet()
	if err != nil {
		return nil, err
	}
	a := &ImageManager{
		clientset: clientset,
		Namespace: namespace,
		Images:    make(map[ImageIdentifier]ImageMap),
	}
	err = a.Scan()
	return a, err
}

// SetLatest sets the latest version on the image
// and flags if it needs update
func (mgr *ImageManager) SetLatest(registry, repository, version string) {
	id := ImageIdentifier{Registry: registry, Repo: repository}
	imap, ok := mgr.Images[id]
	if ok {
		imap.UpdateTo = Version(version)
		imap.NeedsUpdate = false
		versions := imap.Versions()
		for _, v := range versions {
			if v != version {
				imap.NeedsUpdate = true
			}
		}
		mgr.Images[id] = imap
	}
}

// GetImages in alphabetical order
func (mgr *ImageManager) GetImages() []ImageMap {
	// TODO SORTING
	images := make([]ImageMap, 0)
	for _, v := range mgr.Images {
		images = append(images, v)
	}
	return images
}

// Deploy the specified Imagelication to the specified version
func Deploy(Image string, version string) error {
	return nil
}

func parse(url string) (ImageIdentifier, Version) {
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
	return ImageIdentifier{
		Registry: registry,
		Repo:     repo,
	}, Version(version)
}

func resources(name string, spec []corev1.Container) []Resource {
	res := make([]Resource, 0)
	for _, c := range spec {
		id, version := parse(c.Image)
		r := Resource{
			ContainerID: ContainerIdentifier{
				Resource:  name,
				Container: c.Name,
			},
			ImageID: id,
			Current: version,
		}
		res = append(res, r)
	}
	return res
}

func (a *ImageManager) scanDeployments() ([]Resource, error) {
	allResources := make([]Resource, 0)
	client := a.clientset.AppsV1beta1().Deployments(a.Namespace)
	response, err := client.List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for _, item := range response.Items {
		if item.Name != "" {
			for _, r := range resources(item.Name, item.Spec.Template.Spec.Containers) {
				allResources = append(allResources, r)
			}
		}
	}
	return allResources, nil
}

func (a *ImageManager) scanCronjobs() ([]Resource, error) {
	allResources := make([]Resource, 0)
	client := a.clientset.BatchV1beta1().CronJobs(a.Namespace)
	response, err := client.List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for _, item := range response.Items {
		if item.Name != "" {
			for _, r := range resources(item.Name, item.Spec.JobTemplate.Spec.Template.Spec.Containers) {
				allResources = append(allResources, r)
			}
		}
	}
	return allResources, nil
}

func groupResources(deployments []Resource, cronjobs []Resource) map[ImageIdentifier]ImageMap {
	images := make(map[ImageIdentifier]ImageMap)
	for _, item := range deployments {
		image, ok := images[item.ImageID]
		if !ok {
			image = ImageMap{
				ImageID:     item.ImageID,
				Deployments: make([]Resource, 0),
				Cronjobs:    make([]Resource, 0),
			}
		}
		image.Deployments = append(image.Deployments, item)
		images[item.ImageID] = image
	}
	for _, item := range cronjobs {
		image, ok := images[item.ImageID]
		if !ok {
			image = ImageMap{
				ImageID:     item.ImageID,
				Deployments: make([]Resource, 0),
				Cronjobs:    make([]Resource, 0),
			}
		}
		image.Cronjobs = append(image.Cronjobs, item)
		images[item.ImageID] = image
	}
	return images
}

// Scan the specified namespace in the cluster and find
// all the deployments and cronjobs
// then create Imagelications
func (a *ImageManager) Scan() error {
	deployments, err := a.scanDeployments()
	if err != nil {
		return err
	}
	cronjobs, err := a.scanCronjobs()
	if err != nil {
		return err
	}
	a.Images = groupResources(deployments, cronjobs)
	return nil
}
