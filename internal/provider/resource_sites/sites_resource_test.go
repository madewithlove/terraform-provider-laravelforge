package resource_sites

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// An imported site gets its type and in-place settings from Forge's answer, in
// the form the inputs accept, so the first plan after an import changes nothing.
func TestApplyImportedFillsTheInputsFromForge(t *testing.T) {
	d := siteData{ID: "12345", Attributes: map[string]any{
		"app_type":                  "Custom",
		"php_version":               "PHP 8.5",
		"name":                      "example.com",
		"quick_deploy":              true,
		"repository":                map[string]any{"provider": "GitHub", "url": "https://github.com/example/app", "branch": "main"},
		"root_directory":            "/home/forge/example.com",
		"isolated":                  true,
		"user":                      "example",
		"zero_downtime_deployments": false,
	}}
	var m SitesModel
	d.applyImported(&m)

	want := map[string]types.String{"type": types.StringValue("custom"), "php_version": types.StringValue("php85"), "name": types.StringValue("example.com"), "branch": types.StringValue("main"),
		"source_control_provider": types.StringValue("github"), "repository": types.StringValue("example/app"), "isolated_user": types.StringValue("example")}
	got := map[string]types.String{"type": m.Type, "php_version": m.PhpVersion, "name": m.Name, "branch": m.Branch,
		"source_control_provider": m.SourceControlProvider, "repository": m.Repository, "isolated_user": m.IsolatedUser}
	for k, v := range want {
		if !got[k].Equal(v) {
			t.Errorf("%s = %s, want %s", k, got[k], v)
		}
	}
	if !m.PushToDeploy.Equal(types.BoolValue(true)) {
		t.Errorf("push_to_deploy = %s, want true", m.PushToDeploy)
	}
	if !m.IsIsolated.Equal(types.BoolValue(true)) || !m.ZeroDowntimeDeployments.Equal(types.BoolValue(false)) {
		t.Errorf("is_isolated = %s, zero_downtime_deployments = %s, want true and false", m.IsIsolated, m.ZeroDowntimeDeployments)
	}
	if !m.RootDirectory.IsNull() {
		t.Errorf("root_directory = %s, want null: Forge returns an absolute path", m.RootDirectory)
	}
}

func TestSiteTypeAndPhpVersionOnlyTakeWhatTheInputsAccept(t *testing.T) {
	for in, want := range map[string]string{"Laravel": "laravel", "Static HTML": "static-html", "Next.js": "nextjs", "Nuxt.js": "nuxtjs", "WordPress": "wordpress", "Something new": ""} {
		if got := siteType(in); got != want {
			t.Errorf("siteType(%q) = %q, want %q", in, got, want)
		}
	}
	for in, want := range map[string]string{"PHP 8.5": "php85", "PHP 7.4": "php74", "PHP 9.0": "", "": ""} {
		if got := phpVersion(in); got != want {
			t.Errorf("phpVersion(%q) = %q, want %q", in, got, want)
		}
	}
}

// Only a repository on a provider whose input is "owner/name" is read back.
func TestSourceControlOnlyReadsOwnerAndName(t *testing.T) {
	for _, c := range []struct{ provider, url, wantProvider, wantRepository string }{
		{"GitHub", "https://github.com/example/app", "github", "example/app"},
		{"GitLab", "https://gitlab.com/example/app.git", "gitlab", "example/app"},
		{"Bitbucket", "https://bitbucket.org/example/app", "bitbucket", "example/app"},
		{"GitLab", "https://gitlab.com/group/subgroup/app", "", ""},
		{"GitHub", "https://github.example.com/example/app", "", ""},
		{"Custom", "git@example.com:example/app.git", "", ""},
	} {
		provider, repository := sourceControl(map[string]any{"provider": c.provider, "url": c.url})
		if provider != c.wantProvider || repository != c.wantRepository {
			t.Errorf("sourceControl(%s, %s) = %q, %q, want %q, %q", c.provider, c.url, provider, repository, c.wantProvider, c.wantRepository)
		}
	}
}
