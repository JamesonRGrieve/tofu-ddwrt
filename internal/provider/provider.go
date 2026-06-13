// SPDX-License-Identifier: AGPL-3.0-or-later

// Package provider implements the ddwrt OpenTofu/Terraform provider — a native
// client for DD-WRT routers over SSH. DD-WRT has no clean REST API;
// configuration lives in NVRAM, so the provider is generic over NVRAM: the
// ddwrt_nvram resource/data source manage any NVRAM variable
// (manage-declared-only), giving full coverage without per-feature code.
package provider

import (
	"context"

	"github.com/JamesonRGrieve/tofu-ddwrt/internal/ddwrt"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ provider.Provider = (*ddwrtProvider)(nil)

// New returns the provider factory for a given version.
func New(version string) func() provider.Provider {
	return func() provider.Provider { return &ddwrtProvider{version: version} }
}

type ddwrtProvider struct {
	version string
}

type providerModel struct {
	Host           types.String `tfsdk:"host"`
	Username       types.String `tfsdk:"username"`
	KeyFile        types.String `tfsdk:"key_file"`
	SSHBinary      types.String `tfsdk:"ssh_binary"`
	RestartCommand types.String `tfsdk:"restart_command"`
}

func (p *ddwrtProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	// Single-token type name -> resources are `ddwrt_nvram`, so Terraform's
	// prefix-before-first-underscore inference resolves the local name cleanly
	// (the source address is still jamesonrgrieve/ddwrt).
	resp.TypeName = "ddwrt"
	resp.Version = p.version
}

func (p *ddwrtProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Native provider for DD-WRT routers driven over SSH (Dropbear). Manages NVRAM " +
			"variables generically. DD-WRT has no clean REST API, so all config is expressed as NVRAM " +
			"key/value via the `ddwrt_nvram` resource.",
		Attributes: map[string]schema.Attribute{
			"host": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Router address (host or host:port), no scheme. Default SSH port is 22.",
			},
			"username": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "SSH username (default `root` — DD-WRT's Dropbear user).",
			},
			"key_file": schema.StringAttribute{
				Optional: true,
				MarkdownDescription: "Optional SSH identity file (`ssh -i`). When unset, the system ssh " +
					"client resolves the key / agent / OpenBao-signed certificate from ssh_config as usual. " +
					"The provider never handles a private key directly.",
			},
			"ssh_binary": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Path to the ssh executable (default `ssh`).",
			},
			"restart_command": schema.StringAttribute{
				Optional: true,
				MarkdownDescription: "Per-service restart command template applied after `nvram commit`. " +
					"The literal `{service}` is replaced with the resource's `restart` service name. " +
					"Defaults to `stopservice {service}; startservice {service}` (the DD-WRT idiom — DD-WRT " +
					"has no `service <svc> restart` verb). A resource `restart` of `*`, `all`, or `rc` " +
					"instead runs a full `rc restart` regardless of this template.",
			},
		},
	}
}

func (p *ddwrtProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var cfg providerModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client := ddwrt.NewClient(ddwrt.Config{
		Host:           cfg.Host.ValueString(),
		Username:       cfg.Username.ValueString(),
		KeyFile:        cfg.KeyFile.ValueString(),
		SSHBinary:      cfg.SSHBinary.ValueString(),
		RestartCommand: cfg.RestartCommand.ValueString(),
	})
	resp.ResourceData = client
	resp.DataSourceData = client
}

func (p *ddwrtProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{NewObjectResource}
}

func (p *ddwrtProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{NewObjectDataSource}
}
