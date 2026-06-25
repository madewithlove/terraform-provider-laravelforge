package resource_team_invites

// This file implements the team_invites resource. Unlike *_resource_gen.go it
// is NOT regenerated, so the CRUD logic below is safe to edit.

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/madewithlove/terraform-provider-laravelforge/internal/forge"
	"github.com/madewithlove/terraform-provider-laravelforge/internal/providerutil"
)

var (
	_ resource.Resource                = (*teamInvitesResource)(nil)
	_ resource.ResourceWithConfigure   = (*teamInvitesResource)(nil)
	_ resource.ResourceWithImportState = (*teamInvitesResource)(nil)
)

// NewTeamInvitesResource returns a new team_invites resource.
func NewTeamInvitesResource() resource.Resource {
	return &teamInvitesResource{}
}

type teamInvitesResource struct {
	client *forge.Client
}

func (r *teamInvitesResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_team_invites"
}

func (r *teamInvitesResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = providerutil.Client(req, resp)
}

func (r *teamInvitesResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	s := TeamInvitesResourceSchema(ctx)

	// An invitation is scoped to a team within an organization (both path
	// parameters). The invitation is immutable: email and role_id are set only at
	// creation time and cannot be updated in place.
	providerutil.Required(s.Attributes, "organization")
	providerutil.Required(s.Attributes, "team")
	providerutil.RequiresReplace(s.Attributes, "email", "role_id")
	providerutil.UseStateForUnknown(s.Attributes, "id", "invitation")

	s.MarkdownDescription = "Invite a new member to the team."
	resp.Schema = s
}

func (r *teamInvitesResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan TeamInvitesModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]any{
		"email":   plan.Email.ValueString(),
		"role_id": plan.RoleId.ValueInt64(),
	}

	collection := fmt.Sprintf("/orgs/%s/teams/%s/invites",
		plan.Organization.ValueString(), plan.Team.ValueString())

	// Creation is synchronous and returns the created invitation.
	var data inviteData
	if err := r.client.Do(ctx, http.MethodPost, collection, body, &data); err != nil {
		resp.Diagnostics.AddError("Unable to create team invite", err.Error())
		return
	}

	data.apply(&plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *teamInvitesResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state TeamInvitesModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var data inviteData
	endpoint := fmt.Sprintf("/orgs/%s/teams/%s/invites/%s",
		state.Organization.ValueString(), state.Team.ValueString(), state.Id.ValueString())
	if err := r.client.Do(ctx, http.MethodGet, endpoint, nil, &data); err != nil {
		if forge.NotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Unable to read team invite", err.Error())
		return
	}

	data.apply(&state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update is required by the interface but is never invoked with in-place
// changes: email and role_id are marked RequiresReplace, so Terraform destroys
// and recreates the invitation instead.
func (r *teamInvitesResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan TeamInvitesModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *teamInvitesResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state TeamInvitesModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	endpoint := fmt.Sprintf("/orgs/%s/teams/%s/invites/%s",
		state.Organization.ValueString(), state.Team.ValueString(), state.Id.ValueString())
	if err := r.client.Do(ctx, http.MethodDelete, endpoint, nil, nil); err != nil && !forge.NotFound(err) {
		resp.Diagnostics.AddError("Unable to delete team invite", err.Error())
	}
}

func (r *teamInvitesResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts, err := providerutil.ParseID(req.ID, 3)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID",
			fmt.Sprintf("Expected import ID in the form \"organization/team/invitation\": %s.", err))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("organization"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("team"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[2])...)
}

// inviteData is the JSON:API representation of a team invitation. role is a
// relationship identifier returned as a plain string; role_id is a write-only
// input and is not echoed back.
type inviteData struct {
	ID         string `json:"id"`
	Attributes struct {
		Email     string  `json:"email"`
		Role      *string `json:"role"`
		CreatedAt string  `json:"created_at"`
		UpdatedAt string  `json:"updated_at"`
	} `json:"attributes"`
}

func (d *inviteData) apply(m *TeamInvitesModel) {
	m.Id = types.StringValue(d.ID)
	m.Email = types.StringValue(d.Attributes.Email)
	m.CreatedAt = types.StringValue(d.Attributes.CreatedAt)
	m.UpdatedAt = types.StringValue(d.Attributes.UpdatedAt)
	if d.Attributes.Role != nil {
		m.Role = types.StringValue(*d.Attributes.Role)
	} else {
		m.Role = types.StringNull()
	}
	if id, err := providerutil.ParseInt64(d.ID); err == nil {
		m.Invitation = types.Int64Value(id)
	}
}
