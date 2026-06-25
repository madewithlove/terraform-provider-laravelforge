package resource_php_opcache

// This file implements the php_opcache resource. Unlike *_resource_gen.go it is
// NOT regenerated, so the CRUD logic below is safe to edit.

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
	_ resource.Resource                = (*phpOpcacheResource)(nil)
	_ resource.ResourceWithConfigure   = (*phpOpcacheResource)(nil)
	_ resource.ResourceWithImportState = (*phpOpcacheResource)(nil)
)

// NewPhpOpcacheResource returns a new php_opcache resource.
func NewPhpOpcacheResource() resource.Resource {
	return &phpOpcacheResource{}
}

type phpOpcacheResource struct {
	client *forge.Client
}

func (r *phpOpcacheResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_php_opcache"
}

func (r *phpOpcacheResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = providerutil.Client(req, resp)
}

func (r *phpOpcacheResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	s := PhpOpcacheResourceSchema(ctx)

	// php_opcache is a singleton toggled per server: creating the resource
	// enables opcache and deleting it disables opcache. Its presence in state
	// reflects the enabled state, so opcache_enabled is purely computed.
	providerutil.Required(s.Attributes, "organization")
	providerutil.Required(s.Attributes, "server")
	if a, ok := s.Attributes["opcache_enabled"].(schema.BoolAttribute); ok {
		a.Required, a.Optional, a.Computed = false, false, true
		s.Attributes["opcache_enabled"] = a
	}
	providerutil.UseStateForUnknown(s.Attributes, "id")

	resp.Schema = s
}

// path returns the singleton endpoint (collection == item) for the server.
func (r *phpOpcacheResource) path(organization string, server int64) string {
	return fmt.Sprintf("/orgs/%s/servers/%d/php/opcache", organization, server)
}

func (r *phpOpcacheResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan PhpOpcacheModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	endpoint := r.path(plan.Organization.ValueString(), plan.Server.ValueInt64())

	// Enabling opcache is asynchronous (202) and returns no body.
	if err := r.client.Do(ctx, http.MethodPost, endpoint, nil, nil); err != nil {
		resp.Diagnostics.AddError("Unable to enable opcache", err.Error())
		return
	}

	var data opcacheData
	if err := r.client.Do(ctx, http.MethodGet, endpoint, nil, &data); err != nil {
		resp.Diagnostics.AddError("Unable to read opcache state", err.Error())
		return
	}

	data.apply(&plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *phpOpcacheResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state PhpOpcacheModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var data opcacheData
	endpoint := r.path(state.Organization.ValueString(), state.Server.ValueInt64())
	if err := r.client.Do(ctx, http.MethodGet, endpoint, nil, &data); err != nil {
		if forge.NotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Unable to read opcache state", err.Error())
		return
	}

	data.apply(&state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update is never invoked with real changes: the parents force replacement and
// opcache_enabled is computed. It simply persists the plan.
func (r *phpOpcacheResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan PhpOpcacheModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *phpOpcacheResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state PhpOpcacheModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	endpoint := r.path(state.Organization.ValueString(), state.Server.ValueInt64())
	if err := r.client.Do(ctx, http.MethodDelete, endpoint, nil, nil); err != nil && !forge.NotFound(err) {
		resp.Diagnostics.AddError("Unable to disable opcache", err.Error())
	}
}

// ImportState supports `terraform import` using the "organization/server"
// identifier form. The singleton path needs no separate id, so Read populates
// the remaining attributes.
func (r *phpOpcacheResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts, err := providerutil.ParseID(req.ID, 2)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID",
			fmt.Sprintf("Expected import ID in the form \"organization/server\": %s.", err))
		return
	}
	server, err := providerutil.ParseInt64(parts[1])
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("Server must be an integer: %s.", err))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("organization"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("server"), server)...)
}

// opcacheData is the JSON:API representation of the opcache state.
type opcacheData struct {
	ID         string `json:"id"`
	Attributes struct {
		OpcacheEnabled *bool `json:"opcache_enabled"`
	} `json:"attributes"`
}

func (d *opcacheData) apply(m *PhpOpcacheModel) {
	if d.ID != "" {
		m.Id = types.StringValue(d.ID)
	} else {
		m.Id = types.StringValue(fmt.Sprintf("%s/%d", m.Organization.ValueString(), m.Server.ValueInt64()))
	}
	if d.Attributes.OpcacheEnabled != nil {
		m.OpcacheEnabled = types.BoolValue(*d.Attributes.OpcacheEnabled)
	} else {
		m.OpcacheEnabled = types.BoolNull()
	}
}
