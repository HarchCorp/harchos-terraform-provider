package provider

import (
	"context"
	"fmt"

	"github.com/HarchCorp/harchos-terraform-provider/internal/client"
	"github.com/HarchCorp/harchos-terraform-provider/internal/sovereignty"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &DatasetResource{}
	_ resource.ResourceWithConfigure   = &DatasetResource{}
	_ resource.ResourceWithImportState = &DatasetResource{}
	_ resource.ResourceWithModifyPlan  = &DatasetResource{}
)

// DatasetResource defines the resource implementation.
type DatasetResource struct {
	client      *client.Client
	region      string
	sovereignty string
}

// DatasetResourceModel describes the resource data model.
type DatasetResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Format      types.String `tfsdk:"format"`
	SizeBytes   types.Int64  `tfsdk:"size_bytes"`
	Region      types.String `tfsdk:"region"`
	Sovereignty types.String `tfsdk:"sovereignty"`
	StoragePath types.String `tfsdk:"storage_path"`
	Tags        types.Map    `tfsdk:"tags"`
	Status      types.String `tfsdk:"status"`
	CreatedAt   types.String `tfsdk:"created_at"`
	UpdatedAt   types.String `tfsdk:"updated_at"`
}

// NewDatasetResource returns a new dataset resource.
func NewDatasetResource() resource.Resource {
	return &DatasetResource{}
}

// Metadata returns the resource type name.
func (r *DatasetResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dataset"
}

// Schema returns the resource schema.
func (r *DatasetResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a HarchOS dataset. Datasets are sovereign data stores " +
			"that hold training and inference data with strict compliance boundaries.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The unique identifier of the dataset.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the dataset.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 128),
				},
			},
			"format": schema.StringAttribute{
				MarkdownDescription: "The data format (e.g., csv, parquet, jsonl, tfrecord).",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("csv", "parquet", "jsonl", "tfrecord", "webdataset", "arrow"),
				},
			},
			"size_bytes": schema.Int64Attribute{
				MarkdownDescription: "The size of the dataset in bytes. Defaults to 0 for new datasets.",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(0),
				Validators: []validator.Int64{
					int64validator.AtLeast(0),
				},
			},
			"region": schema.StringAttribute{
				MarkdownDescription: "The HarchOS region for the dataset. Defaults to provider region.",
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString(""),
			},
			"sovereignty": schema.StringAttribute{
				MarkdownDescription: "The sovereignty level for this dataset. Datasets with " +
					"strict sovereignty cannot leave the designated region. Cannot be downgraded.",
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString(""),
				Validators: []validator.String{
					sovereignty.SovereigntyLevelValidator(),
				},
			},
			"storage_path": schema.StringAttribute{
				MarkdownDescription: "The storage path where the dataset resides (computed after creation).",
				Computed: true,
			},
			"tags": schema.MapAttribute{
				MarkdownDescription: "Key-value tags for resource organization.",
				Optional:            true,
				Computed:            true,
				Default:             mapdefault.StaticValue(types.MapValueMust(types.StringType, map[string]attr.Value{})),
				ElementType:         types.StringType,
				Validators: []validator.Map{
					mapvalidator.SizeAtMost(32),
				},
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "The current status of the dataset.",
				Computed:            true,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "Timestamp when the dataset was created.",
				Computed:            true,
			},
			"updated_at": schema.StringAttribute{
				MarkdownDescription: "Timestamp when the dataset was last updated.",
				Computed:            true,
			},
		},
	}
}

// Configure adds the provider-configured client to the resource.
func (r *DatasetResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	providerData, ok := req.ProviderData.(*ProviderData)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *ProviderData, got: %T.", req.ProviderData),
		)
		return
	}

	r.client = providerData.Client
	r.region = providerData.Region
	r.sovereignty = providerData.Sovereignty
}

// ModifyPlan enables drift detection and sovereignty enforcement.
func (r *DatasetResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	var plan, state DatasetResourceModel

	if req.Plan.Raw.IsNull() {
		return
	}

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.Region.IsNull() || plan.Region.ValueString() == "" {
		resp.Plan.SetAttribute(ctx, path.Root("region"), r.region)
	}

	if plan.Sovereignty.IsNull() || plan.Sovereignty.ValueString() == "" {
		resp.Plan.SetAttribute(ctx, path.Root("sovereignty"), r.sovereignty)
	}

	if !req.State.Raw.IsNull() {
		diags = req.State.Get(ctx, &state)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		if err := sovereignty.ValidateSovereigntyTransition(state.Sovereignty, plan.Sovereignty); err != nil {
			resp.Diagnostics.AddError(
				"Sovereignty downgrade not allowed",
				fmt.Sprintf("Cannot change dataset sovereignty: %s", err),
			)
		}
	}
}

// Create creates the dataset resource.
func (r *DatasetResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan DatasetResourceModel

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	region := r.region
	if !plan.Region.IsNull() && plan.Region.ValueString() != "" {
		region = plan.Region.ValueString()
	}

	sovereigntyLevel, err := sovereignty.EffectiveSovereignty(r.sovereignty, plan.Sovereignty.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid sovereignty configuration", err.Error())
		return
	}

	datasetReq := &client.Dataset{
		Name:        plan.Name.ValueString(),
		Format:      plan.Format.ValueString(),
		SizeBytes:   plan.SizeBytes.ValueInt64(),
		Region:      region,
		Sovereignty: sovereigntyLevel,
		Tags:        stringMapFromFramework(ctx, plan.Tags, &resp.Diagnostics),
	}

	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.CreateDataset(ctx, datasetReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating dataset",
			fmt.Sprintf("Could not create dataset: %s", err))
		return
	}

	plan.ID = types.StringValue(result.ID)
	plan.Region = types.StringValue(region)
	plan.Sovereignty = types.StringValue(sovereigntyLevel)
	plan.StoragePath = types.StringValue(result.StoragePath)
	plan.SizeBytes = types.Int64Value(result.SizeBytes)
	plan.Status = types.StringValue(result.Status)
	plan.CreatedAt = types.StringValue(result.CreatedAt)
	plan.UpdatedAt = types.StringValue(result.UpdatedAt)
	plan.Tags = frameworkFromStringMap(ctx, result.Tags)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)

	tflog.Info(ctx, "created dataset", map[string]interface{}{
		"id": result.ID, "name": result.Name, "format": result.Format,
	})
}

// Read reads the dataset state from the API.
func (r *DatasetResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state DatasetResourceModel

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.GetDataset(ctx, state.ID.ValueString())
	if err != nil {
		if apiErr, ok := err.(*client.APIError); ok && apiErr.StatusCode == 404 {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading dataset",
			fmt.Sprintf("Could not read dataset %s: %s", state.ID.ValueString(), err))
		return
	}

	state.ID = types.StringValue(result.ID)
	state.Name = types.StringValue(result.Name)
	state.Format = types.StringValue(result.Format)
	state.SizeBytes = types.Int64Value(result.SizeBytes)
	state.Region = types.StringValue(result.Region)
	state.Sovereignty = types.StringValue(result.Sovereignty)
	state.StoragePath = types.StringValue(result.StoragePath)
	state.Status = types.StringValue(result.Status)
	state.CreatedAt = types.StringValue(result.CreatedAt)
	state.UpdatedAt = types.StringValue(result.UpdatedAt)
	state.Tags = frameworkFromStringMap(ctx, result.Tags)

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
}

// Update updates the dataset resource.
func (r *DatasetResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state DatasetResourceModel

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	region := r.region
	if !plan.Region.IsNull() && plan.Region.ValueString() != "" {
		region = plan.Region.ValueString()
	}

	sovereigntyLevel, err := sovereignty.EffectiveSovereignty(r.sovereignty, plan.Sovereignty.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid sovereignty configuration", err.Error())
		return
	}

	datasetReq := &client.Dataset{
		Name:        plan.Name.ValueString(),
		Format:      plan.Format.ValueString(),
		SizeBytes:   plan.SizeBytes.ValueInt64(),
		Region:      region,
		Sovereignty: sovereigntyLevel,
		Tags:        stringMapFromFramework(ctx, plan.Tags, &resp.Diagnostics),
	}

	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.UpdateDataset(ctx, state.ID.ValueString(), datasetReq)
	if err != nil {
		resp.Diagnostics.AddError("Error updating dataset",
			fmt.Sprintf("Could not update dataset %s: %s", state.ID.ValueString(), err))
		return
	}

	plan.ID = types.StringValue(result.ID)
	plan.Region = types.StringValue(region)
	plan.Sovereignty = types.StringValue(sovereigntyLevel)
	plan.StoragePath = types.StringValue(result.StoragePath)
	plan.SizeBytes = types.Int64Value(result.SizeBytes)
	plan.Status = types.StringValue(result.Status)
	plan.CreatedAt = types.StringValue(result.CreatedAt)
	plan.UpdatedAt = types.StringValue(result.UpdatedAt)
	plan.Tags = frameworkFromStringMap(ctx, result.Tags)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

// Delete deletes the dataset resource.
func (r *DatasetResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state DatasetResourceModel

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteDataset(ctx, state.ID.ValueString())
	if err != nil {
		if apiErr, ok := err.(*client.APIError); ok && apiErr.StatusCode == 404 {
			return
		}
		resp.Diagnostics.AddError("Error deleting dataset",
			fmt.Sprintf("Could not delete dataset %s: %s", state.ID.ValueString(), err))
		return
	}

	tflog.Info(ctx, "deleted dataset", map[string]interface{}{"id": state.ID.ValueString()})
}

// ImportState imports an existing dataset into Terraform.
func (r *DatasetResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
