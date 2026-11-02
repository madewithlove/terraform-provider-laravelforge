package provider

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

// forgeSite is a site as Forge's organization API returns it.
const forgeSite = `{"data": {"id": "2505182", "type": "sites", "attributes": {
  "name": "report.example.com",
  "status": "installed",
  "url": "https://report.example.com",
  "user": "forge",
  "https": true,
  "web_directory": "/home/forge/report.example.com/public",
  "root_directory": "/home/forge/report.example.com",
  "aliases": [],
  "php_version": "PHP 8.5",
  "deployment_url": "https://forge.laravel.com/servers/292302/sites/2505182/deploy/http?token=secret",
  "quick_deploy": true,
  "isolated": false,
  "shared_paths": [],
  "repository": {"provider": "GitHub", "url": "https://github.com/example/report", "branch": "main", "status": "installed"},
  "database": "",
  "maintenance_mode": {"enabled": false, "status": null},
  "zero_downtime_deployments": false,
  "wildcards": false,
  "app_type": "Custom",
  "uses_envoyer": false,
  "created_at": "2024-10-21T07:59:09.000000Z",
  "updated_at": "2026-11-02T11:14:39.000000Z"
}}}`

// An imported site plans no change with the configuration Terraform generates
// for it, and changing a setting the update endpoint takes updates the site in
// place: the create-only settings the configuration leaves unset don't force a
// replacement.
func TestSitesImportPlansNoReplacement(t *testing.T) {
	forge := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method + " " + r.URL.Path {
		case "GET /orgs/example/sites/2505182":
			_, _ = w.Write([]byte(forgeSite))
		case "PUT /orgs/example/servers/292302/sites/2505182", "DELETE /orgs/example/servers/292302/sites/2505182":
			w.WriteHeader(http.StatusAccepted)
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer forge.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:             testSitesImportConfig(forge.URL, "custom"),
				ResourceName:       "laravelforge_sites.report",
				ImportState:        true,
				ImportStateId:      "example/292302/2505182",
				ImportStatePersist: true,
			},
			{
				Config:   testSitesImportConfig(forge.URL, "custom"),
				PlanOnly: true,
			},
			{
				Config: testSitesImportConfig(forge.URL, "laravel"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("laravelforge_sites.report", plancheck.ResourceActionUpdate),
					},
				},
			},
		},
	})
}

// testSitesImportConfig is the configuration `terraform plan
// -generate-config-out` writes for forgeSite, with the given type.
func testSitesImportConfig(endpoint, siteType string) string {
	return fmt.Sprintf(`
provider "laravelforge" {
  endpoint  = %[1]q
  api_token = "test"
}

resource "laravelforge_sites" "report" {
  branch                    = "main"
  is_isolated               = false
  name                      = "report.example.com"
  organization              = "example"
  php_version               = "php85"
  push_to_deploy            = true
  repository                = "example/report"
  server                    = "292302"
  site                      = 2505182
  source_control_provider   = "github"
  type                      = %[2]q
  zero_downtime_deployments = false
}
`, endpoint, siteType)
}
