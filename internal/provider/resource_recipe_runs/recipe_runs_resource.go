package resource_recipe_runs

// This file implements the recipe_runs resource. Unlike *_resource_gen.go it is
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
	_ resource.Resource                = (*recipeRunsResource)(nil)
	_ resource.ResourceWithConfigure   = (*recipeRunsResource)(nil)
	_ resource.ResourceWithImportState = (*recipeRunsResource)(nil)
)

// NewRecipeRunsResource returns a new recipe_runs resource.
func NewRecipeRunsResource() resource.Resource {
	return &recipeRunsResource{}
}

type recipeRunsResource struct {
	client *forge.Client
}

func (r *recipeRunsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_recipe_runs"
}

func (r *recipeRunsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = providerutil.Client(req, resp)
}

func (r *recipeRunsResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	s := RecipeRunsResourceSchema(ctx)

	// A recipe run is an action triggered against a recipe. The parents and the
	// write-only inputs (servers, email) fully determine the run, so any change
	// forces a fresh run.
	providerutil.Required(s.Attributes, "organization")
	providerutil.Required(s.Attributes, "recipe")
	providerutil.Input(s.Attributes, "email")
	providerutil.RequiresReplace(s.Attributes, "servers", "email")
	providerutil.UseStateForUnknown(s.Attributes, "id", "log")

	s.MarkdownDescription = "Runs a recipe on one or more servers."
	resp.Schema = s
}

func (r *recipeRunsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan RecipeRunsModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var servers []int64
	resp.Diagnostics.Append(plan.Servers.ElementsAs(ctx, &servers, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]any{
		"servers": servers,
	}
	if !plan.Email.IsNull() && !plan.Email.IsUnknown() {
		body["email"] = plan.Email.ValueBool()
	}

	collection := fmt.Sprintf("/orgs/%s/recipes/%d/runs",
		plan.Organization.ValueString(), plan.Recipe.ValueInt64())

	// Running a recipe is asynchronous and returns no body, so snapshot the
	// existing runs, fire the request, then resolve the new log id.
	before, err := providerutil.ListIDs(ctx, r.client, collection)
	if err != nil {
		resp.Diagnostics.AddError("Unable to list recipe runs", err.Error())
		return
	}
	if err := r.client.Do(ctx, http.MethodPost, collection, body, nil); err != nil {
		resp.Diagnostics.AddError("Unable to run recipe", err.Error())
		return
	}
	id, err := providerutil.ResolveCreatedID(ctx, r.client, collection, "", before)
	if err != nil {
		resp.Diagnostics.AddError("Unable to locate recipe run", err.Error())
		return
	}

	var data recipeRunData
	if err := r.client.Do(ctx, http.MethodGet, collection+"/"+id, nil, &data); err != nil {
		resp.Diagnostics.AddError("Unable to read recipe run", err.Error())
		return
	}

	data.apply(&plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *recipeRunsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state RecipeRunsModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var data recipeRunData
	endpoint := fmt.Sprintf("/orgs/%s/recipes/%d/runs/%s",
		state.Organization.ValueString(), state.Recipe.ValueInt64(), state.Id.ValueString())
	if err := r.client.Do(ctx, http.MethodGet, endpoint, nil, &data); err != nil {
		if forge.NotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Unable to read recipe run", err.Error())
		return
	}

	data.apply(&state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update is never invoked with real changes: every configurable attribute
// forces replacement. It simply persists the plan.
func (r *recipeRunsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan RecipeRunsModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete is a no-op: the Forge API has no endpoint to delete a recipe run, so
// removing the resource only drops it from Terraform state.
func (r *recipeRunsResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
}

// ImportState supports `terraform import` using the
// "organization/recipe/log" identifier form.
func (r *recipeRunsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts, err := providerutil.ParseID(req.ID, 3)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID",
			fmt.Sprintf("Expected import ID in the form \"organization/recipe/log\": %s.", err))
		return
	}
	recipe, err := providerutil.ParseInt64(parts[1])
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("Recipe must be an integer: %s.", err))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("organization"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("recipe"), recipe)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[2])...)
}

// recipeRunData is the JSON:API representation of a recipe run.
type recipeRunData struct {
	ID         string `json:"id"`
	Attributes struct {
		Status     string `json:"status"`
		Output     string `json:"output"`
		RecipeID   *int64 `json:"recipe_id"`
		ServerID   *int64 `json:"server_id"`
		ExecutedBy *int64 `json:"executed_by"`
		StartedAt  string `json:"started_at"`
		FinishedAt string `json:"finished_at"`
	} `json:"attributes"`
}

// apply copies the API response onto the model. The "servers" and "email"
// attributes are write-only inputs and are left untouched.
func (d *recipeRunData) apply(m *RecipeRunsModel) {
	m.Id = types.StringValue(d.ID)
	m.Status = types.StringValue(d.Attributes.Status)
	m.Output = types.StringValue(d.Attributes.Output)
	m.StartedAt = types.StringValue(d.Attributes.StartedAt)
	m.FinishedAt = types.StringValue(d.Attributes.FinishedAt)
	if id, err := providerutil.ParseInt64(d.ID); err == nil {
		m.Log = types.Int64Value(id)
	}
	if d.Attributes.RecipeID != nil {
		m.RecipeId = types.Int64Value(*d.Attributes.RecipeID)
	} else {
		m.RecipeId = types.Int64Null()
	}
	if d.Attributes.ServerID != nil {
		m.ServerId = types.Int64Value(*d.Attributes.ServerID)
	} else {
		m.ServerId = types.Int64Null()
	}
	if d.Attributes.ExecutedBy != nil {
		m.ExecutedBy = types.Int64Value(*d.Attributes.ExecutedBy)
	} else {
		m.ExecutedBy = types.Int64Null()
	}
}
