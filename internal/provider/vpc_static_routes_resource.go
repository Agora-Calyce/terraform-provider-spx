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

// vpcStaticRoutesResource gère l'ensemble des routes statiques d'un VPC.
// C'est une ressource distincte de spx_vpc car chaque route référence un subnet du VPC
// (route_table), ce qui créerait un cycle de dépendances si les routes étaient portées par le VPC.
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
		Description: "Gère l'ensemble des routes statiques d'un VPC SPX. Les routes non déclarées ici sont supprimées ; " +
			"les routes générées automatiquement par les NAT Gateways ne sont pas concernées.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Identique à vpc_id.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"vpc_id": schema.StringAttribute{
				Required:    true,
				Description: "ID du VPC.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"az": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Zone de disponibilité. Si non spécifié, utilise la valeur du provider.",
				PlanModifiers: []planmodifier.String{
					// Sans UseStateForUnknown, la valeur non configurée est inconnue au plan et force un remplacement
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"orga_id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "ID de l'organisation. Si non spécifié, utilise la valeur du provider.",
				PlanModifiers: []planmodifier.String{
					// Sans UseStateForUnknown, la valeur non configurée est inconnue au plan et force un remplacement
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"project_id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "ID du projet. Si non spécifié, utilise la valeur du provider.",
				PlanModifiers: []planmodifier.String{
					// Sans UseStateForUnknown, la valeur non configurée est inconnue au plan et force un remplacement
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"route": schema.SetNestedBlock{
				Description: "Route statique.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"policy": schema.StringAttribute{
							Optional:    true,
							Computed:    true,
							Default:     stringdefault.StaticString("policyDst"),
							Description: "Politique de routage : policyDst (défaut, routage sur la destination) ou policySrc.",
							Validators: []validator.String{
								stringvalidator.OneOf("policyDst", "policySrc"),
							},
						},
						"cidr": schema.StringAttribute{
							Required:    true,
							Description: "CIDR de destination (ou de source pour policySrc), sous forme canonique (ex. 192.168.0.0/24).",
							Validators:  []validator.String{canonicalCIDRValidator{}},
						},
						"next_hop_ip": schema.StringAttribute{
							Required:    true,
							Description: "IP du prochain saut. Doit appartenir à l'un des subnets du VPC.",
							Validators:  []validator.String{ipAddressValidator{}},
						},
						"route_table": schema.StringAttribute{
							Required:    true,
							Description: "ID du subnet du VPC auquel la route s'applique.",
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

// putRoutes remplace les routes du VPC. Le proxy exige le nom du VPC : on le relit pour ne pas le modifier.
func (r *vpcStaticRoutesResource) putRoutes(ctx context.Context, orgaId, az, projectId, vpcId string, routes []client.StaticRoute) (*client.VPC, error) {
	vpc, err := r.client.GetVPC(ctx, orgaId, az, projectId, vpcId)
	if err != nil {
		return nil, err
	}
	if vpc == nil {
		return nil, fmt.Errorf("VPC %s introuvable", vpcId)
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

	// Après un import, seul l'ID (= vpc_id) est connu
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
		// VPC déjà supprimé : rien à faire
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

// canonicalCIDRValidator exige un CIDR valide et canonique (adresse réseau), car l'API SPX
// normalise les CIDR : une forme non canonique provoquerait une différence permanente.
type canonicalCIDRValidator struct{}

func (v canonicalCIDRValidator) Description(ctx context.Context) string {
	return "doit être un CIDR canonique (ex. 192.168.0.0/24)"
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
		resp.Diagnostics.AddAttributeError(req.Path, "CIDR invalide", fmt.Sprintf("%q n'est pas un CIDR valide.", value))
		return
	}
	if !ip.Equal(network.IP) {
		resp.Diagnostics.AddAttributeError(req.Path, "CIDR non canonique",
			fmt.Sprintf("%q n'est pas sous forme canonique, utilisez %q.", value, network.String()))
	}
}

// ipAddressValidator exige une adresse IPv4 ou IPv6 valide
type ipAddressValidator struct{}

func (v ipAddressValidator) Description(ctx context.Context) string {
	return "doit être une adresse IP valide"
}

func (v ipAddressValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v ipAddressValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	if net.ParseIP(req.ConfigValue.ValueString()) == nil {
		resp.Diagnostics.AddAttributeError(req.Path, "Adresse IP invalide",
			fmt.Sprintf("%q n'est pas une adresse IP valide.", req.ConfigValue.ValueString()))
	}
}
