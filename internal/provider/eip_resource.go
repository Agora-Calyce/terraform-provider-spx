package provider

import (
	"context"
	"fmt"

	"github.com/Agora-Calyce/terraform-provider-spx/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type eipResource struct {
	client *client.Client
}

type eipResourceModel struct {
	ID         types.String   `tfsdk:"id"`
	Name       types.String   `tfsdk:"name"`
	SubnetId   types.String   `tfsdk:"subnet_id"`
	InternalIP types.String   `tfsdk:"internal_ip"`
	ExternalIP types.String   `tfsdk:"external_ip"`
	SNAT       types.List     `tfsdk:"snat"`
	DNAT       []eipDNATModel `tfsdk:"dnat"`
	Nat        types.String   `tfsdk:"nat"`
	Az         types.String   `tfsdk:"az"`
	OrgaId     types.String   `tfsdk:"orga_id"`
	ProjectId  types.String   `tfsdk:"project_id"`
}

type eipDNATModel struct {
	ExternalPort types.String `tfsdk:"external_port"`
	InternalIP   types.String `tfsdk:"internal_ip"`
	InternalPort types.String `tfsdk:"internal_port"`
	Protocol     types.String `tfsdk:"protocol"`
}

func NewEipResource() resource.Resource {
	return &eipResource{}
}

func (r *eipResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_eip"
}

func (r *eipResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Gère une ressource Elastic IP sur SPX.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Description: "The ID of the EIP.",
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the EIP.",
			},
			"subnet_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the subnet.",
			},
			"internal_ip": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The internal IP address.",
			},
			"external_ip": schema.StringAttribute{
				Computed:    true,
				Description: "The external IP address.",
			},
			"snat": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Description: "The list of internal IPs for SNAT.",
			},
			"nat": schema.StringAttribute{
				Computed:    true,
				Description: "The NAT type of the EIP (fip or snat).",
			},
			"az": schema.StringAttribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Description: "The availability zone of the EIP. If not set, the provider value is used.",
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
			"dnat": schema.ListNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"external_port": schema.StringAttribute{
							Required: true,
						},
						"internal_ip": schema.StringAttribute{
							Required: true,
						},
						"internal_port": schema.StringAttribute{
							Required: true,
						},
						"protocol": schema.StringAttribute{
							Required: true,
						},
					},
				},
			},
		},
	}
}

func (r *eipResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *eipResource) getParams(data *eipResourceModel) (string, string, string) {
	orgaId := r.client.OrgaId
	az := r.client.Az
	projectId := r.client.ProjectId

	if !data.OrgaId.IsNull() && !data.OrgaId.IsUnknown() {
		orgaId = data.OrgaId.ValueString()
	}
	if !data.Az.IsNull() && !data.Az.IsUnknown() {
		az = data.Az.ValueString()
	}
	if !data.ProjectId.IsNull() && !data.ProjectId.IsUnknown() {
		projectId = data.ProjectId.ValueString()
	}

	return orgaId, az, projectId
}

func (r *eipResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data eipResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	orgaId, az, projectId := r.getParams(&data)

	eipReq := client.EipRequest{
		Name:       data.Name.ValueString(),
		SubnetId:   data.SubnetId.ValueString(),
		InternalIP: data.InternalIP.ValueString(),
	}

	if !data.SNAT.IsNull() && !data.SNAT.IsUnknown() {
		var snat []string
		resp.Diagnostics.Append(data.SNAT.ElementsAs(ctx, &snat, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		eipReq.SNAT = snat
	}

	for _, d := range data.DNAT {
		eipReq.DNAT = append(eipReq.DNAT, client.EipDNAT{
			ExternalPort: d.ExternalPort.ValueString(),
			InternalIP:   d.InternalIP.ValueString(),
			InternalPort: d.InternalPort.ValueString(),
			Protocol:     d.Protocol.ValueString(),
		})
	}

	eip, err := r.client.CreateEip(ctx, orgaId, az, projectId, eipReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create EIP, got error: %s", err))
		return
	}

	r.mapEipToModel(ctx, &data, eip, orgaId, az, projectId, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *eipResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data eipResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	orgaId, az, projectId := r.getParams(&data)

	eip, err := r.client.GetEip(ctx, orgaId, az, projectId, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read EIP, got error: %s", err))
		return
	}

	if eip == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	r.mapEipToModel(ctx, &data, eip, orgaId, az, projectId, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *eipResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data eipResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	orgaId, az, projectId := r.getParams(&data)

	eipReq := client.EipRequest{
		Name:       data.Name.ValueString(),
		SubnetId:   data.SubnetId.ValueString(),
		InternalIP: data.InternalIP.ValueString(),
	}

	if !data.SNAT.IsNull() && !data.SNAT.IsUnknown() {
		var snat []string
		resp.Diagnostics.Append(data.SNAT.ElementsAs(ctx, &snat, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		eipReq.SNAT = snat
	}

	for _, d := range data.DNAT {
		eipReq.DNAT = append(eipReq.DNAT, client.EipDNAT{
			ExternalPort: d.ExternalPort.ValueString(),
			InternalIP:   d.InternalIP.ValueString(),
			InternalPort: d.InternalPort.ValueString(),
			Protocol:     d.Protocol.ValueString(),
		})
	}

	eip, err := r.client.UpdateEip(ctx, orgaId, az, projectId, data.ID.ValueString(), eipReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update EIP, got error: %s", err))
		return
	}

	r.mapEipToModel(ctx, &data, eip, orgaId, az, projectId, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *eipResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data eipResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	orgaId, az, projectId := r.getParams(&data)

	err := r.client.DeleteEip(ctx, orgaId, az, projectId, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete EIP, got error: %s", err))
		return
	}
}

func (r *eipResource) mapEipToModel(ctx context.Context, data *eipResourceModel, eip *client.Eip, orgaId, az, projectId string, diags *diag.Diagnostics) {
	data.ID = types.StringValue(eip.ID)
	data.Name = types.StringValue(eip.Name)
	data.SubnetId = types.StringValue(eip.SubnetId)
	data.InternalIP = types.StringValue(eip.InternalIP)
	data.ExternalIP = types.StringValue(eip.ExternalIP)
	data.Nat = types.StringValue(eip.Nat)
	data.Az = types.StringValue(az)
	data.OrgaId = types.StringValue(orgaId)
	data.ProjectId = types.StringValue(projectId)

	if len(eip.SNAT) > 0 {
		snatList, d := types.ListValueFrom(ctx, types.StringType, eip.SNAT)
		diags.Append(d...)
		data.SNAT = snatList
	} else {
		data.SNAT = types.ListNull(types.StringType)
	}

	data.DNAT = make([]eipDNATModel, 0, len(eip.DNAT))
	for _, d := range eip.DNAT {
		data.DNAT = append(data.DNAT, eipDNATModel{
			ExternalPort: types.StringValue(d.ExternalPort),
			InternalIP:   types.StringValue(d.InternalIP),
			InternalPort: types.StringValue(d.InternalPort),
			Protocol:     types.StringValue(d.Protocol),
		})
	}
}
