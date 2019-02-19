package apps

type ResourceManager struct {
	Kind      string
	Fetcher   func(mgr *AppManager) ([]interface{}, error)
	Generator func(item interface{}) []Container
	Upgrade   func(mgr *AppManager, image *ChangeSet, resource Container) error
}

var resourceManagers = map[string]*ResourceManager{}

func RegisterResource(r *ResourceManager) {
	resourceManagers[r.Kind] = r
}
