package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os/exec"
	"strings"

	"gopkg.in/yaml.v2"
)

// ConfigCommand takes no switches
type ConfigCommand struct {
}

var configCommand ConfigCommand

func getContext() string {
	cmd := exec.Command("kubectl", "config", "current-context")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return ""
	}
	context := strings.TrimSpace(string(output))
	return context
}

func getProfile() string {
	context := getContext()
	config := readConfig()
	return config[context]
}

func readConfig() map[string]string {
	var l = make(map[string]string)
	doc, err := ioutil.ReadFile("k8ecr.yaml")
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
		ioutil.WriteFile("k8ecr.yaml", d, 0644)
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
	parser.AddCommand("config", "Configure", "Link an AWS profile to a kubectl context", &configCommand)
}
