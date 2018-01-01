package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go/service/ecr"

	"github.com/Masterminds/semver"
	"k8s.io/api/apps/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

type DeployCommand struct{}

var deployCommand DeployCommand

func latestVersion(versions []string) (string, error) {
	vs := make([]*semver.Version, len(versions))
	for i, r := range versions {
		v, err := semver.NewVersion(r)
		if err != nil {
			return "", err
		}
		vs[i] = v
	}
	sort.Sort(semver.Collection(vs))
	return vs[len(vs)-1].Original(), nil
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

func getLatestImages() (map[string][]string, error) {
	svc := ecr.New(createSession())
	repositories, err := getAllRepositories(svc)
	if err != nil {
		return nil, err
	}
	imageList := make(map[string][]string)
	for _, r := range repositories {
		imageList[r], err = getTagsForRepository(svc, r)
		if err != nil {
			return nil, err
		}
	}
	return imageList, nil
}

func getDeployments(namespace string) (*v1beta1.DeploymentList, error) {
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		home := homeDir()
		kubeconfig = filepath.Join(home, ".kube", "config")
	}
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, err
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	fmt.Println("Listing deployments for namespace", namespace)
	deployments, err := clientset.AppsV1beta1().Deployments(namespace).List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	fmt.Printf("%+v\n", deployments)
	return deployments, nil
}

type Image struct {
	Original string
	Image    string
	Repo     string
	Version  string
}

func (i Image) New(url string) {
	i.Original = url
	p1 := strings.Split(url, "/")
	i.Repo = p1[0]
	p2 := strings.Split(p1[1], ":")
	i.Image = p2[0]
	if len(p2) == 2 {
		i.Version = p2[1]
	} else {
		i.Version = "latest"
	}
}

func getDeploymentContainerVersions(deployments *v1beta1.DeploymentList) map[string]map[string][]Image {
	images := make(map[string]map[string][]Image)
	for _, d := range deployments.Items {
		if d.Name != "" {
			deploymentName := d.Name
			for _, c := range d.Spec.Template.Spec.Containers {
				containerName := c.Name
				image := Image{}
				image.New(c.Image)
				images[deploymentName][containerName] =
					append(images[deploymentName][containerName], image)
			}
		}
	}
	return images
}

func (x *DeployCommand) Execute(args []string) error {
	namespace := args[0]
	deployments, err := getDeployments(namespace)
	if err != nil {
		return err
	}
	fmt.Println("Getting container versions")
	current := getDeploymentContainerVersions(deployments)
	fmt.Printf("%+v\n", current)
	return nil
}

func init() {
	parser.AddCommand(
		"deploy",
		"Deploy",
		"Deploy an image to your cluster",
		&deployCommand)
}
