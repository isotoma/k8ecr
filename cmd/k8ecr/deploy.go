package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/isotoma/k8ecr/pkg/ecr"
	"github.com/isotoma/k8ecr/pkg/imagemanager"
)

// DeployCommand has no options
type DeployCommand struct{}

var deployCommand DeployCommand

func filter(registry *ecr.Registry, mgr *imagemanager.ImageManager) error {
	for _, repo := range registry.GetRepositories() {
		parts := strings.Split(repo.URI, "/")
		fmt.Printf("%s %s %s\n", parts[0], parts[1], repo.LatestTag)
		mgr.SetLatest(parts[0], parts[1], repo.LatestTag)
	}
	return nil
}

func autodeploy(mgr *imagemanager.ImageManager) error {
	return nil
}

func chooser(mgr *imagemanager.ImageManager) error {
	cyan := color.New(color.FgCyan).Add(color.Underline).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	for _, app := range mgr.Apps {
		for _, image := range app.GetImages() {
			if image.NeedsUpdate {
				fmt.Printf("%-12s:%-30s [%-16s] <- [%-16s] (%s deployments and %s cronjobs) \n",
					green(app.Name),
					cyan(image.ImageID.Repo),
					red(image.UpdateTo),
					green(strings.Join(image.Versions(), ", ")),
					yellow(len(image.Deployments)), yellow(len(image.Cronjobs)))
			}
		}
	}
	var input string
	fmt.Print("image? > ")
	fmt.Scanln(&input)
	for _, image := range mgr.GetImages() {
		if image.ImageID.Repo == input {
			if image.NeedsUpdate {
				return mgr.Upgrade(&image)
			}
			fmt.Printf("Does not require update.\n")
			return nil
		}
	}
	fmt.Printf("Image not known\n")
	return nil
}

func deploy(namespace, image string) error {
	registry := ecr.NewRegistry()
	if err := registry.FetchAll(); err != nil {
		return err
	}
	imagemgr, err := imagemanager.NewImageManager(namespace)
	filter(registry, imagemgr)
	if err != nil {
		return err
	}

	if image == "-" {
		// Autodeploy
		return autodeploy(imagemgr)
	}
	return chooser(imagemgr)
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
	return deploy(namespace, image)
}

func init() {
	parser.AddCommand(
		"deploy",
		"Deploy",
		"Deploy an image to your cluster",
		&deployCommand)
}
