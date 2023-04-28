package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccServerDataSource(t *testing.T) {
	rnd := generateRandomResourceName()
	name := "data.forge_server." + rnd

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: testAccServerDataSourceConfig(rnd),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(name, "id", "123"),
				),
			},
		},
	})
}

func testAccServerDataSourceConfig(resourceName string) string {
	return fmt.Sprintf(`
data "forge_server" "%[1]s" {
	id = "123"
}
	`, resourceName)
}
