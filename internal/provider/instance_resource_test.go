package provider

import (
	"context"
	"fmt"
	"github.com/Agora-Calyce/terraform-provider-spx/internal/client"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestMapInstanceToClient(t *testing.T) {
	model := instanceResourceModel{
		Name:        types.StringValue("test-instance"),
		RunStrategy: types.StringValue("Always"),
		VMType:      types.StringValue("standard"),
		CPU:         types.Int32Value(2),
		Memory:      types.Int32Value(4),
		Network: []instanceNetworkModel{
			{
				Order:    types.Int32Value(0),
				SubnetID: types.StringValue("subnet-1"),
				Model:    types.StringValue("virtio"),
			},
		},
		Disks: []instanceDiskModel{
			{
				Order:  types.Int32Value(0),
				Bus:    types.StringValue("virtio"),
				DiskID: types.StringValue("disk-1"),
				Cdrom:  types.BoolValue(false),
			},
		},
		CloudInit: &instanceCloudInitModel{
			Custom: types.BoolValue(true),
			Bus:    types.StringValue("virtio"),
			Config: types.StringValue("user-data"),
		},
	}

	// Simulation de la logique de mapping dans Create
	instanceReq := client.InstanceRequest{
		Name:        model.Name.ValueString(),
		RunStrategy: model.RunStrategy.ValueString(),
		VMType:      model.VMType.ValueString(),
		CPU:         int(model.CPU.ValueInt32()),
		Memory:      int(model.Memory.ValueInt32()),
	}

	for _, nw := range model.Network {
		instanceReq.Network = append(instanceReq.Network, client.InstanceNetwork{
			Order:    int(nw.Order.ValueInt32()),
			SubnetID: nw.SubnetID.ValueString(),
			Model:    nw.Model.ValueString(),
		})
	}

	for _, d := range model.Disks {
		instanceReq.Disks = append(instanceReq.Disks, client.InstanceDisk{
			Order:  int(d.Order.ValueInt32()),
			Bus:    d.Bus.ValueString(),
			DiskID: d.DiskID.ValueString(),
			Cdrom:  d.Cdrom.ValueBool(),
		})
	}

	if model.CloudInit != nil {
		instanceReq.CloudInit = client.InstanceCloudInit{
			Custom: model.CloudInit.Custom.ValueBool(),
			Bus:    model.CloudInit.Bus.ValueString(),
			Config: model.CloudInit.Config.ValueString(),
		}
	}

	expected := client.InstanceRequest{
		Name:        "test-instance",
		RunStrategy: "Always",
		VMType:      "standard",
		CPU:         2,
		Memory:      4,
		Network: []client.InstanceNetwork{
			{Order: 0, SubnetID: "subnet-1", Model: "virtio"},
		},
		Disks: []client.InstanceDisk{
			{Order: 0, Bus: "virtio", DiskID: "disk-1", Cdrom: false},
		},
		CloudInit: client.InstanceCloudInit{
			Custom: true,
			Bus:    "virtio",
			Config: "user-data",
		},
	}

	if !reflect.DeepEqual(instanceReq, expected) {
		t.Errorf("Mapping to client failed. Expected %+v, got %+v", expected, instanceReq)
	}
}

func TestMapInstanceFromClient(t *testing.T) {
	ctx := context.Background()
	apiInstance := client.Instance{
		ID:          "inst-1",
		Name:        "test-instance",
		RunStrategy: "Always",
		VMType:      "standard",
		CPU:         2,
		Memory:      4,
		Network: []client.InstanceNetwork{
			{Order: 0, SubnetID: "subnet-1", Model: "virtio"},
		},
		Disks: []client.InstanceDisk{
			{Order: 0, Bus: "virtio", DiskID: "disk-1", Cdrom: false},
		},
		CloudInit: client.InstanceCloudInit{
			Custom: true,
			Bus:    "virtio",
			Config: "user-data",
		},
		SSHKeys: []string{"key-1"},
		Labels:  []string{"user.superphenix.net/env=prod"},
	}

	// Simulation de la logique de mapping dans Read
	var model instanceResourceModel
	model.ID = types.StringValue(apiInstance.ID)
	model.Name = types.StringValue(apiInstance.Name)
	model.RunStrategy = types.StringValue(apiInstance.RunStrategy)
	model.VMType = types.StringValue(apiInstance.VMType)
	model.CPU = types.Int32Value(int32(apiInstance.CPU))
	model.Memory = types.Int32Value(int32(apiInstance.Memory))

	sshKeys, _ := types.ListValueFrom(ctx, types.StringType, apiInstance.SSHKeys)
	model.SSHKeys = sshKeys
	labels, _ := types.ListValueFrom(ctx, types.StringType, apiInstance.Labels)
	model.Labels = labels

	for _, nw := range apiInstance.Network {
		model.Network = append(model.Network, instanceNetworkModel{
			Order:    types.Int32Value(int32(nw.Order)),
			SubnetID: types.StringValue(nw.SubnetID),
			Model:    types.StringValue(nw.Model),
		})
	}

	for _, d := range apiInstance.Disks {
		model.Disks = append(model.Disks, instanceDiskModel{
			Order:  types.Int32Value(int32(d.Order)),
			Bus:    types.StringValue(d.Bus),
			DiskID: types.StringValue(d.DiskID),
			Cdrom:  types.BoolValue(d.Cdrom),
		})
	}

	model.CloudInit = &instanceCloudInitModel{
		Custom: types.BoolValue(apiInstance.CloudInit.Custom),
		Bus:    types.StringValue(apiInstance.CloudInit.Bus),
		Config: types.StringValue(apiInstance.CloudInit.Config),
	}

	// Verification
	if model.ID.ValueString() != "inst-1" {
		t.Errorf("Expected ID inst-1, got %s", model.ID.ValueString())
	}
	if len(model.Network) != 1 {
		t.Errorf("Expected 1 network, got %d", len(model.Network))
	}
	if model.CloudInit.Config.ValueString() != "user-data" {
		t.Errorf("Expected cloud-init config user-data, got %s", model.CloudInit.Config.ValueString())
	}
}

func TestAccInstanceResource(t *testing.T) {
	resourceName := "spx_instance.test"
	vpcName := "terraform-test-vpc-for-instance"
	subnetName := "terraform-test-subnet-for-instance"
	instanceName := "terraform-test-instance"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccInstanceResourceConfig(vpcName, subnetName, instanceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", instanceName),
					resource.TestCheckResourceAttr(resourceName, "vm_type", "standard"),
					resource.TestCheckResourceAttr(resourceName, "run_strategy", "Always"),
					resource.TestCheckResourceAttr(resourceName, "cpu", "2"),
					resource.TestCheckResourceAttr(resourceName, "memory", "4"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "network.0.order", "0"),
					resource.TestCheckResourceAttrSet(resourceName, "network.0.subnet_id"),
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

func testAccInstanceResourceConfig(vpcName, subnetName, instanceName string) string {
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
  ipv4       = "10.0.1.0/24"
  private    = true
  protocol   = "IPv4"
  nat_gateway = false
}

resource "spx_instance" "test" {
  name         = "%s"
  vm_type      = "standard"
  run_strategy = "Always"
  cpu          = 2
  memory       = 4
  network = [{
    order     = 0
    subnet_id = spx_subnet.test.id
    model     = "virtio"
  }]
}
`, testOrgaId, testProjectId, testAz, vpcName, subnetName, instanceName)
}
