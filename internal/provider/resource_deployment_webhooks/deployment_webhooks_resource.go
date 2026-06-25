package resource_deployment_webhooks

// This file implements the deployment_webhooks resource. Unlike
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
	_ resource.Resource                = (*deploymentWebhooksResource)(nil)
	_ resource.ResourceWithConfigure   = (*deploymentWebhooksResource)(nil)
	_ resource.ResourceWithImportState = (*deploymentWebhooksResource)(nil)
)

// NewDeploymentWebhooksResource returns a new deployment_webhooks resource.
func NewDeploymentWebhooksResource() resource.Resource {
	return &deploymentWebhooksResource{}
}

type deploymentWebhooksResource struct {
	client *forge.Client
}

func (r *deploymentWebhooksResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_deployment_webhooks"
}

func (r *deploymentWebhooksResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = providerutil.Client(req, resp)
}

func (r *deploymentWebhooksResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	s := DeploymentWebhooksResourceSchema(ctx)

	providerutil.Required(s.Attributes, "organization")
	providerutil.Required(s.Attributes, "server")
	providerutil.Required(s.Attributes, "site")
	// The API has no update endpoint, so the only configurable attribute (url)
	// forces replacement on change.
	providerutil.RequiresReplace(s.Attributes, "url")
	providerutil.UseStateForUnknown(s.Attributes, "id", "deployment_webhook")

	resp.Schema = s
}

func (r *deploymentWebhooksResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan DeploymentWebhooksModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]any{
		"url": plan.Url.ValueString(),
	}

	collection := fmt.Sprintf("/orgs/%s/servers/%d/sites/%d/webhooks",
		plan.Organization.ValueString(), plan.Server.ValueInt64(), plan.Site.ValueInt64())

	// Creation is asynchronous (202, no body) and the webhook has no name, so
	// snapshot the existing ids and resolve the newly-appeared one.
	before, err := providerutil.ListIDs(ctx, r.client, collection)
	if err != nil {
		resp.Diagnostics.AddError("Unable to list deployment webhooks", err.Error())
		return
	}
	if err := r.client.Do(ctx, http.MethodPost, collection, body, nil); err != nil {
		resp.Diagnostics.AddError("Unable to create deployment webhook", err.Error())
		return
	}
	id, err := providerutil.ResolveCreatedID(ctx, r.client, collection, "", before)
	if err != nil {
		resp.Diagnostics.AddError("Unable to locate created deployment webhook", err.Error())
		return
	}

	var data deploymentWebhookData
	if err := r.client.Do(ctx, http.MethodGet, collection+"/"+id, nil, &data); err != nil {
		resp.Diagnostics.AddError("Unable to read created deployment webhook", err.Error())
		return
	}

	data.apply(&plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *deploymentWebhooksResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state DeploymentWebhooksModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var data deploymentWebhookData
	endpoint := fmt.Sprintf("/orgs/%s/servers/%d/sites/%d/webhooks/%s",
		state.Organization.ValueString(), state.Server.ValueInt64(), state.Site.ValueInt64(), state.Id.ValueString())
	if err := r.client.Do(ctx, http.MethodGet, endpoint, nil, &data); err != nil {
		if forge.NotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Unable to read deployment webhook", err.Error())
		return
	}

	data.apply(&state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update is required by the interface but is never invoked with in-place
// changes: the only configurable attribute is marked RequiresReplace, so
// Terraform destroys and recreates the webhook instead.
func (r *deploymentWebhooksResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan DeploymentWebhooksModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *deploymentWebhooksResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state DeploymentWebhooksModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	endpoint := fmt.Sprintf("/orgs/%s/servers/%d/sites/%d/webhooks/%s",
		state.Organization.ValueString(), state.Server.ValueInt64(), state.Site.ValueInt64(), state.Id.ValueString())
	if err := r.client.Do(ctx, http.MethodDelete, endpoint, nil, nil); err != nil && !forge.NotFound(err) {
		resp.Diagnostics.AddError("Unable to delete deployment webhook", err.Error())
	}
}

func (r *deploymentWebhooksResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts, err := providerutil.ParseID(req.ID, 4)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID",
			fmt.Sprintf("Expected import ID in the form \"organization/server/site/deployment_webhook\": %s.", err))
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

// deploymentWebhookData is the JSON:API representation of a deployment webhook.
type deploymentWebhookData struct {
	ID         string `json:"id"`
	Attributes struct {
		Url       string `json:"url"`
		CreatedAt string `json:"created_at"`
		UpdatedAt string `json:"updated_at"`
	} `json:"attributes"`
}

func (d *deploymentWebhookData) apply(m *DeploymentWebhooksModel) {
	m.Id = types.StringValue(d.ID)
	m.Url = types.StringValue(d.Attributes.Url)
	m.CreatedAt = types.StringValue(d.Attributes.CreatedAt)
	m.UpdatedAt = types.StringValue(d.Attributes.UpdatedAt)
	if id, err := providerutil.ParseInt64(d.ID); err == nil {
		m.DeploymentWebhook = types.Int64Value(id)
	}
}
