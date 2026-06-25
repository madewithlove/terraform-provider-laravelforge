package resource_database_schemas

// This file implements the database_schemas resource. Unlike *_resource_gen.go
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
	_ resource.Resource                = (*databaseSchemasResource)(nil)
	_ resource.ResourceWithConfigure   = (*databaseSchemasResource)(nil)
	_ resource.ResourceWithImportState = (*databaseSchemasResource)(nil)
)

// NewDatabaseSchemasResource returns a new database_schemas resource.
func NewDatabaseSchemasResource() resource.Resource {
	return &databaseSchemasResource{}
}

type databaseSchemasResource struct {
	client *forge.Client
}

func (r *databaseSchemasResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_database_schemas"
}

func (r *databaseSchemasResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = providerutil.Client(req, resp)
}

func (r *databaseSchemasResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	s := DatabaseSchemasResourceSchema(ctx)

	// A schema cannot be updated in place, so every input forces replacement.
	providerutil.Required(s.Attributes, "organization")
	providerutil.Required(s.Attributes, "server")
	// user and password are write-only inputs (never returned by the API).
	providerutil.Input(s.Attributes, "user", "password")
	providerutil.Sensitive(s.Attributes, "password")
	providerutil.RequiresReplace(s.Attributes, "name", "user", "password")
	providerutil.UseStateForUnknown(s.Attributes, "id", "database")

	s.MarkdownDescription = "Add a new database schema to the server."
	resp.Schema = s
}

func (r *databaseSchemasResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan DatabaseSchemasModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]any{"name": plan.Name.ValueString()}
	if !plan.User.IsNull() && !plan.User.IsUnknown() {
		body["user"] = plan.User.ValueString()
	}
	if !plan.Password.IsNull() && !plan.Password.IsUnknown() {
		body["password"] = plan.Password.ValueString()
	}

	collection := fmt.Sprintf("/orgs/%s/servers/%d/database/schemas",
		plan.Organization.ValueString(), plan.Server.ValueInt64())

	// Creation is asynchronous (202, no body); resolve the new schema by name.
	before, err := providerutil.ListIDs(ctx, r.client, collection)
	if err != nil {
		resp.Diagnostics.AddError("Unable to list database schemas", err.Error())
		return
	}
	if err := r.client.Do(ctx, http.MethodPost, collection, body, nil); err != nil {
		resp.Diagnostics.AddError("Unable to create database schema", err.Error())
		return
	}
	id, err := providerutil.ResolveCreatedID(ctx, r.client, collection, plan.Name.ValueString(), before)
	if err != nil {
		resp.Diagnostics.AddError("Unable to locate created database schema", err.Error())
		return
	}

	var data databaseData
	if err := r.client.Do(ctx, http.MethodGet, collection+"/"+id, nil, &data); err != nil {
		resp.Diagnostics.AddError("Unable to read created database schema", err.Error())
		return
	}

	data.apply(&plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *databaseSchemasResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state DatabaseSchemasModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var data databaseData
	endpoint := fmt.Sprintf("/orgs/%s/servers/%d/database/schemas/%s",
		state.Organization.ValueString(), state.Server.ValueInt64(), state.Id.ValueString())
	if err := r.client.Do(ctx, http.MethodGet, endpoint, nil, &data); err != nil {
		if forge.NotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Unable to read database schema", err.Error())
		return
	}

	data.apply(&state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update is never invoked with in-place changes: every configurable attribute
// is marked RequiresReplace.
func (r *databaseSchemasResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan DatabaseSchemasModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *databaseSchemasResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state DatabaseSchemasModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	endpoint := fmt.Sprintf("/orgs/%s/servers/%d/database/schemas/%s",
		state.Organization.ValueString(), state.Server.ValueInt64(), state.Id.ValueString())
	if err := r.client.Do(ctx, http.MethodDelete, endpoint, nil, nil); err != nil && !forge.NotFound(err) {
		resp.Diagnostics.AddError("Unable to delete database schema", err.Error())
	}
}

func (r *databaseSchemasResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts, err := providerutil.ParseID(req.ID, 3)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID",
			fmt.Sprintf("Expected import ID in the form \"organization/server/database\": %s.", err))
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

// databaseData is the JSON:API representation of a database schema.
type databaseData struct {
	ID         string `json:"id"`
	Attributes struct {
		Name      string `json:"name"`
		Status    string `json:"status"`
		CreatedAt string `json:"created_at"`
		UpdatedAt string `json:"updated_at"`
	} `json:"attributes"`
}

// apply copies the API response onto the model. The "user" and "password"
// inputs are write-only (not returned), so they are left untouched.
func (d *databaseData) apply(m *DatabaseSchemasModel) {
	m.Id = types.StringValue(d.ID)
	m.Name = types.StringValue(d.Attributes.Name)
	m.Status = types.StringValue(d.Attributes.Status)
	m.CreatedAt = types.StringValue(d.Attributes.CreatedAt)
	m.UpdatedAt = types.StringValue(d.Attributes.UpdatedAt)
	if id, err := providerutil.ParseInt64(d.ID); err == nil {
		m.Database = types.Int64Value(id)
	}
}
