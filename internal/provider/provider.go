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
	"os"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/toalaah/terraform-provider-routeros-firewall-list/internal/client"
)

// Ensure ScaffoldingProvider satisfies various provider interfaces.
var _ provider.Provider = &RouterosFWFLProvider{}

// RouterosFWFLProvider defines the provider implementation.
type RouterosFWFLProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// ScaffoldingProviderModel describes the provider data model.
type ScaffoldingProviderModel struct {
	HostURL  types.String `tfsdk:"hosturl"`
	Username types.String `tfsdk:"username"`
	Password types.String `tfsdk:"password"`
	CA       types.String `tfsdk:"ca_certificate"`
	Insecure types.Bool   `tfsdk:"insecure"`
}

func (p *RouterosFWFLProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "routeros-firewall-list"
	resp.Version = p.version
}

func (p *RouterosFWFLProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"hosturl": schema.StringAttribute{
				MarkdownDescription: "Example provider attribute",
				Optional:            true,
				Description:         "TODO",
			},
			"username": schema.StringAttribute{
				MarkdownDescription: "Example provider attribute",
				Optional:            true,
				Description:         "TODO",
			},
			"password": schema.StringAttribute{
				MarkdownDescription: "Example provider attribute",
				Optional:            true,
				Sensitive:           true,
				Description:         "TODO",
			},
			"ca_certificate": schema.StringAttribute{
				MarkdownDescription: "Example provider attribute",
				Optional:            true,
				Description:         "TODO",
			},
			"insecure": schema.BoolAttribute{
				MarkdownDescription: "Example provider attribute",
				Optional:            true,
				Description:         "TODO",
			},
		},
	}
}

func (p *RouterosFWFLProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config ScaffoldingProviderModel
	var opts client.ClientOpts

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	opts.HostURL = os.Getenv("ROS_HOSTURL")
	if !config.HostURL.IsNull() {
		opts.HostURL = config.HostURL.ValueString()
		if opts.HostURL == "" {
			resp.Diagnostics.AddAttributeError(
				path.Root("host"),
				"Unknown API Host",
				"Cannot create API client, no host value provided",
			)
		}
	}
	// TODO: parse value as URL and check if proto / port are already set
	opts.HostURL = fmt.Sprintf("https://%s:443", opts.HostURL)

	opts.Username = os.Getenv("ROS_USERNAME")
	if !config.Username.IsNull() {
		opts.Username = config.Username.ValueString()
		if opts.Username == "" {
			resp.Diagnostics.AddAttributeError(
				path.Root("username"),
				"Unknown API Username",
				"Cannot create API client, no username value provided",
			)
		}
	}

	opts.Password = os.Getenv("ROS_PASSWORD")
	if !config.Password.IsNull() {
		opts.Password = config.Password.ValueString()
	}

	opts.CA = os.Getenv("ROS_CA_CERTIFICATE")
	if !config.CA.IsNull() {
		opts.CA = config.CA.ValueString()
	}

	if v := os.Getenv("ROS_INSECURE"); v != "" && config.Insecure.IsNull() {
		var err error
		opts.Insecure, err = strconv.ParseBool(v)
		if err != nil {
			resp.Diagnostics.AddAttributeWarning(path.Root("insecure"),
				"Invalid value for parameter `insecure`",
				fmt.Sprintf("Could not parse provided value '%s' for parameter 'insecure' as a boolean", v),
			)
		}
	} else {
		opts.Insecure = config.Insecure.ValueBool()
	}

	if resp.Diagnostics.HasError() {
		return
	}

	client, err := client.New(opts)
	if err != nil {
		resp.Diagnostics.AddError("Client configure error", fmt.Sprintf("Error while configuring client, got err: %s", err))
		return
	}

	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *RouterosFWFLProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewFirewallRuleOrderingResource,
	}
}

func (p *RouterosFWFLProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &RouterosFWFLProvider{
			version: version,
		}
	}
}
