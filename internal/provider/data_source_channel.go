package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/kinvolk/nebraska/backend/pkg/api"
	"github.com/kinvolk/nebraska/backend/pkg/codegen"
)

func dataSourceChannel() *schema.Resource {
	return &schema.Resource{
		Description: "A release channel for package",
		ReadContext: dataSourceChannelRead,
		Schema: map[string]*schema.Schema{
			"application_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "ID of the application this channel belongs to.",
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the channel.",
			},
			"arch": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Arch.",
			},

			"id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Id of the channel",
			},
			"color": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Hex color code of the channel on the UI.",
			},
			"created_ts": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Creation timestamp.",
			},
			"package_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "ID of this channel's package.",
			},
		},
	}
}

func dataSourceChannelRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {

	c := meta.(*apiClient)

	var diags diag.Diagnostics

	appID := d.Get("application_id").(string)
	name := d.Get("name").(string)
	arch := d.Get("arch").(string)

	page := 1
	perPage := 10
	channelsResp, err := c.client.PaginateChannelsWithResponse(ctx, appID, &codegen.PaginateChannelsParams{Page: &page, Perpage: &perPage}, c.reqEditors...)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "Fetching Channels",
			Detail:   fmt.Sprintf("Error fetching channels:%v", err),
		})
		return diags
	}
	if channelsResp.JSON200 == nil {
		diags = append(diags, invalidResponseCodeDiag("Fetching channels", channelsResp.StatusCode()))
		return diags
	}

	totalPages := channelsResp.JSON200.TotalCount / perPage

	var channel *codegen.Channel

	channel = filterChannelByNameArch(channelsResp.JSON200.Channels, name, arch)

	for page <= totalPages && channel == nil {
		page += 1
		channelsResp, err := c.client.PaginateChannelsWithResponse(ctx, appID, &codegen.PaginateChannelsParams{Page: &page, Perpage: &perPage}, c.reqEditors...)
		if err != nil {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  "Fetching Channels",
				Detail:   fmt.Sprintf("Error fetching channels:%v", err),
			})
			return diags
		}
		if channelsResp.JSON200 == nil {
			diags = append(diags, invalidResponseCodeDiag("Fetching channels", channelsResp.StatusCode()))
			return diags
		}

		channel = filterChannelByNameArch(channelsResp.JSON200.Channels, name, arch)
	}

	if channel == nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "Channel not found",
			Detail:   fmt.Sprintf("Channel not found for name: %q, arch: %q", name, arch),
		})
		return diags
	}

	d.SetId(channel.Id)
	channelToResourceData(*channel, d)
	return diags
}

func filterChannelByNameArch(channels []codegen.Channel, name string, arch string) *codegen.Channel {

	for _, channel := range channels {
		if channel.Name == name && api.Arch(int(channel.Arch)).String() == arch {
			return &channel
		}
	}
	return nil
}
