package resource_domain_certificates

// This file implements the domain_certificates resource. Unlike
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

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = (*domainCertificatesResource)(nil)
	_ resource.ResourceWithConfigure   = (*domainCertificatesResource)(nil)
	_ resource.ResourceWithImportState = (*domainCertificatesResource)(nil)
)

// NewDomainCertificatesResource returns a new domain_certificates resource.
func NewDomainCertificatesResource() resource.Resource {
	return &domainCertificatesResource{}
}

type domainCertificatesResource struct {
	client *forge.Client
}

func (r *domainCertificatesResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domain_certificates"
}

func (r *domainCertificatesResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = providerutil.Client(req, resp)
}

func (r *domainCertificatesResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	s := DomainCertificatesResourceSchema(ctx)

	// A certificate is a singleton under a domain record; there is no update
	// endpoint, so every configurable input forces replacement.
	providerutil.Required(s.Attributes, "organization")
	providerutil.Required(s.Attributes, "server")
	providerutil.Required(s.Attributes, "site")
	providerutil.Required(s.Attributes, "domain_record")
	providerutil.RequiresReplace(s.Attributes, "type", "clone", "csr", "existing", "letsencrypt")
	providerutil.UseStateForUnknown(s.Attributes, "id")

	s.MarkdownDescription = "Create a new certificate for a given domain."
	resp.Schema = s
}

// path returns the singleton certificate endpoint (collection == item).
func (r *domainCertificatesResource) path(organization string, server, site, domainRecord int64) string {
	return fmt.Sprintf("/orgs/%s/servers/%d/sites/%d/domains/%d/certificate",
		organization, server, site, domainRecord)
}

func (r *domainCertificatesResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan DomainCertificatesModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]any{
		"type": plan.Type.ValueString(),
	}
	if !plan.Clone.IsNull() && !plan.Clone.IsUnknown() {
		clone := map[string]any{}
		if !plan.Clone.CertificateId.IsNull() && !plan.Clone.CertificateId.IsUnknown() {
			clone["certificate_id"] = plan.Clone.CertificateId.ValueInt64()
		}
		body["clone"] = clone
	}
	if !plan.Existing.IsNull() && !plan.Existing.IsUnknown() {
		existing := map[string]any{}
		if !plan.Existing.Certificate.IsNull() && !plan.Existing.Certificate.IsUnknown() {
			existing["certificate"] = plan.Existing.Certificate.ValueString()
		}
		if !plan.Existing.Key.IsNull() && !plan.Existing.Key.IsUnknown() {
			existing["key"] = plan.Existing.Key.ValueString()
		}
		body["existing"] = existing
	}
	if !plan.Csr.IsNull() && !plan.Csr.IsUnknown() {
		csr := map[string]any{}
		setString(csr, "city", plan.Csr.City)
		setString(csr, "country", plan.Csr.Country)
		setString(csr, "department", plan.Csr.Department)
		setString(csr, "domain", plan.Csr.Domain)
		setString(csr, "organization", plan.Csr.Organization)
		setString(csr, "sans", plan.Csr.Sans)
		setString(csr, "state", plan.Csr.State)
		body["csr"] = csr
	}
	if !plan.Letsencrypt.IsNull() && !plan.Letsencrypt.IsUnknown() {
		le := map[string]any{}
		setString(le, "key_type", plan.Letsencrypt.KeyType)
		setString(le, "preferred_chain", plan.Letsencrypt.PreferredChain)
		setString(le, "verification_method", plan.Letsencrypt.VerificationMethod)
		body["letsencrypt"] = le
	}

	endpoint := r.path(plan.Organization.ValueString(), plan.Server.ValueInt64(),
		plan.Site.ValueInt64(), plan.DomainRecord.ValueInt64())

	// Requesting a certificate is asynchronous and returns no body.
	if err := r.client.Do(ctx, http.MethodPost, endpoint, body, nil); err != nil {
		resp.Diagnostics.AddError("Unable to request certificate", err.Error())
		return
	}

	var data certificateData
	if err := r.client.Do(ctx, http.MethodGet, endpoint, nil, &data); err != nil {
		resp.Diagnostics.AddError("Unable to read created certificate", err.Error())
		return
	}

	data.apply(&plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *domainCertificatesResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state DomainCertificatesModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var data certificateData
	endpoint := r.path(state.Organization.ValueString(), state.Server.ValueInt64(),
		state.Site.ValueInt64(), state.DomainRecord.ValueInt64())
	if err := r.client.Do(ctx, http.MethodGet, endpoint, nil, &data); err != nil {
		if forge.NotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Unable to read certificate", err.Error())
		return
	}

	data.apply(&state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update is never invoked with real changes: every configurable attribute
// forces replacement. It simply persists the plan.
func (r *domainCertificatesResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan DomainCertificatesModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *domainCertificatesResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state DomainCertificatesModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	endpoint := r.path(state.Organization.ValueString(), state.Server.ValueInt64(),
		state.Site.ValueInt64(), state.DomainRecord.ValueInt64())
	if err := r.client.Do(ctx, http.MethodDelete, endpoint, nil, nil); err != nil && !forge.NotFound(err) {
		resp.Diagnostics.AddError("Unable to delete certificate", err.Error())
	}
}

// ImportState supports `terraform import` using the
// "organization/server/site/domain_record" identifier form.
func (r *domainCertificatesResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
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
	domainRecord, err := providerutil.ParseInt64(parts[3])
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("Domain record must be an integer: %s.", err))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("organization"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("server"), server)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("site"), site)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("domain_record"), domainRecord)...)
}

// setString adds a string attribute to the request map when it is known.
func setString(m map[string]any, key string, v types.String) {
	if !v.IsNull() && !v.IsUnknown() {
		m[key] = v.ValueString()
	}
}

// certificateData is the JSON:API representation of a certificate.
type certificateData struct {
	ID         string `json:"id"`
	Attributes struct {
		Type               *string `json:"type"`
		KeyType            *string `json:"key_type"`
		PreferredChain     *string `json:"preferred_chain"`
		RequestStatus      *string `json:"request_status"`
		Status             *string `json:"status"`
		VerificationMethod *string `json:"verification_method"`
		CreatedAt          string  `json:"created_at"`
		UpdatedAt          string  `json:"updated_at"`
	} `json:"attributes"`
}

// apply copies the API response onto the model. The write-only input objects
// (clone, csr, existing, letsencrypt) are left untouched; "type" is echoed back
// by the API but is also the configured value, so it is set from the response
// when present.
func (d *certificateData) apply(m *DomainCertificatesModel) {
	m.Id = types.StringValue(d.ID)
	m.CreatedAt = types.StringValue(d.Attributes.CreatedAt)
	m.UpdatedAt = types.StringValue(d.Attributes.UpdatedAt)
	if d.Attributes.Type != nil {
		m.Type = types.StringValue(*d.Attributes.Type)
	}
	m.KeyType = optString(d.Attributes.KeyType)
	m.PreferredChain = optString(d.Attributes.PreferredChain)
	m.RequestStatus = optString(d.Attributes.RequestStatus)
	m.Status = optString(d.Attributes.Status)
	m.VerificationMethod = optString(d.Attributes.VerificationMethod)
}

func optString(s *string) types.String {
	if s == nil {
		return types.StringNull()
	}
	return types.StringValue(*s)
}
