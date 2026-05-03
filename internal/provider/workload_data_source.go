package provider

import (
	"context"
	"fmt"

	"github.com/HarchCorp/harchos-terraform-provider/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &WorkloadDataSource{}
	_ datasource.DataSourceWithConfigure = &WorkloadDataSource{}
)

// WorkloadDataSource defines the data source implementation.
type WorkloadDataSource struct {
	client *client.Client
}

// WorkloadDataSourceModel describes the data source data model.
type WorkloadDataSourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Image       types.String `tfsdk:"image"`
	Replicas    types.Int64  `tfsdk:"replicas"`
	Region      types.String `tfsdk:"region"`
	Sovereignty types.String `tfsdk:"sovereignty"`
	Env         types.Map    `tfsdk:"env"`
	Tags        types.Map    `tfsdk:"tags"`
	Status      types.String `tfsdk:"status"`
	CreatedAt   types.String `tfsdk:"created_at"`
	UpdatedAt   types.String `tfsdk:"updated_at"`
}

// NewWorkloadDataSource returns a new workload data source.
func NewWorkloadDataSource() datasource.DataSource {
	return &WorkloadDataSource{}
}

// Metadata returns the data source type name.
func (d *WorkloadDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_workload"
}

// Schema returns the data source schema.
func (d *WorkloadDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves information about an existing HarchOS workload. " +
			"Use this data source to reference workload properties in other resources " +
			"without managing the workload's lifecycle.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The unique identifier of the workload to look up.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the workload.",
				Computed:            true,
			},
			"image": schema.StringAttribute{
				MarkdownDescription: "The container image deployed by the workload.",
				Computed:            true,
			},
			"replicas": schema.Int64Attribute{
				MarkdownDescription: "The number of workload replicas.",
				Computed:            true,
			},
			"region": schema.StringAttribute{
				MarkdownDescription: "The region where the workload is deployed.",
				Computed:            true,
			},
			"sovereignty": schema.StringAttribute{
				MarkdownDescription: "The sovereignty level of the workload.",
				Computed:            true,
			},
			"env": schema.MapAttribute{
				MarkdownDescription: "Environment variables injected into the workload.",
				Computed:            true,
				ElementType:         types.StringType,
			},
			"tags": schema.MapAttribute{
				MarkdownDescription: "Key-value tags for resource organization.",
				Computed:            true,
				ElementType:         types.StringType,
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "The current status of the workload.",
				Computed:            true,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "Timestamp when the workload was created.",
				Computed:            true,
			},
			"updated_at": schema.StringAttribute{
				MarkdownDescription: "Timestamp when the workload was last updated.",
				Computed:            true,
			},
		},
	}
}

// Configure adds the provider-configured client to the data source.
func (d *WorkloadDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	providerData, ok := req.ProviderData.(*ProviderData)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *ProviderData, got: %T.", req.ProviderData),
		)
		return
	}

	d.client = providerData.Client
}

// Read reads the workload data from the API.
func (d *WorkloadDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config WorkloadDataSourceModel

	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	workloadID := config.ID.ValueString()

	result, err := d.client.GetWorkload(ctx, workloadID)
	if err != nil {
		if apiErr, ok := err.(*client.APIError); ok && apiErr.StatusCode == 404 {
			resp.Diagnostics.AddError(
				"Workload not found",
				fmt.Sprintf("No workload found with ID %s", workloadID),
			)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading workload",
			fmt.Sprintf("Could not read workload %s: %s", workloadID, err),
		)
		return
	}

	config.ID = types.StringValue(result.ID)
	config.Name = types.StringValue(result.Name)
	config.Image = types.StringValue(result.Image)
	config.Replicas = types.Int64Value(int64(result.Replicas))
	config.Region = types.StringValue(result.Region)
	config.Sovereignty = types.StringValue(result.Sovereignty)
	config.Env = frameworkFromStringMap(ctx, result.Env)
	config.Tags = frameworkFromStringMap(ctx, result.Tags)
	config.Status = types.StringValue(result.Status)
	config.CreatedAt = types.StringValue(result.CreatedAt)
	config.UpdatedAt = types.StringValue(result.UpdatedAt)

	diags = resp.State.Set(ctx, config)
	resp.Diagnostics.Append(diags...)

	tflog.Info(ctx, "read workload data source", map[string]interface{}{
		"id": result.ID, "name": result.Name,
	})
}
