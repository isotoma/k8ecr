package main

import (
	"fmt"

	"github.com/isotoma/k8ecr/pkg/ecr"
)

// LatestCommand is an latest command
type LatestCommand struct{}

var latestCommand LatestCommand

// Execute the latest command
func (*LatestCommand) Execute(args []string) error {
	registry := ecr.NewRegistry()
	if err := registry.FetchAll(); err != nil {
		return err
	}
	for _, repo := range registry.GetRepositories() {
		fmt.Printf("%-30s %s\n", repo.Name, repo.LatestTag)
	}
	return nil
}

func init() {
	parser.AddCommand("latest", "Latest", "List the latest tags available", &latestCommand)
}
