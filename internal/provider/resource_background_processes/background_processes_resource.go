package resource_background_processes

// This file implements the background_processes resource. Unlike *_resource_gen.go
// it is NOT regenerated, so the CRUD logic below is safe to edit.

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
	_ resource.Resource                = (*backgroundProcessesResource)(nil)
	_ resource.ResourceWithConfigure   = (*backgroundProcessesResource)(nil)
	_ resource.ResourceWithImportState = (*backgroundProcessesResource)(nil)
)

// NewBackgroundProcessesResource returns a new background_processes resource.
func NewBackgroundProcessesResource() resource.Resource {
	return &backgroundProcessesResource{}
}

type backgroundProcessesResource struct {
	client *forge.Client
}

func (r *backgroundProcessesResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_background_processes"
}

func (r *backgroundProcessesResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = providerutil.Client(req, resp)
}

func (r *backgroundProcessesResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	s := BackgroundProcessesResourceSchema(ctx)

	providerutil.Required(s.Attributes, "organization")
	providerutil.Required(s.Attributes, "server")
	// The update body only accepts {config, name}. name is updatable; every other
	// create-body field forces replacement.
	providerutil.RequiresReplace(s.Attributes, "command", "directory", "processes",
		"startsecs", "stopsignal", "stopwaitsecs", "user")
	providerutil.UseStateForUnknown(s.Attributes, "background_process", "id")

	s.MarkdownDescription = "Create a new background process from a template."
	resp.Schema = s
}

func (r *backgroundProcessesResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan BackgroundProcessesModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]any{
		"name":      plan.Name.ValueString(),
		"command":   plan.Command.ValueString(),
		"user":      plan.User.ValueString(),
		"processes": plan.Processes.ValueInt64(),
	}
	if !plan.Directory.IsNull() && !plan.Directory.IsUnknown() {
		body["directory"] = plan.Directory.ValueString()
	}
	if !plan.Startsecs.IsNull() && !plan.Startsecs.IsUnknown() {
		body["startsecs"] = plan.Startsecs.ValueInt64()
	}
	if !plan.Stopsignal.IsNull() && !plan.Stopsignal.IsUnknown() {
		body["stopsignal"] = plan.Stopsignal.ValueString()
	}
	if !plan.Stopwaitsecs.IsNull() && !plan.Stopwaitsecs.IsUnknown() {
		body["stopwaitsecs"] = plan.Stopwaitsecs.ValueInt64()
	}

	collection := fmt.Sprintf("/orgs/%s/servers/%d/background-processes",
		plan.Organization.ValueString(), plan.Server.ValueInt64())

	before, err := providerutil.ListIDs(ctx, r.client, collection)
	if err != nil {
		resp.Diagnostics.AddError("Unable to list background processes", err.Error())
		return
	}
	if err := r.client.Do(ctx, http.MethodPost, collection, body, nil); err != nil {
		resp.Diagnostics.AddError("Unable to create background process", err.Error())
		return
	}
	id, err := providerutil.ResolveCreatedID(ctx, r.client, collection, plan.Name.ValueString(), before)
	if err != nil {
		resp.Diagnostics.AddError("Unable to locate created background process", err.Error())
		return
	}

	var data backgroundProcessData
	if err := r.client.Do(ctx, http.MethodGet, collection+"/"+id, nil, &data); err != nil {
		resp.Diagnostics.AddError("Unable to read created background process", err.Error())
		return
	}

	data.apply(&plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *backgroundProcessesResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state BackgroundProcessesModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var data backgroundProcessData
	endpoint := fmt.Sprintf("/orgs/%s/servers/%d/background-processes/%s",
		state.Organization.ValueString(), state.Server.ValueInt64(), state.Id.ValueString())
	if err := r.client.Do(ctx, http.MethodGet, endpoint, nil, &data); err != nil {
		if forge.NotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Unable to read background process", err.Error())
		return
	}

	data.apply(&state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *backgroundProcessesResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan BackgroundProcessesModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]any{
		"name": plan.Name.ValueString(),
	}

	endpoint := fmt.Sprintf("/orgs/%s/servers/%d/background-processes/%s",
		plan.Organization.ValueString(), plan.Server.ValueInt64(), plan.Id.ValueString())
	if err := r.client.Do(ctx, http.MethodPut, endpoint, body, nil); err != nil {
		resp.Diagnostics.AddError("Unable to update background process", err.Error())
		return
	}

	var data backgroundProcessData
	if err := r.client.Do(ctx, http.MethodGet, endpoint, nil, &data); err != nil {
		resp.Diagnostics.AddError("Unable to read updated background process", err.Error())
		return
	}

	data.apply(&plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *backgroundProcessesResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state BackgroundProcessesModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	endpoint := fmt.Sprintf("/orgs/%s/servers/%d/background-processes/%s",
		state.Organization.ValueString(), state.Server.ValueInt64(), state.Id.ValueString())
	if err := r.client.Do(ctx, http.MethodDelete, endpoint, nil, nil); err != nil && !forge.NotFound(err) {
		resp.Diagnostics.AddError("Unable to delete background process", err.Error())
	}
}

func (r *backgroundProcessesResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts, err := providerutil.ParseID(req.ID, 3)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID",
			fmt.Sprintf("Expected import ID in the form \"organization/server/background_process\": %s.", err))
		return
	}
	server, err := providerutil.ParseInt64(parts[1])
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("Server must be an integer: %s.", err))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("organization"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("server"), server)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[2])...)
}

// backgroundProcessData is the JSON:API representation of a background process.
type backgroundProcessData struct {
	ID         string `json:"id"`
	Attributes struct {
		Name         string  `json:"name"`
		Command      string  `json:"command"`
		User         string  `json:"user"`
		Directory    *string `json:"directory"`
		Processes    *int64  `json:"processes"`
		Startsecs    *int64  `json:"startsecs"`
		Stopsignal   *string `json:"stopsignal"`
		Stopwaitsecs *int64  `json:"stopwaitsecs"`
		Status       string  `json:"status"`
		CreatedAt    string  `json:"created_at"`
	} `json:"attributes"`
}

func (d *backgroundProcessData) apply(m *BackgroundProcessesModel) {
	m.Id = types.StringValue(d.ID)
	m.Name = types.StringValue(d.Attributes.Name)
	m.Command = types.StringValue(d.Attributes.Command)
	m.User = types.StringValue(d.Attributes.User)
	m.Status = types.StringValue(d.Attributes.Status)
	m.CreatedAt = types.StringValue(d.Attributes.CreatedAt)
	if d.Attributes.Directory != nil {
		m.Directory = types.StringValue(*d.Attributes.Directory)
	} else {
		m.Directory = types.StringNull()
	}
	if d.Attributes.Processes != nil {
		m.Processes = types.Int64Value(*d.Attributes.Processes)
	}
	if d.Attributes.Startsecs != nil {
		m.Startsecs = types.Int64Value(*d.Attributes.Startsecs)
	} else {
		m.Startsecs = types.Int64Null()
	}
	if d.Attributes.Stopsignal != nil {
		m.Stopsignal = types.StringValue(*d.Attributes.Stopsignal)
	} else {
		m.Stopsignal = types.StringNull()
	}
	if d.Attributes.Stopwaitsecs != nil {
		m.Stopwaitsecs = types.Int64Value(*d.Attributes.Stopwaitsecs)
	} else {
		m.Stopwaitsecs = types.Int64Null()
	}
	if id, err := providerutil.ParseInt64(d.ID); err == nil {
		m.BackgroundProcess = types.Int64Value(id)
	}
}
