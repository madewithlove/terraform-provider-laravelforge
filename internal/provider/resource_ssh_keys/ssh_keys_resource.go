package resource_ssh_keys

// This file implements the ssh_keys resource. Unlike *_resource_gen.go it is
// NOT regenerated, so the CRUD logic below is safe to edit.

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

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = (*sshKeysResource)(nil)
	_ resource.ResourceWithConfigure   = (*sshKeysResource)(nil)
	_ resource.ResourceWithImportState = (*sshKeysResource)(nil)
)

// NewSshKeysResource returns a new ssh_keys resource.
func NewSshKeysResource() resource.Resource {
	return &sshKeysResource{}
}

type sshKeysResource struct {
	client *forge.Client
}

func (r *sshKeysResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ssh_keys"
}

func (r *sshKeysResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = providerutil.Client(req, resp)
}

func (r *sshKeysResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	s := SshKeysResourceSchema(ctx)

	// SSH keys are immutable (the API has no update endpoint) and are scoped to
	// a server within an organization, so every configurable attribute forces
	// replacement on change.
	providerutil.Required(s.Attributes, "organization")
	providerutil.Required(s.Attributes, "server")
	providerutil.RequiresReplace(s.Attributes, "name", "key", "user")
	providerutil.UseStateForUnknown(s.Attributes, "id")

	s.MarkdownDescription = "Manages an SSH key on a Forge server."
	resp.Schema = s
}

func (r *sshKeysResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan SshKeysModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]any{
		"name": plan.Name.ValueString(),
		"key":  plan.Key.ValueString(),
	}
	if !plan.User.IsNull() && !plan.User.IsUnknown() {
		body["user"] = plan.User.ValueString()
	}

	collection := fmt.Sprintf("/orgs/%s/servers/%d/ssh-keys",
		plan.Organization.ValueString(), plan.Server.ValueInt64())

	// Creation is asynchronous and returns 202 with no body, so snapshot the
	// existing keys, create, then resolve the new key's id by name.
	before, err := providerutil.ListIDs(ctx, r.client, collection)
	if err != nil {
		resp.Diagnostics.AddError("Unable to list SSH keys", err.Error())
		return
	}
	if err := r.client.Do(ctx, http.MethodPost, collection, body, nil); err != nil {
		resp.Diagnostics.AddError("Unable to create SSH key", err.Error())
		return
	}
	id, err := providerutil.ResolveCreatedID(ctx, r.client, collection, plan.Name.ValueString(), before)
	if err != nil {
		resp.Diagnostics.AddError("Unable to locate created SSH key", err.Error())
		return
	}

	var data keyData
	if err := r.client.Do(ctx, http.MethodGet, collection+"/"+id, nil, &data); err != nil {
		resp.Diagnostics.AddError("Unable to read created SSH key", err.Error())
		return
	}

	data.apply(&plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *sshKeysResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state SshKeysModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var data keyData
	endpoint := fmt.Sprintf("/orgs/%s/servers/%d/ssh-keys/%s",
		state.Organization.ValueString(), state.Server.ValueInt64(), state.Id.ValueString())
	if err := r.client.Do(ctx, http.MethodGet, endpoint, nil, &data); err != nil {
		if forge.NotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Unable to read SSH key", err.Error())
		return
	}

	data.apply(&state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update is required by the interface but is never invoked with in-place
// changes: every configurable attribute is marked RequiresReplace, so Terraform
// destroys and recreates the key instead.
func (r *sshKeysResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan SshKeysModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *sshKeysResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state SshKeysModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	endpoint := fmt.Sprintf("/orgs/%s/servers/%d/ssh-keys/%s",
		state.Organization.ValueString(), state.Server.ValueInt64(), state.Id.ValueString())
	if err := r.client.Do(ctx, http.MethodDelete, endpoint, nil, nil); err != nil && !forge.NotFound(err) {
		resp.Diagnostics.AddError("Unable to delete SSH key", err.Error())
	}
}

// ImportState supports `terraform import` using the
// "organization/server/key" identifier form.
func (r *sshKeysResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts, err := providerutil.ParseID(req.ID, 3)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			fmt.Sprintf("Expected import ID in the form \"organization/server/key\": %s.", err),
		)
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

// keyData is the JSON:API representation of an SSH key returned by the API.
type keyData struct {
	ID         string `json:"id"`
	Attributes struct {
		Name      string  `json:"name"`
		User      *string `json:"user"`
		Status    string  `json:"status"`
		CreatedBy *int64  `json:"created_by"`
		CreatedAt string  `json:"created_at"`
		UpdatedAt string  `json:"updated_at"`
	} `json:"attributes"`
}

// apply copies the API response onto the Terraform model. The "key" attribute
// is write-only (never returned by the API), so it is left untouched.
func (d *keyData) apply(m *SshKeysModel) {
	m.Id = types.StringValue(d.ID)
	m.Name = types.StringValue(d.Attributes.Name)
	m.Status = types.StringValue(d.Attributes.Status)
	m.CreatedAt = types.StringValue(d.Attributes.CreatedAt)
	m.UpdatedAt = types.StringValue(d.Attributes.UpdatedAt)
	if d.Attributes.User != nil {
		m.User = types.StringValue(*d.Attributes.User)
	}
	if d.Attributes.CreatedBy != nil {
		m.CreatedBy = types.Int64Value(*d.Attributes.CreatedBy)
	} else {
		m.CreatedBy = types.Int64Null()
	}
}
