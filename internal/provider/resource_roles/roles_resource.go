package resource_roles

// This file implements the roles resource. Unlike *_resource_gen.go it is NOT
// regenerated, so the CRUD logic below is safe to edit.

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/madewithlove/terraform-provider-laravelforge/internal/forge"
	"github.com/madewithlove/terraform-provider-laravelforge/internal/providerutil"
)

var (
	_ resource.Resource                = (*rolesResource)(nil)
	_ resource.ResourceWithConfigure   = (*rolesResource)(nil)
	_ resource.ResourceWithImportState = (*rolesResource)(nil)
)

// NewRolesResource returns a new roles resource.
func NewRolesResource() resource.Resource {
	return &rolesResource{}
}

type rolesResource struct {
	client *forge.Client
}

func (r *rolesResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_roles"
}

func (r *rolesResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = providerutil.Client(req, resp)
}

func (r *rolesResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	s := RolesResourceSchema(ctx)

	// A role is scoped to an organization, which is a path parameter. name and
	// permissions are updatable in place via PUT.
	providerutil.Required(s.Attributes, "organization")
	providerutil.UseStateForUnknown(s.Attributes, "id", "role")

	s.MarkdownDescription = "Manages a custom role in the organization."
	resp.Schema = s
}

func (r *rolesResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan RolesModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body, err := roleBody(ctx, &plan)
	if err != nil {
		resp.Diagnostics.AddError("Unable to build role request", err.Error())
		return
	}

	collection := fmt.Sprintf("/orgs/%s/roles", plan.Organization.ValueString())

	// Creation is synchronous and returns the created role.
	var data roleData
	if err := r.client.Do(ctx, http.MethodPost, collection, body, &data); err != nil {
		resp.Diagnostics.AddError("Unable to create role", err.Error())
		return
	}

	data.apply(&plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *rolesResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state RolesModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var data roleData
	endpoint := fmt.Sprintf("/orgs/%s/roles/%s", state.Organization.ValueString(), state.Id.ValueString())
	if err := r.client.Do(ctx, http.MethodGet, endpoint, nil, &data); err != nil {
		if forge.NotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Unable to read role", err.Error())
		return
	}

	data.apply(&state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *rolesResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan RolesModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body, err := roleBody(ctx, &plan)
	if err != nil {
		resp.Diagnostics.AddError("Unable to build role request", err.Error())
		return
	}

	endpoint := fmt.Sprintf("/orgs/%s/roles/%s", plan.Organization.ValueString(), plan.Id.ValueString())
	if err := r.client.Do(ctx, http.MethodPut, endpoint, body, nil); err != nil {
		resp.Diagnostics.AddError("Unable to update role", err.Error())
		return
	}

	var data roleData
	if err := r.client.Do(ctx, http.MethodGet, endpoint, nil, &data); err != nil {
		resp.Diagnostics.AddError("Unable to read updated role", err.Error())
		return
	}

	data.apply(&plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *rolesResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state RolesModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	endpoint := fmt.Sprintf("/orgs/%s/roles/%s", state.Organization.ValueString(), state.Id.ValueString())
	if err := r.client.Do(ctx, http.MethodDelete, endpoint, nil, nil); err != nil && !forge.NotFound(err) {
		resp.Diagnostics.AddError("Unable to delete role", err.Error())
	}
}

func (r *rolesResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts, err := providerutil.ParseID(req.ID, 2)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID",
			fmt.Sprintf("Expected import ID in the form \"organization/role\": %s.", err))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("organization"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}

// roleBody builds the create/update request body, sending both name and
// permissions.
func roleBody(ctx context.Context, plan *RolesModel) (map[string]any, error) {
	body := map[string]any{
		"name": plan.Name.ValueString(),
	}
	if !plan.Permissions.IsNull() && !plan.Permissions.IsUnknown() {
		var perms []string
		if diags := plan.Permissions.ElementsAs(ctx, &perms, false); diags.HasError() {
			return nil, fmt.Errorf("reading permissions: %v", diags)
		}
		body["permissions"] = perms
	}
	return body, nil
}

// roleData is the JSON:API representation of a role returned by the API. The
// permissions are returned as plain string identifiers.
type roleData struct {
	ID         string `json:"id"`
	Attributes struct {
		Name        string   `json:"name"`
		Permissions []string `json:"permissions"`
		CreatedAt   string   `json:"created_at"`
		UpdatedAt   string   `json:"updated_at"`
	} `json:"attributes"`
}

func (d *roleData) apply(m *RolesModel) {
	m.Id = types.StringValue(d.ID)
	m.Name = types.StringValue(d.Attributes.Name)
	m.CreatedAt = types.StringValue(d.Attributes.CreatedAt)
	m.UpdatedAt = types.StringValue(d.Attributes.UpdatedAt)
	if d.Attributes.Permissions != nil {
		elements := make([]attr.Value, len(d.Attributes.Permissions))
		for i, p := range d.Attributes.Permissions {
			elements[i] = types.StringValue(p)
		}
		m.Permissions = types.ListValueMust(types.StringType, elements)
	}
	if id, err := providerutil.ParseInt64(d.ID); err == nil {
		m.Role = types.Int64Value(id)
	}
}
