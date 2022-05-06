package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/kinvolk/nebraska/backend/pkg/api"
	"github.com/kinvolk/nebraska/backend/pkg/codegen"
)

// PackageType is the type of package
type PackageType int

const (
	// PackageTypeFlatcar is a Flatcar update package
	PackageTypeFlatcar PackageType = 1 + iota
	// PackageTypeDocker is a docker container
	PackageTypeDocker
	// PackageTypeRocket is a rkt container
	PackageTypeRocket
	// PackageTypeOther is a generic package type
	PackageTypeOther
)

var (
	// ErrInvalidPackageType is a custom error returned when an unsupported arch is
	// requested
	ErrInvalidPackageType = errors.New("nebraska: invalid/unsupported package type")

	// ValidPackageTypes are the package types that Nebraska supports
	ValidPackageTypes = []string{
		"flatcar",
		"docker",
		"rkt",
		"other",
	}
)

// String returns the string representation of the package type
func (pt PackageType) String() string {
	i := int(pt)

	return ValidPackageTypes[i-1]
}

// PackageTypeFromString parses the string into a PackageType
func PackageTypeFromString(s string) (PackageType, error) {
	for i, sd := range ValidPackageTypes {
		if s == sd {
			return PackageType(i + 1), nil
		}

	}

	return PackageTypeOther, ErrInvalidPackageType
}

func resourcePackage() *schema.Resource {
	return &schema.Resource{
		Description: "A versioned package of the application.",

		CreateContext: resourcePackageCreate,
		ReadContext:   resourcePackageRead,
		UpdateContext: resourcePackageUpdate,
		DeleteContext: resourcePackageDelete,

		Schema: map[string]*schema.Schema{
			"version": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
				Description:  "Package version.",
			},
			"url": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.IsURLWithHTTPorHTTPS,
				Description:  "URL where the package is available.",
			},
			"arch": {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringInSlice([]string{"all", "amd64", "aarch64", "x86"}, false),
				Default:      api.ArchAll.String(),
				Description:  "Package arch.",
			},
			"type": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringInSlice([]string{"flatcar", "docker", "rkt", "other", "git"}, false),
				Default:      PackageTypeFlatcar.String(),
				Description:  "Type of package.",
			},
			"filename": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The filename of the package.",
			},
			"description": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "A description of the package.",
			},
			"size": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The size, in bytes.",
			},
			"hash": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "A base64 encoded sha1 hash of the package digest. Tip: `cat update.gz | openssl dgst -sha1 -binary | base64`.",
			},
			"channels_blacklist": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Description: "A list of channels (by id) that cannot point to this package.",
			},
			"flatcar_action": {
				Type:        schema.TypeList,
				MaxItems:    1,
				Optional:    true,
				Computed:    true,
				Description: "A Flatcar specific Omaha action.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"event": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"chromeos_version": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"sha256": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "A base64 encoded sha256 hash of the action. Tip: `cat update.gz | openssl dgst -sha256 -binary | base64`.",
						},
						"needs_admin": {
							Type:     schema.TypeBool,
							Computed: true,
						},
						"is_delta": {
							Type:     schema.TypeBool,
							Computed: true,
						},
						"disable_payload_backoff": {
							Type:     schema.TypeBool,
							Computed: true,
						},
						"metadata_signature_rsa": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"metadata_size": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"deadline": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"created_ts": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
			"application_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "ID of the application this package belongs to.",
			},
			"created_ts": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Creation timestamp.",
			},
			"nua_commit": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"nua_namespace": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"nua_kustomize_config": {
				Type:     schema.TypeString,
				Optional: true,
			},
		},
	}
}

func resourcePackageRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return dataSourcePackageRead(ctx, d, meta)
}

func resourcePackageCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*apiClient)
	var diags diag.Diagnostics

	applicationID := d.Get("application_id").(string)

	packageConfig, err := resourceToPackageConfig(d)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "Couldn't create package config",
			Detail:   fmt.Sprintf("Got error when generating package config: %v", err),
		})
		return diags
	}
	packageResp, err := c.client.CreatePackageWithResponse(ctx, applicationID, codegen.CreatePackageJSONRequestBody(codegen.CreatePackageJSONBody(*packageConfig)), c.reqEditors...)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "Couldn't create package",
			Detail:   fmt.Sprintf("Got an error when creating package: %v", err),
		})
		return diags
	}
	if packageResp.JSON200 == nil {
		diags = append(diags, invalidResponseCodeDiag("Creating package", packageResp.HTTPResponse))
		return diags
	}
	err = packageToResource(*packageResp.JSON200, d)
	if err != nil {
		return diag.FromErr(err)
	}
	return nil
}

func resourcePackageUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {

	c := meta.(*apiClient)
	var diags diag.Diagnostics

	applicationID := d.Get("application_id").(string)

	packageConfig, err := resourceToPackageConfig(d)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "Couldn't create package config",
			Detail:   fmt.Sprintf("Got error when generating package config: %v", err),
		})
		return diags
	}
	packageResp, err := c.client.UpdatePackageWithResponse(ctx, applicationID, d.Id(), codegen.UpdatePackageJSONRequestBody(codegen.CreatePackageJSONBody(*packageConfig)), c.reqEditors...)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "Couldn't update package",
			Detail:   fmt.Sprintf("Got an error when updating package:%q error:%v", d.Id(), err),
		})
		return diags

	}
	if packageResp.JSON200 == nil {
		diags = append(diags, invalidResponseCodeDiag("Creating package", packageResp.HTTPResponse))
		return diags
	}

	err = packageToResource(*packageResp.JSON200, d)
	if err != nil {
		return diag.FromErr(err)
	}
	return nil
}

func resourcePackageDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*apiClient)

	appID := d.Get("application_id").(string)
	packageID := d.Id()

	_, err := c.client.DeletePackageWithResponse(ctx, appID, packageID, c.reqEditors...)
	if err != nil {
		return diag.FromErr(err)
	}
	return nil
}

func expandFlatcarActionSha256(l []interface{}) string {
	if len(l) == 0 || l[0] == nil {
		return ""
	}

	m := l[0].(map[string]interface{})

	if v, ok := m["sha256"].(string); ok {
		return v
	}

	return ""
}
