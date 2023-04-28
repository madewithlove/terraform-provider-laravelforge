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
var _ datasource.DataSource = &DaemonDataSource{}

func NewDaemonDataSource() datasource.DataSource {
	return &DaemonDataSource{}
}

// DaemonDataSource defines the data source implementation.
type DaemonDataSource struct {
	client *forge.APIClient
}

// DaemonDataSourceModel describes the data source data model.
type DaemonDataSourceModel struct {
	Id           types.Int64  `tfsdk:"id"`
	Command      types.String `tfsdk:"command"`
	User         types.String `tfsdk:"user"`
	Directory    types.String `tfsdk:"directory"`
	Processes    types.Int64  `tfsdk:"processes"`
	Startsecs    types.Int64  `tfsdk:"startsecs"`
	Stopwaitsecs types.Int64  `tfsdk:"stopwaitsecs"`
	Stopsignal   types.String `tfsdk:"stopsignal"`
	Status       types.String `tfsdk:"status"`
	CreatedAt    types.String `tfsdk:"created_at"`
	ServerId     types.Int64  `tfsdk:"server_id"`
}

func (d *DaemonDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_daemon"
}

func (d *DaemonDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Daemon data source",

		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				MarkdownDescription: "",
				Required:            true,
			},
			"command": schema.StringAttribute{
				MarkdownDescription: "",
				Computed:            true,
			},
			"user": schema.StringAttribute{
				MarkdownDescription: "",
				Computed:            true,
			},
			"directory": schema.StringAttribute{
				MarkdownDescription: "",
				Computed:            true,
			},
			"processes": schema.Int64Attribute{
				MarkdownDescription: "",
				Computed:            true,
			},
			"startsecs": schema.Int64Attribute{
				MarkdownDescription: "",
				Computed:            true,
			},
			"stopwaitsecs": schema.Int64Attribute{
				MarkdownDescription: "",
				Computed:            true,
			},
			"stopsignal": schema.StringAttribute{
				MarkdownDescription: "",
				Computed:            true,
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "",
				Computed:            true,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "",
				Computed:            true,
			},
			"server_id": schema.Int64Attribute{
				MarkdownDescription: "",
				Required:            true,
			},
		},
	}
}

func (d *DaemonDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *DaemonDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data DaemonDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	daemon, _, err := d.client.DefaultApi.GetDaemon(ctx, int32(data.ServerId.ValueInt64()), int32(data.Id.ValueInt64()))

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read daemon, got error: %s", err))
		return
	}

	data.Command = types.StringValue(daemon.Command)
	data.User = types.StringValue(daemon.User)
	data.Directory = types.StringValue(daemon.Directory)
	data.Processes = types.Int64Value(int64(daemon.Processes))
	data.Startsecs = types.Int64Value(int64(daemon.Startsecs))
	data.Stopwaitsecs = types.Int64Value(int64(daemon.Stopwaitsecs))
	data.Stopsignal = types.StringValue(daemon.Stopsignal)
	data.Status = types.StringValue(daemon.Status)
	data.CreatedAt = types.StringValue(daemon.CreatedAt)

	// Write logs using the tflog package
	// Documentation: https://terraform.io/plugin/log
	tflog.Trace(ctx, "read a data source")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
