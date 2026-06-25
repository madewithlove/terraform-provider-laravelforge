package resource_site_commands

// This file implements the site_commands resource. Unlike *_resource_gen.go it
// is NOT regenerated, so the CRUD logic below is safe to edit.

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/madewithlove/terraform-provider-laravelforge/internal/forge"
	"github.com/madewithlove/terraform-provider-laravelforge/internal/providerutil"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = (*siteCommandsResource)(nil)
	_ resource.ResourceWithConfigure   = (*siteCommandsResource)(nil)
	_ resource.ResourceWithImportState = (*siteCommandsResource)(nil)
)

// NewSiteCommandsResource returns a new site_commands resource.
func NewSiteCommandsResource() resource.Resource {
	return &siteCommandsResource{}
}

type siteCommandsResource struct {
	client *forge.Client
}

func (r *siteCommandsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_site_commands"
}

func (r *siteCommandsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = providerutil.Client(req, resp)
}

func (r *siteCommandsResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	s := SiteCommandsResourceSchema(ctx)

	providerutil.Required(s.Attributes, "organization")
	providerutil.Required(s.Attributes, "server")
	providerutil.Required(s.Attributes, "site")

	// The generator emits "command" as computed-only, but it is in fact the
	// (required, write-only) input that defines the command to run. Make it a
	// required input that forces replacement, since commands cannot be updated.
	if a, ok := s.Attributes["command"].(schema.StringAttribute); ok {
		a.Computed = false
		s.Attributes["command"] = a
	}
	providerutil.Required(s.Attributes, "command")

	// "opcache_enabled" is a spurious field in the upstream spec for this
	// resource; relax it to optional+computed so it never has to be configured.
	if a, ok := s.Attributes["opcache_enabled"].(schema.BoolAttribute); ok {
		a.Required, a.Optional, a.Computed = false, true, true
		s.Attributes["opcache_enabled"] = a
	}

	providerutil.UseStateForUnknown(s.Attributes, "id")

	s.MarkdownDescription = "Run a command on the site."
	resp.Schema = s
}

func (r *siteCommandsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan SiteCommandsModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]any{
		"command": plan.Command.ValueString(),
	}

	collection := fmt.Sprintf("/orgs/%s/servers/%d/sites/%d/commands",
		plan.Organization.ValueString(), plan.Server.ValueInt64(), plan.Site.ValueInt64())

	// Running a command is asynchronous (202) and returns no body, so snapshot
	// the existing commands, fire the request, then resolve the new id.
	before, err := providerutil.ListIDs(ctx, r.client, collection)
	if err != nil {
		resp.Diagnostics.AddError("Unable to list site commands", err.Error())
		return
	}
	if err := r.client.Do(ctx, http.MethodPost, collection, body, nil); err != nil {
		resp.Diagnostics.AddError("Unable to run site command", err.Error())
		return
	}
	id, err := providerutil.ResolveCreatedID(ctx, r.client, collection, "", before)
	if err != nil {
		resp.Diagnostics.AddError("Unable to locate site command", err.Error())
		return
	}

	var data commandData
	if err := r.client.Do(ctx, http.MethodGet, collection+"/"+id, nil, &data); err != nil {
		resp.Diagnostics.AddError("Unable to read site command", err.Error())
		return
	}

	data.apply(&plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *siteCommandsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state SiteCommandsModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var data commandData
	endpoint := fmt.Sprintf("/orgs/%s/servers/%d/sites/%d/commands/%s",
		state.Organization.ValueString(), state.Server.ValueInt64(), state.Site.ValueInt64(), state.Id.ValueString())
	if err := r.client.Do(ctx, http.MethodGet, endpoint, nil, &data); err != nil {
		if forge.NotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Unable to read site command", err.Error())
		return
	}

	data.apply(&state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update is never invoked with real changes: every configurable attribute
// forces replacement. It simply persists the plan.
func (r *siteCommandsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan SiteCommandsModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *siteCommandsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state SiteCommandsModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	endpoint := fmt.Sprintf("/orgs/%s/servers/%d/sites/%d/commands/%s",
		state.Organization.ValueString(), state.Server.ValueInt64(), state.Site.ValueInt64(), state.Id.ValueString())
	if err := r.client.Do(ctx, http.MethodDelete, endpoint, nil, nil); err != nil && !forge.NotFound(err) {
		resp.Diagnostics.AddError("Unable to delete site command", err.Error())
	}
}

// ImportState supports `terraform import` using the
// "organization/server/site/command" identifier form.
func (r *siteCommandsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts, err := providerutil.ParseID(req.ID, 4)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID",
			fmt.Sprintf("Expected import ID in the form \"organization/server/site/command\": %s.", err))
		return
	}
	server, err := providerutil.ParseInt64(parts[1])
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("Server must be an integer: %s.", err))
		return
	}
	site, err := providerutil.ParseInt64(parts[2])
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("Site must be an integer: %s.", err))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("organization"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("server"), server)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("site"), site)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[3])...)
}

// commandData is the JSON:API representation of a site command.
type commandData struct {
	ID         string `json:"id"`
	Attributes struct {
		Command        string `json:"command"`
		Status         string `json:"status"`
		Duration       string `json:"duration"`
		UserID         *int64 `json:"user_id"`
		OpcacheEnabled *bool  `json:"opcache_enabled"`
		CreatedAt      string `json:"created_at"`
		UpdatedAt      string `json:"updated_at"`
	} `json:"attributes"`
}

// apply copies the API response onto the model. The submitted "command" is
// echoed back by the API, so it is safe to set here.
func (d *commandData) apply(m *SiteCommandsModel) {
	m.Id = types.StringValue(d.ID)
	m.Command = types.StringValue(d.Attributes.Command)
	m.Status = types.StringValue(d.Attributes.Status)
	m.Duration = types.StringValue(d.Attributes.Duration)
	m.CreatedAt = types.StringValue(d.Attributes.CreatedAt)
	m.UpdatedAt = types.StringValue(d.Attributes.UpdatedAt)
	// "user" is returned as a nested resource identifier object that does not
	// map onto the model's string attribute; leave it null but known.
	m.User = types.StringNull()
	if d.Attributes.UserID != nil {
		m.UserId = types.Int64Value(*d.Attributes.UserID)
	} else {
		m.UserId = types.Int64Null()
	}
	if d.Attributes.OpcacheEnabled != nil {
		m.OpcacheEnabled = types.BoolValue(*d.Attributes.OpcacheEnabled)
	} else {
		m.OpcacheEnabled = types.BoolNull()
	}
}
