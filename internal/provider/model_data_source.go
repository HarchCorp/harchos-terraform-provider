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
	_ datasource.DataSource              = &ModelDataSource{}
	_ datasource.DataSourceWithConfigure = &ModelDataSource{}
)

// ModelDataSource defines the data source implementation.
type ModelDataSource struct {
	client *client.Client
}

// ModelDataSourceModel describes the data source data model.
type ModelDataSourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Framework   types.String `tfsdk:"framework"`
	Version     types.String `tfsdk:"version"`
	SourceURI   types.String `tfsdk:"source_uri"`
	Sovereignty types.String `tfsdk:"sovereignty"`
	Region      types.String `tfsdk:"region"`
	Parameters  types.Map    `tfsdk:"parameters"`
	Tags        types.Map    `tfsdk:"tags"`
	Status      types.String `tfsdk:"status"`
	SizeBytes   types.Int64  `tfsdk:"size_bytes"`
	CreatedAt   types.String `tfsdk:"created_at"`
	UpdatedAt   types.String `tfsdk:"updated_at"`
}

// NewModelDataSource returns a new model data source.
func NewModelDataSource() datasource.DataSource {
	return &ModelDataSource{}
}

// Metadata returns the data source type name.
func (d *ModelDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_model"
}

// Schema returns the data source schema.
func (d *ModelDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves information about an existing HarchOS model. " +
			"Use this data source to reference model properties (e.g., for inference endpoints) " +
			"without managing the model's lifecycle.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The unique identifier of the model to look up.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the model.",
				Computed:            true,
			},
			"framework": schema.StringAttribute{
				MarkdownDescription: "The ML framework of the model.",
				Computed:            true,
			},
			"version": schema.StringAttribute{
				MarkdownDescription: "The version of the model.",
				Computed:            true,
			},
			"source_uri": schema.StringAttribute{
				MarkdownDescription: "The URI where the model artifacts are stored.",
				Computed:            true,
			},
			"sovereignty": schema.StringAttribute{
				MarkdownDescription: "The sovereignty level of the model.",
				Computed:            true,
			},
			"region": schema.StringAttribute{
				MarkdownDescription: "The region where the model is stored.",
				Computed:            true,
			},
			"parameters": schema.MapAttribute{
				MarkdownDescription: "Model-specific parameters and hyperparameters.",
				Computed:            true,
				ElementType:         types.StringType,
			},
			"tags": schema.MapAttribute{
				MarkdownDescription: "Key-value tags for resource organization.",
				Computed:            true,
				ElementType:         types.StringType,
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "The current status of the model.",
				Computed:            true,
			},
			"size_bytes": schema.Int64Attribute{
				MarkdownDescription: "The size of the model artifacts in bytes.",
				Computed:            true,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "Timestamp when the model was created.",
				Computed:            true,
			},
			"updated_at": schema.StringAttribute{
				MarkdownDescription: "Timestamp when the model was last updated.",
				Computed:            true,
			},
		},
	}
}

// Configure adds the provider-configured client to the data source.
func (d *ModelDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

// Read reads the model data from the API.
func (d *ModelDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config ModelDataSourceModel

	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	modelID := config.ID.ValueString()

	result, err := d.client.GetModel(ctx, modelID)
	if err != nil {
		if apiErr, ok := err.(*client.APIError); ok && apiErr.StatusCode == 404 {
			resp.Diagnostics.AddError(
				"Model not found",
				fmt.Sprintf("No model found with ID %s", modelID),
			)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading model",
			fmt.Sprintf("Could not read model %s: %s", modelID, err),
		)
		return
	}

	config.ID = types.StringValue(result.ID)
	config.Name = types.StringValue(result.Name)
	config.Framework = types.StringValue(result.Framework)
	config.Version = types.StringValue(result.Version)
	config.SourceURI = types.StringValue(result.SourceURI)
	config.Sovereignty = types.StringValue(result.Sovereignty)
	config.Region = types.StringValue(result.Region)
	config.Parameters = frameworkFromStringMap(ctx, result.Parameters)
	config.Tags = frameworkFromStringMap(ctx, result.Tags)
	config.Status = types.StringValue(result.Status)
	config.SizeBytes = types.Int64Value(result.SizeBytes)
	config.CreatedAt = types.StringValue(result.CreatedAt)
	config.UpdatedAt = types.StringValue(result.UpdatedAt)

	diags = resp.State.Set(ctx, config)
	resp.Diagnostics.Append(diags...)

	tflog.Info(ctx, "read model data source", map[string]interface{}{
		"id": result.ID, "name": result.Name, "framework": result.Framework,
	})
}
