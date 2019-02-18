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
	App         string
	Current     Version
}

// ImageMap is a mapping of containers that are for a specified App and Image
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

func (i *ImageMap) newImage() string {
	return fmt.Sprintf("%s/%s:%s", i.ImageID.Registry, i.ImageID.Repo, i.UpdateTo)
}

// App is a group of images mapped to containers within resources in the app
type App struct {
	Name   string
	Images map[ImageIdentifier]ImageMap
}

func (app *App) GetImages() []ImageMap {
	images := make([]ImageMap, 0)
	for _, v := range app.Images {
		images = append(images, v)
	}
	return images
}

// ImageManager finds and updates Imagelications
// and their deployments and cronjobs
type ImageManager struct {
	clientset *kubernetes.Clientset
	Namespace string
	Apps      map[string]App
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
		Apps:      make(map[string]App),
	}
	err = a.Scan()
	return a, err
}

// UpgradeDeployments upgrades all deployments in the specified imagemap
func (mgr *ImageManager) UpgradeDeployments(image *ImageMap) error {
	client := mgr.clientset.AppsV1beta1().Deployments(mgr.Namespace)
	for _, r := range image.Deployments {
		item, err := client.Get(r.ContainerID.Resource, metav1.GetOptions{})
		if err != nil {
			return err
		}
		for i, container := range item.Spec.Template.Spec.Containers {
			if container.Name == r.ContainerID.Container {
				fmt.Printf("%s/%s image -> %s\n", r.ContainerID.Resource, r.ContainerID.Container, image.newImage())
				item.Spec.Template.Spec.Containers[i].Image = image.newImage()
			}
		}
		_, err = client.Update(item)
		if err != nil {
			return err
		}
	}
	return nil
}

// UpgradeCronjobs upgrades all cronjobs in the specified imagemap
func (mgr *ImageManager) UpgradeCronjobs(image *ImageMap) error {
	client := mgr.clientset.BatchV1beta1().CronJobs(mgr.Namespace)
	for _, r := range image.Cronjobs {
		item, err := client.Get(r.ContainerID.Resource, metav1.GetOptions{})
		if err != nil {
			return err
		}
		for i, container := range item.Spec.JobTemplate.Spec.Template.Spec.Containers {
			if container.Name == r.ContainerID.Container {
				fmt.Printf("%s/%s image -> %s\n", r.ContainerID.Resource, r.ContainerID.Container, image.newImage())
				item.Spec.JobTemplate.Spec.Template.Spec.Containers[i].Image = image.newImage()
			}
		}
		_, err = client.Update(item)
		if err != nil {
			return err
		}
	}
	return nil
}

// Upgrade all of the resources using this image
func (mgr *ImageManager) Upgrade(image *ImageMap) error {
	fmt.Printf("Updating %s\n", image.ImageID.Repo)
	if err := mgr.UpgradeDeployments(image); err != nil {
		return err
	}
	return mgr.UpgradeCronjobs(image)
}

// SetLatest sets the latest version on the image
// and flags if it needs update
func (mgr *ImageManager) SetLatest(registry, repository, version string) {
	id := ImageIdentifier{Registry: registry, Repo: repository}
	for _, app := range mgr.Apps {
		imap, ok := app.Images[id]
		if ok {
			imap.UpdateTo = Version(version)
			imap.NeedsUpdate = false
			versions := imap.Versions()
			for _, v := range versions {
				if v != version {
					imap.NeedsUpdate = true
				}
			}
			app.Images[id] = imap
		}
	}
}

// GetImages in alphabetical order
func (mgr *ImageManager) GetImages() []ImageMap {
	// TODO SORTING
	images := make([]ImageMap, 0)
	for _, a := range mgr.Apps {
		for _, v := range a.Images {
			images = append(images, v)
		}
	}
	return images
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

func (mgr *ImageManager) scanDeployments() ([]Resource, error) {
	allResources := make([]Resource, 0)
	client := mgr.clientset.AppsV1beta1().Deployments(mgr.Namespace)
	response, err := client.List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for _, item := range response.Items {
		if item.Name != "" {
			for _, r := range resources(item.Name, item.ObjectMeta, item.Spec.Template.Spec.Containers) {
				allResources = append(allResources, r)
			}
		}
	}
	return allResources, nil
}

func (mgr *ImageManager) scanCronjobs() ([]Resource, error) {
	allResources := make([]Resource, 0)
	client := mgr.clientset.BatchV1beta1().CronJobs(mgr.Namespace)
	response, err := client.List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for _, item := range response.Items {
		if item.Name != "" {
			for _, r := range resources(item.Name, item.ObjectMeta, item.Spec.JobTemplate.Spec.Template.Spec.Containers) {
				allResources = append(allResources, r)
			}
		}
	}
	return allResources, nil
}

type appender func(m *ImageMap, r *Resource)

func groupResources2(resources []Resource, apps map[string]App, fn appender) {
	for _, item := range resources {
		app, ok := apps[item.App]
		if !ok {
			app = App{
				Name:   item.App,
				Images: make(map[ImageIdentifier]ImageMap),
			}
		}
		image, ok := app.Images[item.ImageID]
		if !ok {
			image = ImageMap{
				ImageID:     item.ImageID,
				Deployments: make([]Resource, 0),
				Cronjobs:    make([]Resource, 0),
			}
		}
		fn(&image, &item)
		app.Images[item.ImageID] = image
		apps[app.Name] = app
	}
}

func groupResources(deployments []Resource, cronjobs []Resource) map[string]App {
	apps := make(map[string]App)
	appendDeployment := func(m *ImageMap, r *Resource) {
		m.Deployments = append(m.Deployments, *r)
	}
	groupResources2(deployments, apps, appendDeployment)
	appendCronjob := func(m *ImageMap, r *Resource) {
		m.Cronjobs = append(m.Cronjobs, *r)
	}
	groupResources2(cronjobs, apps, appendCronjob)
	return apps
}

// Scan the specified namespace in the cluster and find
// all the deployments and cronjobs
// then create Imagelications
func (mgr *ImageManager) Scan() error {
	deployments, err := mgr.scanDeployments()
	if err != nil {
		return err
	}
	cronjobs, err := mgr.scanCronjobs()
	if err != nil {
		return err
	}
	mgr.Apps = groupResources(deployments, cronjobs)
	return nil
}
