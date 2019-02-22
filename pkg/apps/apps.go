package apps

import (
	"k8s.io/client-go/kubernetes"
)

// App is a group of images mapped to containers within resources in the app
type App struct {
	Name       string
	ChangeSets map[ImageIdentifier]*ChangeSet
}

// NewApp returns a new App
func NewApp(name string) *App {
	return &App{
		Name:       name,
		ChangeSets: make(map[ImageIdentifier]*ChangeSet),
	}
}

// GetChangeSets returns all changesets in the App
func (app *App) GetChangeSets() []*ChangeSet {
	cs := make([]*ChangeSet, 0)
	for _, v := range app.ChangeSets {
		cs = append(cs, v)
	}
	return cs
}

// SetLatest sets the latest version on every changeset in this app
func (app *App) SetLatest(registry, repository, version string) {
	id := ImageIdentifier{Registry: registry, Repo: repository}
	cs, ok := app.ChangeSets[id]
	if ok {
		cs.SetLatest(version)
	}

}

// AppManager finds and updates Applications
// and their deployments and cronjobs
type AppManager struct {
	ClientSet kubernetes.Interface
	Namespace string
	Apps      map[string]*App
	Managers  map[string]*ResourceManager
}

// NewAppManager creates a new Image manager
func NewAppManager(namespace string) (*AppManager, error) {
	clientset, err := getClientSet()
	if err != nil {
		return nil, err
	}
	a := &AppManager{
		ClientSet: clientset,
		Namespace: namespace,
		Apps:      make(map[string]*App),
		Managers:  resourceManagers,
	}
	err = a.Scan()
	return a, err
}

// SetLatest calls SetLatest on all contained apps
// Setting the version on all relevant containers
func (mgr *AppManager) SetLatest(registry, repository, version string) {
	for _, app := range mgr.Apps {
		app.SetLatest(registry, repository, version)
	}
}

// AddContainer adds the specified container, from a resource of the specified kind
// To the appropriate app
func (mgr *AppManager) AddContainer(kind string, container Container) {
	_, ok := mgr.Apps[container.App]
	if !ok {
		mgr.Apps[container.App] = NewApp(container.App)
	}
	_, ok = mgr.Apps[container.App].ChangeSets[container.ImageID]
	if !ok {
		mgr.Apps[container.App].ChangeSets[container.ImageID] = NewChangeSet(container.ImageID)
	}
	changeset := mgr.Apps[container.App].ChangeSets[container.ImageID]
	changeset.AddContainer(kind, container)
}

// Scan the cluster and find all resources and containers we manage
func (mgr *AppManager) Scan() error {
	for _, rm := range resourceManagers {
		items, err := rm.Resources(mgr)
		if err != nil {
			return err
		}
		for _, item := range items {
			for _, c := range rm.Generator(item) {
				mgr.AddContainer(rm.Kind, c)
			}
		}
	}
	return nil
}
