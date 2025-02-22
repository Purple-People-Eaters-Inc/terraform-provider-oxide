// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/oxidecomputer/oxide.go/oxide"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource              = (*floatingIpResource)(nil)
	_ resource.ResourceWithConfigure = (*floatingIpResource)(nil)
)

// NewFloatingIpResource is a helper function to simplify the provider implementation.
func NewFloatingIpResource() resource.Resource {
	return &floatingIpResource{}
}

// floatingIpResource is the resource implementation.
type floatingIpResource struct {
	client *oxide.Client
}

type floatingIpResourceModel struct {
	ID           types.String   `tfsdk:"id"`
	Name         types.String   `tfsdk:"name"`
	Description  types.String   `tfsdk:"description"`
	ProjectID    types.String   `tfsdk:"project_id"`
	IPPoolID     types.String   `tfsdk:"ip_pool_id"`
	IP           types.String   `tfsdk:"ip"`
	TimeCreated  types.String   `tfsdk:"time_created"`
	TimeModified types.String   `tfsdk:"time_modified"`
	Timeouts     timeouts.Value `tfsdk:"timeouts"`
}

// Metadata returns the resource type name.
func (r *floatingIpResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "oxide_floating_ip"
}

// Configure adds the provider configured client to the data source.
func (r *floatingIpResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	r.client = req.ProviderData.(*oxide.Client)
}

func (r *floatingIpResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// Schema defines the schema for the resource.
func (r *floatingIpResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Name of the floating IP.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				Required:    true,
				Description: "Description for the floating IP.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"project_id": schema.StringAttribute{
				Required:    true,
				Description: "ID of the project that will contain the floating IP.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"ip": schema.StringAttribute{
				Optional:    true,
				Description: "IP Address of the floating IP.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplaceIfConfigured(),
				},
			},
			"ip_pool_id": schema.StringAttribute{
				Optional:    true,
				Description: "ID of the IP pool that will contain the floating IP.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplaceIfConfigured(),
				},
			},
			"timeouts": timeouts.Attributes(ctx, timeouts.Opts{
				Create: true,
				Read:   true,
				Update: true,
				Delete: true,
			}),
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Unique, immutable, system-controlled identifier of the floating IP.",
			},
			"time_created": schema.StringAttribute{
				Computed:    true,
				Description: "Timestamp of when this image was created.",
			},
			"time_modified": schema.StringAttribute{
				Computed:    true,
				Description: "Timestamp of when this image was last modified.",
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *floatingIpResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan floatingIpResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createTimeout, diags := plan.Timeouts.Create(ctx, defaultTimeout())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()

	params := oxide.FloatingIpCreateParams{
		Project: oxide.NameOrId(plan.ProjectID.ValueString()),
		Body: &oxide.FloatingIpCreate{
			Description: plan.Description.ValueString(),
			Ip:          plan.IP.ValueString(),
			Name:        oxide.Name(plan.Name.ValueString()),
			Pool:        oxide.NameOrId(plan.IPPoolID.ValueString()),
		},
	}

	floatingIp, err := r.client.FloatingIpCreate(ctx, params)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating floatingIp",
			"API error: "+err.Error(),
		)
		return
	}

	tflog.Trace(ctx, fmt.Sprintf("created floatingIp with ID: %v", floatingIp.Id), map[string]any{"success": true})

	// Map response body to schema and populate Computed attribute values
	plan.ID = types.StringValue(floatingIp.Id)
	plan.TimeCreated = types.StringValue(floatingIp.TimeCreated.String())
	plan.TimeModified = types.StringValue(floatingIp.TimeModified.String())

	// Save plan into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *floatingIpResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state floatingIpResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	readTimeout, diags := state.Timeouts.Read(ctx, defaultTimeout())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()

	params := oxide.FloatingIpViewParams{
		FloatingIp: oxide.NameOrId(state.ID.ValueString()),
	}
	floatingIp, err := r.client.FloatingIpView(ctx, params)
	if err != nil {
		if is404(err) {
			// Remove resource from state during a refresh
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Unable to read floatingIp:",
			"API error: "+err.Error(),
		)
		return
	}
	tflog.Trace(ctx, fmt.Sprintf("read floatingIp with ID: %v", floatingIp.Id), map[string]any{"success": true})

	state.Description = types.StringValue(floatingIp.Description)
	state.ID = types.StringValue(floatingIp.Id)
	state.Name = types.StringValue(string(floatingIp.Name))
	state.IP = types.StringValue(floatingIp.Ip)
	state.IPPoolID = types.StringValue(floatingIp.IpPoolId)
	state.Name = types.StringValue(string(floatingIp.Name))
	state.ProjectID = types.StringValue(floatingIp.ProjectId)
	state.TimeCreated = types.StringValue(floatingIp.TimeCreated.String())
	state.TimeModified = types.StringValue(floatingIp.TimeModified.String())

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *floatingIpResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"Error updating floatingIp",
		"the oxide API currently does not support updating floatingIps")
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *floatingIpResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state floatingIpResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	deleteTimeout, diags := state.Timeouts.Delete(ctx, defaultTimeout())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, deleteTimeout)
	defer cancel()

	params := oxide.FloatingIpDeleteParams{
		FloatingIp: oxide.NameOrId(state.ID.ValueString()),
	}
	if err := r.client.FloatingIpDelete(ctx, params); err != nil {
		if !is404(err) {
			resp.Diagnostics.AddError(
				"Unable to delete floatingIp:",
				"API error: "+err.Error(),
			)
			return
		}
	}

	tflog.Trace(ctx, fmt.Sprintf("deleted floatingIp with ID: %v", state.ID.ValueString()), map[string]any{"success": true})
}
