package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/Agora-Calyce/terraform-provider-spx/internal/client"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestMapLoadBalancerToClient(t *testing.T) {
	ctx := context.Background()
	model := loadBalancerResourceModel{
		Name: types.StringValue("test-lb"),
		VIP:  types.StringValue("198.18.0.1"),
		Selectors: []types.String{
			types.StringValue("app:web"),
		},
		Ports: []loadBalancerPortModel{
			{
				Name:       types.StringValue("http"),
				Port:       types.Int32Value(80),
				TargetPort: types.Int32Value(8080),
				Protocol:   types.StringValue("TCP"),
			},
		},
	}

	endpoints := []string{"192.168.1.1", "192.168.1.2"}
	endpointsList, _ := types.ListValueFrom(ctx, types.StringType, endpoints)
	model.Endpoints = endpointsList

	var diags diag.Diagnostics
	lbReq := lbRequestFromModel(ctx, &model, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	expected := client.LoadBalancerRequest{
		Name: "test-lb",
		VIP:  "198.18.0.1",
		Selectors: []client.LBSelector{
			{Key: "app", Value: "web"},
		},
		Endpoints: []string{"192.168.1.1", "192.168.1.2"},
		Ports: []client.LoadBalancerPort{
			{Port: 80, TargetPort: 8080, Protocol: "TCP"},
		},
	}

	if !reflect.DeepEqual(lbReq, expected) {
		t.Errorf("Mapping to client failed. Expected %+v, got %+v", expected, lbReq)
	}

	// The JSON body must match the format expected by spx-tf-api (integer ports, object selectors)
	body, _ := json.Marshal(lbReq)
	want := `{"name":"test-lb","vip":"198.18.0.1","selectors":[{"key":"app","value":"web"}],"endpoints":["192.168.1.1","192.168.1.2"],"ports":[{"port":80,"targetPort":8080,"protocol":"TCP"}]}`
	if string(body) != want {
		t.Errorf("unexpected JSON body:\n got  %s\n want %s", body, want)
	}
}

func TestMapLoadBalancerToClientInvalidSelector(t *testing.T) {
	model := loadBalancerResourceModel{
		Name:      types.StringValue("test-lb"),
		VIP:       types.StringValue("198.18.0.1"),
		Selectors: []types.String{types.StringValue("no-colon")},
		Endpoints: types.ListNull(types.StringType),
	}

	var diags diag.Diagnostics
	lbRequestFromModel(context.Background(), &model, &diags)
	if !diags.HasError() {
		t.Error("expected an error for a selector without ':'")
	}
}

func TestMapLoadBalancerFromClient(t *testing.T) {
	apiLB := client.LoadBalancer{
		ID:   "lb-1",
		Name: "test-lb",
		VIP:  "198.18.0.1",
		Selectors: []client.LBSelector{
			{Key: "app", Value: "web"},
		},
		Ports: []client.LoadBalancerPort{
			{Name: "http", Port: 80, TargetPort: 8080, Protocol: "TCP"},
		},
	}

	var model loadBalancerResourceModel
	var diags diag.Diagnostics
	r := &loadBalancerResource{}
	r.mapLBToModel(context.Background(), &model, &apiLB, "orga", "az", "proj", &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if model.ID.ValueString() != "lb-1" {
		t.Errorf("Expected ID lb-1, got %s", model.ID.ValueString())
	}
	if len(model.Selectors) != 1 || model.Selectors[0].ValueString() != "app:web" {
		t.Errorf("Expected selectors [app:web], got %v", model.Selectors)
	}
	if !model.Endpoints.IsNull() {
		t.Errorf("Expected null endpoints, got %v", model.Endpoints)
	}
	if len(model.Ports) != 1 || model.Ports[0].Port.ValueInt32() != 80 || model.Ports[0].TargetPort.ValueInt32() != 8080 {
		t.Errorf("Unexpected ports: %+v", model.Ports)
	}
}

func TestAccLoadBalancerResource(t *testing.T) {
	resourceName := "spx_load_balancer.test"
	lbName := "terraform-test-lb"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccLoadBalancerResourceConfig(lbName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", lbName),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "selectors.0", "app:web"),
					resource.TestCheckResourceAttr(resourceName, "ports.0.name", "http"),
					resource.TestCheckResourceAttr(resourceName, "ports.0.port", "80"),
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

func testAccLoadBalancerResourceConfig(name string) string {
	return fmt.Sprintf(`
provider "spx" {
  orga_id    = "%s"
  project_id = "%s"
  az         = "%s"
}

resource "spx_load_balancer" "test" {
  name = "%s"
  selectors = ["app:web"]
  ports = [{
    name        = "http"
    port        = 80
    target_port = 8080
    protocol    = "TCP"
  }]
}
`, testOrgaId, testProjectId, testAz, name)
}
