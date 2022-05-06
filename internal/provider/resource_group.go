package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/kinvolk/nebraska/backend/pkg/codegen"
)

func resourceGroup() *schema.Resource {
	return &schema.Resource{
		Description: "A group provides a particular release channel to machines and controls various options that manage the update procedure.",

		CreateContext: resourceGroupCreate,
		ReadContext:   dataSourceGroupRead,
		UpdateContext: resourceGroupUpdate,
		DeleteContext: resourceGroupDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
				Description:  "Name of the group.",
			},
			"track": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Identifier for clients, filled with the group ID if omitted.",
			},
			"application_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "ID of the application this group belongs to.",
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "A description of the group.",
			},
			"created_ts": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Creation timestamp",
			},
			"rollout_in_progress": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Indicates whether a rollout is currently in progress for this group.",
			},
			"channel_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The channel this group provides.",
			},
			"policy_updates_enabled": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Enable updates.",
			},
			"policy_safe_mode": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Safe mode will only update 1 instance at a time, and stop if an update fails.",
			},
			"policy_office_hours": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Only update between 9am and 5pm.",
			},
			"policy_timezone": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "Asia/Calcutta",
				Description: "Timezone used to inform `policy_office_hours`.",
			},
			"policy_period_interval": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "1 hours",
				Description: "Period used in combination with `policy_max_updates_per_period`.",
			},
			"policy_max_updates_per_period": {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     1,
				Description: "The maximum number of updates that can be performed within the `policy_period_interval`.",
			},
			"policy_update_timeout": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "1 days",
				Description: "Timeout for updates",
			},
		},
	}
}

func resourceGroupCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {

	c := meta.(*apiClient)

	applicationID := d.Get("application_id").(string)

	var diags diag.Diagnostics
	groupConfig := resourceToGroupConfig(d)

	group, err := c.client.CreateGroupWithResponse(ctx, applicationID, codegen.CreateGroupJSONRequestBody(*groupConfig), c.reqEditors...)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "Couldn't create group",
			Detail:   fmt.Sprintf("Got an error when creating group: %v", err),
		})
		return diags
	}

	if group.JSON200 == nil {
		diags = append(diags, invalidResponseCodeDiag("Creating group", group.HTTPResponse))
		return diags
	}

	d.SetId(group.JSON200.Id)
	groupToResourceData(*group.JSON200, d)
	return nil
}

func resourceGroupUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {

	c := meta.(*apiClient)

	applicationID := d.Get("application_id").(string)

	var diags diag.Diagnostics
	groupConfig := resourceToGroupConfig(d)

	group, err := c.client.UpdateGroupWithResponse(ctx, applicationID, d.Id(), codegen.UpdateGroupJSONRequestBody(*groupConfig), c.reqEditors...)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "Couldn't update group",
			Detail:   fmt.Sprintf("Got an error when updating group: %q error:%v", d.Id(), err),
		})
		return diags
	}

	if group.JSON200 == nil {
		diags = append(diags, invalidResponseCodeDiag("Updating group", group.HTTPResponse))
		return diags
	}

	d.SetId(group.JSON200.Id)
	groupToResourceData(*group.JSON200, d)
	return nil
}

func resourceGroupDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {

	c := meta.(*apiClient)

	appID := d.Get("application_id").(string)
	groupID := d.Id()

	_, err := c.client.DeleteGroupWithResponse(ctx, appID, groupID, c.reqEditors...)
	if err != nil {
		return diag.FromErr(err)
	}
	return nil
}
