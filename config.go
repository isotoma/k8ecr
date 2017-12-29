package main

import (
	"fmt"
)

type ConfigCommand struct {
}

var configCommand ConfigCommand

func (x *ConfigCommand) Execute(args []string) error {
	fmt.Printf("Setting profile to %s\n", args[0])
	return nil
}

func init() {
	parser.AddCommand("config", "Configure", "Link an AWS profile to a kubectl context", &configCommand)
}
