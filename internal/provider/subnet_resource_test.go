package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestSubnetResourceModel(t *testing.T) {
	model := subnetResourceModel{
		ID:           types.StringValue("sub-123"),
		Name:         types.StringValue("test-subnet"),
		VPCID:        types.StringValue("vpc-123"),
		IPv4:         types.StringValue("192.168.1.0/24"),
		Private:      types.BoolValue(true),
		Protocol:     types.StringValue("IPv4"),
		NatGateway:   types.BoolValue(true),
		NatGatewayIp: types.StringValue("1.2.3.4"),
	}

	if model.ID.ValueString() != "sub-123" {
		t.Errorf("expected ID sub-123, got %s", model.ID.ValueString())
	}
	if !model.Private.ValueBool() {
		t.Error("expected Private to be true")
	}
	if model.NatGatewayIp.ValueString() != "1.2.3.4" {
		t.Errorf("expected NatGatewayIp 1.2.3.4, got %s", model.NatGatewayIp.ValueString())
	}
}

func TestAccSubnetResource(t *testing.T) {
	resourceName := "spx_subnet.test"
	vpcName := "terraform-test-vpc-for-subnet"
	subnetName := "terraform-test-subnet"
	updatedSubnetName := "terraform-test-subnet-updated"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Création VPC + Subnet
			{
				Config: testAccSubnetResourceConfig(vpcName, subnetName, "10.0.1.0/24", true),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", subnetName),
					resource.TestCheckResourceAttr(resourceName, "ipv4", "10.0.1.0/24"),
					resource.TestCheckResourceAttr(resourceName, "private", "true"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "vpc_id"),
				),
			},
			// Mise à jour Subnet
			{
				Config: testAccSubnetResourceConfig(vpcName, updatedSubnetName, "10.0.1.0/24", true),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", updatedSubnetName),
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

func testAccSubnetResourceConfig(vpcName, subnetName, ipv4 string, private bool) string {
	return fmt.Sprintf(`
provider "spx" {
  orga_id    = "%s"
  project_id = "%s"
  az         = "%s"
}

resource "spx_vpc" "test" {
  name = "%s"
}

resource "spx_subnet" "test" {
  name       = "%s"
  vpc_id     = spx_vpc.test.id
  ipv4       = "%s"
  private    = %t
  protocol   = "IPv4"
  nat_gateway = false
}
`, testOrgaId, testProjectId, testAz, vpcName, subnetName, ipv4, private)
}
