package main

import (
	"io/ioutil"
	"log"
	"os"

	"github.com/jessevdk/go-flags"
	"gopkg.in/yaml.v2"
)

// Options is global options
type Options struct {
	Verbose  bool   `short:"v" long:"verbose" description:"Be noisy"`
	Webhooks string `short:"w" long:"webhooks" description:"Webhooks file"`
}

var options Options

var parser = flags.NewParser(&options, flags.Default)

// Verbose is the verbose log
var Verbose = log.New(os.Stderr, "k8ecr: ", log.Lshortfile)

// WebhookMap is a mapping of images to webhooks
type WebhookMap map[string]string

// Webhooks is the configured list of hooks for images
var Webhooks = make(WebhookMap)

func processOptions() {
	if !options.Verbose {
		Verbose.SetFlags(0)
		Verbose.SetOutput(ioutil.Discard)
	}
	if options.Webhooks != "" {
		Verbose.Println("Configuring webhooks from file", options.Webhooks)
		yamlFile, err := ioutil.ReadFile(options.Webhooks)
		if err != nil {
			log.Fatal(err)
		}
		err = yaml.Unmarshal(yamlFile, Webhooks)
		if err != nil {
			log.Fatal(err)
		}
		Verbose.Println(len(Webhooks), "webhooks configured")
	}
}

func main() {
	_, err := parser.Parse()
	if err != nil {
		if flagsErr, ok := err.(*flags.Error); ok && flagsErr.Type == flags.ErrHelp {
			os.Exit(0)
		} else {
			os.Exit(1)
		}
	}
}
