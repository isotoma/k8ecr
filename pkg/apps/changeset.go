package apps

import (
	"fmt"
	"sort"

	"github.com/Masterminds/semver"
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
	Containers  map[string][]Container // Map of Kinds to lists of containers
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

// SetLatest sets the latest version, and checks if this changeset requires update
// Uses SemVer to perform comparisons if possible, otherwise falls back to string
// equality comparison.
func (cs *ChangeSet) SetLatest(version string) {
	cs.UpdateTo = Version(version)
	sv, err := semver.NewVersion(version)
	if err != nil {
		// new version is not semver, we just do a string comparison
		for _, v := range cs.Versions() {
			if v != version {
				cs.NeedsUpdate = true
				return
			}
		}
	} else {
		for _, v := range cs.Versions() {
			oldv, err := semver.NewVersion(v)
			if err != nil {
				cs.NeedsUpdate = true
				return
			}
			if oldv.Compare(sv) < 0 {
				cs.NeedsUpdate = true
				return
			}
		}
	}
}

func (cs *ChangeSet) AddContainer(kind string, container Container) {
	_, ok := cs.Containers[kind]
	if !ok {
		cs.Containers[kind] = make([]Container, 0)
	}
	cs.Containers[kind] = append(cs.Containers[kind], container)
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
	versions := make(map[Version]bool) // using a map as a set
	for _, containers := range cs.Containers {
		for _, c := range containers {
			versions[c.Current] = true
		}
	}
	rv := make([]string, 0)
	for v := range versions {
		rv = append(rv, string(v))
	}
	sort.Strings(rv)
	return rv
}

// RegistryPath returns the full registry path to which all images in the changeset should use
func (cs *ChangeSet) RegistryPath() string {
	return fmt.Sprintf("%s/%s:%s", cs.ImageID.Registry, cs.ImageID.Repo, cs.UpdateTo)
}
