package resource_redirect_rules

// This file implements the redirect_rules resource. Unlike *_resource_gen.go it
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
	_ resource.Resource                = (*redirectRulesResource)(nil)
	_ resource.ResourceWithConfigure   = (*redirectRulesResource)(nil)
	_ resource.ResourceWithImportState = (*redirectRulesResource)(nil)
)

// NewRedirectRulesResource returns a new redirect_rules resource.
func NewRedirectRulesResource() resource.Resource {
	return &redirectRulesResource{}
}

type redirectRulesResource struct {
	client *forge.Client
}

func (r *redirectRulesResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_redirect_rules"
}

func (r *redirectRulesResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = providerutil.Client(req, resp)
}

func (r *redirectRulesResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	s := RedirectRulesResourceSchema(ctx)

	providerutil.Required(s.Attributes, "organization")
	providerutil.Required(s.Attributes, "server")
	providerutil.Required(s.Attributes, "site")
	// Redirect rules are immutable (the API has no update endpoint), so every
	// create-body attribute forces replacement on change.
	providerutil.RequiresReplace(s.Attributes, "from", "to", "type")
	providerutil.UseStateForUnknown(s.Attributes, "id", "redirect_rule")

	s.MarkdownDescription = "Add a new redirect rule to the site."
	resp.Schema = s
}

func (r *redirectRulesResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan RedirectRulesModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]any{
		"from": plan.From.ValueString(),
		"to":   plan.To.ValueString(),
		"type": plan.Type.ValueString(),
	}

	collection := fmt.Sprintf("/orgs/%s/servers/%d/sites/%d/redirect-rules",
		plan.Organization.ValueString(), plan.Server.ValueInt64(), plan.Site.ValueInt64())

	// Creation is asynchronous (202, no body) and the resource has no name, so
	// resolve the new rule by the id that appears after creation.
	before, err := providerutil.ListIDs(ctx, r.client, collection)
	if err != nil {
		resp.Diagnostics.AddError("Unable to list redirect rules", err.Error())
		return
	}
	if err := r.client.Do(ctx, http.MethodPost, collection, body, nil); err != nil {
		resp.Diagnostics.AddError("Unable to create redirect rule", err.Error())
		return
	}
	id, err := providerutil.ResolveCreatedID(ctx, r.client, collection, "", before)
	if err != nil {
		resp.Diagnostics.AddError("Unable to locate created redirect rule", err.Error())
		return
	}

	var data redirectRuleData
	if err := r.client.Do(ctx, http.MethodGet, collection+"/"+id, nil, &data); err != nil {
		resp.Diagnostics.AddError("Unable to read created redirect rule", err.Error())
		return
	}

	data.apply(&plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *redirectRulesResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state RedirectRulesModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var data redirectRuleData
	endpoint := fmt.Sprintf("/orgs/%s/servers/%d/sites/%d/redirect-rules/%s",
		state.Organization.ValueString(), state.Server.ValueInt64(), state.Site.ValueInt64(), state.Id.ValueString())
	if err := r.client.Do(ctx, http.MethodGet, endpoint, nil, &data); err != nil {
		if forge.NotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Unable to read redirect rule", err.Error())
		return
	}

	data.apply(&state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update is required by the interface but is never invoked with in-place
// changes: every configurable attribute is marked RequiresReplace, so Terraform
// destroys and recreates the rule instead.
func (r *redirectRulesResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan RedirectRulesModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *redirectRulesResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state RedirectRulesModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	endpoint := fmt.Sprintf("/orgs/%s/servers/%d/sites/%d/redirect-rules/%s",
		state.Organization.ValueString(), state.Server.ValueInt64(), state.Site.ValueInt64(), state.Id.ValueString())
	if err := r.client.Do(ctx, http.MethodDelete, endpoint, nil, nil); err != nil && !forge.NotFound(err) {
		resp.Diagnostics.AddError("Unable to delete redirect rule", err.Error())
	}
}

func (r *redirectRulesResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts, err := providerutil.ParseID(req.ID, 4)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID",
			fmt.Sprintf("Expected import ID in the form \"organization/server/site/redirect_rule\": %s.", err))
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

// redirectRuleData is the JSON:API representation of a redirect rule.
type redirectRuleData struct {
	ID         string `json:"id"`
	Attributes struct {
		From      string `json:"from"`
		To        string `json:"to"`
		Type      string `json:"type"`
		Status    string `json:"status"`
		CreatedAt string `json:"created_at"`
		UpdatedAt string `json:"updated_at"`
	} `json:"attributes"`
}

func (d *redirectRuleData) apply(m *RedirectRulesModel) {
	m.Id = types.StringValue(d.ID)
	m.From = types.StringValue(d.Attributes.From)
	m.To = types.StringValue(d.Attributes.To)
	m.Type = types.StringValue(d.Attributes.Type)
	m.Status = types.StringValue(d.Attributes.Status)
	m.CreatedAt = types.StringValue(d.Attributes.CreatedAt)
	m.UpdatedAt = types.StringValue(d.Attributes.UpdatedAt)
	if id, err := providerutil.ParseInt64(d.ID); err == nil {
		m.RedirectRule = types.Int64Value(id)
	}
}
