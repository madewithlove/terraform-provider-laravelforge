package resource_php_versions

// This file implements the php_versions resource. Unlike *_resource_gen.go it is
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

var (
	_ resource.Resource                = (*phpVersionsResource)(nil)
	_ resource.ResourceWithConfigure   = (*phpVersionsResource)(nil)
	_ resource.ResourceWithImportState = (*phpVersionsResource)(nil)
)

// NewPhpVersionsResource returns a new php_versions resource.
func NewPhpVersionsResource() resource.Resource {
	return &phpVersionsResource{}
}

type phpVersionsResource struct {
	client *forge.Client
}

func (r *phpVersionsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_php_versions"
}

func (r *phpVersionsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = providerutil.Client(req, resp)
}

func (r *phpVersionsResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	s := PhpVersionsResourceSchema(ctx)

	providerutil.Required(s.Attributes, "organization")
	providerutil.Required(s.Attributes, "server")
	// The version itself is immutable (installing a different version is a new
	// resource); the cli_default/site_default flags can be toggled in place.
	providerutil.RequiresReplace(s.Attributes, "version")
	providerutil.UseStateForUnknown(s.Attributes, "php_version", "id")

	s.MarkdownDescription = "Manages a PHP version installed on a Forge server."
	resp.Schema = s
}

func (r *phpVersionsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan PhpVersionsModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]any{
		"version": plan.Version.ValueString(),
	}
	if !plan.CliDefault.IsNull() && !plan.CliDefault.IsUnknown() {
		body["cli_default"] = plan.CliDefault.ValueBool()
	}
	if !plan.SiteDefault.IsNull() && !plan.SiteDefault.IsUnknown() {
		body["site_default"] = plan.SiteDefault.ValueBool()
	}

	collection := fmt.Sprintf("/orgs/%s/servers/%d/php/versions",
		plan.Organization.ValueString(), plan.Server.ValueInt64())

	before, err := providerutil.ListIDs(ctx, r.client, collection)
	if err != nil {
		resp.Diagnostics.AddError("Unable to list PHP versions", err.Error())
		return
	}
	if err := r.client.Do(ctx, http.MethodPost, collection, body, nil); err != nil {
		resp.Diagnostics.AddError("Unable to create PHP version", err.Error())
		return
	}
	id, err := providerutil.ResolveCreatedID(ctx, r.client, collection, "", before)
	if err != nil {
		resp.Diagnostics.AddError("Unable to locate created PHP version", err.Error())
		return
	}

	var data phpVersionData
	if err := r.client.Do(ctx, http.MethodGet, collection+"/"+id, nil, &data); err != nil {
		resp.Diagnostics.AddError("Unable to read created PHP version", err.Error())
		return
	}

	data.apply(&plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *phpVersionsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state PhpVersionsModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var data phpVersionData
	endpoint := fmt.Sprintf("/orgs/%s/servers/%d/php/versions/%s",
		state.Organization.ValueString(), state.Server.ValueInt64(), state.Id.ValueString())
	if err := r.client.Do(ctx, http.MethodGet, endpoint, nil, &data); err != nil {
		if forge.NotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Unable to read PHP version", err.Error())
		return
	}

	data.apply(&state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *phpVersionsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan PhpVersionsModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]any{}
	if !plan.CliDefault.IsNull() && !plan.CliDefault.IsUnknown() {
		body["cli_default"] = plan.CliDefault.ValueBool()
	}
	if !plan.SiteDefault.IsNull() && !plan.SiteDefault.IsUnknown() {
		body["site_default"] = plan.SiteDefault.ValueBool()
	}

	endpoint := fmt.Sprintf("/orgs/%s/servers/%d/php/versions/%s",
		plan.Organization.ValueString(), plan.Server.ValueInt64(), plan.Id.ValueString())
	if err := r.client.Do(ctx, http.MethodPut, endpoint, body, nil); err != nil {
		resp.Diagnostics.AddError("Unable to update PHP version", err.Error())
		return
	}

	var data phpVersionData
	if err := r.client.Do(ctx, http.MethodGet, endpoint, nil, &data); err != nil {
		resp.Diagnostics.AddError("Unable to read updated PHP version", err.Error())
		return
	}

	data.apply(&plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *phpVersionsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state PhpVersionsModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	endpoint := fmt.Sprintf("/orgs/%s/servers/%d/php/versions/%s",
		state.Organization.ValueString(), state.Server.ValueInt64(), state.Id.ValueString())
	if err := r.client.Do(ctx, http.MethodDelete, endpoint, nil, nil); err != nil && !forge.NotFound(err) {
		resp.Diagnostics.AddError("Unable to delete PHP version", err.Error())
	}
}

func (r *phpVersionsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts, err := providerutil.ParseID(req.ID, 3)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID",
			fmt.Sprintf("Expected import ID in the form \"organization/server/php_version\": %s.", err))
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

// phpVersionData is the JSON:API representation of a PHP version.
type phpVersionData struct {
	ID         string `json:"id"`
	Attributes struct {
		Version     string `json:"version"`
		BinaryName  string `json:"binary_name"`
		CliDefault  *bool  `json:"cli_default"`
		SiteDefault *bool  `json:"site_default"`
		Status      string `json:"status"`
		CreatedAt   string `json:"created_at"`
		UpdatedAt   string `json:"updated_at"`
	} `json:"attributes"`
}

func (d *phpVersionData) apply(m *PhpVersionsModel) {
	m.Id = types.StringValue(d.ID)
	m.Version = types.StringValue(d.Attributes.Version)
	m.BinaryName = types.StringValue(d.Attributes.BinaryName)
	m.Status = types.StringValue(d.Attributes.Status)
	m.CreatedAt = types.StringValue(d.Attributes.CreatedAt)
	m.UpdatedAt = types.StringValue(d.Attributes.UpdatedAt)
	if d.Attributes.CliDefault != nil {
		m.CliDefault = types.BoolValue(*d.Attributes.CliDefault)
	} else {
		m.CliDefault = types.BoolNull()
	}
	if d.Attributes.SiteDefault != nil {
		m.SiteDefault = types.BoolValue(*d.Attributes.SiteDefault)
	} else {
		m.SiteDefault = types.BoolNull()
	}
	if id, err := providerutil.ParseInt64(d.ID); err == nil {
		m.PhpVersion = types.Int64Value(id)
	}
}
