package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceApplication() *schema.Resource {
	return &schema.Resource{
		Description: "A nebraska application",
		ReadContext: dataSourceApplicationRead,
		Schema: map[string]*schema.Schema{
			"created_ts": {
				Type:        schema.TypeString,
				Description: "",
				Computed:    true,
			},
			"description": {
				Type:        schema.TypeString,
				Description: "A description of the application",
				Optional:    true,
			},
			"id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"name": {
				Type:        schema.TypeString,
				Description: "",
				Computed:    true,
			},
			"product_id": {
				Type:        schema.TypeString,
				Description: "product id of app",
				Required:    true,
			},
		},
	}
}

func dataSourceApplicationRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {

	c := meta.(*apiClient)

	var diags diag.Diagnostics

	appID := d.Get("product_id").(string)
	appResp, err := c.client.GetAppWithResponse(ctx, appID, c.reqEditors...)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "Couldn't fetch application",
			Detail:   fmt.Sprintf("Couldnt' fetch application with product id: %s", appID),
		})
	}
	if appResp.JSON200 == nil {
		diags = append(diags, invalidResponseCodeDiag("Fetching application", appResp.StatusCode()))
		return diags
	}

	d.SetId(appResp.JSON200.Id)
	appToResourceData(*appResp.JSON200, d)
	return nil
}
