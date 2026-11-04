package provider

import (
	"context"
	"testing"

	"github.com/Agora-Calyce/terraform-provider-spx/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// setModel writes the model into a state built from the resource's real schema:
// fails if a tfsdk field of the model does not match the schema.
func setModel(t *testing.T, r resource.Resource, model interface{}) tfsdk.State {
	t.Helper()
	ctx := context.Background()
	var sr resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &sr)
	if sr.Diagnostics.HasError() {
		t.Fatalf("schema error: %v", sr.Diagnostics)
	}
	if diags := sr.Schema.ValidateImplementation(ctx); diags.HasError() {
		t.Fatalf("invalid schema: %v", diags)
	}
	state := tfsdk.State{Schema: sr.Schema, Raw: tftypes.NewValue(sr.Schema.Type().TerraformType(ctx), nil)}
	if diags := state.Set(ctx, model); diags.HasError() {
		t.Fatalf("model does not match schema: %v", diags)
	}
	return state
}

func boolPtr(b bool) *bool { return &b }

func TestInstanceResource_ModelMatchesSchema(t *testing.T) {
	setModel(t, NewInstanceResource(), &instanceResourceModel{
		ID:             types.StringValue("inst-1"),
		Name:           types.StringValue("vm"),
		CPU:            types.Int32Value(1),
		Memory:         types.Int32Value(1),
		SSHKeys:        types.ListNull(types.StringType),
		Labels:         types.ListNull(types.StringType),
		ContainerDisks: types.ListValueMust(types.StringType, nil),
		Network: []instanceNetworkModel{{
			Order:      types.Int32Value(0),
			SubnetID:   types.StringValue("sub1"),
			Enabled:    types.BoolValue(false),
			MACAddress: types.StringValue("02:00:00:aa:bb:cc"),
		}},
		Advanced: &instanceAdvancedModel{
			TPM: &instanceTPMModel{Enabled: types.BoolValue(true)},
			EFI: &instanceEFIModel{SecureBoot: types.BoolValue(false)},
		},
	})
}

func TestDiskResource_ModelMatchesSchema(t *testing.T) {
	setModel(t, NewDiskResource(), &DiskResourceModel{
		ID:     types.StringValue("disk-1"),
		Name:   types.StringValue("disk"),
		Size:   types.Int64Value(10),
		Labels: []types.String{types.StringValue("user.superphenix.net/env:prod")},
	})
}

func TestVPCStaticRoutesResource_ModelMatchesSchema(t *testing.T) {
	setModel(t, NewVPCStaticRoutesResource(), &vpcStaticRoutesResourceModel{
		ID:    types.StringValue("vpc-1"),
		VPCID: types.StringValue("vpc-1"),
		Routes: []staticRouteModel{{
			Policy:     types.StringValue("policyDst"),
			CIDR:       types.StringValue("192.168.0.0/24"),
			NextHopIP:  types.StringValue("10.0.0.254"),
			RouteTable: types.StringValue("subnet-1"),
		}},
	})
}

func TestResolveInstanceUnknowns(t *testing.T) {
	ctx := context.Background()
	data := instanceResourceModel{
		RunStrategy:    types.StringUnknown(),
		ContainerDisks: types.ListUnknown(types.StringType),
		Network: []instanceNetworkModel{
			{Model: types.StringUnknown(), Enabled: types.BoolUnknown(), MACAddress: types.StringUnknown()},
			// Configured values: must not be overwritten
			{Model: types.StringValue("e1000"), Enabled: types.BoolValue(false), MACAddress: types.StringValue("02:00:00:00:00:01")},
		},
		Disks:     []instanceDiskModel{{Bus: types.StringUnknown(), Cdrom: types.BoolUnknown()}},
		CloudInit: &instanceCloudInitModel{Custom: types.BoolUnknown(), Bus: types.StringUnknown()},
		Advanced: &instanceAdvancedModel{
			TPM: &instanceTPMModel{Enabled: types.BoolValue(true), Persistent: types.BoolUnknown()},
			EFI: &instanceEFIModel{Enabled: types.BoolUnknown(), SecureBoot: types.BoolValue(false), Persistent: types.BoolUnknown()},
		},
	}
	inst := &client.Instance{
		RunStrategy:    "Always",
		ContainerDisks: []string{"virtio-win"},
		Network: []client.InstanceNetwork{
			{Model: "virtio", Enabled: boolPtr(true)},
			{Model: "e1000", Enabled: boolPtr(false), MACAddress: "02:00:00:00:00:01"},
		},
		Disks:     []client.InstanceDisk{{Bus: "scsi", Cdrom: true}},
		CloudInit: client.InstanceCloudInit{Custom: true, Bus: "sata"},
		Advanced: &client.InstanceAdvanced{
			TPM: client.InstanceTPM{Enabled: boolPtr(true)},
			EFI: client.InstanceEFI{Enabled: boolPtr(true), SecureBoot: boolPtr(false)},
		},
	}

	var diags diag.Diagnostics
	resolveInstanceUnknowns(ctx, &data, inst, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}

	if data.RunStrategy.ValueString() != "Always" {
		t.Errorf("run_strategy not resolved: %v", data.RunStrategy)
	}
	var cds []string
	data.ContainerDisks.ElementsAs(ctx, &cds, false)
	if len(cds) != 1 || cds[0] != "virtio-win" {
		t.Errorf("container_disks not resolved: %v", data.ContainerDisks)
	}
	nw := data.Network[0]
	if nw.Model.ValueString() != "virtio" || !nw.Enabled.ValueBool() || !nw.MACAddress.IsNull() {
		t.Errorf("network[0] not resolved correctly: %+v", nw)
	}
	if data.Network[1].Model.ValueString() != "e1000" || data.Network[1].Enabled.ValueBool() {
		t.Errorf("configured network values must be kept: %+v", data.Network[1])
	}
	if data.Disks[0].Bus.ValueString() != "scsi" || !data.Disks[0].Cdrom.ValueBool() {
		t.Errorf("disk not resolved: %+v", data.Disks[0])
	}
	if !data.CloudInit.Custom.ValueBool() || data.CloudInit.Bus.ValueString() != "sata" {
		t.Errorf("cloud_init not resolved: %+v", data.CloudInit)
	}
	if !data.Advanced.TPM.Persistent.IsNull() {
		t.Errorf("unset tpm.persistent should resolve to null (inherited), got %v", data.Advanced.TPM.Persistent)
	}
	if !data.Advanced.EFI.Enabled.ValueBool() || !data.Advanced.EFI.Persistent.IsNull() {
		t.Errorf("efi not resolved: %+v", data.Advanced.EFI)
	}
}

func TestAdvancedToClient(t *testing.T) {
	if advancedToClient(nil) != nil {
		t.Error("nil advanced block must not send overrides")
	}
	adv := advancedToClient(&instanceAdvancedModel{
		EFI: &instanceEFIModel{Enabled: types.BoolUnknown(), SecureBoot: types.BoolValue(false), Persistent: types.BoolNull()},
	})
	if adv.TPM.Enabled != nil || adv.TPM.Persistent != nil {
		t.Errorf("absent tpm block must be inherited: %+v", adv.TPM)
	}
	if adv.EFI.Enabled != nil || adv.EFI.SecureBoot == nil || *adv.EFI.SecureBoot || adv.EFI.Persistent != nil {
		t.Errorf("unexpected efi: %+v", adv.EFI)
	}
}

func TestKnownStringListPointer(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics
	if knownStringListPointer(ctx, types.ListNull(types.StringType), &diags) != nil {
		t.Error("null list must map to nil (SPX default)")
	}
	if knownStringListPointer(ctx, types.ListUnknown(types.StringType), &diags) != nil {
		t.Error("unknown list must map to nil")
	}
	empty := knownStringListPointer(ctx, types.ListValueMust(types.StringType, nil), &diags)
	if empty == nil || len(*empty) != 0 {
		t.Error("explicit empty list must be sent as [] (detach all)")
	}
}

func TestBuildDiskRequest_SendsStorageClassAndSource(t *testing.T) {
	req := buildDiskRequest(DiskResourceModel{
		Name:         types.StringValue("disk"),
		Size:         types.Int64Value(20),
		StorageClass: types.StringValue("fast"),
		Source:       &DiskSourceModel{Type: types.StringValue("blank")},
		Labels:       []types.String{types.StringValue("user.superphenix.net/env:prod")},
	})
	// The proxy also validates storage_class and source on update
	if req.StorageClass != "fast" || req.Source == nil || req.Source.Type != "blank" {
		t.Errorf("storage_class/source must always be sent: %+v", req)
	}
	if len(req.Labels) != 1 || req.Labels[0] != "user.superphenix.net/env:prod" {
		t.Errorf("labels not sent: %+v", req.Labels)
	}
}

func TestCustomLabelRegexp(t *testing.T) {
	for _, ok := range []string{"user.superphenix.net/env:prod", "user.superphenix.net/env:"} {
		if !customLabelRegexp.MatchString(ok) {
			t.Errorf("%q should be valid", ok)
		}
	}
	for _, bad := range []string{"env:prod", "user.superphenix.net/env", "user.superphenix.net/a:b:c"} {
		if customLabelRegexp.MatchString(bad) {
			t.Errorf("%q should be invalid", bad)
		}
	}
}

func runStringValidator(v validator.String, value string) diag.Diagnostics {
	resp := &validator.StringResponse{}
	v.ValidateString(context.Background(), validator.StringRequest{
		Path:        path.Root("x"),
		ConfigValue: types.StringValue(value),
	}, resp)
	return resp.Diagnostics
}

func TestCanonicalCIDRValidator(t *testing.T) {
	for _, ok := range []string{"192.168.0.0/24", "0.0.0.0/0", "fd00::/64"} {
		if d := runStringValidator(canonicalCIDRValidator{}, ok); d.HasError() {
			t.Errorf("%q should be valid: %v", ok, d)
		}
	}
	for _, bad := range []string{"192.168.0.1/24", "not-a-cidr", "10.0.0.0"} {
		if d := runStringValidator(canonicalCIDRValidator{}, bad); !d.HasError() {
			t.Errorf("%q should be rejected", bad)
		}
	}
}

func TestIPAddressValidator(t *testing.T) {
	if d := runStringValidator(ipAddressValidator{}, "10.0.0.254"); d.HasError() {
		t.Errorf("valid IP rejected: %v", d)
	}
	if d := runStringValidator(ipAddressValidator{}, "10.0.0.256"); !d.HasError() {
		t.Error("invalid IP accepted")
	}
}

func TestStaticRoutesMapping(t *testing.T) {
	sent := staticRoutesToClient(nil)
	if sent == nil || len(sent) != 0 {
		t.Error("no routes must be sent as an empty list, not null")
	}
	read := staticRoutesFromClient([]client.StaticRoute{{CIDR: "10.0.0.0/8", NextHopIP: "10.0.0.1", RouteTable: "s"}})
	if read[0].Policy.ValueString() != "policyDst" {
		t.Errorf("missing policy should default to policyDst, got %v", read[0].Policy)
	}
	if staticRoutesFromClient(nil) != nil {
		t.Error("no routes should map to a null set")
	}
}

func TestCloudInitFromClientKeepsPriorWhenAPIEmpty(t *testing.T) {
	prior := &instanceCloudInitModel{
		Custom: types.BoolValue(true),
		Bus:    types.StringValue("virtio"),
		Config: types.StringValue("#cloud-config"),
	}

	got := cloudInitFromClient(client.InstanceCloudInit{Custom: true}, prior)
	if got.Bus.ValueString() != "virtio" || got.Config.ValueString() != "#cloud-config" {
		t.Errorf("expected prior bus/config to be kept, got %+v", got)
	}

	got = cloudInitFromClient(client.InstanceCloudInit{Custom: false, Bus: "sata", Config: "x"}, prior)
	if got.Custom.ValueBool() || got.Bus.ValueString() != "sata" || got.Config.ValueString() != "x" {
		t.Errorf("expected API values to win, got %+v", got)
	}
}
