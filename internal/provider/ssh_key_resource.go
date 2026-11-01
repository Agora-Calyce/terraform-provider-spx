package provider

import (
	"context"
	"fmt"
	"github.com/Agora-Calyce/terraform-provider-spx/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &SSHKeyResource{}

type SSHKeyResource struct {
	client *client.Client
}

type SSHKeyResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	PublicKey   types.String `tfsdk:"public_key"`
	Fingerprint types.String `tfsdk:"fingerprint"`
}

func NewSSHKeyResource() resource.Resource {
	return &SSHKeyResource{}
}

func (r *SSHKeyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ssh_key"
}

func (r *SSHKeyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Gère une clé SSH dans SPX.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Description: "ID effectif de la clé SSH.",
			},
			"name": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Description: "Nom de la clé SSH.",
			},
			"public_key": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Description: "Contenu de la clé publique SSH.",
			},
			"fingerprint": schema.StringAttribute{
				Computed:    true,
				Description: "Empreinte digitale de la clé publique.",
			},
		},
	}
}

func (r *SSHKeyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *SSHKeyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data SSHKeyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	sshKeyReq := client.SSHKeyRequest{
		Name:      data.Name.ValueString(),
		PublicKey: data.PublicKey.ValueString(),
	}

	sshKey, err := r.client.CreateSSHKey(ctx, r.client.OrgaId, r.client.Az, r.client.ProjectId, sshKeyReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating SSH Key", err.Error())
		return
	}

	data.ID = types.StringValue(sshKey.ID)
	data.Name = types.StringValue(sshKey.Name)
	data.PublicKey = types.StringValue(sshKey.PublicKey)
	data.Fingerprint = types.StringValue(sshKey.Fingerprint)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SSHKeyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data SSHKeyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	sshKey, err := r.client.GetSSHKey(ctx, r.client.OrgaId, r.client.Az, r.client.ProjectId, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading SSH Key", err.Error())
		return
	}

	if sshKey == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	data.Name = types.StringValue(sshKey.Name)
	data.PublicKey = types.StringValue(sshKey.PublicKey)
	data.Fingerprint = types.StringValue(sshKey.Fingerprint)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SSHKeyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Non supporté par SPX, RequiresReplace est utilisé sur les attributs.
}

func (r *SSHKeyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data SSHKeyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteSSHKey(ctx, r.client.OrgaId, r.client.Az, r.client.ProjectId, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error deleting SSH Key", err.Error())
		return
	}
}
