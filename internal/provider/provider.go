package provider

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/kinvolk/nebraska/backend/pkg/codegen"
)

func init() {
	// Set descriptions to support markdown syntax, this will be used in document generation
	// and the language server.
	schema.DescriptionKind = schema.StringMarkdown

	// Customize the content of descriptions when output. For example you can add defaults on
	// to the exported descriptions if present.
	schema.SchemaDescriptionBuilder = func(s *schema.Schema) string {
		desc := s.Description
		if s.Default != nil {
			desc += fmt.Sprintf(" Defaults to `%v`.", s.Default)
		}
		return strings.TrimSpace(desc)
	}
}

func New(version string) func() *schema.Provider {
	return func() *schema.Provider {
		p := &schema.Provider{
			Schema: map[string]*schema.Schema{
				"endpoint": {
					Type:         schema.TypeString,
					Optional:     true,
					DefaultFunc:  schema.EnvDefaultFunc("NEBRASKA_ENDPOINT", "http://localhost:8000"),
					ValidateFunc: validation.IsURLWithHTTPorHTTPS,
					Description:  "The address of Nebraska server. Can be configured using the env variable `NEBRASKA_ENDPOINT`, if not provided defaults to `http://localhost:8000`.",
				},
				"auth_mode": {
					Type:         schema.TypeString,
					Optional:     true,
					DefaultFunc:  schema.EnvDefaultFunc("NEBRASKA_AUTH_MODE", "noop"),
					ValidateFunc: validation.StringInSlice([]string{"noop", "github", "oidc"}, false),
					Description:  "The auth_mode of Nebraska server. Can be configured using the env variable `NEBRASKA_AUTH_MODE`, if not provided defaults to `noop`.",
				},
				"github_token": {
					Type:        schema.TypeString,
					Optional:    true,
					DefaultFunc: schema.EnvDefaultFunc("NEBRASKA_GH_TOKEN", ""),
					Description: "The github_token used to authenticate when the auth_mode is `github`. Can be configured using the env variable `NEBRASKA_GH_TOKEN`",
				},
				"username": {
					Type:        schema.TypeString,
					Optional:    true,
					DefaultFunc: schema.EnvDefaultFunc("NEBRASKA_USERNAME", ""),
					Description: "The username used to authenticate when the auth_mode is `oidc`. Can be configured using the env variable `NEBRASKA_USERNAME` ",
				},
				"password": {
					Type:        schema.TypeString,
					Optional:    true,
					DefaultFunc: schema.EnvDefaultFunc("NEBRASKA_PASSWORD", ""),
					Description: "The password used to authenticate when the auth_mode is `oidc`. Can be configured using the env variable `NEBRASKA_PASSWORD` ",
				},
			},
			DataSourcesMap: map[string]*schema.Resource{
				"nebraska_application": dataSourceApplication(),
				"nebraska_group":       dataSourceGroup(),
				"nebraska_channel":     dataSourceChannel(),
				"nebraska_package":     dataSourcePackage(),
			},
			ResourcesMap: map[string]*schema.Resource{
				"nebraska_application": resourceApplication(),
				"nebraska_channel":     resourceChannel(),
				"nebraska_group":       resourceGroup(),
				"nebraska_package":     resourcePackage(),
			},
		}

		p.ConfigureContextFunc = configure(version, p)

		return p
	}
}

type apiClient struct {
	authMode   string
	reqEditors []codegen.RequestEditorFn
	client     *codegen.ClientWithResponses
}

func configure(version string, p *schema.Provider) func(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
	return func(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {

		var diags diag.Diagnostics
		endpoint := d.Get("endpoint").(string)
		authMode := d.Get("auth_mode").(string)

		// setup client
		client, err := codegen.NewClientWithResponses(endpoint)
		if err != nil {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  "Client init",
				Detail:   fmt.Sprintf("Couldn't initialise client:%v", err),
			})
			return nil, diags
		}

		// fetch server config
		var configRequestEditor []codegen.RequestEditorFn
		if authMode == "github" {
			token := d.Get("github_token").(string)
			if token == "" {
				diags = append(diags, diag.Diagnostic{
					Severity: diag.Error,
					Summary:  "Github Token empty",
					Detail:   "gitub_token is required for github auth_mode",
				})
				return nil, diags
			}
			configRequestEditor = append(configRequestEditor, newBearerTokenRequestEditor(token))
		}

		resp, err := client.GetConfigWithResponse(ctx, configRequestEditor...)
		if err != nil {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  "Config fetch",
				Detail:   fmt.Sprintf("Couldn't fetch the nebraska server config: %v", err),
			})
			return nil, diags
		}

		if authMode != resp.JSON200.AuthMode {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  "Invalid auth_mode",
				Detail:   fmt.Sprintf("The Nebraska server %s supports %s and doesn't support %s auth_mode", endpoint, resp.JSON200.AuthMode, authMode),
			})
			return nil, diags
		}

		apiClient := &apiClient{
			authMode: authMode,
			client:   client,
		}

		if authMode == "noop" {
			return apiClient, diags
		}

		if authMode == "github" {
			token := d.Get("github_token").(string)
			if token == "" {
				diags = append(diags, diag.Diagnostic{
					Severity: diag.Error,
					Summary:  "Github Token empty",
					Detail:   "gitub_token is required for github auth_mode",
				})
				return nil, diags
			}
			apiClient.reqEditors = []codegen.RequestEditorFn{
				newBearerTokenRequestEditor(token),
			}
		}

		if authMode == "oidc" {
			username := d.Get("username").(string)
			password := d.Get("password").(string)

			if username == "" || password == "" {
				diags = append(diags, diag.Diagnostic{
					Severity: diag.Error,
					Summary:  "Username Password empty",
					Detail:   "username, password are required for oidc auth_mode",
				})
				return nil, diags
			}

			requestBody := fmt.Sprintf(`username=%s&password=%s`, url.QueryEscape(username), url.QueryEscape(password))
			loginTokenResp, err := client.LoginTokenWithBodyWithResponse(ctx, "application/x-www-form-urlencoded", strings.NewReader(requestBody))
			if err != nil {
				diags = append(diags, diag.Diagnostic{
					Severity: diag.Error,
					Summary:  "Couldn't fetch login token",
					Detail:   fmt.Sprintf("Login token error failed: %v", err),
				})
			}
			if loginTokenResp.JSON200 == nil {
				diags = append(diags, diag.Diagnostic{
					Severity: diag.Error,
					Summary:  "Couldn't fetch login token",
					Detail:   fmt.Sprintf("Got non 200 status code: %v", loginTokenResp.StatusCode()),
				})
			}
			cookie := loginTokenResp.HTTPResponse.Header.Get("Set-Cookie")
			apiClient.reqEditors = []codegen.RequestEditorFn{
				newBearerTokenRequestEditor(loginTokenResp.JSON200.Token),
				func(ctx context.Context, req *http.Request) error {
					req.Header.Add("Cookie", cookie)
					return nil
				},
			}
		}

		return apiClient, diags
	}
}

func newBearerTokenRequestEditor(token string) codegen.RequestEditorFn {
	if token == "" {
		return nil
	}
	return func(ctx context.Context, req *http.Request) error {
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
		return nil
	}
}
