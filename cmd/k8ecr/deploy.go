package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/gosuri/uitable"
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
	table := uitable.New()
	table.MaxColWidth = 120
	table.AddRow("APP", "IMAGE", "LATEST", "OLD VERSIONS", "DEPLOYMENTS", "CRONJOBS")
	for _, app := range mgr.Apps {
		for _, image := range app.GetImages() {
			if image.NeedsUpdate {
				table.AddRow(app.Name, image.ImageID.Repo, image.UpdateTo, strings.Join(image.Versions(), ", "), len(image.Deployments), len(image.Cronjobs))
			}
		}
	}
	fmt.Println(table)
	var input string
	fmt.Print("app? > ")
	fmt.Scanln(&input)
	app, ok := mgr.Apps[input]
	if ok {
		for _, image := range app.GetImages() {
			if image.NeedsUpdate {
				return mgr.Upgrade(&image)
			}
			fmt.Printf("Does not require update.\n")
			return nil
		}
	}
	fmt.Printf("App not known\n")
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
