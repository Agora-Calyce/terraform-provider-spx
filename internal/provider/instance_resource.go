package provider

import (
	"context"
	"fmt"
	"regexp"

	"github.com/Agora-Calyce/terraform-provider-spx/internal/client"

	"github.com/hashicorp/terraform-plugin-framework-validators/int32validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type instanceResource struct {
	client *client.Client
}

type instanceResourceModel struct {
	ID             types.String            `tfsdk:"id"`
	Name           types.String            `tfsdk:"name"`
	RunStrategy    types.String            `tfsdk:"run_strategy"`
	VMType         types.String            `tfsdk:"vm_type"`
	CPU            types.Int32             `tfsdk:"cpu"`
	Memory         types.Int32             `tfsdk:"memory"`
	Network        []instanceNetworkModel  `tfsdk:"network"`
	Disks          []instanceDiskModel     `tfsdk:"disks"`
	CloudInit      *instanceCloudInitModel `tfsdk:"cloud_init"`
	SSHKeys        types.List              `tfsdk:"ssh_keys"`
	Labels         types.List              `tfsdk:"labels"`
	Ready          types.Bool              `tfsdk:"ready"`
	IPAddress      types.String            `tfsdk:"ip_address"`
	Architecture   types.String            `tfsdk:"architecture"`
	ContainerDisks types.List              `tfsdk:"container_disks"`
	Advanced       *instanceAdvancedModel  `tfsdk:"advanced"`
	Az             types.String            `tfsdk:"az"`
	OrgaId         types.String            `tfsdk:"orga_id"`
	ProjectId      types.String            `tfsdk:"project_id"`
}

type instanceNetworkModel struct {
	Order      types.Int32  `tfsdk:"order"`
	SubnetID   types.String `tfsdk:"subnet_id"`
	Model      types.String `tfsdk:"model"`
	Enabled    types.Bool   `tfsdk:"enabled"`
	IPv4       types.String `tfsdk:"ipv4"`
	IPv6       types.String `tfsdk:"ipv6"`
	MACAddress types.String `tfsdk:"mac_address"`
}

type instanceAdvancedModel struct {
	TPM *instanceTPMModel `tfsdk:"tpm"`
	EFI *instanceEFIModel `tfsdk:"efi"`
}

type instanceTPMModel struct {
	Enabled    types.Bool `tfsdk:"enabled"`
	Persistent types.Bool `tfsdk:"persistent"`
}

type instanceEFIModel struct {
	Enabled    types.Bool `tfsdk:"enabled"`
	SecureBoot types.Bool `tfsdk:"secure_boot"`
	Persistent types.Bool `tfsdk:"persistent"`
}

type instanceDiskModel struct {
	Order  types.Int32  `tfsdk:"order"`
	Bus    types.String `tfsdk:"bus"`
	DiskID types.String `tfsdk:"disk_id"`
	Cdrom  types.Bool   `tfsdk:"cdrom"`
}

type instanceCloudInitModel struct {
	Custom types.Bool   `tfsdk:"custom"`
	Bus    types.String `tfsdk:"bus"`
	Config types.String `tfsdk:"config"`
}

func NewInstanceResource() resource.Resource {
	return &instanceResource{}
}

func (r *instanceResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_instance"
}

func (r *instanceResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an Instance on SPX.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Description: "The ID of the instance.",
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the instance.",
			},
			"run_strategy": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The run strategy of the instance (e.g., Always, Manual, Halted).",
			},
			"vm_type": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The type of VM. Default is 'linux'.",
				Default:     stringdefault.StaticString("linux"),
			},
			"cpu": schema.Int32Attribute{
				Required:    true,
				Description: "The number of CPUs (max 32).",
				Validators: []validator.Int32{
					int32validator.AtLeast(1),
					int32validator.AtMost(32),
				},
			},
			"memory": schema.Int32Attribute{
				Required:    true,
				Description: "The amount of memory in GB (max 64).",
				Validators: []validator.Int32{
					int32validator.AtLeast(1),
					int32validator.AtMost(64),
				},
			},
			"ssh_keys": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Description: "The list of SSH key EIDs to add to the instance.",
			},
			"labels": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Description: "The list of labels for the instance (keys must start with 'user.superphenix.net/').",
				Validators: []validator.List{
					listvalidator.ValueStringsAre(stringvalidator.RegexMatches(regexp.MustCompile(`^user\.superphenix\.net/.*`), "keys must start with 'user.superphenix.net/'")),
				},
			},
			"ready": schema.BoolAttribute{
				Computed:    true,
				Description: "True if the instance is ready.",
			},
			"ip_address": schema.StringAttribute{
				Computed:    true,
				Description: "The primary IP address of the instance.",
			},
			"architecture": schema.StringAttribute{
				Computed:    true,
				Description: "The CPU architecture of the instance.",
			},
			"container_disks": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
				Description: "IDs of the container disks (catalog entries, e.g. drivers) mounted on the instance. " +
					"If not set, the recommended set is mounted at creation and the current mounts are kept on update; an empty list detaches all.",
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
			},
			"az": schema.StringAttribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Description: "The availability zone of the instance. If not set, the provider value is used.",
			},
			"orga_id": schema.StringAttribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Description: "The organization ID. If not set, the provider value is used.",
			},
			"project_id": schema.StringAttribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Description: "The project ID. If not set, the provider value is used.",
			},
		},
		Blocks: map[string]schema.Block{
			"network": schema.ListNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"order": schema.Int32Attribute{
							Required: true,
						},
						"subnet_id": schema.StringAttribute{
							Required: true,
						},
						"model": schema.StringAttribute{
							Optional: true,
							Computed: true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"enabled": schema.BoolAttribute{
							Optional:    true,
							Computed:    true,
							Description: "Link state of the interface (false = link down). Defaults to true.",
							PlanModifiers: []planmodifier.Bool{
								boolplanmodifier.UseStateForUnknown(),
							},
						},
						"ipv4": schema.StringAttribute{
							Optional: true,
						},
						"ipv6": schema.StringAttribute{
							Optional: true,
						},
						"mac_address": schema.StringAttribute{
							Optional:    true,
							Computed:    true,
							Description: "Static MAC address of the interface.",
							Validators: []validator.String{
								stringvalidator.RegexMatches(macAddressRegexp, "must be a MAC address (e.g. 02:00:00:aa:bb:cc)"),
							},
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
					},
				},
			},
			"disks": schema.ListNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"order": schema.Int32Attribute{
							Required: true,
						},
						"bus": schema.StringAttribute{
							Optional: true,
							Computed: true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"disk_id": schema.StringAttribute{
							Optional: true,
						},
						"cdrom": schema.BoolAttribute{
							Optional: true,
							Computed: true,
							PlanModifiers: []planmodifier.Bool{
								boolplanmodifier.UseStateForUnknown(),
							},
						},
					},
				},
			},
			"advanced": schema.SingleNestedBlock{
				Description: "Advanced device/firmware overrides. Any unset value is inherited from the vm_type.",
				Blocks: map[string]schema.Block{
					"tpm": schema.SingleNestedBlock{
						Description: "TPM device.",
						Attributes: map[string]schema.Attribute{
							"enabled":    advancedBoolAttribute("Enable (true) or remove (false) the TPM device."),
							"persistent": advancedBoolAttribute("Persist the TPM state across reboots."),
						},
					},
					"efi": schema.SingleNestedBlock{
						Description: "EFI bootloader (enabled = false forces BIOS).",
						Attributes: map[string]schema.Attribute{
							"enabled":     advancedBoolAttribute("Boot with EFI (true) or BIOS (false)."),
							"secure_boot": advancedBoolAttribute("Enable Secure Boot."),
							"persistent":  advancedBoolAttribute("Persist the EFI variables across reboots."),
						},
					},
				},
			},
			"cloud_init": schema.SingleNestedBlock{
				Attributes: map[string]schema.Attribute{
					"custom": schema.BoolAttribute{
						Optional: true,
						Computed: true,
					},
					"bus": schema.StringAttribute{
						Optional: true,
						Computed: true,
					},
					"config": schema.StringAttribute{
						Optional: true,
					},
				},
			},
		},
	}
}

func (r *instanceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

func (r *instanceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data instanceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgaId := r.client.OrgaId
	if !data.OrgaId.IsUnknown() && !data.OrgaId.IsNull() {
		orgaId = data.OrgaId.ValueString()
	}
	az := r.client.Az
	if !data.Az.IsUnknown() && !data.Az.IsNull() {
		az = data.Az.ValueString()
	}
	projectId := r.client.ProjectId
	if !data.ProjectId.IsUnknown() && !data.ProjectId.IsNull() {
		projectId = data.ProjectId.ValueString()
	}

	instanceReq := client.InstanceRequest{
		Name:        data.Name.ValueString(),
		RunStrategy: data.RunStrategy.ValueString(),
		VMType:      data.VMType.ValueString(),
		CPU:         int(data.CPU.ValueInt32()),
		Memory:      int(data.Memory.ValueInt32()),
	}

	if instanceReq.RunStrategy == "" {
		instanceReq.RunStrategy = "Always"
	}
	if instanceReq.VMType == "" {
		instanceReq.VMType = "standard"
	}

	// Networks
	for _, nw := range data.Network {
		model := nw.Model.ValueString()
		if model == "" {
			model = "virtio"
		}
		instanceReq.Network = append(instanceReq.Network, client.InstanceNetwork{
			Order:      int(nw.Order.ValueInt32()),
			SubnetID:   nw.SubnetID.ValueString(),
			Model:      model,
			Enabled:    knownBoolPointer(nw.Enabled),
			IPv4:       nw.IPv4.ValueString(),
			IPv6:       nw.IPv6.ValueString(),
			MACAddress: knownString(nw.MACAddress),
		})
	}

	// Disks
	for _, d := range data.Disks {
		bus := d.Bus.ValueString()
		if bus == "" {
			bus = "virtio"
		}
		instanceReq.Disks = append(instanceReq.Disks, client.InstanceDisk{
			Order:  int(d.Order.ValueInt32()),
			Bus:    bus,
			DiskID: d.DiskID.ValueString(),
			Cdrom:  d.Cdrom.ValueBool(),
		})
	}

	// CloudInit
	if data.CloudInit != nil {
		bus := data.CloudInit.Bus.ValueString()
		if bus == "" {
			bus = "virtio"
		}
		instanceReq.CloudInit = client.InstanceCloudInit{
			Custom: data.CloudInit.Custom.ValueBool(),
			Bus:    bus,
			Config: data.CloudInit.Config.ValueString(),
		}
	}

	// SSHKeys
	if !data.SSHKeys.IsNull() {
		var sshKeys []string
		resp.Diagnostics.Append(data.SSHKeys.ElementsAs(ctx, &sshKeys, false)...)
		instanceReq.SSHKeys = sshKeys
	}

	// Labels
	if !data.Labels.IsNull() {
		var labels []string
		resp.Diagnostics.Append(data.Labels.ElementsAs(ctx, &labels, false)...)
		instanceReq.Labels = labels
	}

	instanceReq.ContainerDisks = knownStringListPointer(ctx, data.ContainerDisks, &resp.Diagnostics)
	instanceReq.Advanced = advancedToClient(data.Advanced)

	instance, err := r.client.CreateInstance(ctx, orgaId, az, projectId, instanceReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating instance", err.Error())
		return
	}

	data.ID = types.StringValue(instance.ID)
	data.Ready = types.BoolValue(instance.Ready)
	data.IPAddress = types.StringValue(instance.IPAddress)
	data.Architecture = types.StringValue(instance.Architecture)
	data.Az = types.StringValue(az)
	data.OrgaId = types.StringValue(orgaId)
	data.ProjectId = types.StringValue(projectId)
	resolveInstanceUnknowns(ctx, &data, instance, &resp.Diagnostics)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *instanceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data instanceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	instance, err := r.client.GetInstance(ctx, data.OrgaId.ValueString(), data.Az.ValueString(), data.ProjectId.ValueString(), data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading instance", err.Error())
		return
	}

	data.Name = types.StringValue(instance.Name)
	data.RunStrategy = types.StringValue(instance.RunStrategy)
	data.VMType = types.StringValue(instance.VMType)
	data.CPU = types.Int32Value(int32(instance.CPU))
	data.Memory = types.Int32Value(int32(instance.Memory))
	data.Ready = types.BoolValue(instance.Ready)
	data.IPAddress = types.StringValue(instance.IPAddress)
	data.Architecture = types.StringValue(instance.Architecture)

	// Update list fields
	sshKeys, _ := types.ListValueFrom(ctx, types.StringType, instance.SSHKeys)
	data.SSHKeys = sshKeys
	labels, _ := types.ListValueFrom(ctx, types.StringType, instance.Labels)
	data.Labels = labels
	containerDisks, diags := types.ListValueFrom(ctx, types.StringType, nonNilStrings(instance.ContainerDisks))
	resp.Diagnostics.Append(diags...)
	data.ContainerDisks = containerDisks

	// Advanced options: only if managed in the configuration
	if data.Advanced != nil {
		data.Advanced = advancedFromClient(instance.Advanced, data.Advanced)
	}

	// Network mapping back to state
	newNetwork := make([]instanceNetworkModel, 0, len(instance.Network))
	for _, nw := range instance.Network {
		newNetwork = append(newNetwork, instanceNetworkModel{
			Order:      types.Int32Value(int32(nw.Order)),
			SubnetID:   types.StringValue(nw.SubnetID),
			Model:      types.StringValue(nw.Model),
			Enabled:    networkEnabledFromClient(nw.Enabled),
			IPv4:       types.StringValue(nw.IPv4),
			IPv6:       types.StringValue(nw.IPv6),
			MACAddress: optionalStringFromClient(nw.MACAddress),
		})
	}
	data.Network = newNetwork

	// Disks mapping back to state
	newDisks := make([]instanceDiskModel, 0, len(instance.Disks))
	for _, d := range instance.Disks {
		newDisks = append(newDisks, instanceDiskModel{
			Order:  types.Int32Value(int32(d.Order)),
			Bus:    types.StringValue(d.Bus),
			DiskID: types.StringValue(d.DiskID),
			Cdrom:  types.BoolValue(d.Cdrom),
		})
	}
	data.Disks = newDisks

	// CloudInit: only if managed in the configuration (otherwise a permanent diff on an absent block).
	// The SPX API no longer always returns bus/config: the known value is kept in that case.
	if data.CloudInit != nil {
		data.CloudInit = cloudInitFromClient(instance.CloudInit, data.CloudInit)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *instanceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data instanceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	instanceReq := client.InstanceRequest{
		Name:        data.Name.ValueString(),
		RunStrategy: data.RunStrategy.ValueString(),
		VMType:      data.VMType.ValueString(),
		CPU:         int(data.CPU.ValueInt32()),
		Memory:      int(data.Memory.ValueInt32()),
	}

	// Similar logic to Create for mapping fields...
	for _, nw := range data.Network {
		instanceReq.Network = append(instanceReq.Network, client.InstanceNetwork{
			Order:      int(nw.Order.ValueInt32()),
			SubnetID:   nw.SubnetID.ValueString(),
			Model:      nw.Model.ValueString(),
			Enabled:    knownBoolPointer(nw.Enabled),
			IPv4:       nw.IPv4.ValueString(),
			IPv6:       nw.IPv6.ValueString(),
			MACAddress: knownString(nw.MACAddress),
		})
	}
	for _, d := range data.Disks {
		instanceReq.Disks = append(instanceReq.Disks, client.InstanceDisk{
			Order:  int(d.Order.ValueInt32()),
			Bus:    d.Bus.ValueString(),
			DiskID: d.DiskID.ValueString(),
			Cdrom:  d.Cdrom.ValueBool(),
		})
	}
	if data.CloudInit != nil {
		instanceReq.CloudInit = client.InstanceCloudInit{
			Custom: data.CloudInit.Custom.ValueBool(),
			Bus:    data.CloudInit.Bus.ValueString(),
			Config: data.CloudInit.Config.ValueString(),
		}
	}
	if !data.SSHKeys.IsNull() {
		var sshKeys []string
		data.SSHKeys.ElementsAs(ctx, &sshKeys, false)
		instanceReq.SSHKeys = sshKeys
	}
	if !data.Labels.IsNull() {
		var labels []string
		data.Labels.ElementsAs(ctx, &labels, false)
		instanceReq.Labels = labels
	}

	instanceReq.ContainerDisks = knownStringListPointer(ctx, data.ContainerDisks, &resp.Diagnostics)
	instanceReq.Advanced = advancedToClient(data.Advanced)

	instance, err := r.client.UpdateInstance(ctx, data.OrgaId.ValueString(), data.Az.ValueString(), data.ProjectId.ValueString(), data.ID.ValueString(), instanceReq)
	if err != nil {
		resp.Diagnostics.AddError("Error updating instance", err.Error())
		return
	}

	data.Ready = types.BoolValue(instance.Ready)
	data.IPAddress = types.StringValue(instance.IPAddress)
	data.Architecture = types.StringValue(instance.Architecture)
	resolveInstanceUnknowns(ctx, &data, instance, &resp.Diagnostics)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *instanceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data instanceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteInstance(ctx, data.OrgaId.ValueString(), data.Az.ValueString(), data.ProjectId.ValueString(), data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error deleting instance", err.Error())
		return
	}
}
