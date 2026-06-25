package resource_site_domains

// This file implements the site_domains resource. Unlike *_resource_gen.go it
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
	_ resource.Resource                = (*siteDomainsResource)(nil)
	_ resource.ResourceWithConfigure   = (*siteDomainsResource)(nil)
	_ resource.ResourceWithImportState = (*siteDomainsResource)(nil)
)

// NewSiteDomainsResource returns a new site_domains resource.
func NewSiteDomainsResource() resource.Resource {
	return &siteDomainsResource{}
}

type siteDomainsResource struct {
	client *forge.Client
}

func (r *siteDomainsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_site_domains"
}

func (r *siteDomainsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = providerutil.Client(req, resp)
}

func (r *siteDomainsResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	s := SiteDomainsResourceSchema(ctx)

	providerutil.Required(s.Attributes, "organization")
	providerutil.Required(s.Attributes, "server")
	providerutil.Required(s.Attributes, "site")
	// name is a create-only field (the API's update endpoint only accepts
	// allow_wildcard_subdomains and www_redirect_type), so it forces replacement.
	providerutil.RequiresReplace(s.Attributes, "name")
	providerutil.UseStateForUnknown(s.Attributes, "id", "domain_record")

	s.MarkdownDescription = "Add a new domain to the site"
	resp.Schema = s
}

func (r *siteDomainsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan SiteDomainsModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]any{
		"name": plan.Name.ValueString(),
	}
	if !plan.AllowWildcardSubdomains.IsNull() && !plan.AllowWildcardSubdomains.IsUnknown() {
		body["allow_wildcard_subdomains"] = plan.AllowWildcardSubdomains.ValueBool()
	}
	if !plan.WwwRedirectType.IsNull() && !plan.WwwRedirectType.IsUnknown() {
		body["www_redirect_type"] = plan.WwwRedirectType.ValueString()
	}

	collection := fmt.Sprintf("/orgs/%s/servers/%d/sites/%d/domains",
		plan.Organization.ValueString(), plan.Server.ValueInt64(), plan.Site.ValueInt64())

	// Creation is asynchronous (202, no body); resolve the new domain by name.
	before, err := providerutil.ListIDs(ctx, r.client, collection)
	if err != nil {
		resp.Diagnostics.AddError("Unable to list site domains", err.Error())
		return
	}
	if err := r.client.Do(ctx, http.MethodPost, collection, body, nil); err != nil {
		resp.Diagnostics.AddError("Unable to create site domain", err.Error())
		return
	}
	id, err := providerutil.ResolveCreatedID(ctx, r.client, collection, plan.Name.ValueString(), before)
	if err != nil {
		resp.Diagnostics.AddError("Unable to locate created site domain", err.Error())
		return
	}

	var data siteDomainData
	if err := r.client.Do(ctx, http.MethodGet, collection+"/"+id, nil, &data); err != nil {
		resp.Diagnostics.AddError("Unable to read created site domain", err.Error())
		return
	}

	data.apply(&plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *siteDomainsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state SiteDomainsModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var data siteDomainData
	endpoint := fmt.Sprintf("/orgs/%s/servers/%d/sites/%d/domains/%s",
		state.Organization.ValueString(), state.Server.ValueInt64(), state.Site.ValueInt64(), state.Id.ValueString())
	if err := r.client.Do(ctx, http.MethodGet, endpoint, nil, &data); err != nil {
		if forge.NotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Unable to read site domain", err.Error())
		return
	}

	data.apply(&state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *siteDomainsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan SiteDomainsModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]any{}
	if !plan.AllowWildcardSubdomains.IsNull() && !plan.AllowWildcardSubdomains.IsUnknown() {
		body["allow_wildcard_subdomains"] = plan.AllowWildcardSubdomains.ValueBool()
	}
	if !plan.WwwRedirectType.IsNull() && !plan.WwwRedirectType.IsUnknown() {
		body["www_redirect_type"] = plan.WwwRedirectType.ValueString()
	}

	endpoint := fmt.Sprintf("/orgs/%s/servers/%d/sites/%d/domains/%s",
		plan.Organization.ValueString(), plan.Server.ValueInt64(), plan.Site.ValueInt64(), plan.Id.ValueString())
	if err := r.client.Do(ctx, http.MethodPut, endpoint, body, nil); err != nil {
		resp.Diagnostics.AddError("Unable to update site domain", err.Error())
		return
	}

	// The update may be applied asynchronously; re-read to refresh state.
	var data siteDomainData
	if err := r.client.Do(ctx, http.MethodGet, endpoint, nil, &data); err != nil {
		resp.Diagnostics.AddError("Unable to read updated site domain", err.Error())
		return
	}

	data.apply(&plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *siteDomainsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state SiteDomainsModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	endpoint := fmt.Sprintf("/orgs/%s/servers/%d/sites/%d/domains/%s",
		state.Organization.ValueString(), state.Server.ValueInt64(), state.Site.ValueInt64(), state.Id.ValueString())
	if err := r.client.Do(ctx, http.MethodDelete, endpoint, nil, nil); err != nil && !forge.NotFound(err) {
		resp.Diagnostics.AddError("Unable to delete site domain", err.Error())
	}
}

func (r *siteDomainsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts, err := providerutil.ParseID(req.ID, 4)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID",
			fmt.Sprintf("Expected import ID in the form \"organization/server/site/domain_record\": %s.", err))
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

// siteDomainData is the JSON:API representation of a site domain.
type siteDomainData struct {
	ID         string `json:"id"`
	Attributes struct {
		Name                    string  `json:"name"`
		AllowWildcardSubdomains *bool   `json:"allow_wildcard_subdomains"`
		WwwRedirectType         *string `json:"www_redirect_type"`
		Status                  string  `json:"status"`
		CreatedAt               string  `json:"created_at"`
		UpdatedAt               string  `json:"updated_at"`
	} `json:"attributes"`
}

func (d *siteDomainData) apply(m *SiteDomainsModel) {
	m.Id = types.StringValue(d.ID)
	m.Name = types.StringValue(d.Attributes.Name)
	m.Status = types.StringValue(d.Attributes.Status)
	m.CreatedAt = types.StringValue(d.Attributes.CreatedAt)
	m.UpdatedAt = types.StringValue(d.Attributes.UpdatedAt)
	if d.Attributes.AllowWildcardSubdomains != nil {
		m.AllowWildcardSubdomains = types.BoolValue(*d.Attributes.AllowWildcardSubdomains)
	}
	if d.Attributes.WwwRedirectType != nil {
		m.WwwRedirectType = types.StringValue(*d.Attributes.WwwRedirectType)
	}
	if id, err := providerutil.ParseInt64(d.ID); err == nil {
		m.DomainRecord = types.Int64Value(id)
	}
}
