package main

import (
	"errors"
	"fmt"

	"github.com/isotoma/k8ecr/pkg/ecr"
)

// CreateCommand is a create command
type CreateCommand struct{}

var createCommand CreateCommand

// Execute the create repository command
func (x *CreateCommand) Execute(args []string) error {
	if len(args) == 0 {
		return errors.New("No repository name specified")
	}
	registry := ecr.NewRegistry()
	repository, err := registry.CreateRepository(args[0])
	if err != nil {
		return err
	}
	fmt.Printf("Created repository %s\n", *repository.RepositoryUri)
	return nil
}

func init() {
	parser.AddCommand("create",
		"Create",
		"Create an ECR repository and grant read permissions to your cluster",
		&createCommand)
}
