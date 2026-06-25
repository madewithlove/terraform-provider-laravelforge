package resource_composer_credentials

// This file implements the composer_credentials resource. Unlike
// *_resource_gen.go it is NOT regenerated, so the CRUD logic below is safe to
// edit.

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
	_ resource.Resource                = (*composerCredentialsResource)(nil)
	_ resource.ResourceWithConfigure   = (*composerCredentialsResource)(nil)
	_ resource.ResourceWithImportState = (*composerCredentialsResource)(nil)
)

// NewComposerCredentialsResource returns a new composer_credentials resource.
func NewComposerCredentialsResource() resource.Resource {
	return &composerCredentialsResource{}
}

type composerCredentialsResource struct {
	client *forge.Client
}

func (r *composerCredentialsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_composer_credentials"
}

func (r *composerCredentialsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = providerutil.Client(req, resp)
}

func (r *composerCredentialsResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	s := ComposerCredentialsResourceSchema(ctx)

	providerutil.Required(s.Attributes, "organization")
	providerutil.Required(s.Attributes, "server")
	providerutil.Required(s.Attributes, "site")
	// password, repository and username are all updatable in place via the
	// API's PUT endpoint, so none of them force replacement. The password is
	// write-only and never echoed back.
	providerutil.Sensitive(s.Attributes, "password")
	providerutil.UseStateForUnknown(s.Attributes, "id")

	s.MarkdownDescription = "Create composer credentials for the site"
	resp.Schema = s
}

func (r *composerCredentialsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ComposerCredentialsModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]any{
		"repository": plan.Repository.ValueString(),
		"username":   plan.Username.ValueString(),
		"password":   plan.Password.ValueString(),
	}

	collection := fmt.Sprintf("/orgs/%s/servers/%d/sites/%d/composer/credentials",
		plan.Organization.ValueString(), plan.Server.ValueInt64(), plan.Site.ValueInt64())

	// Creation is asynchronous (202, no body); resolve the new credential by its
	// repository name, which the API exposes as the item's "name".
	before, err := providerutil.ListIDs(ctx, r.client, collection)
	if err != nil {
		resp.Diagnostics.AddError("Unable to list composer credentials", err.Error())
		return
	}
	if err := r.client.Do(ctx, http.MethodPost, collection, body, nil); err != nil {
		resp.Diagnostics.AddError("Unable to create composer credential", err.Error())
		return
	}
	id, err := providerutil.ResolveCreatedID(ctx, r.client, collection, plan.Repository.ValueString(), before)
	if err != nil {
		resp.Diagnostics.AddError("Unable to locate created composer credential", err.Error())
		return
	}

	var data composerCredentialData
	if err := r.client.Do(ctx, http.MethodGet, collection+"/"+id, nil, &data); err != nil {
		resp.Diagnostics.AddError("Unable to read created composer credential", err.Error())
		return
	}

	data.apply(&plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *composerCredentialsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ComposerCredentialsModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var data composerCredentialData
	endpoint := fmt.Sprintf("/orgs/%s/servers/%d/sites/%d/composer/credentials/%s",
		state.Organization.ValueString(), state.Server.ValueInt64(), state.Site.ValueInt64(), state.Id.ValueString())
	if err := r.client.Do(ctx, http.MethodGet, endpoint, nil, &data); err != nil {
		if forge.NotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Unable to read composer credential", err.Error())
		return
	}

	data.apply(&state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *composerCredentialsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ComposerCredentialsModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]any{
		"repository": plan.Repository.ValueString(),
		"username":   plan.Username.ValueString(),
		"password":   plan.Password.ValueString(),
	}

	endpoint := fmt.Sprintf("/orgs/%s/servers/%d/sites/%d/composer/credentials/%s",
		plan.Organization.ValueString(), plan.Server.ValueInt64(), plan.Site.ValueInt64(), plan.Id.ValueString())
	if err := r.client.Do(ctx, http.MethodPut, endpoint, body, nil); err != nil {
		resp.Diagnostics.AddError("Unable to update composer credential", err.Error())
		return
	}

	var data composerCredentialData
	if err := r.client.Do(ctx, http.MethodGet, endpoint, nil, &data); err != nil {
		resp.Diagnostics.AddError("Unable to read updated composer credential", err.Error())
		return
	}

	data.apply(&plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *composerCredentialsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ComposerCredentialsModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	endpoint := fmt.Sprintf("/orgs/%s/servers/%d/sites/%d/composer/credentials/%s",
		state.Organization.ValueString(), state.Server.ValueInt64(), state.Site.ValueInt64(), state.Id.ValueString())
	if err := r.client.Do(ctx, http.MethodDelete, endpoint, nil, nil); err != nil && !forge.NotFound(err) {
		resp.Diagnostics.AddError("Unable to delete composer credential", err.Error())
	}
}

func (r *composerCredentialsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts, err := providerutil.ParseID(req.ID, 4)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID",
			fmt.Sprintf("Expected import ID in the form \"organization/server/site/repository\": %s.", err))
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

// composerCredentialData is the JSON:API representation of a composer
// credential. The password is write-only and never returned by the API.
type composerCredentialData struct {
	ID         string `json:"id"`
	Attributes struct {
		Repository string `json:"repository"`
		Username   string `json:"username"`
	} `json:"attributes"`
}

// apply copies the API response onto the Terraform model. The password is
// write-only (never returned), so it is left untouched.
func (d *composerCredentialData) apply(m *ComposerCredentialsModel) {
	m.Id = types.StringValue(d.ID)
	m.Repository = types.StringValue(d.Attributes.Repository)
	m.Username = types.StringValue(d.Attributes.Username)
}
