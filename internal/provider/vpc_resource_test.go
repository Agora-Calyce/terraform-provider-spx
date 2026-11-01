package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestVPCResourceModel(t *testing.T) {
	model := vpcResourceModel{
		ID:        types.StringValue("vpc-123"),
		Name:      types.StringValue("test-vpc"),
		Az:        types.StringValue("fr-par-1"),
		OrgaId:    types.StringValue("orga-123"),
		ProjectId: types.StringValue("proj-123"),
	}

	if model.ID.ValueString() != "vpc-123" {
		t.Errorf("expected ID vpc-123, got %s", model.ID.ValueString())
	}
	if model.Name.ValueString() != "test-vpc" {
		t.Errorf("expected Name test-vpc, got %s", model.Name.ValueString())
	}
}

func TestAccVPCResource(t *testing.T) {
	resourceName := "spx_vpc.test"
	vpcName := "terraform-test-vpc"
	updatedVpcName := "terraform-test-vpc-updated"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Création et vérification
			{
				Config: testAccVPCResourceConfig(vpcName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", vpcName),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "orga_id", testOrgaId),
					resource.TestCheckResourceAttr(resourceName, "project_id", testProjectId),
					resource.TestCheckResourceAttr(resourceName, "az", testAz),
				),
			},
			// Mise à jour et vérification
			{
				Config: testAccVPCResourceConfig(updatedVpcName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", updatedVpcName),
				),
			},
			// Importation
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccVPCResourceConfig(name string) string {
	return fmt.Sprintf(`
provider "spx" {
  orga_id    = "%s"
  project_id = "%s"
  az         = "%s"
}

resource "spx_vpc" "test" {
  name = "%s"
}
`, testOrgaId, testProjectId, testAz, name)
}
