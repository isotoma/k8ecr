GO ?= go
BINDIR := $(CURDIR)/bin
GOFLAGS :=

build:
	CGO_ENABLED=0 GOBIN=$(BINDIR) $(GO) install $(GOFLAGS) github.com/isotoma/k8ecr/cmd/...
	strip bin/k8ecr
	
vendor:
	dep ensure	

test:
	go test -timeout 30s github.com/isotoma/k8ecr/pkg/...