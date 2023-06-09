package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccServerResource(t *testing.T) {
	rnd := generateRandomResourceName()
	name := "laravelforge_server." + rnd

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testServerConfig(rnd, "ocean2"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(name, "platform", "ocean2"),
					resource.TestCheckResourceAttr(name, "id", "123"),
				),
			},
		},
	})
}

func testServerConfig(resourceName string, platformName string) string {
	return fmt.Sprintf(`
resource "laravelforge_server" "%[1]s" {
  platform = %[2]q
  credential_id = 1
}
`, resourceName, platformName)
}
