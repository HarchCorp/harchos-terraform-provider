package provider

import (
	"context"
	"fmt"

	"github.com/HarchCorp/harchos-terraform-provider/internal/client"
	"github.com/HarchCorp/harchos-terraform-provider/internal/sovereignty"
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
	_ resource.Resource                = &ModelResource{}
	_ resource.ResourceWithConfigure   = &ModelResource{}
	_ resource.ResourceWithImportState = &ModelResource{}
	_ resource.ResourceWithModifyPlan  = &ModelResource{}
)

// ModelResource defines the resource implementation.
type ModelResource struct {
	client      *client.Client
	region      string
	sovereignty string
}

// ModelResourceModel describes the resource data model.
type ModelResourceModel struct {
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

// NewModelResource returns a new model resource.
func NewModelResource() resource.Resource {
	return &ModelResource{}
}

// Metadata returns the resource type name.
func (r *ModelResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_model"
}

// Schema returns the resource schema.
func (r *ModelResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a HarchOS model. Models represent ML/AI models stored " +
			"in the HarchOS model registry, with sovereignty controls governing their " +
			"deployment and replication boundaries.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The unique identifier of the model.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the model. Must be unique within the region.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 128),
				},
			},
			"framework": schema.StringAttribute{
				MarkdownDescription: "The ML framework for this model (e.g., pytorch, tensorflow, onnx).",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("pytorch", "tensorflow", "onnx", "jax", "huggingface"),
				},
			},
			"version": schema.StringAttribute{
				MarkdownDescription: "The semantic version of the model (e.g., v1.0.0).",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"source_uri": schema.StringAttribute{
				MarkdownDescription: "The URI where the model artifacts are stored (e.g., s3://, gs://, harchos://).",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"sovereignty": schema.StringAttribute{
				MarkdownDescription: "The sovereignty level for this model. Must be one of: " +
					"strict, regional, global. Cannot be downgraded after creation.",
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString(""),
				Validators: []validator.String{
					sovereignty.SovereigntyLevelValidator(),
				},
			},
			"region": schema.StringAttribute{
				MarkdownDescription: "The HarchOS region for the model. Defaults to the provider region.",
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString(""),
			},
			"parameters": schema.MapAttribute{
				MarkdownDescription: "Model-specific parameters and hyperparameters.",
				Optional:            true,
				Computed:            true,
				Default:             mapdefault.StaticValue(types.MapValueMust(types.StringType, map[string]attr.Value{})),
				ElementType:         types.StringType,
				Validators: []validator.Map{
					mapvalidator.SizeAtMost(32),
				},
			},
			"tags": schema.MapAttribute{
				MarkdownDescription: "Key-value tags for resource organization.",
				Optional:            true,
				Computed:            true,
				Default:             mapdefault.StaticValue(types.MapValueMust(types.StringType, map[string]attr.Value{})),
				ElementType:         types.StringType,
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "The current status of the model.",
				Computed:            true,
			},
			"size_bytes": schema.Int64Attribute{
				MarkdownDescription: "The size of the model artifacts in bytes.",
				Computed:            true,
				Default:             int64default.StaticInt64(0),
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

// Configure adds the provider-configured client to the resource.
func (r *ModelResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// ModifyPlan enables drift detection and sovereignty enforcement at plan time.
func (r *ModelResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	var plan, state ModelResourceModel

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
				fmt.Sprintf("Cannot change model sovereignty: %s", err),
			)
		}
	}
}

// Create creates the model resource.
func (r *ModelResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ModelResourceModel

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

	modelReq := &client.Model{
		Name:        plan.Name.ValueString(),
		Framework:   plan.Framework.ValueString(),
		Version:     plan.Version.ValueString(),
		SourceURI:   plan.SourceURI.ValueString(),
		Sovereignty: sovereigntyLevel,
		Region:      region,
		Parameters:  stringMapFromFramework(ctx, plan.Parameters, &resp.Diagnostics),
		Tags:        stringMapFromFramework(ctx, plan.Tags, &resp.Diagnostics),
	}

	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.CreateModel(ctx, modelReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating model", fmt.Sprintf("Could not create model: %s", err))
		return
	}

	plan.ID = types.StringValue(result.ID)
	plan.Region = types.StringValue(region)
	plan.Sovereignty = types.StringValue(sovereigntyLevel)
	plan.Status = types.StringValue(result.Status)
	plan.SizeBytes = types.Int64Value(result.SizeBytes)
	plan.CreatedAt = types.StringValue(result.CreatedAt)
	plan.UpdatedAt = types.StringValue(result.UpdatedAt)
	plan.Parameters = frameworkFromStringMap(ctx, result.Parameters)
	plan.Tags = frameworkFromStringMap(ctx, result.Tags)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)

	tflog.Info(ctx, "created model", map[string]interface{}{
		"id": result.ID, "name": result.Name, "framework": result.Framework,
	})
}

// Read reads the model state from the API.
func (r *ModelResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ModelResourceModel

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.GetModel(ctx, state.ID.ValueString())
	if err != nil {
		if apiErr, ok := err.(*client.APIError); ok && apiErr.StatusCode == 404 {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading model",
			fmt.Sprintf("Could not read model %s: %s", state.ID.ValueString(), err))
		return
	}

	state.ID = types.StringValue(result.ID)
	state.Name = types.StringValue(result.Name)
	state.Framework = types.StringValue(result.Framework)
	state.Version = types.StringValue(result.Version)
	state.SourceURI = types.StringValue(result.SourceURI)
	state.Region = types.StringValue(result.Region)
	state.Sovereignty = types.StringValue(result.Sovereignty)
	state.Status = types.StringValue(result.Status)
	state.SizeBytes = types.Int64Value(result.SizeBytes)
	state.CreatedAt = types.StringValue(result.CreatedAt)
	state.UpdatedAt = types.StringValue(result.UpdatedAt)
	state.Parameters = frameworkFromStringMap(ctx, result.Parameters)
	state.Tags = frameworkFromStringMap(ctx, result.Tags)

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
}

// Update updates the model resource.
func (r *ModelResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state ModelResourceModel

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

	modelReq := &client.Model{
		Name:        plan.Name.ValueString(),
		Framework:   plan.Framework.ValueString(),
		Version:     plan.Version.ValueString(),
		SourceURI:   plan.SourceURI.ValueString(),
		Sovereignty: sovereigntyLevel,
		Region:      region,
		Parameters:  stringMapFromFramework(ctx, plan.Parameters, &resp.Diagnostics),
		Tags:        stringMapFromFramework(ctx, plan.Tags, &resp.Diagnostics),
	}

	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.UpdateModel(ctx, state.ID.ValueString(), modelReq)
	if err != nil {
		resp.Diagnostics.AddError("Error updating model",
			fmt.Sprintf("Could not update model %s: %s", state.ID.ValueString(), err))
		return
	}

	plan.ID = types.StringValue(result.ID)
	plan.Region = types.StringValue(region)
	plan.Sovereignty = types.StringValue(sovereigntyLevel)
	plan.Status = types.StringValue(result.Status)
	plan.SizeBytes = types.Int64Value(result.SizeBytes)
	plan.CreatedAt = types.StringValue(result.CreatedAt)
	plan.UpdatedAt = types.StringValue(result.UpdatedAt)
	plan.Parameters = frameworkFromStringMap(ctx, result.Parameters)
	plan.Tags = frameworkFromStringMap(ctx, result.Tags)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

// Delete deletes the model resource.
func (r *ModelResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ModelResourceModel

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteModel(ctx, state.ID.ValueString())
	if err != nil {
		if apiErr, ok := err.(*client.APIError); ok && apiErr.StatusCode == 404 {
			return
		}
		resp.Diagnostics.AddError("Error deleting model",
			fmt.Sprintf("Could not delete model %s: %s", state.ID.ValueString(), err))
		return
	}

	tflog.Info(ctx, "deleted model", map[string]interface{}{"id": state.ID.ValueString()})
}

// ImportState imports an existing model into Terraform.
func (r *ModelResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
