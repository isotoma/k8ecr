package imagemanager

import (
	"reflect"
	"testing"

	"k8s.io/client-go/kubernetes/fake"
)

func TestScanDeployments(t *testing.T) {
	mgr := ImageManager{
		clientset: fake.NewSimpleClientset(),
		Namespace: "test",
		Apps:      make(map[string]App),
	}
}

func TestGroupResources(t *testing.T) {
	r1 := Resource{
		ContainerID: ContainerIdentifier{
			Resource:  "r1",
			Container: "c1",
		},
		ImageID: ImageIdentifier{
			Repo:     "repo1",
			Registry: "image1",
		},
		App:     "app1",
		Current: "1.0.0",
	}
	r2 := Resource{
		ContainerID: ContainerIdentifier{
			Resource:  "r2",
			Container: "c1",
		},
		ImageID: ImageIdentifier{
			Repo:     "repo1",
			Registry: "image1",
		},
		App:     "app1",
		Current: "1.0.0",
	}
	a1d := Image{
		Name:        "Image1",
		Deployments: []Resource{r1},
		Cronjobs:    []Resource{},
	}
	a1c := Image{
		Name:        "Image1",
		Deployments: []Resource{},
		Cronjobs:    []Resource{r1},
	}
	a2 := Image{
		Name:        "Image1",
		Deployments: []Resource{r1},
		Cronjobs:    []Resource{r2},
	}
	tests := []struct {
		name        string
		deployments []Resource
		cronjobs    []Resource
		result      map[string]Image
	}{
		{"empty", []Resource{}, []Resource{}, map[string]Image{}},
		{"one deployment", []Resource{r1}, []Resource{}, map[string]Image{"Image1": a1d}},
		{"one cronjob", []Resource{}, []Resource{r1}, map[string]Image{"Image1": a1c}},
		{"one of each", []Resource{r1}, []Resource{r2}, map[string]Image{"Image1": a2}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := groupResources(tt.deployments, tt.cronjobs)
			if !reflect.DeepEqual(got, tt.result) {
				t.Errorf("groupResources got\n%+v\n, want\n%+v", got, tt.result)
			}
		})
	}
}
