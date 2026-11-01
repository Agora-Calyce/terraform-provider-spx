package provider

import (
	"context"
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"

	"github.com/Agora-Calyce/terraform-provider-spx/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// customLabelRegexp valide un label utilisateur SPX : "user.superphenix.net/clé:valeur" (un seul ':')
var customLabelRegexp = regexp.MustCompile(`^user\.superphenix\.net/[^:]+:[^:]*$`)

var _ resource.Resource = &DiskResource{}

func NewDiskResource() resource.Resource {
	return &DiskResource{}
}

type DiskResource struct {
	client *client.Client
}

type DiskResourceModel struct {
	ID           types.String     `tfsdk:"id"`
	Name         types.String     `tfsdk:"name"`
	Size         types.Int64      `tfsdk:"size"`
	StorageClass types.String     `tfsdk:"storage_class"`
	AZ           types.String     `tfsdk:"az"`
	Source       *DiskSourceModel `tfsdk:"source"`
	Labels       []types.String   `tfsdk:"labels"`
}

type DiskSourceModel struct {
	Type     types.String `tfsdk:"type"`
	URL      types.String `tfsdk:"url"`
	Clone    types.String `tfsdk:"clone"`
	Snapshot types.String `tfsdk:"snapshot"`
}

func (r *DiskResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_disk"
}

func (r *DiskResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Gère une ressource Disque sur SPX.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "L'ID effectif du disque.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Le nom du disque.",
			},
			"size": schema.Int64Attribute{
				Required:    true,
				Description: "La taille du disque en Gi (ex: 10).",
			},
			"storage_class": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "La StorageClass à utiliser pour le disque.",
				Default:     stringdefault.StaticString("default"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"az": schema.StringAttribute{
				Computed:    true,
				Description: "La zone de disponibilité du disque.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"labels": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Description: "Labels du disque au format 'user.superphenix.net/clé:valeur'.",
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
					listvalidator.ValueStringsAre(stringvalidator.RegexMatches(customLabelRegexp, "keys must start with 'user.superphenix.net/' and use the key:value format")),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"source": schema.SingleNestedBlock{
				Description: "La source du disque.",
				Attributes: map[string]schema.Attribute{
					"type": schema.StringAttribute{
						Required:    true,
						Description: "Le type de source (blank, http, snapshot, clone, registry).",
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
					},
					"url": schema.StringAttribute{
						Optional:    true,
						Description: "L'URL de la source (obligatoire pour http et registry).",
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
					},
					"clone": schema.StringAttribute{
						Optional:    true,
						Description: "L'EID du disque à cloner (obligatoire pour clone).",
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
					},
					"snapshot": schema.StringAttribute{
						Optional:    true,
						Description: "L'EID du snapshot à utiliser (obligatoire pour snapshot).",
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
					},
				},
			},
		},
	}
}

func (r *DiskResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *DiskResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data DiskResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	diskReq := buildDiskRequest(data)

	disk, err := r.client.CreateDisk(ctx, r.client.OrgaId, r.client.Az, r.client.ProjectId, diskReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create disk, got error: %s", err))
		return
	}

	data.ID = types.StringValue(disk.ID)
	data.AZ = types.StringValue(disk.Az)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DiskResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data DiskResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	disk, err := r.client.GetDisk(ctx, r.client.OrgaId, r.client.Az, r.client.ProjectId, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read disk, got error: %s", err))
		return
	}

	if disk == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	data.Name = types.StringValue(disk.Name)
	data.Size = types.Int64Value(int64(disk.Size))
	data.StorageClass = types.StringValue(disk.StorageClass)
	data.AZ = types.StringValue(disk.Az)
	data.Labels = mapStringsFromClient(disk.Labels)

	if disk.Source != nil {
		// "" => null : sinon url/clone/snapshot (RequiresReplace) diffèrent de la config et forcent le remplacement du disque
		data.Source = &DiskSourceModel{
			Type:     types.StringValue(disk.Source.Type),
			URL:      optionalStringFromClient(disk.Source.URL),
			Clone:    optionalStringFromClient(disk.Source.Clone),
			Snapshot: optionalStringFromClient(disk.Source.Snapshot),
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DiskResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data DiskResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// storage_class et source ne sont pas modifiables, mais le proxy les valide aussi à la mise à jour
	diskReq := buildDiskRequest(data)

	disk, err := r.client.UpdateDisk(ctx, r.client.OrgaId, r.client.Az, r.client.ProjectId, data.ID.ValueString(), diskReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update disk, got error: %s", err))
		return
	}

	// az est calculé : il ne doit pas rester inconnu dans l'état
	if data.AZ.IsUnknown() {
		az := r.client.Az
		if disk != nil && disk.Az != "" {
			az = disk.Az
		}
		data.AZ = types.StringValue(az)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DiskResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data DiskResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteDisk(ctx, r.client.OrgaId, r.client.Az, r.client.ProjectId, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete disk, got error: %s", err))
		return
	}
}

func buildDiskRequest(data DiskResourceModel) client.DiskRequest {
	diskReq := client.DiskRequest{
		Name:         data.Name.ValueString(),
		Size:         int(data.Size.ValueInt64()),
		StorageClass: data.StorageClass.ValueString(),
		Labels:       mapStringsToClient(data.Labels),
	}

	if data.Source != nil {
		diskReq.Source = &client.DiskSource{
			Type:     data.Source.Type.ValueString(),
			URL:      data.Source.URL.ValueString(),
			Clone:    data.Source.Clone.ValueString(),
			Snapshot: data.Source.Snapshot.ValueString(),
		}
	}

	return diskReq
}
