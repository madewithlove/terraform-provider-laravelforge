package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// protoV6ProviderFactories wires the provider for acceptance tests.
var protoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"laravelforge": providerserver.NewProtocol6WithError(New("test")()),
}

// testAccPreCheck verifies the environment is configured for live tests.
func testAccPreCheck(t *testing.T) {
	if os.Getenv("FORGE_API_TOKEN") == "" {
		t.Fatal("FORGE_API_TOKEN must be set for acceptance tests")
	}
	if os.Getenv("FORGE_ORGANIZATION") == "" {
		t.Fatal("FORGE_ORGANIZATION must be set for acceptance tests")
	}
}

// TestAccRecipesResource exercises the full create/update lifecycle of the
// recipes resource against the live Forge API. It runs only when TF_ACC=1.
func TestAccRecipesResource(t *testing.T) {
	org := os.Getenv("FORGE_ORGANIZATION")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRecipeConfig(org, "echo one"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("laravelforge_recipes.test", "id"),
					resource.TestCheckResourceAttr("laravelforge_recipes.test", "script", "echo one"),
				),
			},
			{
				Config: testAccRecipeConfig(org, "echo two"),
				Check: resource.TestCheckResourceAttr(
					"laravelforge_recipes.test", "script", "echo two"),
			},
		},
	})
}

func testAccRecipeConfig(org, script string) string {
	return fmt.Sprintf(`
resource "laravelforge_recipes" "test" {
  organization = %[1]q
  name         = "tf-acc-recipes-test"
  user         = "forge"
  script       = %[2]q
}
`, org, script)
}
