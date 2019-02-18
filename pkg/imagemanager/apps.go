package imagemanager

import (
	"fmt"

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
	Resources   map[string][]Resource
}

func NewImageMap(ID ImageIdentifier) *ImageMap {
	return &ImageMap{
		ImageID:     ID,
		NeedsUpdate: false,
		UpdateTo:    "",
		Resources:   make(map[string][]Resource),
	}
}

// Versions returns all versions in use for the image
func (i *ImageMap) Versions() []string {
	versions := make(map[Version]bool)
	for _, resources := range i.Resources {
		for _, item := range resources {
			versions[item.Current] = true
		}
	}
	rv := make([]string, 0)
	for v := range versions {
		rv = append(rv, string(v))
	}
	return rv
}

func (i *ImageMap) NewImage() string {
	return fmt.Sprintf("%s/%s:%s", i.ImageID.Registry, i.ImageID.Repo, i.UpdateTo)
}

// App is a group of images mapped to containers within resources in the app
type App struct {
	Name   string
	Images map[ImageIdentifier]ImageMap
}

func NewApp(name string) *App {
	return &App{
		Name:   name,
		Images: make(map[ImageIdentifier]ImageMap),
	}
}

func (app *App) GetImages() []ImageMap {
	images := make([]ImageMap, 0)
	for _, v := range app.Images {
		images = append(images, v)
	}
	return images
}

type ResourceManager struct {
	Kind      string
	Fetcher   func(mgr *ImageManager) ([]interface{}, error)
	Generator func(item interface{}) []Resource
	Upgrade   func(mgr *ImageManager, image *ImageMap, resource Resource) error
}

var resourceManagers = map[string]*ResourceManager{}

func RegisterResource(r *ResourceManager) {
	resourceManagers[r.Kind] = r
}

// ImageManager finds and updates Applications
// and their deployments and cronjobs
type ImageManager struct {
	ClientSet kubernetes.Interface
	Namespace string
	Apps      map[string]App
	Managers  map[string]*ResourceManager
}

// NewImageManager creates a new Image manager
func NewImageManager(namespace string) (*ImageManager, error) {
	clientset, err := getClientSet()
	if err != nil {
		return nil, err
	}
	a := &ImageManager{
		ClientSet: clientset,
		Namespace: namespace,
		Apps:      make(map[string]App),
		Managers:  resourceManagers,
	}
	err = a.Scan()
	return a, err
}

// Upgrade all of the resources using this image
func (mgr *ImageManager) Upgrade(image *ImageMap) error {
	fmt.Printf("Updating image %s:\n", image.ImageID.Repo)
	for kind, resources := range image.Resources {
		for _, resource := range resources {
			fmt.Printf("    %s %s/%s\n", kind, resource.ContainerID.Resource, resource.ContainerID.Container)
			err := mgr.Managers[kind].Upgrade(mgr, image, resource)
			if err != nil {
				return err
			}
		}
	}
	return nil
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

func groupResources(resources map[string][]Resource) map[string]App {
	apps := make(map[string]App)
	for kind, resources := range resources {
		for _, item := range resources {
			app, ok := apps[item.App]
			if !ok {
				app = *NewApp(item.App)
			}
			image, ok := app.Images[item.ImageID]
			if !ok {
				image = *NewImageMap(item.ImageID)
			}
			image.Resources[kind] = append(image.Resources[kind], item)
			app.Images[item.ImageID] = image
			apps[app.Name] = app
		}
	}
	return apps
}

func (mgr *ImageManager) Scan() error {
	resources := make(map[string][]Resource)
	for _, rm := range resourceManagers {
		resources[rm.Kind] = make([]Resource, 0)
		items, err := rm.Fetcher(mgr)
		if err != nil {
			return err
		}
		for _, item := range items {
			for _, r := range rm.Generator(item) {
				resources[rm.Kind] = append(resources[rm.Kind], r)
			}
		}
	}
	mgr.Apps = groupResources(resources)
	return nil
}
