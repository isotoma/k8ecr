package main

import (
	"io/ioutil"
	"log"
	"os"

	"github.com/jessevdk/go-flags"
)

// Options is global options
type Options struct {
	Verbose bool `short:"v" long:"verbose" description:"Be noisy"`
}

var options Options

var parser = flags.NewParser(&options, flags.Default)

// Verbose is the verbose log
var Verbose = log.New(os.Stderr, "k8ecr: ", log.Lshortfile)

func processOptions() {
	if !options.Verbose {
		Verbose.SetFlags(0)
		Verbose.SetOutput(ioutil.Discard)
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
