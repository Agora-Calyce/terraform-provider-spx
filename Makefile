BINARY=terraform-provider-spx
VERSION=1.0.0
HOSTNAME=registry.terraform.io
NAMESPACE=agora-calyce
NAME=spx

GOOS=$(shell go env GOOS)
GOARCH=$(shell go env GOARCH)

TERRAFORM_PLUGINS_DIR=$(HOME)/.terraform.d/plugins/$(HOSTNAME)/$(NAMESPACE)/$(NAME)/$(VERSION)/$(GOOS)_$(GOARCH)

default: build

build:
	go build -ldflags "-X main.version=$(VERSION)" -o $(BINARY)
	$(MAKE) install

install:
	go build -ldflags "-X main.version=$(VERSION)" -o $(BINARY)
	mkdir -p $(TERRAFORM_PLUGINS_DIR)
	cp $(BINARY) $(TERRAFORM_PLUGINS_DIR)

test:
	go test -v ./...

clean:
	rm -f $(BINARY)

docs:
	~/go/bin/tfplugindocs generate --provider-name spx

.PHONY: default build install test clean docs
