package resource_security_rules

// This file implements the security_rules resource. Unlike *_resource_gen.go it
// is NOT regenerated, so the CRUD logic below is safe to edit.

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/madewithlove/terraform-provider-laravelforge/internal/forge"
	"github.com/madewithlove/terraform-provider-laravelforge/internal/providerutil"
)

var (
	_ resource.Resource                = (*securityRulesResource)(nil)
	_ resource.ResourceWithConfigure   = (*securityRulesResource)(nil)
	_ resource.ResourceWithImportState = (*securityRulesResource)(nil)
)

// NewSecurityRulesResource returns a new security_rules resource.
func NewSecurityRulesResource() resource.Resource {
	return &securityRulesResource{}
}

type securityRulesResource struct {
	client *forge.Client
}

func (r *securityRulesResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_security_rules"
}

func (r *securityRulesResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = providerutil.Client(req, resp)
}

func (r *securityRulesResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	s := SecurityRulesResourceSchema(ctx)

	providerutil.Required(s.Attributes, "organization")
	providerutil.Required(s.Attributes, "server")
	providerutil.Required(s.Attributes, "site")
	// name, path and credentials are all updatable in place via the API's PUT
	// endpoint, so none of them force replacement.
	providerutil.UseStateForUnknown(s.Attributes, "id", "security_rule")
	// credentials is a write-only input (the API never echoes the passwords back)
	// and is sensitive.
	if a, ok := s.Attributes["credentials"].(schema.ListNestedAttribute); ok {
		a.Sensitive = true
		s.Attributes["credentials"] = a
	}

	resp.Schema = s
}

func (r *securityRulesResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan SecurityRulesModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]any{
		"name": plan.Name.ValueString(),
	}
	if !plan.Path.IsNull() && !plan.Path.IsUnknown() {
		body["path"] = plan.Path.ValueString()
	}
	creds, diags := credentialsBody(ctx, plan.Credentials)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if creds != nil {
		body["credentials"] = creds
	}

	collection := fmt.Sprintf("/orgs/%s/servers/%d/sites/%d/security-rules",
		plan.Organization.ValueString(), plan.Server.ValueInt64(), plan.Site.ValueInt64())

	// Creation is asynchronous (202, no body); resolve the new rule by name.
	before, err := providerutil.ListIDs(ctx, r.client, collection)
	if err != nil {
		resp.Diagnostics.AddError("Unable to list security rules", err.Error())
		return
	}
	if err := r.client.Do(ctx, http.MethodPost, collection, body, nil); err != nil {
		resp.Diagnostics.AddError("Unable to create security rule", err.Error())
		return
	}
	id, err := providerutil.ResolveCreatedID(ctx, r.client, collection, plan.Name.ValueString(), before)
	if err != nil {
		resp.Diagnostics.AddError("Unable to locate created security rule", err.Error())
		return
	}

	var data securityRuleData
	if err := r.client.Do(ctx, http.MethodGet, collection+"/"+id, nil, &data); err != nil {
		resp.Diagnostics.AddError("Unable to read created security rule", err.Error())
		return
	}

	data.apply(&plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *securityRulesResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state SecurityRulesModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var data securityRuleData
	endpoint := fmt.Sprintf("/orgs/%s/servers/%d/sites/%d/security-rules/%s",
		state.Organization.ValueString(), state.Server.ValueInt64(), state.Site.ValueInt64(), state.Id.ValueString())
	if err := r.client.Do(ctx, http.MethodGet, endpoint, nil, &data); err != nil {
		if forge.NotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Unable to read security rule", err.Error())
		return
	}

	data.apply(&state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *securityRulesResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan SecurityRulesModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]any{
		"name": plan.Name.ValueString(),
	}
	if !plan.Path.IsNull() && !plan.Path.IsUnknown() {
		body["path"] = plan.Path.ValueString()
	}
	creds, diags := credentialsBody(ctx, plan.Credentials)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if creds != nil {
		body["credentials"] = creds
	}

	endpoint := fmt.Sprintf("/orgs/%s/servers/%d/sites/%d/security-rules/%s",
		plan.Organization.ValueString(), plan.Server.ValueInt64(), plan.Site.ValueInt64(), plan.Id.ValueString())
	if err := r.client.Do(ctx, http.MethodPut, endpoint, body, nil); err != nil {
		resp.Diagnostics.AddError("Unable to update security rule", err.Error())
		return
	}

	// The update may be applied asynchronously; re-read to refresh state.
	var data securityRuleData
	if err := r.client.Do(ctx, http.MethodGet, endpoint, nil, &data); err != nil {
		resp.Diagnostics.AddError("Unable to read updated security rule", err.Error())
		return
	}

	data.apply(&plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *securityRulesResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state SecurityRulesModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	endpoint := fmt.Sprintf("/orgs/%s/servers/%d/sites/%d/security-rules/%s",
		state.Organization.ValueString(), state.Server.ValueInt64(), state.Site.ValueInt64(), state.Id.ValueString())
	if err := r.client.Do(ctx, http.MethodDelete, endpoint, nil, nil); err != nil && !forge.NotFound(err) {
		resp.Diagnostics.AddError("Unable to delete security rule", err.Error())
	}
}

func (r *securityRulesResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts, err := providerutil.ParseID(req.ID, 4)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID",
			fmt.Sprintf("Expected import ID in the form \"organization/server/site/security_rule\": %s.", err))
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

// credentialsBody extracts the credentials list into the API request shape,
// returning nil when the list is null or unknown.
func credentialsBody(ctx context.Context, list types.List) ([]map[string]any, diag.Diagnostics) {
	if list.IsNull() || list.IsUnknown() {
		return nil, nil
	}
	var values []CredentialsValue
	diags := list.ElementsAs(ctx, &values, false)
	if diags.HasError() {
		return nil, diags
	}
	creds := make([]map[string]any, 0, len(values))
	for _, v := range values {
		creds = append(creds, map[string]any{
			"username": v.Username.ValueString(),
			"password": v.Password.ValueString(),
		})
	}
	return creds, diags
}

// securityRuleData is the JSON:API representation of a security rule. The
// credentials (passwords in particular) are write-only and not returned.
type securityRuleData struct {
	ID         string `json:"id"`
	Attributes struct {
		Name      string  `json:"name"`
		Path      *string `json:"path"`
		Status    string  `json:"status"`
		CreatedAt string  `json:"created_at"`
		UpdatedAt string  `json:"updated_at"`
	} `json:"attributes"`
}

func (d *securityRuleData) apply(m *SecurityRulesModel) {
	m.Id = types.StringValue(d.ID)
	m.Name = types.StringValue(d.Attributes.Name)
	m.Status = types.StringValue(d.Attributes.Status)
	m.CreatedAt = types.StringValue(d.Attributes.CreatedAt)
	m.UpdatedAt = types.StringValue(d.Attributes.UpdatedAt)
	if d.Attributes.Path != nil {
		m.Path = types.StringValue(*d.Attributes.Path)
	}
	if id, err := providerutil.ParseInt64(d.ID); err == nil {
		m.SecurityRule = types.Int64Value(id)
	}
}
