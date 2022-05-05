package provider

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/url"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/kinvolk/nebraska/backend/pkg/api"
	"github.com/kinvolk/nebraska/backend/pkg/codegen"
)

var pkgTypeToString = map[int]string{
	0: "flatcar",
	1: "docker",
	2: "rkt",
	3: "other",
}

func dataSourcePackage() *schema.Resource {
	return &schema.Resource{
		Description: "Package of the application",
		ReadContext: dataSourcePackageRead,
		Schema: map[string]*schema.Schema{
			"application_id": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validateProductID,
				Description:  "Application ID",
			},
			"version": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Package version.",
			},
			"arch": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Package arch.",
			},
			"id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Package ID",
			},
			"type": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Type of package.",
			},
			"url": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "URL where the package is available.",
			},
			"filename": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The filename of the package.",
			},
			"description": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "A description of the package.",
			},
			"size": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The size, in bytes.",
			},
			"hash": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "A base64 encoded sha1 hash of the package digest.",
			},
			"created_ts": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Creation timestamp.",
			},
			"flatcar_action": {
				Type:        schema.TypeList,
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
							Type:     schema.TypeString,
							Computed: true,
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
						"nua_commit": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"nua_namespace": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"nua_kustomize_config": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
			"channels_blacklist": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Description: "A list of channels (by id) that cannot point to this package.",
			},
		},
	}
}

func dataSourcePackageRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*apiClient)

	var diags diag.Diagnostics

	appID := d.Get("application_id").(string)
	version := d.Get("version").(string)
	arch := d.Get("arch").(string)

	page := 1
	perPage := 10

	packagesPage, err := c.client.PaginatePackagesWithResponse(ctx, appID, &codegen.PaginatePackagesParams{Page: &page, Perpage: &perPage}, c.reqEditors...)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "Fetching packages",
			Detail:   fmt.Sprintf("Error fetching packages:%v", err),
		})
		return diags
	}
	if packagesPage.JSON200 == nil {
		diags = append(diags, invalidResponseCodeDiag("Fetching packages", packagesPage.StatusCode()))
		return diags
	}

	totalPages := packagesPage.JSON200.TotalCount / perPage

	var nebraskaPackage *codegen.Package

	nebraskaPackage = filterPackageByVersionArch(packagesPage.JSON200.Packages, version, arch)

	for page <= totalPages && nebraskaPackage == nil {
		page += 1
		packagesPage, err := c.client.PaginatePackagesWithResponse(ctx, appID, &codegen.PaginatePackagesParams{Page: &page, Perpage: &perPage}, c.reqEditors...)
		if err != nil {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  "Fetching packages",
				Detail:   fmt.Sprintf("Error fetching packages:%v", err),
			})
			return diags
		}
		if packagesPage.JSON200 == nil {
			diags = append(diags, invalidResponseCodeDiag("Fetching packages", packagesPage.StatusCode()))
			return diags
		}
		nebraskaPackage = filterPackageByVersionArch(packagesPage.JSON200.Packages, version, arch)
	}

	if nebraskaPackage == nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "Package not found",
			Detail:   fmt.Sprintf("Package not found for version: %q, arch: %q", version, arch),
		})
		return diags
	}

	d.SetId(nebraskaPackage.Id)
	packageToResource(*nebraskaPackage, d)
	return diags
}

func filterPackageByVersionArch(packages []codegen.Package, version string, arch string) *codegen.Package {

	for _, nebraskaPackage := range packages {
		if nebraskaPackage.Version == version && api.Arch(nebraskaPackage.Arch).String() == arch {
			return &nebraskaPackage
		}
	}
	return nil
}

func flattenFlatcarAction(action *codegen.FlatcarAction) []map[string]interface{} {
	if action == nil {
		return []map[string]interface{}{}
	}

	return []map[string]interface{}{
		{
			"id":                      action.Id,
			"event":                   action.Event,
			"chromeos_version":        action.ChromeOSVersion,
			"sha256":                  action.Sha256,
			"needs_admin":             action.NeedsAdmin,
			"is_delta":                action.IsDelta,
			"disable_payload_backoff": action.DisablePayloadBackoff,
			"metadata_signature_rsa":  action.MetadataSignatureRsa,
			"metadata_size":           action.MetadataSize,
			"deadline":                action.Deadline,
			"created_ts":              action.CreatedTs.String(),
		},
	}
}

func base64Encode(value string) string {
	return base64.StdEncoding.EncodeToString([]byte(value))
}

func base64Decode(value string) (string, error) {
	decodeBytes, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return "", err
	}
	return string(decodeBytes), nil
}

func encodeNUAURL(ghUrl string, commit string, path string, kustomize string) (string, error) {
	finalURL, err := url.Parse(ghUrl)
	if err != nil {
		return "", err
	}
	query := finalURL.Query()
	query.Add("nua_commit", base64Encode(commit))
	query.Add("nua_namespace", base64Encode(path))
	query.Add("nua_kustomize_config", base64Encode(kustomize))
	finalURL.RawQuery = query.Encode()
	return finalURL.String(), nil
}

func decodeNUAURL(encodedURL string) (ghUrl string, commit string, path string, kustomize string, err error) {

	finalURL, err := url.Parse(encodedURL)
	if err != nil {
		return
	}

	encodedCommit := finalURL.Query().Get("nua_commit")
	encodedPath := finalURL.Query().Get("nua_namespace")
	encodedKustomize := finalURL.Query().Get("nua_kustomize_config")

	query := finalURL.Query()
	query.Del("nua_commit")
	query.Del("nua_namespace")
	query.Del("nua_kustomize_config")
	finalURL.RawQuery = query.Encode()

	ghUrl = finalURL.String()

	commit, err = base64Decode(encodedCommit)
	if err != nil {
		return
	}

	path, err = base64Decode(encodedPath)
	if err != nil {
		return
	}

	kustomize, err = base64Decode(encodedKustomize)
	if err != nil {
		return
	}
	return
}
