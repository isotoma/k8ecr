package main

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/service/ecr"
)

// Image represents an image currently deployed to a container
type Image struct {
	Original string
	Repo     string
	Registry string
	Version  string
}

func newImage(url string) Image {
	p1 := strings.Split(url, "/")
	registry := p1[0]
	var p2 []string
	switch {
	case len(p1) == 1:
		p2 = strings.Split(p1[0], ":")
	case len(p1) == 2:
		p2 = strings.Split(p1[1], ":")
	default:
		panic(fmt.Errorf("Unexpected number of / in image"))
	}
	repo := p2[0]
	version := "latest"
	if len(p2) == 2 {
		version = p2[1]
	}
	return Image{
		Original: url,
		Registry: registry,
		Repo:     repo,
		Version:  version,
	}
}

func getLatestImage() (map[string]string, error) {
	svc := ecr.New(createSession())
	repositories, err := getAllRepositories(svc)
	if err != nil {
		return nil, err
	}
	l := make(map[string]string)
	for _, r := range repositories {
		all, err := getTagsForRepository(svc, r)
		if err != nil {
			return nil, err
		}
		l[r] = latestVersion(all)
	}
	return l, nil
}
