package resource_database_users

// This file implements the database_users resource. Unlike *_resource_gen.go it
// is NOT regenerated, so the CRUD logic below is safe to edit.

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/madewithlove/terraform-provider-laravelforge/internal/forge"
	"github.com/madewithlove/terraform-provider-laravelforge/internal/providerutil"
)

var (
	_ resource.Resource                = (*databaseUsersResource)(nil)
	_ resource.ResourceWithConfigure   = (*databaseUsersResource)(nil)
	_ resource.ResourceWithImportState = (*databaseUsersResource)(nil)
)

// NewDatabaseUsersResource returns a new database_users resource.
func NewDatabaseUsersResource() resource.Resource {
	return &databaseUsersResource{}
}

type databaseUsersResource struct {
	client *forge.Client
}

func (r *databaseUsersResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_database_users"
}

func (r *databaseUsersResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = providerutil.Client(req, resp)
}

func (r *databaseUsersResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	s := DatabaseUsersResourceSchema(ctx)

	providerutil.Required(s.Attributes, "organization")
	providerutil.Required(s.Attributes, "server")
	// name and read_only cannot be changed in place; password and database_ids
	// are updatable via the API's PUT endpoint. All four are write-only inputs
	// (never returned by the API).
	providerutil.Input(s.Attributes, "read_only", "database_ids")
	providerutil.RequiresReplace(s.Attributes, "name", "read_only")
	providerutil.Sensitive(s.Attributes, "password")
	providerutil.UseStateForUnknown(s.Attributes, "id", "database_user")

	s.MarkdownDescription = "Manages a database user on a Forge server."
	resp.Schema = s
}

func (r *databaseUsersResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan DatabaseUsersModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]any{
		"name":     plan.Name.ValueString(),
		"password": plan.Password.ValueString(),
	}
	if !plan.ReadOnly.IsNull() && !plan.ReadOnly.IsUnknown() {
		body["read_only"] = plan.ReadOnly.ValueBool()
	}
	ids, diags := databaseIDs(ctx, plan.DatabaseIds)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if ids != nil {
		body["database_ids"] = ids
	}

	collection := fmt.Sprintf("/orgs/%s/servers/%d/database/users",
		plan.Organization.ValueString(), plan.Server.ValueInt64())

	// Creation is asynchronous (202, no body); resolve the new user by name.
	before, err := providerutil.ListIDs(ctx, r.client, collection)
	if err != nil {
		resp.Diagnostics.AddError("Unable to list database users", err.Error())
		return
	}
	if err := r.client.Do(ctx, http.MethodPost, collection, body, nil); err != nil {
		resp.Diagnostics.AddError("Unable to create database user", err.Error())
		return
	}
	id, err := providerutil.ResolveCreatedID(ctx, r.client, collection, plan.Name.ValueString(), before)
	if err != nil {
		resp.Diagnostics.AddError("Unable to locate created database user", err.Error())
		return
	}

	var data databaseUserData
	if err := r.client.Do(ctx, http.MethodGet, collection+"/"+id, nil, &data); err != nil {
		resp.Diagnostics.AddError("Unable to read created database user", err.Error())
		return
	}

	data.apply(&plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *databaseUsersResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state DatabaseUsersModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var data databaseUserData
	endpoint := fmt.Sprintf("/orgs/%s/servers/%d/database/users/%s",
		state.Organization.ValueString(), state.Server.ValueInt64(), state.Id.ValueString())
	if err := r.client.Do(ctx, http.MethodGet, endpoint, nil, &data); err != nil {
		if forge.NotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Unable to read database user", err.Error())
		return
	}

	data.apply(&state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *databaseUsersResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan DatabaseUsersModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]any{}
	if !plan.Password.IsNull() && !plan.Password.IsUnknown() {
		body["password"] = plan.Password.ValueString()
	}
	ids, diags := databaseIDs(ctx, plan.DatabaseIds)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if ids != nil {
		body["database_ids"] = ids
	}

	endpoint := fmt.Sprintf("/orgs/%s/servers/%d/database/users/%s",
		plan.Organization.ValueString(), plan.Server.ValueInt64(), plan.Id.ValueString())
	if err := r.client.Do(ctx, http.MethodPut, endpoint, body, nil); err != nil {
		resp.Diagnostics.AddError("Unable to update database user", err.Error())
		return
	}

	// The update may be applied asynchronously; re-read to refresh state.
	var data databaseUserData
	if err := r.client.Do(ctx, http.MethodGet, endpoint, nil, &data); err != nil {
		resp.Diagnostics.AddError("Unable to read updated database user", err.Error())
		return
	}

	data.apply(&plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *databaseUsersResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state DatabaseUsersModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	endpoint := fmt.Sprintf("/orgs/%s/servers/%d/database/users/%s",
		state.Organization.ValueString(), state.Server.ValueInt64(), state.Id.ValueString())
	if err := r.client.Do(ctx, http.MethodDelete, endpoint, nil, nil); err != nil && !forge.NotFound(err) {
		resp.Diagnostics.AddError("Unable to delete database user", err.Error())
	}
}

func (r *databaseUsersResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts, err := providerutil.ParseID(req.ID, 3)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID",
			fmt.Sprintf("Expected import ID in the form \"organization/server/database_user\": %s.", err))
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

// databaseIDs extracts the database_ids list as a []int64, returning nil when
// the list is null or unknown.
func databaseIDs(ctx context.Context, list types.List) ([]int64, diag.Diagnostics) {
	if list.IsNull() || list.IsUnknown() {
		return nil, nil
	}
	var ids []int64
	diags := list.ElementsAs(ctx, &ids, false)
	return ids, diags
}

// databaseUserData is the JSON:API representation of a database user. The
// read_only flag, password and database_ids are write-only and not returned.
type databaseUserData struct {
	ID         string `json:"id"`
	Attributes struct {
		Name      string `json:"name"`
		Status    string `json:"status"`
		CreatedAt string `json:"created_at"`
		UpdatedAt string `json:"updated_at"`
	} `json:"attributes"`
}

func (d *databaseUserData) apply(m *DatabaseUsersModel) {
	m.Id = types.StringValue(d.ID)
	m.Name = types.StringValue(d.Attributes.Name)
	m.Status = types.StringValue(d.Attributes.Status)
	m.CreatedAt = types.StringValue(d.Attributes.CreatedAt)
	m.UpdatedAt = types.StringValue(d.Attributes.UpdatedAt)
	if id, err := providerutil.ParseInt64(d.ID); err == nil {
		m.DatabaseUser = types.Int64Value(id)
	}
}
