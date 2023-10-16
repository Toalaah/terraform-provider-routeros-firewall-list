/*
 * terraform-provider-routeros-firewall-list
 * Copyright (C) 2023  Samuel Kunst
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/toalaah/terraform-provider-routeros-firewall-list/internal/client"

	"github.com/google/uuid"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &FirewallRuleOrderingResource{}

func NewFirewallRuleOrderingResource() resource.Resource {
	return &FirewallRuleOrderingResource{}
}

// FirewallRuleOrderingResource defines the resource implementation.
type FirewallRuleOrderingResource struct {
	client *client.Client
}

// FirewallRuleOrderingResourceModel describes the resource data model.
type FirewallRuleOrderingResourceModel struct {
	RuleType types.String `tfsdk:"rule_type"`
	Rules    types.List   `tfsdk:"rules"`
	ID       types.String `tfsdk:"id"`
}

func (r *FirewallRuleOrderingResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_rule_ordering"
}

func (r *FirewallRuleOrderingResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}
	r.client = client
}

func (r *FirewallRuleOrderingResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Firewall rule ordering",
		Description:         "Firewall rule ordering",
		Attributes: map[string]schema.Attribute{
			"rule_type": schema.StringAttribute{
				MarkdownDescription: "The rule type to apply ordering to",
				Description:         "The rule type to apply ordering to",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("filter", "nat", "mangle", "raw"),
				},
			},
			"rules": schema.ListAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "List of rules arranged in their desired order",
				Description:         "List of rules arranged in their desired order",
				Required:            true,
			},
			"id": schema.StringAttribute{
				Computed:            true,
				Description:         "Identifier of resource",
				MarkdownDescription: "Identifier of resource",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *FirewallRuleOrderingResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data FirewallRuleOrderingResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(r.createOrdering(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.ID = types.StringValue(uuid.New().String())

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *FirewallRuleOrderingResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data FirewallRuleOrderingResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	rules, diags := r.rulesFromTerraformValue(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	match, err := r.client.RuleOrderExists(data.RuleType.ValueString(), rules)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read ordering, got error: %s", err))
		return
	}

	if !match {
		// force recreation of entire ordering. A little blunt but does the job
		data.Rules = types.ListNull(types.StringType)
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	}
}

func (r *FirewallRuleOrderingResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data FirewallRuleOrderingResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(r.createOrdering(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete removes the ordering lock.
//
// Note that since this is a pseudo-resource, no API call / further cleanup is
// necessary upon deletion. This does imply, however, that original state (in
// terms of the original rule ordering) is not restored. This is still
// appropriate given that this "resource" is simply meant to represent a lock /
// ordering guarantee between *two* firewall rules, and not some absolute
// ordering of the entire chain.
func (r *FirewallRuleOrderingResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
}

// createOrdering orders rules in accordance to the passed resource model. It
// *does not* set or otherwise interact with state; this responsibility is left
// to the caller.
func (r *FirewallRuleOrderingResource) createOrdering(ctx context.Context, data *FirewallRuleOrderingResourceModel) (diags diag.Diagnostics) {
	var rules []client.FirewallRule

	rules, err := r.rulesFromTerraformValue(ctx, data)
	diags.Append(err...)
	if diags.HasError() {
		return
	}

	if err := r.client.OrderRules(data.RuleType.ValueString(), rules...); err != nil {
		diags.AddError("Client Error", fmt.Sprintf("Unable to create ordering, got error(s): %s", err))
	}

	return
}

// rulesFromTerraformValue converts Terraform's internal list representation to
// a usable array of FirewallRules which the client can understand.
func (r *FirewallRuleOrderingResource) rulesFromTerraformValue(ctx context.Context, data *FirewallRuleOrderingResourceModel) ([]client.FirewallRule, diag.Diagnostics) {
	var rules []client.FirewallRule
	var diags diag.Diagnostics

	arr := make([]types.String, 0, len(data.Rules.Elements()))
	diags.Append(data.Rules.ElementsAs(ctx, &arr, false)...)

	for _, v := range arr {
		rule, err := r.client.GetRule(data.RuleType.ValueString(), v.ValueString())
		if err != nil {
			diags.AddError("Client Error", fmt.Sprintf("Unable to create ordering, got error: %s", err))
		}
		rules = append(rules, rule)
	}

	return rules, diags
}
