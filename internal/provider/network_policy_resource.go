package provider

import (
        "context"
        "fmt"

        "github.com/HarchCorp/harchos-terraform-provider/internal/client"
        "github.com/HarchCorp/harchos-terraform-provider/internal/sovereignty"
        "github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
        "github.com/hashicorp/terraform-plugin-framework/diag"
        "github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
        "github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
        "github.com/hashicorp/terraform-plugin-framework/attr"
        "github.com/hashicorp/terraform-plugin-framework/path"
        "github.com/hashicorp/terraform-plugin-framework/resource"
        "github.com/hashicorp/terraform-plugin-framework/resource/schema"
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
        _ resource.Resource                = &NetworkPolicyResource{}
        _ resource.ResourceWithConfigure   = &NetworkPolicyResource{}
        _ resource.ResourceWithImportState = &NetworkPolicyResource{}
        _ resource.ResourceWithModifyPlan  = &NetworkPolicyResource{}
)

// NetworkPolicyResource defines the resource implementation.
type NetworkPolicyResource struct {
        client      *client.Client
        region      string
        sovereignty string
}

// NetworkPolicyResourceModel describes the resource data model.
type NetworkPolicyResourceModel struct {
        ID          types.String `tfsdk:"id"`
        Name        types.String `tfsdk:"name"`
        Region      types.String `tfsdk:"region"`
        Sovereignty types.String `tfsdk:"sovereignty"`
        Ingress     types.List   `tfsdk:"ingress"`
        Egress      types.List   `tfsdk:"egress"`
        Tags        types.Map    `tfsdk:"tags"`
        Status      types.String `tfsdk:"status"`
        CreatedAt   types.String `tfsdk:"created_at"`
        UpdatedAt   types.String `tfsdk:"updated_at"`
}

// NetworkRuleModel describes a single network rule in the schema.
type NetworkRuleModel struct {
        Protocol types.String `tfsdk:"protocol"`
        Port     types.Int64  `tfsdk:"port"`
        From     types.List   `tfsdk:"from"`
        To       types.List   `tfsdk:"to"`
        Action   types.String `tfsdk:"action"`
}

// NewNetworkPolicyResource returns a new network policy resource.
func NewNetworkPolicyResource() resource.Resource {
        return &NetworkPolicyResource{}
}

// Metadata returns the resource type name.
func (r *NetworkPolicyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
        resp.TypeName = req.ProviderTypeName + "_network_policy"
}

// Schema returns the resource schema.
func (r *NetworkPolicyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
        resp.Schema = schema.Schema{
                MarkdownDescription: "Manages a HarchOS network policy. Network policies control " +
                        "ingress and egress traffic for workloads within a sovereign boundary, " +
                        "enforcing zero-trust networking principles.",

                Attributes: map[string]schema.Attribute{
                        "id": schema.StringAttribute{
                                MarkdownDescription: "The unique identifier of the network policy.",
                                Computed:            true,
                                PlanModifiers: []planmodifier.String{
                                        stringplanmodifier.UseStateForUnknown(),
                                },
                        },
                        "name": schema.StringAttribute{
                                MarkdownDescription: "The name of the network policy.",
                                Required:            true,
                                Validators: []validator.String{
                                        stringvalidator.LengthBetween(1, 128),
                                },
                        },
                        "region": schema.StringAttribute{
                                MarkdownDescription: "The HarchOS region for the network policy. Defaults to provider region.",
                                Optional: true,
                                Computed: true,
                                Default:  stringdefault.StaticString(""),
                        },
                        "sovereignty": schema.StringAttribute{
                                MarkdownDescription: "The sovereignty level for this policy. Cannot be downgraded.",
                                Optional: true,
                                Computed: true,
                                Default:  stringdefault.StaticString(""),
                                Validators: []validator.String{
                                        sovereignty.SovereigntyLevelValidator(),
                                },
                        },
                        "ingress": schema.ListNestedAttribute{
                                MarkdownDescription: "List of ingress rules for incoming traffic.",
                                Optional:            true,
                                NestedObject: schema.NestedAttributeObject{
                                        Attributes: map[string]schema.Attribute{
                                                "protocol": schema.StringAttribute{
                                                        MarkdownDescription: "The network protocol (tcp, udp, icmp).",
                                                        Required:            true,
                                                        Validators: []validator.String{
                                                                stringvalidator.OneOf("tcp", "udp", "icmp"),
                                                        },
                                                },
                                                "port": schema.Int64Attribute{
                                                        MarkdownDescription: "The port number. 0 means all ports.",
                                                        Optional:            true,
                                                        Validators: []validator.Int64{
                                                                int64validator.Between(0, 65535),
                                                        },
                                                },
                                                "from": schema.ListAttribute{
                                                        MarkdownDescription: "Source CIDR blocks or workload selectors.",
                                                        Optional:            true,
                                                        ElementType:         types.StringType,
                                                },
                                                "to": schema.ListAttribute{
                                                        MarkdownDescription: "Destination CIDR blocks or workload selectors.",
                                                        Optional:            true,
                                                        ElementType:         types.StringType,
                                                },
                                                "action": schema.StringAttribute{
                                                        MarkdownDescription: "The action to take (allow, deny).",
                                                        Required:            true,
                                                        Validators: []validator.String{
                                                                stringvalidator.OneOf("allow", "deny"),
                                                        },
                                                },
                                        },
                                },
                        },
                        "egress": schema.ListNestedAttribute{
                                MarkdownDescription: "List of egress rules for outgoing traffic.",
                                Optional:            true,
                                NestedObject: schema.NestedAttributeObject{
                                        Attributes: map[string]schema.Attribute{
                                                "protocol": schema.StringAttribute{
                                                        MarkdownDescription: "The network protocol (tcp, udp, icmp).",
                                                        Required:            true,
                                                        Validators: []validator.String{
                                                                stringvalidator.OneOf("tcp", "udp", "icmp"),
                                                        },
                                                },
                                                "port": schema.Int64Attribute{
                                                        MarkdownDescription: "The port number. 0 means all ports.",
                                                        Optional:            true,
                                                        Validators: []validator.Int64{
                                                                int64validator.Between(0, 65535),
                                                        },
                                                },
                                                "from": schema.ListAttribute{
                                                        MarkdownDescription: "Source CIDR blocks or workload selectors.",
                                                        Optional:            true,
                                                        ElementType:         types.StringType,
                                                },
                                                "to": schema.ListAttribute{
                                                        MarkdownDescription: "Destination CIDR blocks or workload selectors.",
                                                        Optional:            true,
                                                        ElementType:         types.StringType,
                                                },
                                                "action": schema.StringAttribute{
                                                        MarkdownDescription: "The action to take (allow, deny).",
                                                        Required:            true,
                                                        Validators: []validator.String{
                                                                stringvalidator.OneOf("allow", "deny"),
                                                        },
                                                },
                                        },
                                },
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
                                MarkdownDescription: "The current status of the network policy.",
                                Computed:            true,
                        },
                        "created_at": schema.StringAttribute{
                                MarkdownDescription: "Timestamp when the network policy was created.",
                                Computed:            true,
                        },
                        "updated_at": schema.StringAttribute{
                                MarkdownDescription: "Timestamp when the network policy was last updated.",
                                Computed:            true,
                        },
                },
        }
}

// Configure adds the provider-configured client to the resource.
func (r *NetworkPolicyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *NetworkPolicyResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
        var plan, state NetworkPolicyResourceModel

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
                                fmt.Sprintf("Cannot change network policy sovereignty: %s", err),
                        )
                }
        }
}

// Create creates the network policy resource.
func (r *NetworkPolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
        var plan NetworkPolicyResourceModel

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

        ingressRules := networkRulesFromList(ctx, plan.Ingress, &resp.Diagnostics)
        egressRules := networkRulesFromList(ctx, plan.Egress, &resp.Diagnostics)

        if resp.Diagnostics.HasError() {
                return
        }

        policyReq := &client.NetworkPolicy{
                Name:        plan.Name.ValueString(),
                Region:      region,
                Sovereignty: sovereigntyLevel,
                Ingress:     ingressRules,
                Egress:      egressRules,
                Tags:        stringMapFromFramework(ctx, plan.Tags, &resp.Diagnostics),
        }

        if resp.Diagnostics.HasError() {
                return
        }

        result, err := r.client.CreateNetworkPolicy(ctx, policyReq)
        if err != nil {
                resp.Diagnostics.AddError("Error creating network policy",
                        fmt.Sprintf("Could not create network policy: %s", err))
                return
        }

        plan.ID = types.StringValue(result.ID)
        plan.Region = types.StringValue(region)
        plan.Sovereignty = types.StringValue(sovereigntyLevel)
        plan.Status = types.StringValue(result.Status)
        plan.CreatedAt = types.StringValue(result.CreatedAt)
        plan.UpdatedAt = types.StringValue(result.UpdatedAt)
        plan.Tags = frameworkFromStringMap(ctx, result.Tags)

        diags = resp.State.Set(ctx, plan)
        resp.Diagnostics.Append(diags...)

        tflog.Info(ctx, "created network policy", map[string]interface{}{
                "id": result.ID, "name": result.Name,
        })
}

// Read reads the network policy state from the API.
func (r *NetworkPolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
        var state NetworkPolicyResourceModel

        diags := req.State.Get(ctx, &state)
        resp.Diagnostics.Append(diags...)
        if resp.Diagnostics.HasError() {
                return
        }

        result, err := r.client.GetNetworkPolicy(ctx, state.ID.ValueString())
        if err != nil {
                if apiErr, ok := err.(*client.APIError); ok && apiErr.StatusCode == 404 {
                        resp.State.RemoveResource(ctx)
                        return
                }
                resp.Diagnostics.AddError("Error reading network policy",
                        fmt.Sprintf("Could not read network policy %s: %s", state.ID.ValueString(), err))
                return
        }

        state.ID = types.StringValue(result.ID)
        state.Name = types.StringValue(result.Name)
        state.Region = types.StringValue(result.Region)
        state.Sovereignty = types.StringValue(result.Sovereignty)
        state.Status = types.StringValue(result.Status)
        state.CreatedAt = types.StringValue(result.CreatedAt)
        state.UpdatedAt = types.StringValue(result.UpdatedAt)
        state.Tags = frameworkFromStringMap(ctx, result.Tags)

        // Map ingress rules from API response back to state
        if len(result.Ingress) > 0 {
                ingressRules := make([]NetworkRuleModel, 0, len(result.Ingress))
                for _, rule := range result.Ingress {
                        rm := NetworkRuleModel{
                                Protocol: types.StringValue(rule.Protocol),
                                Port:     types.Int64Value(int64(rule.Port)),
                                Action:   types.StringValue(rule.Action),
                        }
                        if len(rule.From) > 0 {
                                fromVals := make([]attr.Value, 0, len(rule.From))
                                for _, f := range rule.From {
                                        fromVals = append(fromVals, types.StringValue(f))
                                }
                                fromList, d := types.ListValue(types.StringType, fromVals)
                                resp.Diagnostics.Append(d...)
                                rm.From = fromList
                        } else {
                                rm.From = types.ListNull(types.StringType)
                        }
                        if len(rule.To) > 0 {
                                toVals := make([]attr.Value, 0, len(rule.To))
                                for _, to := range rule.To {
                                        toVals = append(toVals, types.StringValue(to))
                                }
                                toList, d := types.ListValue(types.StringType, toVals)
                                resp.Diagnostics.Append(d...)
                                rm.To = toList
                        } else {
                                rm.To = types.ListNull(types.StringType)
                        }
                        ingressRules = append(ingressRules, rm)
                }
                ingressList, d := types.ListValueFrom(ctx, state.Ingress.ElementType(ctx), ingressRules)
                resp.Diagnostics.Append(d...)
                state.Ingress = ingressList
        } else {
                state.Ingress = types.ListNull(state.Ingress.ElementType(ctx))
        }

        // Map egress rules from API response back to state
        if len(result.Egress) > 0 {
                egressRules := make([]NetworkRuleModel, 0, len(result.Egress))
                for _, rule := range result.Egress {
                        rm := NetworkRuleModel{
                                Protocol: types.StringValue(rule.Protocol),
                                Port:     types.Int64Value(int64(rule.Port)),
                                Action:   types.StringValue(rule.Action),
                        }
                        if len(rule.From) > 0 {
                                fromVals := make([]attr.Value, 0, len(rule.From))
                                for _, f := range rule.From {
                                        fromVals = append(fromVals, types.StringValue(f))
                                }
                                fromList, d := types.ListValue(types.StringType, fromVals)
                                resp.Diagnostics.Append(d...)
                                rm.From = fromList
                        } else {
                                rm.From = types.ListNull(types.StringType)
                        }
                        if len(rule.To) > 0 {
                                toVals := make([]attr.Value, 0, len(rule.To))
                                for _, to := range rule.To {
                                        toVals = append(toVals, types.StringValue(to))
                                }
                                toList, d := types.ListValue(types.StringType, toVals)
                                resp.Diagnostics.Append(d...)
                                rm.To = toList
                        } else {
                                rm.To = types.ListNull(types.StringType)
                        }
                        egressRules = append(egressRules, rm)
                }
                egressList, d := types.ListValueFrom(ctx, state.Egress.ElementType(ctx), egressRules)
                resp.Diagnostics.Append(d...)
                state.Egress = egressList
        } else {
                state.Egress = types.ListNull(state.Egress.ElementType(ctx))
        }

        diags = resp.State.Set(ctx, state)
        resp.Diagnostics.Append(diags...)
}

// Update updates the network policy resource.
func (r *NetworkPolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
        var plan, state NetworkPolicyResourceModel

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

        ingressRules := networkRulesFromList(ctx, plan.Ingress, &resp.Diagnostics)
        egressRules := networkRulesFromList(ctx, plan.Egress, &resp.Diagnostics)

        if resp.Diagnostics.HasError() {
                return
        }

        policyReq := &client.NetworkPolicy{
                Name:        plan.Name.ValueString(),
                Region:      region,
                Sovereignty: sovereigntyLevel,
                Ingress:     ingressRules,
                Egress:      egressRules,
                Tags:        stringMapFromFramework(ctx, plan.Tags, &resp.Diagnostics),
        }

        if resp.Diagnostics.HasError() {
                return
        }

        result, err := r.client.UpdateNetworkPolicy(ctx, state.ID.ValueString(), policyReq)
        if err != nil {
                resp.Diagnostics.AddError("Error updating network policy",
                        fmt.Sprintf("Could not update network policy %s: %s", state.ID.ValueString(), err))
                return
        }

        plan.ID = types.StringValue(result.ID)
        plan.Region = types.StringValue(region)
        plan.Sovereignty = types.StringValue(sovereigntyLevel)
        plan.Status = types.StringValue(result.Status)
        plan.CreatedAt = types.StringValue(result.CreatedAt)
        plan.UpdatedAt = types.StringValue(result.UpdatedAt)
        plan.Tags = frameworkFromStringMap(ctx, result.Tags)

        diags = resp.State.Set(ctx, plan)
        resp.Diagnostics.Append(diags...)
}

// Delete deletes the network policy resource.
func (r *NetworkPolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
        var state NetworkPolicyResourceModel

        diags := req.State.Get(ctx, &state)
        resp.Diagnostics.Append(diags...)
        if resp.Diagnostics.HasError() {
                return
        }

        err := r.client.DeleteNetworkPolicy(ctx, state.ID.ValueString())
        if err != nil {
                if apiErr, ok := err.(*client.APIError); ok && apiErr.StatusCode == 404 {
                        return
                }
                resp.Diagnostics.AddError("Error deleting network policy",
                        fmt.Sprintf("Could not delete network policy %s: %s", state.ID.ValueString(), err))
                return
        }

        tflog.Info(ctx, "deleted network policy", map[string]interface{}{"id": state.ID.ValueString()})
}

// ImportState imports an existing network policy into Terraform.
func (r *NetworkPolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
        resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// networkRulesFromList converts a types.List of NetworkRuleModel to []client.NetworkRule.
func networkRulesFromList(ctx context.Context, l types.List, diags *diag.Diagnostics) []client.NetworkRule {
        if l.IsNull() || l.IsUnknown() {
                return nil
        }

        var rules []NetworkRuleModel
        d := l.ElementsAs(ctx, &rules, false)
        diags.Append(d...)
        if diags.HasError() {
                return nil
        }

        result := make([]client.NetworkRule, 0, len(rules))
        for _, rule := range rules {
                nr := client.NetworkRule{
                        Protocol: rule.Protocol.ValueString(),
                        Port:     int(rule.Port.ValueInt64()),
                        Action:   rule.Action.ValueString(),
                }

                if !rule.From.IsNull() {
                        var fromList []string
                        d := rule.From.ElementsAs(ctx, &fromList, false)
                        diags.Append(d...)
                        nr.From = fromList
                }

                if !rule.To.IsNull() {
                        var toList []string
                        d := rule.To.ElementsAs(ctx, &toList, false)
                        diags.Append(d...)
                        nr.To = toList
                }

                result = append(result, nr)
        }

        return result
}
