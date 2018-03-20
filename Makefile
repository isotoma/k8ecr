k8ecr: config.go create.go deploy.go main.go push.go util.go
	CGO_ENABLED=0 go build
	strip k8ecr
