package apps

import "testing"

var (
	id1 = ImageIdentifier{Registry: "reg1", Repo: "repo1"}

	container1 = Container{
		ImageID: id1,
		ContainerID: ContainerIdentifier{
			Resource:  "Resource1",
			Container: "app",
		},
		App:     "App1",
		Current: "0.1.0",
	}

	container2 = Container{
		ImageID: id1,
		ContainerID: ContainerIdentifier{
			Resource:  "Resource2",
			Container: "app",
		},
		App:     "App1",
		Current: "1.0.0",
	}
	container3 = Container{
		ImageID: id1,
		ContainerID: ContainerIdentifier{
			Resource:  "Resource2",
			Container: "app",
		},
		App:     "App1",
		Current: "latest",
	}
)

func TestSetLatest(T *testing.T) {
	cs := NewChangeSet(id1)
	cs.SetLatest("1.0.0")
	if cs.UpdateTo != Version("1.0.0") {
		T.Errorf("SetLatest fails to set UpdateTo")
	}
	if cs.NeedsUpdate {
		T.Errorf("Empty changeset needs update when it shouldn't")
	}
	cs2 := NewChangeSet(id1)
	cs2.AddContainer("Foo", container1)
	cs2.SetLatest("1.0.1")
	if cs2.UpdateTo != Version("1.0.1") {
		T.Errorf("SetLatest fails to set UpdateTo")
	}
	if !cs2.NeedsUpdate {
		T.Errorf("Changeset should need update")
	}
	cs2.SetLatest("latest")
	if !cs2.NeedsUpdate {
		T.Errorf("Should do string comparison for non semver input")
	}
	cs3 := NewChangeSet(id1)
	cs3.AddContainer("Foo", container2)
	cs3.SetLatest("0.5.0")
	if cs3.NeedsUpdate {
		T.Errorf("Changeset should not need update")
	}
	cs4 := NewChangeSet(id1)
	cs4.AddContainer("Foo", container3)
	cs4.SetLatest("0.5.0")
	if !cs4.NeedsUpdate {
		T.Errorf("Should always update from latest")
	}
}
