package provider

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/Agora-Calyce/terraform-provider-spx/internal/client"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var lbSelectorRegexp = regexp.MustCompile(`^[^:]+:.*$`)

type loadBalancerResource struct {
	client *client.Client
}

type loadBalancerResourceModel struct {
	ID        types.String            `tfsdk:"id"`
	Name      types.String            `tfsdk:"name"`
	VIP       types.String            `tfsdk:"vip"`
	Selectors []types.String          `tfsdk:"selectors"`
	Endpoints types.List              `tfsdk:"endpoints"`
	Ports     []loadBalancerPortModel `tfsdk:"ports"`
	Az        types.String            `tfsdk:"az"`
	OrgaId    types.String            `tfsdk:"orga_id"`
	ProjectId types.String            `tfsdk:"project_id"`
}

type loadBalancerPortModel struct {
	Name       types.String `tfsdk:"name"`
	Port       types.Int32  `tfsdk:"port"`
	TargetPort types.Int32  `tfsdk:"target_port"`
	Protocol   types.String `tfsdk:"protocol"`
}

func NewLoadBalancerResource() resource.Resource {
	return &loadBalancerResource{}
}

func (r *loadBalancerResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_load_balancer"
}

func (r *loadBalancerResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Load Balancer on SPX.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Description: "The ID of the Load Balancer.",
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the Load Balancer.",
			},
			"vip": schema.StringAttribute{
				Required:    true,
				Description: "The Virtual IP of the Load Balancer.",
			},
			"endpoints": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Description: "The list of IP endpoints.",
			},
			"az": schema.StringAttribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Description: "The availability zone of the Load Balancer. If not set, the provider value is used.",
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
			"selectors": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Description: "The list of selectors (key:value).",
				Validators: []validator.List{
					listvalidator.ValueStringsAre(stringvalidator.RegexMatches(lbSelectorRegexp, "must use the key:value format")),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"ports": schema.ListNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Computed:    true,
							Description: "The name of the port (computed).",
						},
						"port": schema.Int32Attribute{
							Required: true,
						},
						"target_port": schema.Int32Attribute{
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

func (r *loadBalancerResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *loadBalancerResource) getParams(data *loadBalancerResourceModel) (string, string, string) {
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

func (r *loadBalancerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data loadBalancerResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	orgaId, az, projectId := r.getParams(&data)

	lbReq := lbRequestFromModel(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	lb, err := r.client.CreateLoadBalancer(ctx, orgaId, az, projectId, lbReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create Load Balancer, got error: %s", err))
		return
	}

	r.mapLBToModel(ctx, &data, lb, orgaId, az, projectId, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *loadBalancerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data loadBalancerResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	orgaId, az, projectId := r.getParams(&data)

	lb, err := r.client.GetLoadBalancer(ctx, orgaId, az, projectId, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read Load Balancer, got error: %s", err))
		return
	}

	if lb == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	r.mapLBToModel(ctx, &data, lb, orgaId, az, projectId, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *loadBalancerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data loadBalancerResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	orgaId, az, projectId := r.getParams(&data)

	lbReq := lbRequestFromModel(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	lb, err := r.client.UpdateLoadBalancer(ctx, orgaId, az, projectId, data.ID.ValueString(), lbReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update Load Balancer, got error: %s", err))
		return
	}

	r.mapLBToModel(ctx, &data, lb, orgaId, az, projectId, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *loadBalancerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data loadBalancerResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	orgaId, az, projectId := r.getParams(&data)

	err := r.client.DeleteLoadBalancer(ctx, orgaId, az, projectId, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete Load Balancer, got error: %s", err))
		return
	}
}

func (r *loadBalancerResource) mapLBToModel(ctx context.Context, data *loadBalancerResourceModel, lb *client.LoadBalancer, orgaId, az, projectId string, diags *diag.Diagnostics) {
	data.ID = types.StringValue(lb.ID)
	data.Name = types.StringValue(lb.Name)
	data.VIP = types.StringValue(lb.VIP)
	data.Az = types.StringValue(az)
	data.OrgaId = types.StringValue(orgaId)
	data.ProjectId = types.StringValue(projectId)

	if len(lb.Endpoints) > 0 {
		endpointsList, d := types.ListValueFrom(ctx, types.StringType, lb.Endpoints)
		diags.Append(d...)
		data.Endpoints = endpointsList
	} else {
		data.Endpoints = types.ListNull(types.StringType)
	}

	// The API represents selectors as {key, value}; the Terraform schema as "key:value"
	if len(lb.Selectors) > 0 {
		data.Selectors = make([]types.String, 0, len(lb.Selectors))
		for _, s := range lb.Selectors {
			data.Selectors = append(data.Selectors, types.StringValue(s.Key+":"+s.Value))
		}
	} else {
		data.Selectors = nil
	}

	data.Ports = make([]loadBalancerPortModel, 0, len(lb.Ports))
	for _, p := range lb.Ports {
		data.Ports = append(data.Ports, loadBalancerPortModel{
			Name:       types.StringValue(p.Name),
			Port:       types.Int32Value(p.Port),
			TargetPort: types.Int32Value(p.TargetPort),
			Protocol:   types.StringValue(p.Protocol),
		})
	}
}

// lbRequestFromModel builds the create/update request body from the Terraform model.
func lbRequestFromModel(ctx context.Context, data *loadBalancerResourceModel, diags *diag.Diagnostics) client.LoadBalancerRequest {
	lbReq := client.LoadBalancerRequest{
		Name: data.Name.ValueString(),
		VIP:  data.VIP.ValueString(),
	}

	for _, s := range data.Selectors {
		key, value, ok := strings.Cut(s.ValueString(), ":")
		if !ok {
			diags.AddError("Invalid selector", fmt.Sprintf("selector %q must use the key:value format", s.ValueString()))
			continue
		}
		lbReq.Selectors = append(lbReq.Selectors, client.LBSelector{Key: key, Value: value})
	}

	if !data.Endpoints.IsNull() && !data.Endpoints.IsUnknown() {
		var endpoints []string
		diags.Append(data.Endpoints.ElementsAs(ctx, &endpoints, false)...)
		lbReq.Endpoints = endpoints
	}

	for _, p := range data.Ports {
		lbReq.Ports = append(lbReq.Ports, client.LoadBalancerPort{
			Port:       p.Port.ValueInt32(),
			TargetPort: p.TargetPort.ValueInt32(),
			Protocol:   p.Protocol.ValueString(),
		})
	}

	return lbReq
}
