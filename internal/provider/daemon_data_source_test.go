package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDaemonDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: testAccDaemonDataSourceConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.forge_daemon.test", "id", "48122"),
					resource.TestCheckResourceAttr("data.forge_daemon.test", "server_id", "230361"),
				),
			},
		},
	})
}

const testAccDaemonDataSourceConfig = `
data "forge_daemon" "test" {
  id = 48122
  server_id = 230361
}
`
