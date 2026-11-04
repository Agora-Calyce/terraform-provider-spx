package provider

import (
	"context"
	"regexp"

	"github.com/Agora-Calyce/terraform-provider-spx/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var macAddressRegexp = regexp.MustCompile(`^([0-9A-Fa-f]{2}[:-]){5}[0-9A-Fa-f]{2}$`)

// advancedBoolAttribute: optional advanced option; null = inherited from the vm_type
func advancedBoolAttribute(description string) schema.BoolAttribute {
	return schema.BoolAttribute{
		Optional:    true,
		Computed:    true,
		Description: description,
		PlanModifiers: []planmodifier.Bool{
			boolplanmodifier.UseStateForUnknown(),
		},
	}
}

// knownBoolPointer returns nil for a null or unknown value
func knownBoolPointer(v types.Bool) *bool {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	b := v.ValueBool()
	return &b
}

// knownString returns "" for a null or unknown value
func knownString(v types.String) string {
	if v.IsNull() || v.IsUnknown() {
		return ""
	}
	return v.ValueString()
}

// knownStringListPointer returns nil if the list is null/unknown (let SPX decide),
// otherwise a non-nil slice (an empty list is meaningful).
func knownStringListPointer(ctx context.Context, v types.List, diags *diag.Diagnostics) *[]string {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	values := []string{}
	diags.Append(v.ElementsAs(ctx, &values, false)...)
	return &values
}

func optionalStringFromClient(s string) types.String {
	if s == "" {
		return types.StringNull()
	}
	return types.StringValue(s)
}

func nonNilStrings(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

func advancedToClient(m *instanceAdvancedModel) *client.InstanceAdvanced {
	if m == nil {
		return nil
	}
	adv := &client.InstanceAdvanced{}
	if m.TPM != nil {
		adv.TPM = client.InstanceTPM{
			Enabled:    knownBoolPointer(m.TPM.Enabled),
			Persistent: knownBoolPointer(m.TPM.Persistent),
		}
	}
	if m.EFI != nil {
		adv.EFI = client.InstanceEFI{
			Enabled:    knownBoolPointer(m.EFI.Enabled),
			SecureBoot: knownBoolPointer(m.EFI.SecureBoot),
			Persistent: knownBoolPointer(m.EFI.Persistent),
		}
	}
	return adv
}

// advancedFromClient reads the advanced options back, preserving the presence of the configuration's sub-blocks
func advancedFromClient(api *client.InstanceAdvanced, prior *instanceAdvancedModel) *instanceAdvancedModel {
	if api == nil {
		api = &client.InstanceAdvanced{}
	}
	res := &instanceAdvancedModel{}
	if prior.TPM != nil {
		res.TPM = &instanceTPMModel{
			Enabled:    types.BoolPointerValue(api.TPM.Enabled),
			Persistent: types.BoolPointerValue(api.TPM.Persistent),
		}
	}
	if prior.EFI != nil {
		res.EFI = &instanceEFIModel{
			Enabled:    types.BoolPointerValue(api.EFI.Enabled),
			SecureBoot: types.BoolPointerValue(api.EFI.SecureBoot),
			Persistent: types.BoolPointerValue(api.EFI.Persistent),
		}
	}
	return res
}

func resolveBool(v types.Bool, api *bool) types.Bool {
	if v.IsUnknown() {
		return types.BoolPointerValue(api)
	}
	return v
}

// resolveInstanceUnknowns replaces computed values still unknown after a create/update
// with those returned by the API (Terraform rejects a state containing unknown values).
func resolveInstanceUnknowns(ctx context.Context, data *instanceResourceModel, inst *client.Instance, diags *diag.Diagnostics) {
	if data.RunStrategy.IsUnknown() {
		data.RunStrategy = types.StringValue(inst.RunStrategy)
	}
	if data.ContainerDisks.IsUnknown() {
		v, d := types.ListValueFrom(ctx, types.StringType, nonNilStrings(inst.ContainerDisks))
		diags.Append(d...)
		data.ContainerDisks = v
	}

	for i := range data.Network {
		nw := &data.Network[i]
		var api client.InstanceNetwork
		if i < len(inst.Network) {
			api = inst.Network[i]
		}
		if nw.Model.IsUnknown() {
			nw.Model = types.StringValue(api.Model)
		}
		if nw.Enabled.IsUnknown() {
			nw.Enabled = networkEnabledFromClient(api.Enabled)
		}
		if nw.MACAddress.IsUnknown() {
			nw.MACAddress = optionalStringFromClient(api.MACAddress)
		}
	}

	for i := range data.Disks {
		d := &data.Disks[i]
		var api client.InstanceDisk
		if i < len(inst.Disks) {
			api = inst.Disks[i]
		}
		if d.Bus.IsUnknown() {
			d.Bus = types.StringValue(api.Bus)
		}
		if d.Cdrom.IsUnknown() {
			d.Cdrom = types.BoolValue(api.Cdrom)
		}
	}

	if data.CloudInit != nil {
		if data.CloudInit.Custom.IsUnknown() {
			data.CloudInit.Custom = types.BoolValue(inst.CloudInit.Custom)
		}
		if data.CloudInit.Bus.IsUnknown() {
			data.CloudInit.Bus = types.StringValue(inst.CloudInit.Bus)
		}
	}

	if data.Advanced != nil {
		api := inst.Advanced
		if api == nil {
			api = &client.InstanceAdvanced{}
		}
		if t := data.Advanced.TPM; t != nil {
			t.Enabled = resolveBool(t.Enabled, api.TPM.Enabled)
			t.Persistent = resolveBool(t.Persistent, api.TPM.Persistent)
		}
		if e := data.Advanced.EFI; e != nil {
			e.Enabled = resolveBool(e.Enabled, api.EFI.Enabled)
			e.SecureBoot = resolveBool(e.SecureBoot, api.EFI.SecureBoot)
			e.Persistent = resolveBool(e.Persistent, api.EFI.Persistent)
		}
	}
}

func cloudInitFromClient(api client.InstanceCloudInit, prior *instanceCloudInitModel) *instanceCloudInitModel {
	m := &instanceCloudInitModel{
		Custom: types.BoolValue(api.Custom),
		Bus:    types.StringValue(api.Bus),
		Config: types.StringValue(api.Config),
	}
	if api.Bus == "" {
		m.Bus = prior.Bus
	}
	if api.Config == "" {
		m.Config = prior.Config
	}
	return m
}

func networkEnabledFromClient(enabled *bool) types.Bool {
	return types.BoolPointerValue(enabled)
}
