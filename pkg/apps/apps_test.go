package apps

import (
	"testing"

	"k8s.io/client-go/kubernetes/fake"
)

var (
	c1 = Container{
		ImageID: id1,
		App:     "App1",
		ContainerID: ContainerIdentifier{
			Resource:  "res1",
			Container: "c1",
		},
	}
)

func TestAddContainer(T *testing.T) {
	mgr := AppManager{
		ClientSet: fake.NewSimpleClientset(),
		Namespace: "default",
		Apps:      make(map[string]*App),
		Managers:  make(map[string]*ResourceManager),
	}
	mgr.AddContainer("Foo", c1)
	if mgr.Apps["App1"].ChangeSets[id1].Containers["Foo"][0].ContainerID.Container != "c1" {
		T.Errorf("AddContainer failed")
	}
}
