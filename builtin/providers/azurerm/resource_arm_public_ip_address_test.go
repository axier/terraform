package azurerm

import (
	"fmt"
	"testing"

	"github.com/Azure/azure-sdk-for-go/core/http"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAzureRMPublicIPAddress_basic(t *testing.T) {
	name := "azurerm_public_ip_address.test"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMPublicIPAddressDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAzureRMPublicIPAddressConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMPublicIPAddressExists(name),
					resource.TestCheckResourceAttr(name, "public_ip_allocation_method", "Dynamic"),
				),
			},
		},
	})
}

func testCheckAzureRMPublicIPAddressExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// first check within the schema for the local network gateway:
		res, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Public ip address '%s' not found.", name)
		}

		// then, extract the name and the resource group:
		id, err := parseAzureResourceID(res.Primary.ID)
		if err != nil {
			return err
		}
		publicIPName := id.Path["publicIPAddresses"]
		resGrp := id.ResourceGroup

		// and finally, check that it exists on Azure:
		publicIpClient := testAccProvider.Meta().(*ArmClient).publicIPClient

		resp, err := publicIpClient.Get(resGrp, publicIPName)
		if err != nil {
			if resp.StatusCode == http.StatusNotFound {
				return fmt.Errorf("Public ip address '%s' (resource group '%s') does not exist on Azure.", publicIPName, resGrp)
			}

			return fmt.Errorf("Error reading the state of public ip address '%s'.", publicIPName)
		}

		return nil
	}
}

func testCheckAzureRMPublicIPAddressDestroy(s *terraform.State) error {
	for _, res := range s.RootModule().Resources {
		if res.Type != "azurerm_public_ip_address" {
			continue
		}

		id, err := parseAzureResourceID(res.Primary.ID)
		if err != nil {
			return err
		}
		publicIPName := id.Path["publicIPAddresses"]
		resGrp := id.ResourceGroup

		publicIpClient := testAccProvider.Meta().(*ArmClient).publicIPClient
		resp, err := publicIpClient.Get(resGrp, publicIPName)

		if err != nil {
			return nil
		}

		if resp.StatusCode != http.StatusNotFound {
			return fmt.Errorf("Public ip address still exists:\n%#v", resp.Properties)
		}
	}

	return nil
}

var testAccAzureRMPublicIPAddressConfig_basic = `
resource "azurerm_resource_group" "test" {
    name = "tftestingResourceGroup"
    location = "West US"
}

resource "azurerm_public_ip_network" "test" {
    name = "tftestingPublicIPAddress"
    location = "${azurerm_resource_group.test.location}"
    resource_group_name = "${azurerm_resource_group.test.name}"
    public_ip_allocation_method = "Dynamic"
}
`
