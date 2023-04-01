package provider

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/madewithlove/forge-go-sdk"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &ServerDataSource{}

func NewServerDataSource() datasource.DataSource {
	return &ServerDataSource{}
}

// ServerDataSource defines the data source implementation.
type ServerDataSource struct {
	client *forge.APIClient
}

// ServerDataSourceModel describes the data source data model.
type ServerDataSourceModel struct {
	Id               types.Int64    `tfsdk:"id"`
	CredentialId     types.Int64    `tfsdk:"credential_id"`
	Name             types.String   `tfsdk:"name"`
	Size             types.String   `tfsdk:"size"`
	Region           types.String   `tfsdk:"region"`
	PhpVersion       types.String   `tfsdk:"php_version"`
	PhpCliVersion    types.String   `tfsdk:"php_cli_version"`
	OpcacheStatus    types.String   `tfsdk:"opcache_status"`
	DatabaseType     types.String   `tfsdk:"database_type"`
	IpAddress        types.String   `tfsdk:"ip_address"`
	PrivateIpAddress types.String   `tfsdk:"private_ip_address"`
	BlackfireStatus  types.String   `tfsdk:"blackfire_status"`
	PapertrailStatus types.String   `tfsdk:"papertrail_status"`
	Revoked          types.String   `tfsdk:"revoked"`
	CreatedAt        types.String   `tfsdk:"created_at"`
	IsReady          types.String   `tfsdk:"is_ready"`
	Network          types.ListType `tfsdk:"network"`
	Tags             types.ListType `tfsdk:"tags"`
}

func (d *ServerDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_server"
}

func (d *ServerDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Server data source",

		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				MarkdownDescription: "",
				Computed:            true,
			},
			"credential_id": schema.Int64Attribute{
				MarkdownDescription: "",
				Optional:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "",
				Optional:            true,
			},
			"size": schema.StringAttribute{
				MarkdownDescription: "",
				Optional:            true,
			},
			"region": schema.StringAttribute{
				MarkdownDescription: "",
				Optional:            true,
			},
			"php_version": schema.StringAttribute{
				MarkdownDescription: "",
				Optional:            true,
			},
			"php_cli_version": schema.StringAttribute{
				MarkdownDescription: "",
				Optional:            true,
			},
			"opcache_status": schema.StringAttribute{
				MarkdownDescription: "",
				Optional:            true,
			},
			"database_type": schema.StringAttribute{
				MarkdownDescription: "",
				Optional:            true,
			},
			"ip_address": schema.StringAttribute{
				MarkdownDescription: "",
				Optional:            true,
			},
			"private_ip_address": schema.StringAttribute{
				MarkdownDescription: "",
				Optional:            true,
			},
			"blackfire_status": schema.StringAttribute{
				MarkdownDescription: "",
				Optional:            true,
			},
			"papertrail_status": schema.StringAttribute{
				MarkdownDescription: "",
				Optional:            true,
			},
			"revoked": schema.StringAttribute{
				MarkdownDescription: "",
				Optional:            true,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "",
				Optional:            true,
			},
			"is_ready": schema.StringAttribute{
				MarkdownDescription: "",
				Optional:            true,
			},
			"network": schema.ListAttribute{
				ElementType:         types.Int64Type,
				MarkdownDescription: "",
				Optional:            true,
			},
			"tags": schema.ListAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "",
				Optional:            true,
			},
		},
	}
}

func (d *ServerDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*forge.APIClient)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *forge.APIClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	d.client = client
}

func (d *ServerDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ServerDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	server, _, err := d.client.DefaultApi.GetServer(ctx, int32(data.Id.ValueInt64()))

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read server, got error: %s", err))
		return
	}

	data.CredentialId = types.Int64Value(int64(server.CredentialId))
	data.Name = types.StringValue(string(server.Name))
	data.Size = types.StringValue(string(server.Size))
	data.Region = types.StringValue(string(server.Region))
	data.PhpVersion = types.StringValue(string(server.PhpVersion))
	data.PhpCliVersion = types.StringValue(string(server.PhpCliVersion))
	data.OpcacheStatus = types.StringValue(string(server.OpcacheStatus))
	data.DatabaseType = types.StringValue(string(server.DatabaseType))
	data.IpAddress = types.StringValue(string(server.IpAddress))
	data.PrivateIpAddress = types.StringValue(string(server.PrivateIpAddress))
	data.BlackfireStatus = types.StringValue(string(server.BlackfireStatus))
	data.PapertrailStatus = types.StringValue(string(server.PapertrailStatus))
	data.Revoked = types.StringValue(string(server.Revoked))
	data.CreatedAt = types.StringValue(string(server.CreatedAt))
	data.IsReady = types.StringValue(string(server.IsReady))
	//data.Network = types.ListValue(types.Int64Type, server.Network)
	//data.Tags = types.ListValue(types.StringType, server.Tags)

	// Write logs using the tflog package
	// Documentation: https://terraform.io/plugin/log
	tflog.Trace(ctx, "read a data source")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
