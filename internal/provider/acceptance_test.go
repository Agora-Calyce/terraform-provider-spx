package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

const (
	// Default values can be set here for the tests
	testOrgaId    = "test-orga"
	testProjectId = "test-project"
	testAz        = "fr-par-1"
)

var (
	// testAccProtoV6ProviderFactories defines the factories for the provider under test.
	testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
		"spx": func() (tfprotov6.ProviderServer, error) {
			return providerserver.NewProtocol6(New("test")())(), nil
		},
	}
)

func testAccPreCheck(t *testing.T) {
	// The required environment variables can be checked here
	// Acceptance tests need SPX_API_TOKEN and SPX_API_ENDPOINT
}
