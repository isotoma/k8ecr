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
	"github.com/ashwanthkumar/slack-go-webhook"
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

func getTagsForRepositoryPage(svc *ecr.ECR, repository string, tagList []string, nextToken *string) ([]string, *string, error) {
	response, err := svc.DescribeImages(&ecr.DescribeImagesInput{
		RepositoryName: &repository,
		NextToken:      nextToken,
	})
	if err != nil {
		return tagList, nil, err
	}
	for _, i := range response.ImageDetails {
		for _, t := range i.ImageTags {
			if *t != "latest" {
				tagList = append(tagList, *t)
			}
		}
	}
	return tagList, response.NextToken, nil
}

func getTagsForRepository(svc *ecr.ECR, repository string) ([]string, error) {
	tagList := make([]string, 0)
	tagList, nextToken, err := getTagsForRepositoryPage(svc, repository, tagList, nil)
	if err != nil {
		return nil, err
	}
	for nextToken != nil {
		tagList, nextToken, err = getTagsForRepositoryPage(svc, repository, tagList, nextToken)
		if err != nil {
			return nil, err
		}

	}
	return tagList, nil
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

func getClientSet() (*kubernetes.Clientset, error) {
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
	return kubernetes.NewForConfig(config)
}

type ResourceType int

const (
	DeploymentResource ResourceType = 0
	CronjobResource    ResourceType = 1
)

// Option is an option for upgrade
type Option struct {
	Type      ResourceType
	Name      string
	App       string
	Container string
	Current   Image
	Latest    string
}

// TypeName returns the type as a string
func (o *Option) TypeName() string {
	switch {
	case o.Type == DeploymentResource:
		return "Deployment"
	case o.Type == CronjobResource:
		return "Cronjob"
	default:
		return "ERROR"
	}
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
					Type:      DeploymentResource,
					Name:      deploymentName,
					Container: containerName,
					Current:   newImage(c.Image),
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
			Verbose.Printf("Ignoring %s %s/%s %s === %s\n", o.TypeName(), o.Name, o.Container, o.Current.Version, latest)
		}
	}
	return choices
}

func getUpgradeChoices(namespace string) (OptionList, error) {
	clientset, err := getClientSet()
	if err != nil {
		return nil, err
	}
	client := clientset.AppsV1beta1().Deployments(namespace)
	deployments, err := client.List(metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("Error listing deployments: %s", err)
	}
	deployments := getDeploymentContainerVersions(deployments)
	images, err := getLatestImage()
	if err != nil {
		return nil, err
	}
	choices := getUpgradeOptions(&current, images)
	return choices, nil
}

func displayChoices(choices OptionList) {
	for i, c := range choices {
		fmt.Printf("%d> %s %s/%s %s -> %s \n", i, c.TypeName(), c.Name, c.Container, c.Current.Version, c.Latest)
	}
}

func updateDeployment(client typed.DeploymentInterface, choice Option) error {
	fmt.Printf("Updating %s %s/%s to %s\n", choice.TypeName(), choice.Name, choice.Container, choice.Latest)
	deployment, err := client.Get(choice.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	for i, container := range deployment.Spec.Template.Spec.Containers {
		if container.Name == choice.Container {
			Verbose.Printf("%s/%s Image was %s\n", choice.Name, choice.Container, deployment.Spec.Template.Spec.Containers[0].Image)
			newImage := fmt.Sprintf("%s/%s:%s", choice.Current.Registry, choice.Current.Repo, choice.Latest)
			Verbose.Printf("%s/%s new Image %s\n", choice.Name, choice.Container, newImage)
			deployment.Spec.Template.Spec.Containers[i].Image = newImage
			_, err = client.Update(deployment)
			if err != nil {
				return err
			}
			if hook, ok := Webhooks[choice.Current.Repo]; ok {
				payload := slack.Payload{
					Text: fmt.Sprintf("%s updated to %s", choice.Name, choice.Latest),
				}
				err := slack.Send(hook, "", payload)
				if len(err) > 0 {
					fmt.Printf("error: %s\n", err)
				}
			}

		}
	}
	return nil
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
	choices, err := getUpgradeChoices(namespace)
	if err != nil {
		return err
	}
	clientset, err := getClientSet()
	if err != nil {
		return err
	}
	client := clientset.AppsV1beta1().Deployments(namespace)
	if image != "" {
		for _, choice := range choices {
			if image == "-" {
				fmt.Println("Autodeploying to", choice.Name)
				err := updateDeployment(client, choice)
				if err != nil {
					fmt.Println("Error updating", choice.Name)
				}
			} else {
				if choice.Current.Repo == image {
					fmt.Println("Autodeploying to", choice.Name)
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
