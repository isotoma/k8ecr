package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go/service/ecr"

	"github.com/Masterminds/semver"
	"k8s.io/api/apps/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	typed "k8s.io/client-go/kubernetes/typed/apps/v1beta1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// DeployCommand has no options
type DeployCommand struct{}

var deployCommand DeployCommand

// Sorts by semantic version, if there are any, otherwise resorts to a string sort
func latestVersion(versions []string) string {
	if len(versions) == 0 {
		return ""
	}
	vs := make([]*semver.Version, 0)
	for _, r := range versions {
		v, err := semver.NewVersion(r)
		if err == nil {
			vs = append(vs, v)
		}
	}
	if len(vs) > 0 {
		sort.Sort(semver.Collection(vs))
		return vs[len(vs)-1].Original()
	}
	sort.Strings(versions)
	return versions[len(versions)-1]
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}

func getAllRepositories(svc *ecr.ECR) ([]string, error) {
	response, err := svc.DescribeRepositories(&ecr.DescribeRepositoriesInput{})
	if err != nil {
		return nil, err
	}
	repositories := make([]string, len(response.Repositories))
	for i, r := range response.Repositories {
		repositories[i] = *r.RepositoryName
	}
	return repositories, nil
}

func getTagsForRepository(svc *ecr.ECR, repository string) ([]string, error) {
	response, err := svc.DescribeImages(&ecr.DescribeImagesInput{
		RepositoryName: &repository,
	})
	if err != nil {
		return nil, err
	}
	tagList := make([]string, 0)
	for _, i := range response.ImageDetails {
		for _, t := range i.ImageTags {
			if *t != "latest" {
				tagList = append(tagList, *t)
			}
		}
	}
	return tagList, nil
}

func getLatestImage() (map[string]string, error) {
	svc := ecr.New(createSession())
	repositories, err := getAllRepositories(svc)
	if err != nil {
		return nil, err
	}
	l := make(map[string]string)
	for _, r := range repositories {
		all, err := getTagsForRepository(svc, r)
		if err != nil {
			return nil, err
		}
		l[r] = latestVersion(all)
	}
	return l, nil
}

func getClusterConfig(kubeconfig string) (*rest.Config, error) {
	if kubeconfig == "" {
		// We are running in-cluster
		config, err := rest.InClusterConfig()
		if err != nil {
			return nil, err
		}
		return config, nil
	} else {
		config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, err
		}
		return config, nil
	}
}

func getDeploymentsClient(namespace string) (typed.DeploymentInterface, error) {
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		home := homeDir()
		defpath := filepath.Join(home, ".kube", "config")
		if _, err := os.Stat(defpath); err == nil {
			kubeconfig = defpath
		}
	}
	config, err := getClusterConfig(kubeconfig)
	if err != nil {
		return nil, err
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return clientset.AppsV1beta1().Deployments(namespace), nil

}

// Image represents an image currently deployed to a container
type Image struct {
	Original string
	Repo     string
	Registry string
	Version  string
}

func newImage(url string) Image {
	p1 := strings.Split(url, "/")
	p2 := strings.Split(p1[1], ":")
	version := "latest"
	if len(p2) == 2 {
		version = p2[1]
	}
	return Image{
		Original: url,
		Registry: p1[0],
		Repo:     p2[0],
		Version:  version,
	}
}

// Option is an option for upgrade
type Option struct {
	Deployment string
	Container  string
	Current    Image
	Latest     string
}

// OptionList is a list of Options
type OptionList []Option

func getDeploymentContainerVersions(deployments *v1beta1.DeploymentList) OptionList {
	images := make(OptionList, 0)
	for _, d := range deployments.Items {
		if d.Name != "" {
			deploymentName := d.Name
			for _, c := range d.Spec.Template.Spec.Containers {
				containerName := c.Name
				images = append(images, Option{
					Deployment: deploymentName,
					Container:  containerName,
					Current:    newImage(c.Image),
				})
			}
		}
	}
	return images
}

func getUpgradeOptions(current *OptionList, images map[string]string) OptionList {
	choices := make(OptionList, 0)
	for _, o := range *current {
		latest := images[o.Current.Repo]
		o.Latest = latest
		if latest != "" && latest != o.Current.Version {
			choices = append(choices, o)
		} else {
			Verbose.Printf("Ignoring %s/%s %s === %s\n", o.Deployment, o.Container, o.Current.Version, latest)
		}
	}
	return choices
}

func getUpgradeChoices(client typed.DeploymentInterface) (OptionList, error) {
	deployments, err := client.List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	current := getDeploymentContainerVersions(deployments)
	images, err := getLatestImage()
	if err != nil {
		return nil, err
	}
	choices := getUpgradeOptions(&current, images)
	return choices, nil
}

func displayChoices(choices OptionList) {
	for i, c := range choices {
		fmt.Printf("%d> %s/%s %s -> %s \n", i, c.Deployment, c.Container, c.Current.Version, c.Latest)
	}
}

func updateDeployment(client typed.DeploymentInterface, choice Option) {
	fmt.Printf("Updating %s/%s to %s\n", choice.Deployment, choice.Container, choice.Latest)
	deployment, err := client.Get(choice.Deployment, metav1.GetOptions{})
	if err != nil {
		panic(err)
	}
	for i, container := range deployment.Spec.Template.Spec.Containers {
		if container.Name == choice.Container {
			Verbose.Printf("%s/%s Image was %s\n", choice.Deployment, choice.Container, deployment.Spec.Template.Spec.Containers[0].Image)
			newImage := fmt.Sprintf("%s/%s:%s", choice.Current.Registry, choice.Current.Repo, choice.Latest)
			Verbose.Printf("%s/%s new Image %s\n", choice.Deployment, choice.Container, newImage)
			deployment.Spec.Template.Spec.Containers[i].Image = newImage
			_, err = client.Update(deployment)
			if err != nil {
				panic(err)
			}

		}
	}
}

func getChosen(choices OptionList) OptionList {
	fmt.Print("> ")
	reader := bufio.NewReader(os.Stdin)
	text, err := reader.ReadString('\n')
	if err != nil {
		panic(err)
	}
	rv := make(OptionList, 0)
	chosen := strings.Split(text, ",")
	for _, c := range chosen {
		i, err := strconv.ParseInt(strings.TrimSpace(c), 0, 64)
		if err != nil {
			panic(err)
		}
		rv = append(rv, choices[i])
	}
	return rv
}

// Execute the deploy command
func (x *DeployCommand) Execute(args []string) error {
	processOptions()
	if len(args) != 1 && len(args) != 2 {
		return errors.New("Usage: k8ecr deploy NAMESPACE [IMAGE]")
	}
	namespace := args[0]
	image := ""
	if len(args) == 2 {
		image = args[1]
	}
	client, err := getDeploymentsClient(namespace)
	if err != nil {
		return err
	}
	choices, err := getUpgradeChoices(client)
	if err != nil {
		return err
	}
	if image != "" {
		for _, choice := range choices {
			if image == "-" {
				fmt.Println("Autodeploying to", choice.Deployment)
				updateDeployment(client, choice)
			} else {
				if choice.Current.Repo == image {
					fmt.Println("Autodeploying to", choice.Deployment)
					updateDeployment(client, choice)
				}
			}
		}
	} else {
		if len(choices) == 0 {
			fmt.Println("No containers require upgrade")
		} else {
			displayChoices(choices)
			chosen := getChosen(choices)
			for _, c := range chosen {
				updateDeployment(client, c)
			}
		}
	}
	return nil
}

func init() {
	parser.AddCommand(
		"deploy",
		"Deploy",
		"Deploy an image to your cluster",
		&deployCommand)
}
