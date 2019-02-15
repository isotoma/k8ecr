package apps

import (
	"fmt"
	"strings"
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
