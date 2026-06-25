package resource_deployments

// This file implements the deployments resource. Unlike *_resource_gen.go it is
// NOT regenerated, so the CRUD logic below is safe to edit.

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

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = (*deploymentsResource)(nil)
	_ resource.ResourceWithConfigure   = (*deploymentsResource)(nil)
	_ resource.ResourceWithImportState = (*deploymentsResource)(nil)
)

// NewDeploymentsResource returns a new deployments resource.
func NewDeploymentsResource() resource.Resource {
	return &deploymentsResource{}
}

type deploymentsResource struct {
	client *forge.Client
}

func (r *deploymentsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_deployments"
}

func (r *deploymentsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = providerutil.Client(req, resp)
}

func (r *deploymentsResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	s := DeploymentsResourceSchema(ctx)

	// A deployment is an action triggered against a site; the parents are the
	// only configurable inputs and everything else is reported by the API.
	providerutil.Required(s.Attributes, "organization")
	providerutil.Required(s.Attributes, "server")
	providerutil.Required(s.Attributes, "site")
	providerutil.UseStateForUnknown(s.Attributes, "id", "deployment")

	resp.Schema = s
}

func (r *deploymentsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan DeploymentsModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	collection := fmt.Sprintf("/orgs/%s/servers/%d/sites/%d/deployments",
		plan.Organization.ValueString(), plan.Server.ValueInt64(), plan.Site.ValueInt64())

	// Triggering a deployment is asynchronous and returns no body, so snapshot
	// the existing deployments, fire the request, then resolve the new id.
	before, err := providerutil.ListIDs(ctx, r.client, collection)
	if err != nil {
		resp.Diagnostics.AddError("Unable to list deployments", err.Error())
		return
	}
	if err := r.client.Do(ctx, http.MethodPost, collection, nil, nil); err != nil {
		resp.Diagnostics.AddError("Unable to trigger deployment", err.Error())
		return
	}
	id, err := providerutil.ResolveCreatedID(ctx, r.client, collection, "", before)
	if err != nil {
		resp.Diagnostics.AddError("Unable to locate triggered deployment", err.Error())
		return
	}

	var data deploymentData
	if err := r.client.Do(ctx, http.MethodGet, collection+"/"+id, nil, &data); err != nil {
		resp.Diagnostics.AddError("Unable to read triggered deployment", err.Error())
		return
	}

	data.apply(&plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *deploymentsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state DeploymentsModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var data deploymentData
	endpoint := fmt.Sprintf("/orgs/%s/servers/%d/sites/%d/deployments/%s",
		state.Organization.ValueString(), state.Server.ValueInt64(), state.Site.ValueInt64(), state.Id.ValueString())
	if err := r.client.Do(ctx, http.MethodGet, endpoint, nil, &data); err != nil {
		if forge.NotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Unable to read deployment", err.Error())
		return
	}

	data.apply(&state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update is never invoked with real changes: the parents force replacement and
// every other attribute is computed. It simply persists the plan.
func (r *deploymentsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan DeploymentsModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete is a no-op: the Forge API has no endpoint to delete a deployment
// record, so removing the resource only drops it from Terraform state.
func (r *deploymentsResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
}

// ImportState supports `terraform import` using the
// "organization/server/site/deployment" identifier form.
func (r *deploymentsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts, err := providerutil.ParseID(req.ID, 4)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID",
			fmt.Sprintf("Expected import ID in the form \"organization/server/site/deployment\": %s.", err))
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

// deploymentData is the JSON:API representation of a deployment.
type deploymentData struct {
	ID         string `json:"id"`
	Attributes struct {
		Status    string `json:"status"`
		StartedAt string `json:"started_at"`
		EndedAt   string `json:"ended_at"`
		CreatedAt string `json:"created_at"`
		UpdatedAt string `json:"updated_at"`
		Commit    *struct {
			Author  string `json:"author"`
			Branch  string `json:"branch"`
			Hash    string `json:"hash"`
			Message string `json:"message"`
		} `json:"commit"`
	} `json:"attributes"`
}

func (d *deploymentData) apply(m *DeploymentsModel) {
	m.Id = types.StringValue(d.ID)
	m.Status = types.StringValue(d.Attributes.Status)
	m.StartedAt = types.StringValue(d.Attributes.StartedAt)
	m.EndedAt = types.StringValue(d.Attributes.EndedAt)
	m.CreatedAt = types.StringValue(d.Attributes.CreatedAt)
	m.UpdatedAt = types.StringValue(d.Attributes.UpdatedAt)
	if id, err := providerutil.ParseInt64(d.ID); err == nil {
		m.Deployment = types.Int64Value(id)
	}
	if d.Attributes.Commit != nil {
		m.Commit = NewCommitValueMust(
			CommitValue{}.AttributeTypes(context.Background()),
			map[string]attr.Value{
				"author":  types.StringValue(d.Attributes.Commit.Author),
				"branch":  types.StringValue(d.Attributes.Commit.Branch),
				"hash":    types.StringValue(d.Attributes.Commit.Hash),
				"message": types.StringValue(d.Attributes.Commit.Message),
			},
		)
	} else {
		m.Commit = NewCommitValueNull()
	}
}
