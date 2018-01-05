package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

// ConfigCommand takes no switches
type ConfigCommand struct {
}

var configCommand ConfigCommand

func readConfig() map[string]string {
	var l = make(map[string]string)
	home := homeDir()
	config := filepath.Join(home, ".k8ecr.yaml")
	doc, err := ioutil.ReadFile(config)
	if err == nil {
		yaml.Unmarshal(doc, &l)
	}
	return l
}

func setProfile(context string, profile string) {
	fmt.Printf("Setting profile to %s for context %s\n", profile, context)
	l := readConfig()
	l[context] = profile
	d, err := yaml.Marshal(&l)
	if err == nil {
		home := homeDir()
		config := filepath.Join(home, ".k8ecr.yaml")
		ioutil.WriteFile(config, d, 0644)
	} else {
		fmt.Println(err.Error())
	}
}

// Execute Config command
func (x *ConfigCommand) Execute(args []string) error {
	context := getContext()
	if context == "" {
		return errors.New("Unable to read kubectl context")
	}
	if len(args) == 0 {
		fmt.Println(getProfile())
	} else {
		setProfile(context, args[0])
	}
	return nil
}

func init() {
	parser.AddCommand("config",
		"Configure",
		"Link an AWS profile to a kubectl context",
		&configCommand)
}
