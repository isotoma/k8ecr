package apps

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// func updateDeployment(client typed.DeploymentInterface, choice Option) error {
// 	fmt.Printf("Updating %s %s/%s to %s\n", choice.TypeName(), choice.Name, choice.Container, choice.Latest)
// 	deployment, err := client.Get(choice.Name, metav1.GetOptions{})
// 	if err != nil {
// 		return err
// 	}
// 	for i, container := range deployment.Spec.Template.Spec.Containers {
// 		if container.Name == choice.Container {
// 			Verbose.Printf("%s/%s Image was %s\n", choice.Name, choice.Container, deployment.Spec.Template.Spec.Containers[0].Image)
// 			newImage := fmt.Sprintf("%s/%s:%s", choice.Current.Registry, choice.Current.Repo, choice.Latest)
// 			Verbose.Printf("%s/%s new Image %s\n", choice.Name, choice.Container, newImage)
// 			deployment.Spec.Template.Spec.Containers[i].Image = newImage
// 			_, err = client.Update(deployment)
// 			if err != nil {
// 				return err
// 			}
// 			if hook, ok := Webhooks[choice.Current.Repo]; ok {
// 				payload := slack.Payload{
// 					Text: fmt.Sprintf("%s updated to %s", choice.Name, choice.Latest),
// 				}
// 				err := slack.Send(hook, "", payload)
// 				if len(err) > 0 {
// 					fmt.Printf("error: %s\n", err)
// 				}
// 			}

// 		}
// 	}
// 	return nil
// }

// func getChosen(choices OptionList) OptionList {
// 	fmt.Print("> ")
// 	reader := bufio.NewReader(os.Stdin)
// 	text, err := reader.ReadString('\n')
// 	if err != nil {
// 		panic(err)
// 	}
// 	rv := make(OptionList, 0)
// 	chosen := strings.Split(text, ",")
// 	for _, c := range chosen {
// 		i, err := strconv.ParseInt(strings.TrimSpace(c), 0, 64)
// 		if err != nil {
// 			panic(err)
// 		}
// 		rv = append(rv, choices[i])
// 	}
// 	return rv
// }

// Container is a container found in a Deployment or Cronjob
type Container struct {
	Name    string
	Current Image
	Latest  string
}

// Resource is something that has an image
type Resource struct {
	Name       string
	App        string
	Containers []Container
}

// App is a grouping of deployments and cronjobs with the same app label
// in the same namespace
type App struct {
	Name        string
	Deployments []Resource
	Cronjobs    []Resource
}

// AppManager finds and updates applications
// and their deployments and cronjobs
type AppManager struct {
	clientset *kubernetes.Clientset
	Namespace string
	Apps      map[string]App
}

// Init initialises the application manager
func (a *AppManager) Init(namespace string) error {
	clientset, err := getClientSet()
	if err != nil {
		return err
	}
	a.clientset = clientset
	a.Namespace = namespace
	a.Apps = make(map[string]App)
	return a.Scan()
}

// Deploy the specified application to the specified version
func Deploy(app string, version string) error {
	return nil
}

func makeContainerList(spec []corev1.Container) []Container {
	containers := make([]Container, len(spec))
	for _, c := range spec {
		container := Container{
			Name:    c.Name,
			Current: newImage(c.Image),
		}
		containers = append(containers, container)
	}
	return containers
}

func (a *AppManager) scanDeployments() ([]Resource, error) {
	resources := make([]Resource, 0)
	client := a.clientset.AppsV1().Deployments(a.Namespace)
	response, err := client.List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for _, item := range response.Items {
		if item.Name != "" {
			resource := Resource{
				Name:       item.Name,
				App:        item.ObjectMeta.Labels["App"],
				Containers: makeContainerList(item.Spec.Template.Spec.Containers),
			}
			resources = append(resources, resource)
		}
	}
	return resources, nil
}

func (a *AppManager) scanCronjobs() ([]Resource, error) {
	resources := make([]Resource, 0)
	client := a.clientset.BatchV1beta1().CronJobs(a.Namespace)
	response, err := client.List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for _, item := range response.Items {
		if item.Name != "" {
			resource := Resource{
				Name:       item.Name,
				App:        item.ObjectMeta.Labels["App"],
				Containers: makeContainerList(item.Spec.JobTemplate.Spec.Template.Spec.Containers),
			}
			resources = append(resources, resource)
		}
	}
	return resources, nil
}

func groupResources(deployments []Resource, cronjobs []Resource) map[string]App {
	apps := make(map[string]App)
	for _, item := range deployments {
		app, ok := apps[item.App]
		if !ok {
			app = App{
				Name:        item.App,
				Deployments: make([]Resource, 0),
				Cronjobs:    make([]Resource, 0),
			}
		}
		app.Deployments = append(app.Deployments, item)
		apps[item.App] = app
	}
	for _, item := range cronjobs {
		app, ok := apps[item.App]
		if !ok {
			app = App{
				Name:        item.App,
				Deployments: make([]Resource, 0),
				Cronjobs:    make([]Resource, 0),
			}
		}
		app.Cronjobs = append(app.Cronjobs, item)
		apps[item.App] = app
	}
	return apps
}

// Scan the specified namespace in the cluster and find
// all the deployments and cronjobs
// then create applications
func (a *AppManager) Scan() error {
	deployments, err := a.scanDeployments()
	if err != nil {
		return err
	}
	cronjobs, err := a.scanCronjobs()
	if err != nil {
		return err
	}
	a.Apps = groupResources(deployments, cronjobs)
	return nil
}
