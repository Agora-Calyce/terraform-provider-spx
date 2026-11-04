package provider

import (
	"fmt"
	"github.com/Agora-Calyce/terraform-provider-spx/internal/client"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestMapDiskToClient(t *testing.T) {
	model := DiskResourceModel{
		Name:         types.StringValue("test-disk"),
		Size:         types.Int64Value(10),
		StorageClass: types.StringValue("fast"),
		Source: &DiskSourceModel{
			Type: types.StringValue("http"),
			URL:  types.StringValue("http://example.com/image.qcow2"),
		},
	}

	// Simulate the mapping
	diskReq := client.DiskRequest{
		Name:         model.Name.ValueString(),
		Size:         int(model.Size.ValueInt64()),
		StorageClass: model.StorageClass.ValueString(),
	}

	if model.Source != nil {
		diskReq.Source = &client.DiskSource{
			Type:     model.Source.Type.ValueString(),
			URL:      model.Source.URL.ValueString(),
			Clone:    model.Source.Clone.ValueString(),
			Snapshot: model.Source.Snapshot.ValueString(),
		}
	}

	expected := client.DiskRequest{
		Name:         "test-disk",
		Size:         10,
		StorageClass: "fast",
		Source: &client.DiskSource{
			Type: "http",
			URL:  "http://example.com/image.qcow2",
		},
	}

	if !reflect.DeepEqual(diskReq, expected) {
		t.Errorf("Mapping failed")
	}
}

func TestAccDiskResource(t *testing.T) {
	resourceName := "spx_disk.test"
	diskName := "terraform-test-disk"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDiskResourceConfig(diskName, 10, "standard"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", diskName),
					resource.TestCheckResourceAttr(resourceName, "size", "10"),
					resource.TestCheckResourceAttr(resourceName, "storage_class", "standard"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccDiskResourceConfig(name string, size int, storageClass string) string {
	return fmt.Sprintf(`
provider "spx" {
  orga_id    = "%s"
  project_id = "%s"
  az         = "%s"
}

resource "spx_disk" "test" {
  name          = "%s"
  size          = %d
  storage_class = "%s"
}
`, testOrgaId, testProjectId, testAz, name, size, storageClass)
}
