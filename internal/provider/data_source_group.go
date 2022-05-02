package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/kinvolk/nebraska/backend/pkg/codegen"
)

func dataSourceGroup() *schema.Resource {
	return &schema.Resource{
		Description: "A group is used by machines to track releases",
		ReadContext: dataSourceGroupRead,
		Schema: map[string]*schema.Schema{
			"application_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Application Id of the group",
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the group.",
			},
			"id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Id of the group",
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "A description of the group.",
			},
			"created_ts": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Creation timestamp.",
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
				Description: "Are updates enabled?",
			},
			"policy_safe_mode": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Safe mode will only update 1 instance at a time, and stop if an update fails.",
			},
			"policy_office_hours": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Only update between 9am and 5pm.",
			},
			"policy_timezone": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Timezone used to inform `policy_office_hours`.",
			},
			"policy_period_interval": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Period used in combination with `policy_max_updates_per_period`.",
			},
			"policy_max_updates_per_period": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "The maximum number of updates that can be performed within the `policy_period_interval`.",
			},
			"policy_update_timeout": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Timeout for updates.",
			},
			"track": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Identifier for machines.",
			},
		},
	}
}

func dataSourceGroupRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {

	c := meta.(*apiClient)

	var diags diag.Diagnostics

	appID := d.Get("application_id").(string)
	name := d.Get("name").(string)

	page := 1
	perPage := 10
	groupsResp, err := c.client.PaginateGroupsWithResponse(ctx, appID, &codegen.PaginateGroupsParams{Page: &page, Perpage: &perPage}, c.reqEditors...)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "Fetching Groups",
			Detail:   fmt.Sprintf("Error fetching groups: %v", err),
		})
		return diags
	}
	if groupsResp.JSON200 == nil {
		diags = append(diags, invalidResponseCodeDiag("Fetching group", groupsResp.StatusCode()))
		return diags
	}

	totalPages := groupsResp.JSON200.TotalCount / perPage

	var group *codegen.Group

	group = filterGroupByName(groupsResp.JSON200.Groups, name)

	for page <= totalPages && group == nil {
		page += 1
		groupsResp, err := c.client.PaginateGroupsWithResponse(ctx, appID, &codegen.PaginateGroupsParams{Page: &page, Perpage: &perPage}, c.reqEditors...)
		if err != nil {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  "Fetching Groups",
				Detail:   fmt.Sprintf("Error fetching groups: %v", err),
			})
			return diags
		}
		if groupsResp.JSON200 == nil {
			diags = append(diags, invalidResponseCodeDiag("Fetching group", groupsResp.StatusCode()))
			return diags
		}
		group = filterGroupByName(groupsResp.JSON200.Groups, name)
	}
	if group == nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "Group not found",
			Detail:   fmt.Sprintf("Group not found for name: %q, appId: %q", name, appID),
		})
		return diags
	}

	d.SetId(group.Id)
	groupToResourceData(*group, d)
	return diags
}

func filterGroupByName(groups []codegen.Group, name string) *codegen.Group {

	for _, group := range groups {
		if group.Name == name {
			return &group
		}
	}
	return nil
}
