package provider

import (
	"context"
	"fmt"
	"sort"

	"github.com/Agora-Calyce/terraform-provider-spx/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &StorageClassesDataSource{}

func NewStorageClassesDataSource() datasource.DataSource {
	return &StorageClassesDataSource{}
}

type StorageClassesDataSource struct {
	client *client.Client
}

type StorageClassesDataSourceModel struct {
	Names types.List `tfsdk:"names"`
}

func (d *StorageClassesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_storage_classes"
}

func (d *StorageClassesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Récupère la liste des StorageClasses disponibles pour un projet et une AZ.",
		Attributes: map[string]schema.Attribute{
			"names": schema.ListAttribute{
				ElementType: types.StringType,
				Computed:    true,
				Description: "Liste des noms de StorageClasses.",
			},
		},
	}
}

func (d *StorageClassesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *StorageClassesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data StorageClassesDataSourceModel

	storageClasses, err := d.client.ListStorageClasses(ctx, d.client.OrgaId, d.client.Az, d.client.ProjectId)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list storage classes, got error: %s", err))
		return
	}

	// Ordre stable : l'API ne garantit pas l'ordre, ce qui changerait names[0] d'un plan à l'autre
	sort.Strings(storageClasses)

	names, diags := types.ListValueFrom(ctx, types.StringType, storageClasses)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.Names = names

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
