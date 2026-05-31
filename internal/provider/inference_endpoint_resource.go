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
        _ resource.Resource                = &InferenceEndpointResource{}
        _ resource.ResourceWithConfigure   = &InferenceEndpointResource{}
        _ resource.ResourceWithImportState = &InferenceEndpointResource{}
        _ resource.ResourceWithModifyPlan  = &InferenceEndpointResource{}
)

// InferenceEndpointResource defines the resource implementation.
type InferenceEndpointResource struct {
        client      *client.Client
        region      string
        sovereignty string
}

// InferenceEndpointResourceModel describes the resource data model.
type InferenceEndpointResourceModel struct {
        ID           types.String `tfsdk:"id"`
        Name         types.String `tfsdk:"name"`
        ModelID      types.String `tfsdk:"model_id"`
        InstanceType types.String `tfsdk:"instance_type"`
        MinReplicas  types.Int64  `tfsdk:"min_replicas"`
        MaxReplicas  types.Int64  `tfsdk:"max_replicas"`
        Region       types.String `tfsdk:"region"`
        Sovereignty  types.String `tfsdk:"sovereignty"`
        EndpointURL  types.String `tfsdk:"endpoint_url"`
        Tags         types.Map    `tfsdk:"tags"`
        Status       types.String `tfsdk:"status"`
        CreatedAt    types.String `tfsdk:"created_at"`
        UpdatedAt    types.String `tfsdk:"updated_at"`
}

// NewInferenceEndpointResource returns a new inference endpoint resource.
func NewInferenceEndpointResource() resource.Resource {
        return &InferenceEndpointResource{}
}

// Metadata returns the resource type name.
func (r *InferenceEndpointResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
        resp.TypeName = req.ProviderTypeName + "_inference_endpoint"
}

// Schema returns the resource schema.
func (r *InferenceEndpointResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
        resp.Schema = schema.Schema{
                MarkdownDescription: "Manages a HarchOS inference endpoint. Inference endpoints expose " +
                        "a hosted model for real-time predictions with auto-scaling capabilities. " +
                        "The endpoint's sovereignty must be at least as restrictive as the model it serves.",

                Attributes: map[string]schema.Attribute{
                        "id": schema.StringAttribute{
                                MarkdownDescription: "The unique identifier of the inference endpoint.",
                                Computed:            true,
                                PlanModifiers: []planmodifier.String{
                                        stringplanmodifier.UseStateForUnknown(),
                                },
                        },
                        "name": schema.StringAttribute{
                                MarkdownDescription: "The name of the inference endpoint.",
                                Required:            true,
                                Validators: []validator.String{
                                        stringvalidator.LengthBetween(1, 128),
                                },
                        },
                        "model_id": schema.StringAttribute{
                                MarkdownDescription: "The ID of the model to serve. The endpoint's sovereignty " +
                                        "must be at least as restrictive as the model's sovereignty. " +
                                        "Changing this field requires resource replacement.",
                                Required: true,
                                Validators: []validator.String{
                                        stringvalidator.LengthAtLeast(1),
                                },
                                PlanModifiers: []planmodifier.String{
                                        stringplanmodifier.RequiresReplace(),
                                },
                        },
                        "instance_type": schema.StringAttribute{
                                MarkdownDescription: "The compute instance type for the endpoint " +
                                        "(e.g., gpu.small, gpu.large, cpu.medium).",
                                Required: true,
                                Validators: []validator.String{
                                        stringvalidator.OneOf(
                                                "cpu.small", "cpu.medium", "cpu.large",
                                                "gpu.small", "gpu.medium", "gpu.large", "gpu.xlarge",
                                        ),
                                },
                        },
                        "min_replicas": schema.Int64Attribute{
                                MarkdownDescription: "Minimum number of replicas for auto-scaling. Defaults to 1.",
                                Optional:            true,
                                Computed:            true,
                                Default:             int64default.StaticInt64(1),
                                Validators: []validator.Int64{
                                        int64validator.AtLeast(0),
                                        int64validator.AtMost(50),
                                },
                        },
                        "max_replicas": schema.Int64Attribute{
                                MarkdownDescription: "Maximum number of replicas for auto-scaling. Defaults to 5.",
                                Optional:            true,
                                Computed:            true,
                                Default:             int64default.StaticInt64(5),
                                Validators: []validator.Int64{
                                        int64validator.AtLeast(1),
                                        int64validator.AtMost(100),
                                },
                        },
                        "region": schema.StringAttribute{
                                MarkdownDescription: "The HarchOS region for the endpoint. Defaults to provider region.",
                                Optional: true,
                                Computed: true,
                                Default:  stringdefault.StaticString(""),
                        },
                        "sovereignty": schema.StringAttribute{
                                MarkdownDescription: "The sovereignty level for this endpoint. Must be at least " +
                                        "as restrictive as the model it serves. Cannot be downgraded.",
                                Optional: true,
                                Computed: true,
                                Default:  stringdefault.StaticString(""),
                                Validators: []validator.String{
                                        sovereignty.SovereigntyLevelValidator(),
                                },
                        },
                        "endpoint_url": schema.StringAttribute{
                                MarkdownDescription: "The URL where the inference endpoint is accessible.",
                                Computed:            true,
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
                                MarkdownDescription: "The current status of the inference endpoint.",
                                Computed:            true,
                        },
                        "created_at": schema.StringAttribute{
                                MarkdownDescription: "Timestamp when the endpoint was created.",
                                Computed:            true,
                        },
                        "updated_at": schema.StringAttribute{
                                MarkdownDescription: "Timestamp when the endpoint was last updated.",
                                Computed:            true,
                        },
                },
        }
}

// Configure adds the provider-configured client to the resource.
func (r *InferenceEndpointResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *InferenceEndpointResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
        var plan, state InferenceEndpointResourceModel

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

        // Validate min_replicas <= max_replicas
        if !plan.MinReplicas.IsNull() && !plan.MaxReplicas.IsNull() &&
                plan.MinReplicas.ValueInt64() > plan.MaxReplicas.ValueInt64() {
                resp.Diagnostics.AddError(
                        "Invalid replica configuration",
                        fmt.Sprintf("min_replicas (%d) cannot be greater than max_replicas (%d)",
                                plan.MinReplicas.ValueInt64(), plan.MaxReplicas.ValueInt64()),
                )
        }

        // Sovereignty downgrade prevention
        if !req.State.Raw.IsNull() {
                diags = req.State.Get(ctx, &state)
                resp.Diagnostics.Append(diags...)
                if resp.Diagnostics.HasError() {
                        return
                }

                if err := sovereignty.ValidateSovereigntyTransition(state.Sovereignty, plan.Sovereignty); err != nil {
                        resp.Diagnostics.AddError(
                                "Sovereignty downgrade not allowed",
                                fmt.Sprintf("Cannot change inference endpoint sovereignty: %s", err),
                        )
                }
        }
}

// Create creates the inference endpoint resource.
func (r *InferenceEndpointResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
        var plan InferenceEndpointResourceModel

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

        endpointReq := &client.InferenceEndpoint{
                Name:         plan.Name.ValueString(),
                ModelID:      plan.ModelID.ValueString(),
                InstanceType: plan.InstanceType.ValueString(),
                MinReplicas:  int(plan.MinReplicas.ValueInt64()),
                MaxReplicas:  int(plan.MaxReplicas.ValueInt64()),
                Region:       region,
                Sovereignty:  sovereigntyLevel,
                Tags:         stringMapFromFramework(ctx, plan.Tags, &resp.Diagnostics),
        }

        if resp.Diagnostics.HasError() {
                return
        }

        result, err := r.client.CreateInferenceEndpoint(ctx, endpointReq)
        if err != nil {
                resp.Diagnostics.AddError("Error creating inference endpoint",
                        fmt.Sprintf("Could not create inference endpoint: %s", err))
                return
        }

        plan.ID = types.StringValue(result.ID)
        plan.Region = types.StringValue(region)
        plan.Sovereignty = types.StringValue(sovereigntyLevel)
        plan.EndpointURL = types.StringValue(result.EndpointURL)
        plan.Status = types.StringValue(result.Status)
        plan.CreatedAt = types.StringValue(result.CreatedAt)
        plan.UpdatedAt = types.StringValue(result.UpdatedAt)
        plan.Tags = frameworkFromStringMap(ctx, result.Tags)

        diags = resp.State.Set(ctx, plan)
        resp.Diagnostics.Append(diags...)

        tflog.Info(ctx, "created inference endpoint", map[string]interface{}{
                "id": result.ID, "name": result.Name, "model_id": result.ModelID,
        })
}

// Read reads the inference endpoint state from the API.
func (r *InferenceEndpointResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
        var state InferenceEndpointResourceModel

        diags := req.State.Get(ctx, &state)
        resp.Diagnostics.Append(diags...)
        if resp.Diagnostics.HasError() {
                return
        }

        result, err := r.client.GetInferenceEndpoint(ctx, state.ID.ValueString())
        if err != nil {
                if apiErr, ok := err.(*client.APIError); ok && apiErr.StatusCode == 404 {
                        resp.State.RemoveResource(ctx)
                        return
                }
                resp.Diagnostics.AddError("Error reading inference endpoint",
                        fmt.Sprintf("Could not read inference endpoint %s: %s", state.ID.ValueString(), err))
                return
        }

        state.ID = types.StringValue(result.ID)
        state.Name = types.StringValue(result.Name)
        state.ModelID = types.StringValue(result.ModelID)
        state.InstanceType = types.StringValue(result.InstanceType)
        state.MinReplicas = types.Int64Value(int64(result.MinReplicas))
        state.MaxReplicas = types.Int64Value(int64(result.MaxReplicas))
        state.Region = types.StringValue(result.Region)
        state.Sovereignty = types.StringValue(result.Sovereignty)
        state.EndpointURL = types.StringValue(result.EndpointURL)
        state.Status = types.StringValue(result.Status)
        state.CreatedAt = types.StringValue(result.CreatedAt)
        state.UpdatedAt = types.StringValue(result.UpdatedAt)
        state.Tags = frameworkFromStringMap(ctx, result.Tags)

        diags = resp.State.Set(ctx, state)
        resp.Diagnostics.Append(diags...)
}

// Update updates the inference endpoint resource.
func (r *InferenceEndpointResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
        var plan, state InferenceEndpointResourceModel

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

        endpointReq := &client.InferenceEndpoint{
                Name:         plan.Name.ValueString(),
                ModelID:      plan.ModelID.ValueString(),
                InstanceType: plan.InstanceType.ValueString(),
                MinReplicas:  int(plan.MinReplicas.ValueInt64()),
                MaxReplicas:  int(plan.MaxReplicas.ValueInt64()),
                Region:       region,
                Sovereignty:  sovereigntyLevel,
                Tags:         stringMapFromFramework(ctx, plan.Tags, &resp.Diagnostics),
        }

        if resp.Diagnostics.HasError() {
                return
        }

        result, err := r.client.UpdateInferenceEndpoint(ctx, state.ID.ValueString(), endpointReq)
        if err != nil {
                resp.Diagnostics.AddError("Error updating inference endpoint",
                        fmt.Sprintf("Could not update inference endpoint %s: %s", state.ID.ValueString(), err))
                return
        }

        plan.ID = types.StringValue(result.ID)
        plan.Region = types.StringValue(region)
        plan.Sovereignty = types.StringValue(sovereigntyLevel)
        plan.EndpointURL = types.StringValue(result.EndpointURL)
        plan.Status = types.StringValue(result.Status)
        plan.CreatedAt = types.StringValue(result.CreatedAt)
        plan.UpdatedAt = types.StringValue(result.UpdatedAt)
        plan.Tags = frameworkFromStringMap(ctx, result.Tags)

        diags = resp.State.Set(ctx, plan)
        resp.Diagnostics.Append(diags...)
}

// Delete deletes the inference endpoint resource.
func (r *InferenceEndpointResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
        var state InferenceEndpointResourceModel

        diags := req.State.Get(ctx, &state)
        resp.Diagnostics.Append(diags...)
        if resp.Diagnostics.HasError() {
                return
        }

        err := r.client.DeleteInferenceEndpoint(ctx, state.ID.ValueString())
        if err != nil {
                if apiErr, ok := err.(*client.APIError); ok && apiErr.StatusCode == 404 {
                        return
                }
                resp.Diagnostics.AddError("Error deleting inference endpoint",
                        fmt.Sprintf("Could not delete inference endpoint %s: %s", state.ID.ValueString(), err))
                return
        }

        tflog.Info(ctx, "deleted inference endpoint", map[string]interface{}{"id": state.ID.ValueString()})
}

// ImportState imports an existing inference endpoint into Terraform.
func (r *InferenceEndpointResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
        resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
