package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

const (
	// On peut définir des valeurs par défaut pour les tests
	testOrgaId    = "test-orga"
	testProjectId = "test-project"
	testAz        = "fr-par-1"
)

var (
	// testAccProtoV6ProviderFactories définit les factories pour le provider en test.
	testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
		"spx": func() (tfprotov6.ProviderServer, error) {
			return providerserver.NewProtocol6(New("test")())(), nil
		},
	}
)

func testAccPreCheck(t *testing.T) {
	// Ici on peut vérifier que les variables d'environnement nécessaires sont présentes
	// Pour les tests d'acceptation, on a besoin de SPX_API_TOKEN et SPX_API_ENDPOINT
}
