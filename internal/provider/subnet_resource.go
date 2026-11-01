package provider

import (
	"context"
	"fmt"

	"github.com/Agora-Calyce/terraform-provider-spx/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type subnetResource struct {
	client *client.Client
}

type subnetResourceModel struct {
	ID           types.String `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	VPCID        types.String `tfsdk:"vpc_id"`
	IPv4         types.String `tfsdk:"ipv4"`
	IPv6         types.String `tfsdk:"ipv6"`
	Private      types.Bool   `tfsdk:"private"`
	Protocol     types.String `tfsdk:"protocol"`
	DNSV4        types.String `tfsdk:"dns_v4"`
	DNSV6        types.String `tfsdk:"dns_v6"`
	NatGateway   types.Bool   `tfsdk:"nat_gateway"`
	NatGatewayIp types.String `tfsdk:"nat_gateway_ip"`
	Az           types.String `tfsdk:"az"`
	OrgaId       types.String `tfsdk:"orga_id"`
	ProjectId    types.String `tfsdk:"project_id"`
}

func NewSubnetResource() resource.Resource {
	return &subnetResource{}
}

func (r *subnetResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_subnet"
}

func (r *subnetResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Gère une ressource de type Subnet sur SPX.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Description: "The ID of the Subnet.",
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the Subnet.",
			},
			"vpc_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the parent VPC.",
			},
			"ipv4": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The IPv4 CIDR block.",
			},
			"ipv6": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The IPv6 CIDR block.",
			},
			"private": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
				Description: "Whether the subnet is private.",
			},
			"protocol": schema.StringAttribute{
				Required:    true,
				Description: "The protocol (IPv4, IPv6, Dual).",
			},
			"dns_v4": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The IPv4 DNS server.",
			},
			"dns_v6": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The IPv6 DNS server.",
			},
			"nat_gateway": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
				Description: "Whether to enable NAT Gateway.",
			},
			"nat_gateway_ip": schema.StringAttribute{
				Computed:    true,
				Description: "The IP address of the NAT Gateway.",
			},
			"az": schema.StringAttribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Description: "The availability zone of the Subnet. If not set, the provider value is used.",
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
	}
}

func (r *subnetResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *subnetResource) getParams(data *subnetResourceModel) (string, string, string) {
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

func (r *subnetResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data subnetResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	orgaId, az, projectId := r.getParams(&data)

	subnetReq := client.SubnetRequest{
		Name:       data.Name.ValueString(),
		VPCID:      data.VPCID.ValueString(),
		IPv4:       data.IPv4.ValueString(),
		IPv6:       data.IPv6.ValueString(),
		Private:    data.Private.ValueBool(),
		Protocol:   data.Protocol.ValueString(),
		DNSV4:      data.DNSV4.ValueString(),
		DNSV6:      data.DNSV6.ValueString(),
		NatGateway: data.NatGateway.ValueBool(),
	}

	subnet, err := r.client.CreateSubnet(ctx, orgaId, az, projectId, subnetReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create Subnet, got error: %s", err))
		return
	}

	data.ID = types.StringValue(subnet.ID)
	data.Name = types.StringValue(subnet.Name)
	data.VPCID = types.StringValue(subnet.VPCID)
	data.IPv4 = types.StringValue(subnet.IPv4)
	data.IPv6 = types.StringValue(subnet.IPv6)
	data.Private = types.BoolValue(subnet.Private)
	data.Protocol = types.StringValue(subnet.Protocol)
	data.DNSV4 = types.StringValue(subnet.DNSV4)
	data.DNSV6 = types.StringValue(subnet.DNSV6)
	data.NatGateway = types.BoolValue(subnet.NatGateway)
	data.NatGatewayIp = types.StringValue(subnet.NatGatewayIp)
	data.Az = types.StringValue(az)
	data.OrgaId = types.StringValue(orgaId)
	data.ProjectId = types.StringValue(projectId)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *subnetResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data subnetResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	orgaId, az, projectId := r.getParams(&data)

	subnet, err := r.client.GetSubnet(ctx, orgaId, az, projectId, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read Subnet, got error: %s", err))
		return
	}

	if subnet == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	data.ID = types.StringValue(subnet.ID)
	data.Name = types.StringValue(subnet.Name)
	data.VPCID = types.StringValue(subnet.VPCID)
	data.IPv4 = types.StringValue(subnet.IPv4)
	data.IPv6 = types.StringValue(subnet.IPv6)
	data.Private = types.BoolValue(subnet.Private)
	data.Protocol = types.StringValue(subnet.Protocol)
	data.DNSV4 = types.StringValue(subnet.DNSV4)
	data.DNSV6 = types.StringValue(subnet.DNSV6)
	data.NatGateway = types.BoolValue(subnet.NatGateway)
	data.NatGatewayIp = types.StringValue(subnet.NatGatewayIp)
	data.Az = types.StringValue(az)
	data.OrgaId = types.StringValue(orgaId)
	data.ProjectId = types.StringValue(projectId)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *subnetResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data subnetResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	orgaId, az, projectId := r.getParams(&data)

	subnetReq := client.SubnetRequest{
		Name:       data.Name.ValueString(),
		VPCID:      data.VPCID.ValueString(),
		IPv4:       data.IPv4.ValueString(),
		IPv6:       data.IPv6.ValueString(),
		Private:    data.Private.ValueBool(),
		Protocol:   data.Protocol.ValueString(),
		DNSV4:      data.DNSV4.ValueString(),
		DNSV6:      data.DNSV6.ValueString(),
		NatGateway: data.NatGateway.ValueBool(),
	}

	subnet, err := r.client.UpdateSubnet(ctx, orgaId, az, projectId, data.ID.ValueString(), subnetReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update Subnet, got error: %s", err))
		return
	}

	data.ID = types.StringValue(subnet.ID)
	data.Name = types.StringValue(subnet.Name)
	data.VPCID = types.StringValue(subnet.VPCID)
	data.IPv4 = types.StringValue(subnet.IPv4)
	data.IPv6 = types.StringValue(subnet.IPv6)
	data.Private = types.BoolValue(subnet.Private)
	data.Protocol = types.StringValue(subnet.Protocol)
	data.DNSV4 = types.StringValue(subnet.DNSV4)
	data.DNSV6 = types.StringValue(subnet.DNSV6)
	data.NatGateway = types.BoolValue(subnet.NatGateway)
	data.NatGatewayIp = types.StringValue(subnet.NatGatewayIp)
	data.Az = types.StringValue(az)
	data.OrgaId = types.StringValue(orgaId)
	data.ProjectId = types.StringValue(projectId)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *subnetResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data subnetResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	orgaId, az, projectId := r.getParams(&data)

	err := r.client.DeleteSubnet(ctx, orgaId, az, projectId, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete Subnet, got error: %s", err))
		return
	}
}
