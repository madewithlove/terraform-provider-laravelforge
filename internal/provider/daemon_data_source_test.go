package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDaemonDataSource(t *testing.T) {
	rnd := generateRandomResourceName()
	name := "data.forge_daemon." + rnd

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: testAccDaemonDataSourceConfig(rnd),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(name, "id", "123"),
					resource.TestCheckResourceAttr(name, "server_id", "123"),
				),
			},
		},
	})
}

func testAccDaemonDataSourceConfig(resourceName string) string {
	return fmt.Sprintf(`
data "forge_daemon" "%[1]s" {
	id = "123"
	server_id = "123"
}
	`, resourceName)
}
