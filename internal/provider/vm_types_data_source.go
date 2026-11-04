package provider

import (
	"context"
	"fmt"

	"github.com/Agora-Calyce/terraform-provider-spx/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &vmTypesDataSource{}
	_ datasource.DataSourceWithConfigure = &vmTypesDataSource{}
)

// NewVMTypesDataSource is a helper function to simplify the provider implementation.
func NewVMTypesDataSource() datasource.DataSource {
	return &vmTypesDataSource{}
}

// vmTypesDataSource is the data source implementation.
type vmTypesDataSource struct {
	client *client.Client
}

// vmTypesDataSourceModel maps the data source schema data.
type vmTypesDataSourceModel struct {
	Types types.List `tfsdk:"types"`
}

// Metadata returns the data source type name.
func (d *vmTypesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vm_types"
}

// Schema defines the schema for the data source.
func (d *vmTypesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves the list of Instance types available for a project and an AZ.",
		Attributes: map[string]schema.Attribute{
			"types": schema.ListAttribute{
				Description: "The list of available Instance types.",
				Computed:    true,
				ElementType: types.StringType,
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *vmTypesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	d.client = client
}

// Read refreshes the Terraform state with the latest data.
func (d *vmTypesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state vmTypesDataSourceModel

	vmTypes, err := d.client.ListVMTypes(ctx, d.client.OrgaId, d.client.Az, d.client.ProjectId)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read SPX VM Types",
			err.Error(),
		)
		return
	}

	vmTypesList, diags := types.ListValueFrom(ctx, types.StringType, vmTypes)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state.Types = vmTypesList

	// Set state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
