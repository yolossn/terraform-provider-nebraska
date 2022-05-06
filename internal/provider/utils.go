package provider

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/kinvolk/nebraska/backend/pkg/api"
	"github.com/kinvolk/nebraska/backend/pkg/codegen"
)

func invalidResponseCodeDiag(step string, resp *http.Response) diag.Diagnostic {
	respBody, _ := ioutil.ReadAll(resp.Body)
	return diag.Diagnostic{
		Severity: diag.Error,
		Summary:  step,
		Detail:   fmt.Sprintf("Got invalid response code:%d\n resp:%s", resp.StatusCode, string(respBody)),
	}
}

func keyToStringPointer(d *schema.ResourceData, key string) *string {
	if v, ok := d.Get(key).(string); ok {
		return &v
	}
	return nil
}

func keyToBoolPointer(d *schema.ResourceData, key string) *bool {
	if v, ok := d.Get(key).(bool); ok {
		return &v
	}
	return nil
}

func arrInterfaceToarrString(interfaces []interface{}) []string {
	strings := []string{}
	for _, i := range interfaces {
		if v, ok := i.(string); ok {
			strings = append(strings, v)
		}
	}
	return strings
}

// app
func resourceToAppConfig(d *schema.ResourceData) (*codegen.AppConfig, error) {

	return &codegen.AppConfig{
		Name:        d.Get("name").(string),
		Description: keyToStringPointer(d, "description"),
		ProductId:   keyToStringPointer(d, "product_id"),
	}, nil
}

func appToResourceData(app codegen.Application, d *schema.ResourceData) {
	d.Set("created_ts", app.CreatedTs.String())
	d.Set("description", app.Description)
	d.Set("name", app.Name)
	d.Set("product_id", app.ProductId)
}

// channel

func resourceToChannelConfig(d *schema.ResourceData) (*codegen.ChannelConfig, error) {

	arch, err := api.ArchFromString(d.Get("arch").(string))
	if err != nil {
		return nil, err
	}

	return &codegen.ChannelConfig{
		Name:          d.Get("name").(string),
		Color:         d.Get("color").(string),
		PackageId:     keyToStringPointer(d, "package_id"),
		Arch:          uint(arch),
		ApplicationId: d.Get("application_id").(string),
	}, nil
}

func channelToResourceData(channel codegen.Channel, d *schema.ResourceData) {
	d.Set("name", channel.Name)
	d.Set("arch", api.Arch(channel.Arch).String())
	d.Set("application_id", channel.ApplicationID)
	d.Set("color", channel.Color)
	d.Set("created_ts", channel.CreatedTs.String())
	d.Set("package_id", channel.PackageID)
}

// group

func resourceToGroupConfig(d *schema.ResourceData) *codegen.GroupConfig {
	return &codegen.GroupConfig{
		Name:                      d.Get("name").(string),
		PolicyMaxUpdatesPerPeriod: d.Get("policy_max_updates_per_period").(int),
		PolicyPeriodInterval:      d.Get("policy_period_interval").(string),
		PolicyTimezone:            d.Get("policy_timezone").(string),
		PolicyUpdateTimeout:       d.Get("policy_update_timeout").(string),
		ChannelId:                 keyToStringPointer(d, "channel_id"),
		Description:               keyToStringPointer(d, "description"),
		Track:                     keyToStringPointer(d, "track"),
		PolicyOfficeHours:         keyToBoolPointer(d, "policy_office_hours"),
		PolicySafeMode:            keyToBoolPointer(d, "policy_safe_mode"),
		PolicyUpdatesEnabled:      keyToBoolPointer(d, "policy_updates_enabled"),
	}
}

func groupToResourceData(group codegen.Group, d *schema.ResourceData) {
	d.Set("name", group.Name)
	d.Set("description", group.Description)
	d.Set("created_ts", group.CreatedTs.String())
	d.Set("rollout_in_progress", group.RolloutInProgress)
	d.Set("channel_id", group.ChannelID)
	d.Set("policy_updates_enabled", group.PolicyUpdatesEnabled)
	d.Set("policy_safe_mode", group.PolicySafeMode)
	d.Set("policy_office_hours", group.PolicyOfficeHours)
	d.Set("policy_timezone", group.PolicyTimezone)
	d.Set("policy_period_interval", group.PolicyPeriodInterval)
	d.Set("policy_max_updates_per_period", group.PolicyMaxUpdatesPerPeriod)
	d.Set("policy_update_timeout", group.PolicyUpdateTimeout)
	d.Set("track", group.Track)

}

// package

func resourceToPackageConfig(d *schema.ResourceData) (*codegen.PackageConfig, error) {

	version, _ := d.GetOk("version")
	packageURL := d.Get("url").(string)
	packageType := d.Get("type").(string)

	if packageType == "git" {
		fmt.Println("encoding url")
		nuaCommit, ok := d.Get("nua_commit").(string)
		if !ok {
			return nil, errors.New("'nua_commit' is required for package type 'git'")
		}
		nuaNamespace, ok := d.Get("nua_namespace").(string)
		if !ok {
			return nil, errors.New("'nua_namespace' is required for package type 'git'")
		}
		nuaKustomizeConfig := d.Get("nua_kustomize_config").(string)
		if !ok {
			return nil, errors.New("'nua_kustomize_config' is required for package type 'git'")
		}
		encodedURL, err := encodeNUAURL(packageURL, nuaCommit, nuaNamespace, nuaKustomizeConfig)
		if err != nil {
			return nil, err
		}
		fmt.Println("encoded url", encodedURL)
		d.Set("url", packageURL)
		d.Set("type", "other")
		packageURL = encodedURL
		packageType = "other"
	}

	arch, err := api.ArchFromString(d.Get("arch").(string))
	if err != nil {
		return nil, err
	}
	pkgType, err := PackageTypeFromString(packageType)
	if err != nil {
		return nil, err
	}

	flatcarSha := expandFlatcarActionSha256(d.Get("flatcar_action").([]interface{}))
	return &codegen.PackageConfig{
		ApplicationId:     d.Get("application_id").(string),
		Arch:              int(arch),
		ChannelsBlacklist: arrInterfaceToarrString(d.Get("channels_blacklist").([]interface{})),
		Description:       d.Get("description").(string),
		Filename:          d.Get("filename").(string),
		Hash:              d.Get("hash").(string),
		Size:              d.Get("size").(string),
		Type:              int(pkgType),
		Url:               packageURL,
		Version:           version.(string),
		FlatcarAction: &codegen.FlatcarActionPackage{
			Sha256: &flatcarSha,
		},
	}, nil
}

func packageToResource(nebraskaPackage codegen.Package, d *schema.ResourceData) error {

	if strings.Contains(nebraskaPackage.Url, "nua_commit") || strings.Contains(nebraskaPackage.Url, "nua_namespace") || strings.Contains(nebraskaPackage.Url, "nua_kustomize_config") {
		ghUrl, commit, path, kustomize, err := decodeNUAURL(nebraskaPackage.Url)
		if err != nil {
			return err
		}
		d.Set("nua_commit", commit)
		d.Set("nua_namespace", path)
		d.Set("nua_kustomize_config", kustomize)
		nebraskaPackage.Url = ghUrl
		d.Set("type", "git")
		fmt.Println("set type to git")
	} else {
		d.Set("type", pkgTypeToString[nebraskaPackage.Type])
	}
	d.SetId(nebraskaPackage.Id)
	d.Set("url", nebraskaPackage.Url)
	d.Set("filename", nebraskaPackage.Filename)
	d.Set("description", nebraskaPackage.Description)
	d.Set("size", nebraskaPackage.Size)
	d.Set("hash", nebraskaPackage.Hash)
	d.Set("created_ts", nebraskaPackage.CreatedTs.String())
	if nebraskaPackage.ChannelsBlacklist == nil {
		nebraskaPackage.ChannelsBlacklist = []string{}
	}
	d.Set("channels_blacklist", nebraskaPackage.ChannelsBlacklist)
	d.Set("flatcar_action", flattenFlatcarAction(nebraskaPackage.FlatcarAction))
	d.Set("version", nebraskaPackage.Version)
	return nil
}
