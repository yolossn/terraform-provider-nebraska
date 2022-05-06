package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/kinvolk/nebraska/backend/pkg/codegen"
)

func resourceChannel() *schema.Resource {
	return &schema.Resource{
		Description: "A release channel that provides a particular package version.",

		CreateContext: resourceChannelCreate,
		ReadContext:   dataSourceChannelRead,
		UpdateContext: resourceChannelUpdate,
		DeleteContext: resourceChannelDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
				Description:  "Name of the channel. Can be an existing one as long as the arch is different.",
			},
			"arch": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringInSlice([]string{"all", "amd64", "aarch64", "x86"}, false),
				Description:  "Arch. Cannot be changed once created.",
			},
			"application_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "ID of the application this channel belongs to.",
			},
			"color": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Hex color code that informs the color of the channel in the UI.",
			},
			"created_ts": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Creation timestamp.",
			},
			"package_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The id of the package this channel provides.",
			},
		},
	}
}

func resourceChannelCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*apiClient)

	var diags diag.Diagnostics
	appID := d.Get("application_id").(string)

	channelConfig, err := resourceToChannelConfig(d)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "Error generating channel config",
			Detail:   fmt.Sprintf("Error generating channel config to create channel: %v", err),
		})
		return diags
	}

	channel, err := c.client.CreateChannelWithResponse(ctx, appID, codegen.CreateChannelJSONRequestBody(*channelConfig), c.reqEditors...)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "Couldn't create channel",
			Detail:   fmt.Sprintf("Got an error when creating channel: %v", err),
		})
		return diags
	}
	if channel.JSON200 == nil {
		diags = append(diags, invalidResponseCodeDiag("Couldn't create channel", channel.HTTPResponse))
		return diags
	}

	d.SetId(channel.JSON200.Id)
	channelToResourceData(*channel.JSON200, d)
	return diags
}

func resourceChannelUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*apiClient)

	ID := d.Id()
	appID := d.Get("application_id").(string)
	var diags diag.Diagnostics

	channelConfig, err := resourceToChannelConfig(d)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "Error generating channel config",
			Detail:   fmt.Sprintf("Error generating channel config to update channel:%q error: %v", ID, err),
		})
		return diags
	}

	channel, err := c.client.UpdateChannelWithResponse(ctx, appID, ID, codegen.UpdateChannelJSONRequestBody(*channelConfig), c.reqEditors...)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "Couldn't update channel",
			Detail:   fmt.Sprintf("Got an error when updating channel: %v", err),
		})
		return diags
	}
	if channel.JSON200 == nil {
		diags = append(diags, invalidResponseCodeDiag("Couldn't update channel", channel.HTTPResponse))
		return diags
	}

	d.SetId(channel.JSON200.Id)
	channelToResourceData(*channel.JSON200, d)
	return diags
}

func resourceChannelDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {

	c := meta.(*apiClient)

	appID := d.Get("application_id").(string)
	channelID := d.Id()

	_, err := c.client.DeleteChannelWithResponse(ctx, appID, channelID, c.reqEditors...)
	if err != nil {
		return diag.FromErr(err)
	}
	return nil
}
