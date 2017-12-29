package main

import (
	"fmt"
	"io/ioutil"
	"os/exec"
	"strings"

	"gopkg.in/yaml.v2"
)

type Link struct {
	Profile string
	Context string
}

type ConfigCommand struct {
}

var configCommand ConfigCommand

func setContext(profile string, context string) {
	fmt.Printf("Setting profile to %s for context %s\n", profile, context)
	var l []Link
	doc, err := ioutil.ReadFile("k8ecr.yaml")
	if err == nil {
		yaml.Unmarshal(doc, &l)
	}
	k := append(l, Link{profile, context})
	d, err := yaml.Marshal(&k)
	if err == nil {
		ioutil.WriteFile("k8ecr.yaml", d, 0644)
	} else {
		fmt.Println(err.Error())
	}
}

func (x *ConfigCommand) Execute(args []string) error {
	cmd := exec.Command("kubectl", "config", "current-context")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return err
	}
	context := strings.TrimSpace(string(output))
	setContext(args[0], context)
	return nil
}

func init() {
	parser.AddCommand("config", "Configure", "Link an AWS profile to a kubectl context", &configCommand)
}
