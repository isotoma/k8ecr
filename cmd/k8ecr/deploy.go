package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/gosuri/uitable"
	"github.com/isotoma/k8ecr/pkg/apps"
	"github.com/isotoma/k8ecr/pkg/ecr"
	"github.com/isotoma/k8ecr/pkg/resources"
)

// DeployCommand has no options
type DeployCommand struct{}

var deployCommand DeployCommand

func filter(registry *ecr.Registry, mgr *apps.AppManager) error {
	for _, repo := range registry.GetRepositories() {
		parts := strings.Split(repo.URI, "/")
		mgr.SetLatest(parts[0], parts[1], repo.LatestTag)
	}
	return nil
}

func autodeploy(mgr *apps.AppManager) error {
	return nil
}

func chooser(mgr *apps.AppManager) error {
	table := uitable.New()
	table.MaxColWidth = 120
	cols := []interface{}{"APP", "IMAGE", "LATEST", "OLD VERSIONS"}
	kinds := make([]string, 0)
	for kind := range mgr.Managers {
		kinds = append(kinds, kind)
		cols = append(cols, fmt.Sprintf("%sS", strings.ToUpper(kind)))
	}
	table.AddRow(cols...)
	for _, app := range mgr.Apps {
		for _, image := range app.GetContainers() {
			if image.NeedsUpdate {
				row := []interface{}{app.Name, image.ImageID.Repo, image.UpdateTo, strings.Join(image.Versions(), ", ")}
				for _, kind := range kinds {
					row = append(row, len(image.Containers[kind]))
				}
				table.AddRow(row...)
			}
		}
	}
	fmt.Println(table)
	var input string
	fmt.Print("app? > ")
	fmt.Scanln(&input)
	app, ok := mgr.Apps[input]
	if ok {
		for _, image := range app.GetContainers() {
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
	imagemgr, err := apps.NewAppManager(namespace)
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
	resources.Register()
	parser.AddCommand(
		"deploy",
		"Deploy",
		"Deploy an image to your cluster",
		&deployCommand)
}
