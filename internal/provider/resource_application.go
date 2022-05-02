package provider

import (
	"context"
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/kinvolk/nebraska/backend/pkg/codegen"
)

func resourceApplication() *schema.Resource {
	return &schema.Resource{
		Description:   "A nebraska application",
		CreateContext: resourceApplicationCreate,
		ReadContext:   resourceApplicationRead,
		UpdateContext: resourceApplicationUpdate,
		DeleteContext: resourceApplicationDelete,
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
				Type:         schema.TypeString,
				Description:  "",
				ValidateFunc: validation.StringIsNotEmpty,
				Required:     true,
			},
			"product_id": {
				Type:         schema.TypeString,
				Description:  "product id of app",
				ValidateFunc: validateProductID,
				Required:     true,
			},
		},
	}
}

func validateProductID(pID interface{}, key string) ([]string, []error) {

	productID, ok := pID.(string)
	if !ok {
		return nil, []error{fmt.Errorf("expected type of %q to be string", key)}
	}

	if len(productID) > 155 {
		return nil, []error{fmt.Errorf("product ID %v is not valid (max length 155)", productID)}
	}

	regMatcher := "^[a-zA-Z]+([a-zA-Z0-9\\-]*[a-zA-Z0-9])*(\\.[a-zA-Z]+([a-zA-Z0-1\\-]*[a-zA-Z0-9])*)+$"
	matches, err := regexp.MatchString(regMatcher, productID)
	if err != nil {
		return nil, []error{err}
	}

	if !matches {
		return nil, []error{fmt.Errorf("product ID %v is not valid (has to be in the form e.g. io.example.App)", productID)}
	}

	return nil, nil
}

func resourceApplicationRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return dataSourceApplicationRead(ctx, d, meta)
}

func resourceApplicationCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {

	c := meta.(*apiClient)

	var diags diag.Diagnostics

	appConfig, err := resourceToAppConfig(d)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "Error generating app config",
			Detail:   fmt.Sprintf("Error generating app config to Create Application: %v", err),
		})
		return diags
	}
	app, err := c.client.CreateAppWithResponse(ctx, &codegen.CreateAppParams{}, codegen.CreateAppJSONRequestBody(*appConfig), c.reqEditors...)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "Couldn't create application",
			Detail:   fmt.Sprintf("Got an error when creating application: %v", err),
		})
		return diags
	}
	if app.JSON200 == nil {
		diags = append(diags, invalidResponseCodeDiag("Couldn't create application", app.StatusCode()))
		return diags
	}

	d.SetId(app.JSON200.Id)
	appToResourceData(*app.JSON200, d)
	return nil
}

func resourceApplicationUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*apiClient)

	var appID = d.Id()

	var diags diag.Diagnostics

	appConfig, err := resourceToAppConfig(d)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "Error generating appConfig",
			Detail:   fmt.Sprintf("Error generating appConfig to Update Application:%q error: %v", appID, err),
		})
		return diags
	}
	app, err := c.client.UpdateAppWithResponse(ctx, appID, codegen.UpdateAppJSONRequestBody(*appConfig))
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "Couldn't update application",
			Detail:   fmt.Sprintf("Got an error when updating application: %v", err),
		})
		return diags
	}
	if app.JSON200 == nil {
		diags = append(diags, invalidResponseCodeDiag("Couldn't update application", app.StatusCode()))
		return diags
	}

	d.SetId(app.JSON200.Id)
	appToResourceData(*app.JSON200, d)
	return nil
}

func resourceApplicationDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {

	c := meta.(*apiClient)

	var appID = d.Id()

	_, err := c.client.DeleteAppWithResponse(ctx, appID, c.reqEditors...)
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}
