package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDeploymentDataSource(t *testing.T) {
	rnd := generateRandomResourceName()
	name := "data.forge_deployment." + rnd

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: testAccDeploymentDataSourceConfig(rnd),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(name, "id", "123"),
					resource.TestCheckResourceAttr(name, "site_id", "123"),
					resource.TestCheckResourceAttr(name, "server_id", "123"),
				),
			},
		},
	})
}

func testAccDeploymentDataSourceConfig(resourceName string) string {
	return fmt.Sprintf(`
data "forge_deployment" "%[1]s" {
	id = "123"
	site_id = "123"
	server_id = "123"
}
	`, resourceName)
}
