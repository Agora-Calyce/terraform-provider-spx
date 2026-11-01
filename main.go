package main

import (
	"context"
	"flag"
	"log"

	"github.com/Agora-Calyce/terraform-provider-spx/internal/provider"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
)

// Run the docs generation tool, check its pride for more details:
// https://github.com/hashicorp/terraform-plugin-docs
//go:generate go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs generate --provider-name spx

var (
	// version is set at build time via -ldflags "-X main.version=..." (GoReleaser, Makefile)
	version string = "dev"
)

func main() {
	var debug bool

	flag.BoolVar(&debug, "debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.Parse()

	opts := providerserver.ServeOpts{
		Address: "registry.terraform.io/agora-calyce/spx",
		Debug:   debug,
	}

	err := providerserver.Serve(context.Background(), provider.New(version), opts)

	if err != nil {
		log.Fatal(err.Error())
	}
}
