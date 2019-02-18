GO ?= go
BINDIR := $(CURDIR)/bin
GOFLAGS :=

build:
	CGO_ENABLED=0 GOBIN=$(BINDIR) $(GO) install $(GOFLAGS) github.com/isotoma/k8ecr/cmd/...

vendor:
	dep ensure	