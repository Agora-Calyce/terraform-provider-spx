package provider

import (
	"context"
	"fmt"
	"net"

	"github.com/Agora-Calyce/terraform-provider-spx/internal/client"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &vpcStaticRoutesResource{}
var _ resource.ResourceWithImportState = &vpcStaticRoutesResource{}

// vpcStaticRoutesResource manages the full set of static routes of a VPC.
// It is a separate resource from spx_vpc because each route references a subnet of the VPC
// (route_table), which would create a dependency cycle if the routes were held by the VPC.
type vpcStaticRoutesResource struct {
	client *client.Client
}

type vpcStaticRoutesResourceModel struct {
	ID        types.String       `tfsdk:"id"`
	VPCID     types.String       `tfsdk:"vpc_id"`
	Routes    []staticRouteModel `tfsdk:"route"`
	Az        types.String       `tfsdk:"az"`
	OrgaId    types.String       `tfsdk:"orga_id"`
	ProjectId types.String       `tfsdk:"project_id"`
}

type staticRouteModel struct {
	Policy     types.String `tfsdk:"policy"`
	CIDR       types.String `tfsdk:"cidr"`
	NextHopIP  types.String `tfsdk:"next_hop_ip"`
	RouteTable types.String `tfsdk:"route_table"`
}

func NewVPCStaticRoutesResource() resource.Resource {
	return &vpcStaticRoutesResource{}
}

func (r *vpcStaticRoutesResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vpc_static_routes"
}

func (r *vpcStaticRoutesResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages the full set of static routes of an SPX VPC. Routes not declared here are removed; " +
			"routes generated automatically by NAT Gateways are not affected.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Same as vpc_id.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"vpc_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the VPC.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"az": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The availability zone. If not set, the provider value is used.",
				PlanModifiers: []planmodifier.String{
					// Without UseStateForUnknown, the unconfigured value is unknown at plan time and forces a replacement
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"orga_id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The organization ID. If not set, the provider value is used.",
				PlanModifiers: []planmodifier.String{
					// Without UseStateForUnknown, the unconfigured value is unknown at plan time and forces a replacement
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"project_id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The project ID. If not set, the provider value is used.",
				PlanModifiers: []planmodifier.String{
					// Without UseStateForUnknown, the unconfigured value is unknown at plan time and forces a replacement
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"route": schema.SetNestedBlock{
				Description: "A static route.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"policy": schema.StringAttribute{
							Optional:    true,
							Computed:    true,
							Default:     stringdefault.StaticString("policyDst"),
							Description: "The routing policy: policyDst (default, routes on the destination) or policySrc.",
							Validators: []validator.String{
								stringvalidator.OneOf("policyDst", "policySrc"),
							},
						},
						"cidr": schema.StringAttribute{
							Required:    true,
							Description: "The destination CIDR (or source CIDR for policySrc), in canonical form (e.g. 192.168.0.0/24).",
							Validators:  []validator.String{canonicalCIDRValidator{}},
						},
						"next_hop_ip": schema.StringAttribute{
							Required:    true,
							Description: "The next hop IP. Must belong to one of the VPC subnets.",
							Validators:  []validator.String{ipAddressValidator{}},
						},
						"route_table": schema.StringAttribute{
							Required:    true,
							Description: "The ID of the VPC subnet the route applies to.",
						},
					},
				},
			},
		},
	}
}

func (r *vpcStaticRoutesResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *vpcStaticRoutesResource) getParams(data *vpcStaticRoutesResourceModel) (string, string, string) {
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

// putRoutes replaces the VPC routes. The proxy requires the VPC name: it is read back so it stays unchanged.
func (r *vpcStaticRoutesResource) putRoutes(ctx context.Context, orgaId, az, projectId, vpcId string, routes []client.StaticRoute) (*client.VPC, error) {
	vpc, err := r.client.GetVPC(ctx, orgaId, az, projectId, vpcId)
	if err != nil {
		return nil, err
	}
	if vpc == nil {
		return nil, fmt.Errorf("VPC %s not found", vpcId)
	}
	if routes == nil {
		routes = []client.StaticRoute{}
	}
	return r.client.UpdateVPC(ctx, orgaId, az, projectId, vpcId, client.VPCRequest{
		Name:         vpc.Name,
		StaticRoutes: &routes,
	})
}

func (r *vpcStaticRoutesResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data vpcStaticRoutesResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgaId, az, projectId := r.getParams(&data)

	vpc, err := r.putRoutes(ctx, orgaId, az, projectId, data.VPCID.ValueString(), staticRoutesToClient(data.Routes))
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to set VPC static routes, got error: %s", err))
		return
	}

	data.ID = data.VPCID
	data.Az = types.StringValue(az)
	data.OrgaId = types.StringValue(orgaId)
	data.ProjectId = types.StringValue(projectId)
	data.Routes = staticRoutesFromClient(vpc.StaticRoutes)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *vpcStaticRoutesResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data vpcStaticRoutesResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// After an import, only the ID (= vpc_id) is known
	if data.VPCID.IsNull() || data.VPCID.ValueString() == "" {
		data.VPCID = data.ID
	}
	orgaId, az, projectId := r.getParams(&data)

	vpc, err := r.client.GetVPC(ctx, orgaId, az, projectId, data.VPCID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read VPC static routes, got error: %s", err))
		return
	}
	if vpc == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	data.ID = data.VPCID
	data.Az = types.StringValue(az)
	data.OrgaId = types.StringValue(orgaId)
	data.ProjectId = types.StringValue(projectId)
	data.Routes = staticRoutesFromClient(vpc.StaticRoutes)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *vpcStaticRoutesResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data vpcStaticRoutesResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgaId, az, projectId := r.getParams(&data)

	vpc, err := r.putRoutes(ctx, orgaId, az, projectId, data.VPCID.ValueString(), staticRoutesToClient(data.Routes))
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update VPC static routes, got error: %s", err))
		return
	}

	data.ID = data.VPCID
	data.Routes = staticRoutesFromClient(vpc.StaticRoutes)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *vpcStaticRoutesResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data vpcStaticRoutesResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgaId, az, projectId := r.getParams(&data)

	vpc, err := r.client.GetVPC(ctx, orgaId, az, projectId, data.VPCID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read VPC before removing static routes, got error: %s", err))
		return
	}
	if vpc == nil {
		// VPC already deleted: nothing to do
		return
	}

	if _, err := r.putRoutes(ctx, orgaId, az, projectId, data.VPCID.ValueString(), []client.StaticRoute{}); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to remove VPC static routes, got error: %s", err))
		return
	}
}

func (r *vpcStaticRoutesResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("vpc_id"), req.ID)...)
}

func staticRoutesToClient(routes []staticRouteModel) []client.StaticRoute {
	res := make([]client.StaticRoute, 0, len(routes))
	for _, rt := range routes {
		res = append(res, client.StaticRoute{
			Policy:     rt.Policy.ValueString(),
			CIDR:       rt.CIDR.ValueString(),
			NextHopIP:  rt.NextHopIP.ValueString(),
			RouteTable: rt.RouteTable.ValueString(),
		})
	}
	return res
}

func staticRoutesFromClient(routes []client.StaticRoute) []staticRouteModel {
	if len(routes) == 0 {
		return nil
	}
	res := make([]staticRouteModel, 0, len(routes))
	for _, rt := range routes {
		policy := rt.Policy
		if policy == "" {
			policy = "policyDst"
		}
		res = append(res, staticRouteModel{
			Policy:     types.StringValue(policy),
			CIDR:       types.StringValue(rt.CIDR),
			NextHopIP:  types.StringValue(rt.NextHopIP),
			RouteTable: types.StringValue(rt.RouteTable),
		})
	}
	return res
}

// canonicalCIDRValidator requires a valid, canonical CIDR (network address), because the SPX API
// normalizes CIDRs: a non-canonical form would cause a permanent diff.
type canonicalCIDRValidator struct{}

func (v canonicalCIDRValidator) Description(ctx context.Context) string {
	return "must be a canonical CIDR (e.g. 192.168.0.0/24)"
}

func (v canonicalCIDRValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v canonicalCIDRValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	value := req.ConfigValue.ValueString()
	ip, network, err := net.ParseCIDR(value)
	if err != nil {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid CIDR", fmt.Sprintf("%q is not a valid CIDR.", value))
		return
	}
	if !ip.Equal(network.IP) {
		resp.Diagnostics.AddAttributeError(req.Path, "Non-canonical CIDR",
			fmt.Sprintf("%q is not in canonical form, use %q.", value, network.String()))
	}
}

// ipAddressValidator requires a valid IPv4 or IPv6 address
type ipAddressValidator struct{}

func (v ipAddressValidator) Description(ctx context.Context) string {
	return "must be a valid IP address"
}

func (v ipAddressValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v ipAddressValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	if net.ParseIP(req.ConfigValue.ValueString()) == nil {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid IP Address",
			fmt.Sprintf("%q is not a valid IP address.", req.ConfigValue.ValueString()))
	}
}
