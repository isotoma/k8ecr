package apps

import (
	"fmt"
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

// Container represents a container
type Container struct {
	ContainerID ContainerIdentifier
	ImageID     ImageIdentifier
	App         string
	Current     Version
}

// ChangeSet contains resources that share an image identifier
type ChangeSet struct {
	ImageID     ImageIdentifier
	NeedsUpdate bool
	UpdateTo    Version
	Containers  map[string][]Container
}

// NewChangeSet creates a new changeset
func NewChangeSet(ID ImageIdentifier) *ChangeSet {
	return &ChangeSet{
		ImageID:     ID,
		NeedsUpdate: false,
		UpdateTo:    "",
		Containers:  make(map[string][]Container),
	}
}

// Upgrade all of the resources in this changeset, using the managers in the appmanager
func (cs *ChangeSet) Upgrade(mgr *AppManager) error {
	fmt.Printf("Updating image %s:\n", cs.ImageID.Repo)
	for kind, resources := range cs.Containers {
		for _, resource := range resources {
			fmt.Printf("    %s %s/%s\n", kind, resource.ContainerID.Resource, resource.ContainerID.Container)
			err := mgr.Managers[kind].Upgrade(mgr, cs, resource)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// Versions returns all versions in use for the image
func (cs *ChangeSet) Versions() []string {
	versions := make(map[Version]bool)
	for _, resources := range cs.Containers {
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

// RegistryPath returns the full registry path to which all images in the changeset should use
func (cs *ChangeSet) RegistryPath() string {
	return fmt.Sprintf("%s/%s:%s", cs.ImageID.Registry, cs.ImageID.Repo, cs.UpdateTo)
}
