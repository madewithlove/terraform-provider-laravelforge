package resource_site_scheduled_jobs

// This file implements the site_scheduled_jobs resource. Unlike *_resource_gen.go
// it is NOT regenerated, so the CRUD logic below is safe to edit.

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
	_ resource.Resource                = (*siteScheduledJobsResource)(nil)
	_ resource.ResourceWithConfigure   = (*siteScheduledJobsResource)(nil)
	_ resource.ResourceWithImportState = (*siteScheduledJobsResource)(nil)
)

// NewSiteScheduledJobsResource returns a new site_scheduled_jobs resource.
func NewSiteScheduledJobsResource() resource.Resource {
	return &siteScheduledJobsResource{}
}

type siteScheduledJobsResource struct {
	client *forge.Client
}

func (r *siteScheduledJobsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_site_scheduled_jobs"
}

func (r *siteScheduledJobsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = providerutil.Client(req, resp)
}

func (r *siteScheduledJobsResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	s := SiteScheduledJobsResourceSchema(ctx)

	providerutil.Required(s.Attributes, "organization")
	providerutil.Required(s.Attributes, "server")
	providerutil.Required(s.Attributes, "site")
	// Scheduled jobs are immutable (the API has no update endpoint), so every
	// create-body attribute forces replacement on change.
	providerutil.RequiresReplace(s.Attributes, "command", "cron", "frequency", "grace_period", "heartbeat", "name", "user")
	providerutil.UseStateForUnknown(s.Attributes, "id", "job")

	s.MarkdownDescription = "Manages a scheduled job (cron) for a site."
	resp.Schema = s
}

func (r *siteScheduledJobsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan SiteScheduledJobsModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]any{
		"command":   plan.Command.ValueString(),
		"frequency": plan.Frequency.ValueString(),
		"user":      plan.User.ValueString(),
	}
	if !plan.Cron.IsNull() && !plan.Cron.IsUnknown() {
		body["cron"] = plan.Cron.ValueString()
	}
	if !plan.GracePeriod.IsNull() && !plan.GracePeriod.IsUnknown() {
		body["grace_period"] = plan.GracePeriod.ValueString()
	}
	if !plan.Heartbeat.IsNull() && !plan.Heartbeat.IsUnknown() {
		body["heartbeat"] = plan.Heartbeat.ValueBool()
	}
	if !plan.Name.IsNull() && !plan.Name.IsUnknown() {
		body["name"] = plan.Name.ValueString()
	}

	collection := fmt.Sprintf("/orgs/%s/servers/%d/sites/%d/scheduled-jobs",
		plan.Organization.ValueString(), plan.Server.ValueInt64(), plan.Site.ValueInt64())

	// Creation is asynchronous (202, no body); resolve the new job by name.
	before, err := providerutil.ListIDs(ctx, r.client, collection)
	if err != nil {
		resp.Diagnostics.AddError("Unable to list scheduled jobs", err.Error())
		return
	}
	if err := r.client.Do(ctx, http.MethodPost, collection, body, nil); err != nil {
		resp.Diagnostics.AddError("Unable to create scheduled job", err.Error())
		return
	}
	id, err := providerutil.ResolveCreatedID(ctx, r.client, collection, plan.Name.ValueString(), before)
	if err != nil {
		resp.Diagnostics.AddError("Unable to locate created scheduled job", err.Error())
		return
	}

	var data scheduledJobData
	if err := r.client.Do(ctx, http.MethodGet, collection+"/"+id, nil, &data); err != nil {
		resp.Diagnostics.AddError("Unable to read created scheduled job", err.Error())
		return
	}

	data.apply(&plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *siteScheduledJobsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state SiteScheduledJobsModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var data scheduledJobData
	endpoint := fmt.Sprintf("/orgs/%s/servers/%d/sites/%d/scheduled-jobs/%s",
		state.Organization.ValueString(), state.Server.ValueInt64(), state.Site.ValueInt64(), state.Id.ValueString())
	if err := r.client.Do(ctx, http.MethodGet, endpoint, nil, &data); err != nil {
		if forge.NotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Unable to read scheduled job", err.Error())
		return
	}

	data.apply(&state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update is required by the interface but is never invoked with in-place
// changes: every configurable attribute is marked RequiresReplace, so Terraform
// destroys and recreates the job instead.
func (r *siteScheduledJobsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan SiteScheduledJobsModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *siteScheduledJobsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state SiteScheduledJobsModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	endpoint := fmt.Sprintf("/orgs/%s/servers/%d/sites/%d/scheduled-jobs/%s",
		state.Organization.ValueString(), state.Server.ValueInt64(), state.Site.ValueInt64(), state.Id.ValueString())
	if err := r.client.Do(ctx, http.MethodDelete, endpoint, nil, nil); err != nil && !forge.NotFound(err) {
		resp.Diagnostics.AddError("Unable to delete scheduled job", err.Error())
	}
}

func (r *siteScheduledJobsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts, err := providerutil.ParseID(req.ID, 4)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID",
			fmt.Sprintf("Expected import ID in the form \"organization/server/site/job\": %s.", err))
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

// scheduledJobData is the JSON:API representation of a site scheduled job.
type scheduledJobData struct {
	ID         string `json:"id"`
	Attributes struct {
		Command     string  `json:"command"`
		Cron        *string `json:"cron"`
		Frequency   string  `json:"frequency"`
		GracePeriod *string `json:"grace_period"`
		Heartbeat   *bool   `json:"heartbeat"`
		Name        *string `json:"name"`
		User        string  `json:"user"`
		NextRunTime *string `json:"next_run_time"`
		Status      string  `json:"status"`
		CreatedAt   string  `json:"created_at"`
		UpdatedAt   string  `json:"updated_at"`
	} `json:"attributes"`
}

func (d *scheduledJobData) apply(m *SiteScheduledJobsModel) {
	m.Id = types.StringValue(d.ID)
	m.Command = types.StringValue(d.Attributes.Command)
	m.Frequency = types.StringValue(d.Attributes.Frequency)
	m.User = types.StringValue(d.Attributes.User)
	m.Status = types.StringValue(d.Attributes.Status)
	m.CreatedAt = types.StringValue(d.Attributes.CreatedAt)
	m.UpdatedAt = types.StringValue(d.Attributes.UpdatedAt)
	if d.Attributes.Cron != nil {
		m.Cron = types.StringValue(*d.Attributes.Cron)
	}
	if d.Attributes.GracePeriod != nil {
		m.GracePeriod = types.StringValue(*d.Attributes.GracePeriod)
	}
	if d.Attributes.Heartbeat != nil {
		m.Heartbeat = types.BoolValue(*d.Attributes.Heartbeat)
	}
	if d.Attributes.Name != nil {
		m.Name = types.StringValue(*d.Attributes.Name)
	}
	if d.Attributes.NextRunTime != nil {
		m.NextRunTime = types.StringValue(*d.Attributes.NextRunTime)
	} else {
		m.NextRunTime = types.StringNull()
	}
	if id, err := providerutil.ParseInt64(d.ID); err == nil {
		m.Job = types.Int64Value(id)
	}
}
