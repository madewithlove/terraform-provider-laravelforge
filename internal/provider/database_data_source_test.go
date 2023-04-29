package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDatabaseDataSource(t *testing.T) {
	rnd := generateRandomResourceName()
	name := "data.forge_database." + rnd

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: testAccDatabaseDataSourceConfig(rnd),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(name, "id", "123"),
					resource.TestCheckResourceAttr(name, "server_id", "123"),
				),
			},
		},
	})
}

func testAccDatabaseDataSourceConfig(resourceName string) string {
	return fmt.Sprintf(`
data "forge_database" "%[1]s" {
	id = "123"
	server_id = "123"
}
	`, resourceName)
}
