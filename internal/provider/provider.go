package provider

import (
	"context"
	"os"

	"github.com/Agora-Calyce/terraform-provider-spx/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type spxProvider struct {
	version string
}

type spxProviderModel struct {
	Endpoint  types.String `tfsdk:"endpoint"`
	ApiToken  types.String `tfsdk:"api_token"`
	Az        types.String `tfsdk:"az"`
	OrgaId    types.String `tfsdk:"orga_id"`
	ProjectId types.String `tfsdk:"project_id"`
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &spxProvider{
			version: version,
		}
	}
}

func (p *spxProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "spx"
	resp.Version = p.version
}

func (p *spxProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The SPX provider is used to interact with the Superphenix API.",
		Attributes: map[string]schema.Attribute{
			"endpoint": schema.StringAttribute{
				Optional:    true,
				Description: "SPX API Endpoint. Can also be set via the `SPX_API_ENDPOINT` environment variable.",
			},
			"api_token": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "SPX API Token. Can also be set via the `SPX_API_TOKEN` environment variable.",
			},
			"az": schema.StringAttribute{
				Optional:    true,
				Description: "Availability Zone. Can also be set via the `SPX_AZ` environment variable.",
			},
			"orga_id": schema.StringAttribute{
				Optional:    true,
				Description: "Organization ID. Can also be set via the `SPX_ORGA_ID` environment variable.",
			},
			"project_id": schema.StringAttribute{
				Optional:    true,
				Description: "Project ID. Can also be set via the `SPX_PROJECT_ID` environment variable.",
			},
		},
	}
}

func (p *spxProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data spxProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	endpoint := os.Getenv("SPX_API_ENDPOINT")
	apiToken := os.Getenv("SPX_API_TOKEN")
	az := os.Getenv("SPX_AZ")
	orgaId := os.Getenv("SPX_ORGA_ID")
	projectId := os.Getenv("SPX_PROJECT_ID")

	if !data.Endpoint.IsNull() {
		endpoint = data.Endpoint.ValueString()
	}
	if !data.ApiToken.IsNull() {
		apiToken = data.ApiToken.ValueString()
	}
	if !data.Az.IsNull() {
		az = data.Az.ValueString()
	}
	if !data.OrgaId.IsNull() {
		orgaId = data.OrgaId.ValueString()
	}
	if !data.ProjectId.IsNull() {
		projectId = data.ProjectId.ValueString()
	}

	if endpoint == "" {
		endpoint = "http://localhost:8080"
	}

	client := client.NewClient(endpoint, apiToken, az, orgaId, projectId)

	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *spxProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewVPCResource,
		NewVPCStaticRoutesResource,
		NewSubnetResource,
		NewDiskResource,
		NewEipResource,
		NewLoadBalancerResource,
		NewSSHKeyResource,
		NewInstanceResource,
		NewSecurityGroupResource,
	}
}

func (p *spxProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewStorageClassesDataSource,
		NewVMTypesDataSource,
	}
}
