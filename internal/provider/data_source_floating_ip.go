// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/datasource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/oxidecomputer/oxide.go/oxide"
)

var (
	_ datasource.DataSource              = (*floatingIpDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*floatingIpDataSource)(nil)
)

// NewFloatingIpDataSource initialises an images datasource
func NewFloatingIpDataSource() datasource.DataSource {
	return &floatingIpDataSource{}
}

type floatingIpDataSource struct {
	client *oxide.Client
}

type floatingIpDataSourceModel struct {
	ID           types.String   `tfsdk:"id"`
	Name         types.String   `tfsdk:"name"`
	Description  types.String   `tfsdk:"description"`
	ProjectID    types.String   `tfsdk:"project_id"`
	ProjectName  types.String   `tfsdk:"project_name"`
	IP           types.String   `tfsdk:"ip"`
	IPPoolID     types.String   `tfsdk:"ip_pool_id"`
	InstanceId   types.String   `tfsdk:"instance_id"`
	TimeCreated  types.String   `tfsdk:"time_created"`
	TimeModified types.String   `tfsdk:"time_modified"`
	Timeouts     timeouts.Value `tfsdk:"timeouts"`
}

func (d *floatingIpDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "oxide_floating_ip"
}

// Configure adds the provider configured client to the data source.
func (d *floatingIpDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	d.client = req.ProviderData.(*oxide.Client)
}

func (d *floatingIpDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"project_name": schema.StringAttribute{
				Required:    true,
				Description: "Name of the project that contains the floating IP.",
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Name of the floating IP.",
			},
			"project_id": schema.StringAttribute{
				Computed:    true,
				Description: "ID of the project that contains the floating IP.",
			},
			"description": schema.StringAttribute{
				Computed:    true,
				Description: "Description for the floating IP.",
			},
			"timeouts": timeouts.Attributes(ctx),
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Unique, immutable, system-controlled identifier of the floating IP.",
			},
			"ip": schema.StringAttribute{
				Computed:    true,
				Description: "IP address of the floating IP.",
			},
			"ip_pool_id": schema.StringAttribute{
				Computed:    true,
				Description: "ID of the IP Pool containing the floating IP.",
			},
			"instance_id": schema.StringAttribute{
				Computed:    true,
				Description: "ID of the instance using the floating IP.",
			},
			"time_created": schema.StringAttribute{
				Computed:    true,
				Description: "Timestamp of when this floating IP was created.",
			},
			"time_modified": schema.StringAttribute{
				Computed:    true,
				Description: "Timestamp of when this floating IP was last modified.",
			},
		},
	}
}

func (d *floatingIpDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state floatingIpDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
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
		FloatingIp: oxide.NameOrId(state.Name.ValueString()),
		Project:    oxide.NameOrId(state.ProjectName.ValueString()),
	}
	floatingIp, err := d.client.FloatingIpView(ctx, params)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to read floating IP:",
			"API error: "+err.Error(),
		)
		return
	}
	tflog.Trace(ctx, fmt.Sprintf("read floating IP with ID: %v", floatingIp.Id), map[string]any{"success": true})

	state.ID = types.StringValue(floatingIp.Id)
	state.Name = types.StringValue(string(floatingIp.Name))
	state.Description = types.StringValue(floatingIp.Description)
	state.ProjectID = types.StringValue(floatingIp.ProjectId)
	state.IP = types.StringValue(floatingIp.Ip)
	state.IPPoolID = types.StringValue(floatingIp.IpPoolId)
	state.InstanceId = types.StringValue(floatingIp.InstanceId)
	state.TimeCreated = types.StringValue(floatingIp.TimeCreated.String())
	state.TimeModified = types.StringValue(floatingIp.TimeModified.String())

	// Save state into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
}
