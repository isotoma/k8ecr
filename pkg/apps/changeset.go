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

func NewChangeSet(ID ImageIdentifier) *ChangeSet {
	return &ChangeSet{
		ImageID:     ID,
		NeedsUpdate: false,
		UpdateTo:    "",
		Containers:  make(map[string][]Container),
	}
}

// Versions returns all versions in use for the image
func (i *ChangeSet) Versions() []string {
	versions := make(map[Version]bool)
	for _, resources := range i.Containers {
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

func (i *ChangeSet) RegistryPath() string {
	return fmt.Sprintf("%s/%s:%s", i.ImageID.Registry, i.ImageID.Repo, i.UpdateTo)
}
