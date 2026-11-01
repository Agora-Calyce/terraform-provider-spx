package provider

import (
	"context"
	"fmt"
	"github.com/Agora-Calyce/terraform-provider-spx/internal/client"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	tfresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func securityGroupSchema(t *testing.T) resource.SchemaResponse {
	t.Helper()
	var resp resource.SchemaResponse
	NewSecurityGroupResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema error: %v", resp.Diagnostics)
	}
	return resp
}

func TestSecurityGroupResource_SchemaValid(t *testing.T) {
	resp := securityGroupSchema(t)
	if diags := resp.Schema.ValidateImplementation(context.Background()); diags.HasError() {
		t.Fatalf("invalid schema implementation: %v", diags)
	}
}

// Le modèle doit correspondre exactement au schéma (champs tfsdk), sinon toute lecture de plan/état échoue.
func TestSecurityGroupResource_ModelMatchesSchema(t *testing.T) {
	ctx := context.Background()
	s := securityGroupSchema(t).Schema
	state := tfsdk.State{Schema: s, Raw: tftypes.NewValue(s.Type().TerraformType(ctx), nil)}

	model := SecurityGroupResourceModel{
		ID:        types.StringValue("sg-1"),
		Name:      types.StringValue("sg"),
		SubnetIDs: []types.String{types.StringValue("subnet-a")},
		Target:    &TargetModel{MatchLabels: []types.String{types.StringValue("app:web")}},
		Ingress: []IngressRuleModel{{
			AllowAll: types.BoolValue(false),
			DenyAll:  types.BoolValue(false),
			Ports:    []SgPortModel{{Port: types.Int32Value(80), EndPort: types.Int32Value(80), Protocol: types.StringValue("TCP")}},
			From: []PeerModel{
				{PodSelector: &TargetModel{MatchLabels: []types.String{types.StringValue("role:db")}}},
				{IPBlock: &IPBlockModel{CIDR: types.StringValue("10.0.0.0/8")}},
			},
		}},
		Egress: []EgressRuleModel{{AllowAll: types.BoolValue(true), DenyAll: types.BoolValue(false)}},
	}
	if diags := state.Set(ctx, &model); diags.HasError() {
		t.Fatalf("model does not match schema: %v", diags)
	}

	var back SecurityGroupResourceModel
	if diags := state.Get(ctx, &back); diags.HasError() {
		t.Fatalf("cannot read model back: %v", diags)
	}
	if back.Target.MatchLabels[0].ValueString() != "app:web" || back.SubnetIDs[0].ValueString() != "subnet-a" {
		t.Errorf("unexpected round trip: %+v", back)
	}
}

func TestMapTargetToClient(t *testing.T) {
	model := TargetModel{
		MatchLabels: []types.String{
			types.StringValue("app:web"),
		},
		MatchExpressions: []MatchExpressionModel{
			{
				Key:      types.StringValue("env"),
				Operator: types.StringValue("In"),
				Values:   []types.String{types.StringValue("prod"), types.StringValue("staging")},
			},
		},
	}

	expected := client.LabelSelector{
		MatchLabels: []string{"app:web"},
		MatchExpressions: []client.MatchExpression{
			{Key: "env", Operator: "In", Values: []string{"prod", "staging"}},
		},
	}

	result := mapTargetToClient(model)

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Expected %+v, got %+v", expected, result)
	}
}

func TestMapTargetFromClient(t *testing.T) {
	ls := client.LabelSelector{
		MatchLabels: []string{"app:web"},
		MatchExpressions: []client.MatchExpression{
			{Key: "env", Operator: "In", Values: []string{"prod", "staging"}},
		},
	}

	result := mapTargetFromClient(ls)

	if len(result.MatchLabels) != 1 || result.MatchLabels[0].ValueString() != "app:web" {
		t.Errorf("Labels mismatch: %+v", result.MatchLabels)
	}
	if len(result.MatchExpressions) != 1 || len(result.MatchExpressions[0].Values) != 2 {
		t.Errorf("Expressions mismatch: %+v", result.MatchExpressions)
	}
}

func TestMapPorts(t *testing.T) {
	sent := mapPortsToClient([]SgPortModel{
		{Port: types.Int32Value(443), EndPort: types.Int32Unknown(), Protocol: types.StringValue("tcp")},
		{Port: types.Int32Value(8000), EndPort: types.Int32Value(8100), Protocol: types.StringValue("UDP")},
	})
	expected := []client.SgPort{
		{Port: 443, EndPort: 0, Protocol: "TCP"},
		{Port: 8000, EndPort: 8100, Protocol: "UDP"},
	}
	if !reflect.DeepEqual(sent, expected) {
		t.Errorf("Expected %+v, got %+v", expected, sent)
	}

	read := mapPortsFromClient([]client.SgPort{{Port: 22}})
	if read[0].EndPort.ValueInt32() != 22 || read[0].Protocol.ValueString() != "TCP" {
		t.Errorf("end_port/protocol should default to port/TCP, got %+v", read[0])
	}
}

func TestApplySecurityGroupFromClient_PreservesProtocolCase(t *testing.T) {
	data := SecurityGroupResourceModel{
		Ingress: []IngressRuleModel{{
			Ports: []SgPortModel{{Port: types.Int32Value(80), Protocol: types.StringValue("tcp")}},
		}},
	}
	applySecurityGroupFromClient(&data, &client.SecurityGroup{
		Name:    "sg",
		Ingress: []client.IngressRule{{Ports: []client.SgPort{{Port: 80, EndPort: 80, Protocol: "TCP"}}}},
	})
	if got := data.Ingress[0].Ports[0].Protocol.ValueString(); got != "tcp" {
		t.Errorf("protocol case should be preserved, got %q", got)
	}
	if data.SubnetIDs != nil {
		t.Errorf("empty subnet list should map to null, got %+v", data.SubnetIDs)
	}
}

func TestLegacyFirewallStateToModel(t *testing.T) {
	// Ancien état avec match_labels sous forme d'objets {key, value} et ports en chaînes
	raw := []byte(`{
		"id": "fw-1",
		"name": "legacy",
		"description": "old firewall",
		"orga_id": "orga",
		"az": "az1",
		"project_id": "proj",
		"target": {"match_labels": [{"key": "user.superphenix.net/app", "value": "web"}], "match_expressions": []},
		"ingress": [{
			"allow_all": false,
			"deny_all": false,
			"ports": [{"port": 80, "end_port": 81, "protocol": "TCP"}, {"port": "443", "end_port": null, "protocol": null}],
			"from": [
				{"pod_selector": {"match_labels": ["role:db"]}, "ip_block": null},
				{"pod_selector": null, "ip_block": {"cidr": "0.0.0.0/0", "except": ["10.0.0.0/8"]}}
			]
		}],
		"egress": [{"allow_all": true, "deny_all": false, "ports": [], "to": []}],
		"match_labels": null
	}`)

	m, err := legacyFirewallStateToModel(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if m.ID.ValueString() != "fw-1" || m.Name.ValueString() != "legacy" || m.OrgaId.ValueString() != "orga" {
		t.Errorf("unexpected scalar fields: %+v", m)
	}
	if len(m.Target.MatchLabels) != 1 || m.Target.MatchLabels[0].ValueString() != "user.superphenix.net/app:web" {
		t.Errorf("object labels should be converted to key:value, got %+v", m.Target.MatchLabels)
	}
	ports := m.Ingress[0].Ports
	if ports[0].Port.ValueInt32() != 80 || ports[0].EndPort.ValueInt32() != 81 {
		t.Errorf("unexpected first port: %+v", ports[0])
	}
	if ports[1].Port.ValueInt32() != 443 || !ports[1].EndPort.IsNull() || ports[1].Protocol.ValueString() != "TCP" {
		t.Errorf("unexpected second port: %+v", ports[1])
	}
	if m.Ingress[0].From[0].PodSelector.MatchLabels[0].ValueString() != "role:db" {
		t.Errorf("string labels should be kept as is")
	}
	if m.Ingress[0].From[1].IPBlock.CIDR.ValueString() != "0.0.0.0/0" {
		t.Errorf("ip_block not migrated")
	}
	if !m.Egress[0].AllowAll.ValueBool() {
		t.Errorf("egress allow_all not migrated")
	}
	if m.SubnetIDs != nil {
		t.Errorf("subnet_ids should be null after migration")
	}
}

func TestLegacyFirewallStateToModel_MissingID(t *testing.T) {
	if _, err := legacyFirewallStateToModel([]byte(`{"name": "x"}`)); err == nil {
		t.Error("expected an error when id is missing")
	}
}

// Exécute le StateMover comme le ferait Terraform pour `moved { from = spx_firewall.x  to = spx_security_group.x }`
func TestSecurityGroupResource_MoveState(t *testing.T) {
	ctx := context.Background()
	s := securityGroupSchema(t).Schema
	r := NewSecurityGroupResource().(*SecurityGroupResource)
	movers := r.MoveState(ctx)
	if len(movers) != 1 {
		t.Fatalf("expected 1 state mover, got %d", len(movers))
	}

	newResp := func() *resource.MoveStateResponse {
		return &resource.MoveStateResponse{
			TargetState: tfsdk.State{Schema: s, Raw: tftypes.NewValue(s.Type().TerraformType(ctx), nil)},
		}
	}

	// Source gérée
	resp := newResp()
	movers[0].StateMover(ctx, resource.MoveStateRequest{
		SourceProviderAddress: "registry.terraform.io/agora-calyce/spx",
		SourceTypeName:        "spx_firewall",
		SourceRawState: &tfprotov6.RawState{
			JSON: []byte(`{"id": "fw-1", "name": "legacy", "orga_id": "o", "az": "a", "project_id": "p",
				"target": {"match_labels": ["app:web"]},
				"ingress": [{"ports": [{"port": 22, "end_port": 22, "protocol": "TCP"}], "from": [{"ip_block": {"cidr": "0.0.0.0/0"}}]}]}`),
		},
	}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("move failed: %v", resp.Diagnostics)
	}
	var moved SecurityGroupResourceModel
	if diags := resp.TargetState.Get(ctx, &moved); diags.HasError() {
		t.Fatalf("cannot read moved state: %v", diags)
	}
	if moved.ID.ValueString() != "fw-1" || moved.Ingress[0].Ports[0].Port.ValueInt32() != 22 {
		t.Errorf("unexpected moved state: %+v", moved)
	}

	// Source non gérée : le mover ne fait rien (le framework signalera l'absence de mover)
	resp = newResp()
	movers[0].StateMover(ctx, resource.MoveStateRequest{
		SourceProviderAddress: "registry.terraform.io/hashicorp/aws",
		SourceTypeName:        "aws_security_group",
		SourceRawState:        &tfprotov6.RawState{JSON: []byte(`{"id": "x"}`)},
	}, resp)
	if resp.Diagnostics.HasError() || !resp.TargetState.Raw.IsNull() {
		t.Errorf("foreign source should be ignored, got diags=%v state=%v", resp.Diagnostics, resp.TargetState.Raw)
	}
}

func TestAccSecurityGroupResource(t *testing.T) {
	resourceName := "spx_security_group.test"
	sgName := "terraform-test-sg"

	tfresource.Test(t, tfresource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []tfresource.TestStep{
			{
				Config: testAccSecurityGroupResourceConfig(sgName),
				Check: tfresource.ComposeTestCheckFunc(
					tfresource.TestCheckResourceAttr(resourceName, "name", sgName),
					tfresource.TestCheckResourceAttrSet(resourceName, "id"),
					tfresource.TestCheckResourceAttr(resourceName, "target.match_labels.0", "app:web"),
					tfresource.TestCheckResourceAttr(resourceName, "ingress.0.ports.0.protocol", "TCP"),
					tfresource.TestCheckResourceAttr(resourceName, "ingress.0.ports.0.end_port", "443"),
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

func testAccSecurityGroupResourceConfig(name string) string {
	return fmt.Sprintf(`
provider "spx" {
  orga_id    = "%s"
  project_id = "%s"
  az         = "%s"
}

resource "spx_security_group" "test" {
  name = "%s"

  target {
    match_labels = ["app:web"]
  }

  ingress {
    ports {
      port = 443
    }
    from {
      ip_block {
        cidr = "0.0.0.0/0"
      }
    }
  }
}
`, testOrgaId, testProjectId, testAz, name)
}

// Après un import, l'état ne contient que l'id : la lecture du modèle ne doit pas échouer (bloc target null).
func TestSecurityGroupResource_ImportedStateReadable(t *testing.T) {
	ctx := context.Background()
	s := securityGroupSchema(t).Schema
	state := tfsdk.State{Schema: s, Raw: tftypes.NewValue(s.Type().TerraformType(ctx), nil)}
	if diags := state.SetAttribute(ctx, path.Root("id"), "sg-1"); diags.HasError() {
		t.Fatalf("cannot set id: %v", diags)
	}
	var m SecurityGroupResourceModel
	if diags := state.Get(ctx, &m); diags.HasError() {
		t.Fatalf("imported state not readable: %v", diags)
	}
	if m.Target != nil || m.ID.ValueString() != "sg-1" {
		t.Errorf("unexpected model: %+v", m)
	}
}
