package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

type LaravelforgeProvider struct {
	version string
}

// New returns a new provider instance.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &LaravelforgeProvider{version: version}
	}
}

func (p *LaravelforgeProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "laravelforge"
	resp.Version = p.version
}

func (p *LaravelforgeProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{}
}

func (p *LaravelforgeProvider) Configure(_ context.Context, _ provider.ConfigureRequest, _ *provider.ConfigureResponse) {
	// No-op: add client initialization here when available.
}

func (p *LaravelforgeProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{}
}

func (p *LaravelforgeProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}
