package main

import (
	"errors"

	"github.com/isotoma/k8ecr/pkg/ecr"
)

// PushCommand is the push command
type PushCommand struct{}

var pushCommand PushCommand

// Execute the push command
func (x *PushCommand) Execute(args []string) error {
	if len(args) < 2 {
		return errors.New("push REPOSITORY VERSION")
	}
	registry := ecr.NewRegistry()
	return registry.PushRepository(args[0], args[1:])

}

func init() {
	parser.AddCommand("push",
		"Push",
		"Push an image to ECR",
		&pushCommand)
}
