package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/Agora-Calyce/terraform-provider-spx/internal/client"

	"github.com/hashicorp/terraform-plugin-framework-validators/int32validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &SecurityGroupResource{}
var _ resource.ResourceWithImportState = &SecurityGroupResource{}
var _ resource.ResourceWithValidateConfig = &SecurityGroupResource{}
var _ resource.ResourceWithMoveState = &SecurityGroupResource{}

// matchLabelRegexp valide un label au format "clé:valeur"
var matchLabelRegexp = regexp.MustCompile(`^[^:]+:.*$`)

// legacyFirewallTypeName est l'ancien nom de la ressource, avant le renommage upstream Firewall -> Security Group
const legacyFirewallTypeName = "spx_firewall"

func NewSecurityGroupResource() resource.Resource {
	return &SecurityGroupResource{}
}

type SecurityGroupResource struct {
	client *client.Client
}

type SecurityGroupResourceModel struct {
	ID          types.String       `tfsdk:"id"`
	Name        types.String       `tfsdk:"name"`
	Description types.String       `tfsdk:"description"`
	Target      *TargetModel       `tfsdk:"target"` // pointeur : null après un import
	Ingress     []IngressRuleModel `tfsdk:"ingress"`
	Egress      []EgressRuleModel  `tfsdk:"egress"`
	SubnetIDs   []types.String     `tfsdk:"subnet_ids"`
	OrgaId      types.String       `tfsdk:"orga_id"`
	Az          types.String       `tfsdk:"az"`
	ProjectId   types.String       `tfsdk:"project_id"`
}

type TargetModel struct {
	MatchLabels      []types.String         `tfsdk:"match_labels"`
	MatchExpressions []MatchExpressionModel `tfsdk:"match_expressions"`
}

type MatchExpressionModel struct {
	Key      types.String   `tfsdk:"key"`
	Operator types.String   `tfsdk:"operator"`
	Values   []types.String `tfsdk:"values"`
}

type IngressRuleModel struct {
	Ports    []SgPortModel `tfsdk:"ports"`
	From     []PeerModel   `tfsdk:"from"`
	AllowAll types.Bool    `tfsdk:"allow_all"`
	DenyAll  types.Bool    `tfsdk:"deny_all"`
}

type EgressRuleModel struct {
	Ports    []SgPortModel `tfsdk:"ports"`
	To       []PeerModel   `tfsdk:"to"`
	AllowAll types.Bool    `tfsdk:"allow_all"`
	DenyAll  types.Bool    `tfsdk:"deny_all"`
}

type SgPortModel struct {
	Port     types.Int32  `tfsdk:"port"`
	EndPort  types.Int32  `tfsdk:"end_port"`
	Protocol types.String `tfsdk:"protocol"`
}

type PeerModel struct {
	PodSelector *TargetModel  `tfsdk:"pod_selector"`
	IPBlock     *IPBlockModel `tfsdk:"ip_block"`
}

type IPBlockModel struct {
	CIDR   types.String   `tfsdk:"cidr"`
	Except []types.String `tfsdk:"except"`
}

func (r *SecurityGroupResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_security_group"
}

func (r *SecurityGroupResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Gère un groupe de sécurité (anciennement pare-feu, `spx_firewall`) sur SPX.",
		MarkdownDescription: "Gère un groupe de sécurité (anciennement pare-feu, `spx_firewall`) sur SPX.\n\n" +
			"## Migration depuis `spx_firewall`\n\n" +
			"La ressource `spx_firewall` a été renommée `spx_security_group`. Pour migrer sans recréer le groupe " +
			"de sécurité (Terraform >= 1.8), renommez le bloc `resource` puis ajoutez un bloc `moved` :\n\n" +
			"```terraform\nmoved {\n  from = spx_firewall.example\n  to   = spx_security_group.example\n}\n```\n\n" +
			"`terraform plan` doit alors indiquer que la ressource a été déplacée, sans autre modification.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "ID effectif du groupe de sécurité.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Nom du groupe de sécurité.",
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 63),
				},
			},
			"description": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Description du groupe de sécurité.",
			},
			"subnet_ids": schema.ListAttribute{
				ElementType:         types.StringType,
				Optional:            true,
				MarkdownDescription: "IDs des subnets auxquels le groupe de sécurité s'applique (10 maximum). Si non spécifié, il s'applique à tous les subnets du projet.",
				Validators: []validator.List{
					listvalidator.SizeBetween(1, 10),
				},
			},
			"orga_id": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "ID de l'organisation. Si non spécifié, utilise la valeur du provider.",
				PlanModifiers: []planmodifier.String{
					// Sans UseStateForUnknown, la valeur non configurée est inconnue au plan et force un remplacement
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"az": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Zone de disponibilité. Si non spécifié, utilise la valeur du provider.",
				PlanModifiers: []planmodifier.String{
					// Sans UseStateForUnknown, la valeur non configurée est inconnue au plan et force un remplacement
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"project_id": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "ID du projet. Si non spécifié, utilise la valeur du provider.",
				PlanModifiers: []planmodifier.String{
					// Sans UseStateForUnknown, la valeur non configurée est inconnue au plan et force un remplacement
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"target": schema.SingleNestedBlock{
				MarkdownDescription: "Sélecteur des instances ciblées par le groupe de sécurité.",
				Attributes:          labelSelectorAttributes(),
				Blocks:              labelSelectorBlocks(),
			},
			"ingress": schema.ListNestedBlock{
				MarkdownDescription: "Règles de trafic entrant.",
				NestedObject:        ruleNestedBlock("from"),
			},
			"egress": schema.ListNestedBlock{
				MarkdownDescription: "Règles de trafic sortant.",
				NestedObject:        ruleNestedBlock("to"),
			},
		},
	}
}

// ruleNestedBlock construit le schéma d'une règle ingress/egress ; peerBlockName vaut "from" ou "to".
func ruleNestedBlock(peerBlockName string) schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"allow_all": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
			},
			"deny_all": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
			},
		},
		Blocks: map[string]schema.Block{
			"ports": schema.ListNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"port": schema.Int32Attribute{
							Required: true,
							Validators: []validator.Int32{
								int32validator.Between(1, 65535),
							},
						},
						"end_port": schema.Int32Attribute{
							Optional:            true,
							Computed:            true,
							MarkdownDescription: "Fin de la plage de ports. Vaut `port` si non spécifié.",
							PlanModifiers: []planmodifier.Int32{
								endPortPlanModifier{},
							},
							Validators: []validator.Int32{
								int32validator.Between(1, 65535),
								int32validator.AtLeastSumOf(path.MatchRelative().AtParent().AtName("port")),
							},
						},
						"protocol": schema.StringAttribute{
							Optional:            true,
							Computed:            true,
							Default:             stringdefault.StaticString("TCP"),
							MarkdownDescription: "Protocole : `TCP` (défaut), `UDP` ou `SCTP`.",
							Validators: []validator.String{
								stringvalidator.OneOfCaseInsensitive("TCP", "UDP", "SCTP"),
							},
							PlanModifiers: []planmodifier.String{
								caseInsensitivePlanModifier{},
							},
						},
					},
				},
			},
			peerBlockName: schema.ListNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Blocks: map[string]schema.Block{
						"pod_selector": schema.SingleNestedBlock{
							Attributes: labelSelectorAttributes(),
							Blocks:     labelSelectorBlocks(),
						},
						"ip_block": schema.SingleNestedBlock{
							Attributes: map[string]schema.Attribute{
								"cidr":   schema.StringAttribute{Optional: true},
								"except": schema.ListAttribute{ElementType: types.StringType, Optional: true},
							},
						},
					},
				},
			},
		},
	}
}

func labelSelectorAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"match_labels": schema.ListAttribute{
			ElementType:         types.StringType,
			Optional:            true,
			MarkdownDescription: "Liste de labels de correspondance au format `clé:valeur`.",
			Validators: []validator.List{
				listvalidator.ConflictsWith(path.MatchRelative().AtParent().AtName("match_expressions")),
				listvalidator.ValueStringsAre(stringvalidator.RegexMatches(matchLabelRegexp, "doit être au format clé:valeur")),
			},
		},
	}
}

func labelSelectorBlocks() map[string]schema.Block {
	return map[string]schema.Block{
		"match_expressions": schema.ListNestedBlock{
			NestedObject: schema.NestedBlockObject{
				Attributes: map[string]schema.Attribute{
					"key": schema.StringAttribute{Required: true},
					"operator": schema.StringAttribute{
						Required: true,
						Validators: []validator.String{
							stringvalidator.OneOf("In", "NotIn", "Exists", "DoesNotExists"),
						},
					},
					"values": schema.ListAttribute{ElementType: types.StringType, Optional: true},
				},
			},
		},
	}
}

func (r *SecurityGroupResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Data Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

func (r *SecurityGroupResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var model SecurityGroupResourceModel
	diags := req.Config.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	hasLabels := model.Target != nil && len(model.Target.MatchLabels) > 0
	hasExpressions := model.Target != nil && len(model.Target.MatchExpressions) > 0
	if !hasLabels && !hasExpressions {
		resp.Diagnostics.AddAttributeError(
			path.Root("target"),
			"Configuration Invalide",
			"Le bloc 'target' doit contenir soit 'match_labels', soit 'match_expressions'.",
		)
	}

	if len(model.Ingress) > 1 {
		for _, ingress := range model.Ingress {
			if !ingress.AllowAll.IsNull() && ingress.AllowAll.ValueBool() {
				resp.Diagnostics.AddError(
					"Configuration d'ingress invalide",
					"Une règle ingress.allow_all=true ne peut pas coexister avec d'autres règles ingress.",
				)
				break
			}
		}
	}

	if len(model.Egress) > 1 {
		for _, egress := range model.Egress {
			if !egress.AllowAll.IsNull() && egress.AllowAll.ValueBool() {
				resp.Diagnostics.AddError(
					"Configuration d'egress invalide",
					"Une règle egress.allow_all=true ne peut pas coexister avec d'autres règles egress.",
				)
				break
			}
		}
	}

	for _, ingress := range model.Ingress {
		for _, from := range ingress.From {
			if !peerHasSelector(from) {
				resp.Diagnostics.AddError(
					"Configuration Invalide",
					"Chaque bloc 'from' dans une règle ingress doit contenir soit 'pod_selector', soit 'ip_block.cidr'.",
				)
			}
		}
	}

	for _, egress := range model.Egress {
		for _, to := range egress.To {
			if !peerHasSelector(to) {
				resp.Diagnostics.AddError(
					"Configuration Invalide",
					"Chaque bloc 'to' dans une règle egress doit contenir soit 'pod_selector', soit 'ip_block.cidr'.",
				)
			}
		}
	}
}

func peerHasSelector(p PeerModel) bool {
	hasPodSelector := p.PodSelector != nil && (len(p.PodSelector.MatchLabels) > 0 || len(p.PodSelector.MatchExpressions) > 0)
	// Une valeur inconnue (issue d'une autre ressource) est considérée comme renseignée
	hasIPBlock := p.IPBlock != nil && (p.IPBlock.CIDR.IsUnknown() || (!p.IPBlock.CIDR.IsNull() && p.IPBlock.CIDR.ValueString() != ""))
	return hasPodSelector || hasIPBlock
}

// scope retourne orga/az/projet du modèle, ou les valeurs du provider à défaut
func (r *SecurityGroupResource) scope(data *SecurityGroupResourceModel) (string, string, string) {
	orgaId := data.OrgaId.ValueString()
	if orgaId == "" {
		orgaId = r.client.OrgaId
	}
	az := data.Az.ValueString()
	if az == "" {
		az = r.client.Az
	}
	projectId := data.ProjectId.ValueString()
	if projectId == "" {
		projectId = r.client.ProjectId
	}
	return orgaId, az, projectId
}

func buildSecurityGroupRequest(data SecurityGroupResourceModel) client.SecurityGroupRequest {
	req := client.SecurityGroupRequest{
		Name:        data.Name.ValueString(),
		Description: data.Description.ValueString(),
		Ingress:     mapIngressToClient(data.Ingress),
		Egress:      mapEgressToClient(data.Egress),
		SubnetIDs:   mapStringsToClient(data.SubnetIDs),
	}
	if data.Target != nil {
		req.Target = mapTargetToClient(*data.Target)
	}
	return req
}

// applySecurityGroupFromClient met à jour le modèle à partir de la réponse de l'API,
// en conservant la casse du protocole saisie par l'utilisateur.
func applySecurityGroupFromClient(data *SecurityGroupResourceModel, sg *client.SecurityGroup) {
	prior := *data
	data.Name = types.StringValue(sg.Name)
	if sg.Description != "" || !data.Description.IsNull() {
		data.Description = types.StringValue(sg.Description)
	}
	target := mapTargetFromClient(sg.Target)
	data.Target = &target
	data.Ingress = mapIngressFromClient(sg.Ingress)
	data.Egress = mapEgressFromClient(sg.Egress)
	data.SubnetIDs = mapStringsFromClient(sg.SubnetIDs)

	for i := range data.Ingress {
		if i < len(prior.Ingress) {
			preserveProtocolCase(prior.Ingress[i].Ports, data.Ingress[i].Ports)
		}
	}
	for i := range data.Egress {
		if i < len(prior.Egress) {
			preserveProtocolCase(prior.Egress[i].Ports, data.Egress[i].Ports)
		}
	}
}

func preserveProtocolCase(prior, fresh []SgPortModel) {
	for i := range fresh {
		if i < len(prior) && strings.EqualFold(prior[i].Protocol.ValueString(), fresh[i].Protocol.ValueString()) {
			fresh[i].Protocol = prior[i].Protocol
		}
	}
}

func (r *SecurityGroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data SecurityGroupResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgaId, az, projectId := r.scope(&data)

	sg, err := r.client.CreateSecurityGroup(ctx, orgaId, az, projectId, buildSecurityGroupRequest(data))
	if err != nil {
		resp.Diagnostics.AddError("Erreur lors de la création du groupe de sécurité", err.Error())
		return
	}

	data.ID = types.StringValue(sg.ID)
	data.OrgaId = types.StringValue(orgaId)
	data.Az = types.StringValue(az)
	data.ProjectId = types.StringValue(projectId)
	applySecurityGroupFromClient(&data, sg)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SecurityGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data SecurityGroupResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Après un import, seul l'ID est connu : on complète avec les valeurs du provider
	orgaId, az, projectId := r.scope(&data)

	sg, err := r.client.GetSecurityGroup(ctx, orgaId, az, projectId, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Erreur lors de la lecture du groupe de sécurité", err.Error())
		return
	}

	if sg == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	data.OrgaId = types.StringValue(orgaId)
	data.Az = types.StringValue(az)
	data.ProjectId = types.StringValue(projectId)
	applySecurityGroupFromClient(&data, sg)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SecurityGroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data SecurityGroupResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	sg, err := r.client.UpdateSecurityGroup(ctx, data.OrgaId.ValueString(), data.Az.ValueString(), data.ProjectId.ValueString(), data.ID.ValueString(), buildSecurityGroupRequest(data))
	if err != nil {
		resp.Diagnostics.AddError("Erreur lors de la mise à jour du groupe de sécurité", err.Error())
		return
	}

	applySecurityGroupFromClient(&data, sg)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SecurityGroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data SecurityGroupResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteSecurityGroup(ctx, data.OrgaId.ValueString(), data.Az.ValueString(), data.ProjectId.ValueString(), data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Erreur lors de la suppression du groupe de sécurité", err.Error())
		return
	}
}

func (r *SecurityGroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// MoveState permet de migrer une ressource spx_firewall existante avec un bloc `moved` :
//
//	moved {
//	  from = spx_firewall.example
//	  to   = spx_security_group.example
//	}
//
// L'état brut est relu manuellement car l'ancien schéma a varié (match_labels sous forme
// de chaînes "clé:valeur" ou d'objets {key, value}).
func (r *SecurityGroupResource) MoveState(ctx context.Context) []resource.StateMover {
	return []resource.StateMover{
		{
			StateMover: func(ctx context.Context, req resource.MoveStateRequest, resp *resource.MoveStateResponse) {
				if req.SourceTypeName != legacyFirewallTypeName || !strings.HasSuffix(req.SourceProviderAddress, "/spx") {
					// Pas une source gérée par ce mover : le framework renverra une erreur explicite
					return
				}
				if req.SourceRawState == nil {
					resp.Diagnostics.AddError("État source manquant", "Impossible de migrer spx_firewall : état source absent.")
					return
				}

				model, err := legacyFirewallStateToModel(req.SourceRawState.JSON)
				if err != nil {
					resp.Diagnostics.AddError("Impossible de migrer spx_firewall vers spx_security_group", err.Error())
					return
				}

				resp.Diagnostics.Append(resp.TargetState.Set(ctx, model)...)
			},
		},
	}
}

// --- Migration de l'ancien état spx_firewall ---

type legacyLabelSelector struct {
	MatchLabels      []json.RawMessage `json:"match_labels"`
	MatchExpressions []struct {
		Key      string   `json:"key"`
		Operator string   `json:"operator"`
		Values   []string `json:"values"`
	} `json:"match_expressions"`
}

type legacyPort struct {
	Port     json.RawMessage `json:"port"`
	EndPort  json.RawMessage `json:"end_port"`
	Protocol *string         `json:"protocol"`
}

type legacyPeer struct {
	PodSelector *legacyLabelSelector `json:"pod_selector"`
	IPBlock     *struct {
		CIDR   *string  `json:"cidr"`
		Except []string `json:"except"`
	} `json:"ip_block"`
}

type legacyRule struct {
	Ports    []legacyPort `json:"ports"`
	From     []legacyPeer `json:"from"`
	To       []legacyPeer `json:"to"`
	AllowAll *bool        `json:"allow_all"`
	DenyAll  *bool        `json:"deny_all"`
}

type legacyFirewallState struct {
	ID          *string              `json:"id"`
	Name        *string              `json:"name"`
	Description *string              `json:"description"`
	OrgaId      *string              `json:"orga_id"`
	Az          *string              `json:"az"`
	ProjectId   *string              `json:"project_id"`
	Target      *legacyLabelSelector `json:"target"`
	Ingress     []legacyRule         `json:"ingress"`
	Egress      []legacyRule         `json:"egress"`
}

func legacyFirewallStateToModel(raw []byte) (SecurityGroupResourceModel, error) {
	var st legacyFirewallState
	if err := json.Unmarshal(raw, &st); err != nil {
		return SecurityGroupResourceModel{}, fmt.Errorf("état spx_firewall illisible : %w", err)
	}
	if st.ID == nil || *st.ID == "" {
		return SecurityGroupResourceModel{}, fmt.Errorf("l'état spx_firewall ne contient pas d'id")
	}

	model := SecurityGroupResourceModel{
		ID:          types.StringPointerValue(st.ID),
		Name:        types.StringPointerValue(st.Name),
		Description: types.StringPointerValue(st.Description),
		OrgaId:      types.StringPointerValue(st.OrgaId),
		Az:          types.StringPointerValue(st.Az),
		ProjectId:   types.StringPointerValue(st.ProjectId),
	}

	if st.Target != nil {
		t, err := legacySelectorToModel(*st.Target)
		if err != nil {
			return model, err
		}
		model.Target = &t
	}

	for _, r := range st.Ingress {
		ports, err := legacyPortsToModel(r.Ports)
		if err != nil {
			return model, err
		}
		peers, err := legacyPeersToModel(r.From)
		if err != nil {
			return model, err
		}
		model.Ingress = append(model.Ingress, IngressRuleModel{
			Ports:    ports,
			From:     peers,
			AllowAll: types.BoolPointerValue(r.AllowAll),
			DenyAll:  types.BoolPointerValue(r.DenyAll),
		})
	}
	for _, r := range st.Egress {
		ports, err := legacyPortsToModel(r.Ports)
		if err != nil {
			return model, err
		}
		peers, err := legacyPeersToModel(r.To)
		if err != nil {
			return model, err
		}
		model.Egress = append(model.Egress, EgressRuleModel{
			Ports:    ports,
			To:       peers,
			AllowAll: types.BoolPointerValue(r.AllowAll),
			DenyAll:  types.BoolPointerValue(r.DenyAll),
		})
	}

	return model, nil
}

func legacySelectorToModel(ls legacyLabelSelector) (TargetModel, error) {
	m := TargetModel{}
	for _, raw := range ls.MatchLabels {
		var s string
		if err := json.Unmarshal(raw, &s); err == nil {
			m.MatchLabels = append(m.MatchLabels, types.StringValue(s))
			continue
		}
		var kv struct {
			Key   string `json:"key"`
			Value string `json:"value"`
		}
		if err := json.Unmarshal(raw, &kv); err != nil {
			return m, fmt.Errorf("match_labels illisible : %s", string(raw))
		}
		m.MatchLabels = append(m.MatchLabels, types.StringValue(kv.Key+":"+kv.Value))
	}
	for _, e := range ls.MatchExpressions {
		m.MatchExpressions = append(m.MatchExpressions, MatchExpressionModel{
			Key:      types.StringValue(e.Key),
			Operator: types.StringValue(e.Operator),
			Values:   mapStringsFromClient(e.Values),
		})
	}
	return m, nil
}

func legacyPortsToModel(ports []legacyPort) ([]SgPortModel, error) {
	var res []SgPortModel
	for _, p := range ports {
		port, err := legacyInt32(p.Port)
		if err != nil {
			return nil, fmt.Errorf("port illisible : %w", err)
		}
		endPort, err := legacyInt32(p.EndPort)
		if err != nil {
			return nil, fmt.Errorf("end_port illisible : %w", err)
		}
		protocol := types.StringValue("TCP")
		if p.Protocol != nil && *p.Protocol != "" {
			protocol = types.StringValue(*p.Protocol)
		}
		res = append(res, SgPortModel{Port: port, EndPort: endPort, Protocol: protocol})
	}
	return res, nil
}

// legacyInt32 accepte un nombre, une chaîne numérique ou null
func legacyInt32(raw json.RawMessage) (types.Int32, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return types.Int32Null(), nil
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		if s == "" {
			return types.Int32Null(), nil
		}
		n, err := strconv.ParseInt(s, 10, 32)
		if err != nil {
			return types.Int32Null(), err
		}
		return types.Int32Value(int32(n)), nil
	}
	var n int32
	if err := json.Unmarshal(raw, &n); err != nil {
		return types.Int32Null(), err
	}
	return types.Int32Value(n), nil
}

func legacyPeersToModel(peers []legacyPeer) ([]PeerModel, error) {
	var res []PeerModel
	for _, p := range peers {
		peer := PeerModel{}
		if p.PodSelector != nil {
			ls, err := legacySelectorToModel(*p.PodSelector)
			if err != nil {
				return nil, err
			}
			peer.PodSelector = &ls
		}
		if p.IPBlock != nil {
			peer.IPBlock = &IPBlockModel{
				CIDR:   types.StringPointerValue(p.IPBlock.CIDR),
				Except: mapStringsFromClient(p.IPBlock.Except),
			}
		}
		res = append(res, peer)
	}
	return res, nil
}

// --- Helpers de conversion modèle <-> client ---

func mapStringsToClient(values []types.String) []string {
	if values == nil {
		return nil
	}
	res := make([]string, len(values))
	for i, v := range values {
		res[i] = v.ValueString()
	}
	return res
}

// mapStringsFromClient retourne nil (null côté Terraform) pour une liste vide
func mapStringsFromClient(values []string) []types.String {
	if len(values) == 0 {
		return nil
	}
	res := make([]types.String, len(values))
	for i, v := range values {
		res[i] = types.StringValue(v)
	}
	return res
}

func mapTargetToClient(m TargetModel) client.LabelSelector {
	ls := client.LabelSelector{}
	for _, l := range m.MatchLabels {
		ls.MatchLabels = append(ls.MatchLabels, l.ValueString())
	}
	for _, e := range m.MatchExpressions {
		ls.MatchExpressions = append(ls.MatchExpressions, client.MatchExpression{
			Key:      e.Key.ValueString(),
			Operator: e.Operator.ValueString(),
			Values:   mapStringsToClient(e.Values),
		})
	}
	return ls
}

func mapTargetFromClient(ls client.LabelSelector) TargetModel {
	m := TargetModel{}
	for _, l := range ls.MatchLabels {
		m.MatchLabels = append(m.MatchLabels, types.StringValue(l))
	}
	for _, e := range ls.MatchExpressions {
		m.MatchExpressions = append(m.MatchExpressions, MatchExpressionModel{
			Key:      types.StringValue(e.Key),
			Operator: types.StringValue(e.Operator),
			Values:   mapStringsFromClient(e.Values),
		})
	}
	return m
}

func mapIngressToClient(rules []IngressRuleModel) []client.IngressRule {
	if rules == nil {
		return nil
	}
	res := make([]client.IngressRule, len(rules))
	for i, r := range rules {
		res[i] = client.IngressRule{
			AllowAll: r.AllowAll.ValueBool(),
			DenyAll:  r.DenyAll.ValueBool(),
			Ports:    mapPortsToClient(r.Ports),
			From:     mapPeersToClient(r.From),
		}
	}
	return res
}

func mapIngressFromClient(rules []client.IngressRule) []IngressRuleModel {
	if rules == nil {
		return nil
	}
	res := make([]IngressRuleModel, len(rules))
	for i, r := range rules {
		res[i] = IngressRuleModel{
			AllowAll: types.BoolValue(r.AllowAll),
			DenyAll:  types.BoolValue(r.DenyAll),
			Ports:    mapPortsFromClient(r.Ports),
			From:     mapPeersFromClient(r.From),
		}
	}
	return res
}

func mapEgressToClient(rules []EgressRuleModel) []client.EgressRule {
	if rules == nil {
		return nil
	}
	res := make([]client.EgressRule, len(rules))
	for i, r := range rules {
		res[i] = client.EgressRule{
			AllowAll: r.AllowAll.ValueBool(),
			DenyAll:  r.DenyAll.ValueBool(),
			Ports:    mapPortsToClient(r.Ports),
			To:       mapPeersToClient(r.To),
		}
	}
	return res
}

func mapEgressFromClient(rules []client.EgressRule) []EgressRuleModel {
	if rules == nil {
		return nil
	}
	res := make([]EgressRuleModel, len(rules))
	for i, r := range rules {
		res[i] = EgressRuleModel{
			AllowAll: types.BoolValue(r.AllowAll),
			DenyAll:  types.BoolValue(r.DenyAll),
			Ports:    mapPortsFromClient(r.Ports),
			To:       mapPeersFromClient(r.To),
		}
	}
	return res
}

func mapPortsToClient(ports []SgPortModel) []client.SgPort {
	if ports == nil {
		return nil
	}
	res := make([]client.SgPort, len(ports))
	for i, p := range ports {
		// end_port null/inconnu => 0, le proxy le normalise à la valeur de port
		res[i] = client.SgPort{
			Port:     p.Port.ValueInt32(),
			EndPort:  p.EndPort.ValueInt32(),
			Protocol: strings.ToUpper(p.Protocol.ValueString()),
		}
	}
	return res
}

func mapPortsFromClient(ports []client.SgPort) []SgPortModel {
	if ports == nil {
		return nil
	}
	res := make([]SgPortModel, len(ports))
	for i, p := range ports {
		endPort := p.EndPort
		if endPort == 0 {
			endPort = p.Port
		}
		protocol := p.Protocol
		if protocol == "" {
			protocol = "TCP"
		}
		res[i] = SgPortModel{
			Port:     types.Int32Value(p.Port),
			EndPort:  types.Int32Value(endPort),
			Protocol: types.StringValue(protocol),
		}
	}
	return res
}

func mapPeersToClient(peers []PeerModel) []client.Peer {
	if peers == nil {
		return nil
	}
	res := make([]client.Peer, len(peers))
	for i, p := range peers {
		if p.PodSelector != nil && (len(p.PodSelector.MatchLabels) > 0 || len(p.PodSelector.MatchExpressions) > 0) {
			ls := mapTargetToClient(*p.PodSelector)
			res[i].PodSelector = &ls
		}
		if p.IPBlock != nil && !p.IPBlock.CIDR.IsNull() && p.IPBlock.CIDR.ValueString() != "" {
			res[i].IPBlock = &client.IPBlock{
				CIDR:   p.IPBlock.CIDR.ValueString(),
				Except: mapStringsToClient(p.IPBlock.Except),
			}
		}
	}
	return res
}

func mapPeersFromClient(peers []client.Peer) []PeerModel {
	if peers == nil {
		return nil
	}
	res := make([]PeerModel, len(peers))
	for i, p := range peers {
		if p.PodSelector != nil {
			ls := mapTargetFromClient(*p.PodSelector)
			if len(ls.MatchLabels) > 0 || len(ls.MatchExpressions) > 0 {
				res[i].PodSelector = &ls
			}
		}
		if p.IPBlock != nil && p.IPBlock.CIDR != "" {
			res[i].IPBlock = &IPBlockModel{
				CIDR:   types.StringValue(p.IPBlock.CIDR),
				Except: mapStringsFromClient(p.IPBlock.Except),
			}
		}
	}
	return res
}

// caseInsensitivePlanModifier conserve la valeur de l'état quand la configuration ne diffère que par la casse
// (l'API renvoie le protocole en majuscules, y compris après un import).
type caseInsensitivePlanModifier struct{}

func (m caseInsensitivePlanModifier) Description(ctx context.Context) string {
	return "Ignore les différences de casse avec la valeur de l'état."
}

func (m caseInsensitivePlanModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m caseInsensitivePlanModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if req.StateValue.IsNull() || req.StateValue.IsUnknown() || req.PlanValue.IsNull() || req.PlanValue.IsUnknown() {
		return
	}
	if req.PlanValue.ValueString() != req.StateValue.ValueString() && strings.EqualFold(req.PlanValue.ValueString(), req.StateValue.ValueString()) {
		resp.PlanValue = req.StateValue
	}
}

// endPortPlanModifier : end_port non configuré vaut port côté API. Tant que port ne change pas,
// on reprend la valeur de l'état plutôt que de l'afficher comme inconnue à chaque plan.
// (UseStateForUnknown serait faux : si port change, end_port doit être recalculé.)
type endPortPlanModifier struct{}

func (m endPortPlanModifier) Description(ctx context.Context) string {
	return "Conserve end_port de l'état tant que port est inchangé."
}

func (m endPortPlanModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m endPortPlanModifier) PlanModifyInt32(ctx context.Context, req planmodifier.Int32Request, resp *planmodifier.Int32Response) {
	if !req.ConfigValue.IsNull() || !req.PlanValue.IsUnknown() || req.StateValue.IsNull() || req.StateValue.IsUnknown() {
		return
	}
	portPath := req.Path.ParentPath().AtName("port")
	var planPort, statePort types.Int32
	if diags := req.Plan.GetAttribute(ctx, portPath, &planPort); diags.HasError() {
		return
	}
	if diags := req.State.GetAttribute(ctx, portPath, &statePort); diags.HasError() {
		return
	}
	if !planPort.IsUnknown() && planPort.Equal(statePort) && req.StateValue.Equal(statePort) {
		resp.PlanValue = req.StateValue
	}
}
