package azurerm

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/Azure/azure-sdk-for-go/arm/network"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceArmPublicIPAddress() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmPublicIPAddressCreate,
		Read:   resourceArmPublicIPAddressRead,
		Update: resourceArmPublicIPAddressCreate,
		Delete: resourceArmPublicIPAddressDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"public_ip_allocation_method": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"location": &schema.Schema{
				Type:      schema.TypeString,
				Required:  true,
				ForceNew:  true,
				StateFunc: azureRMNormalizeLocation,
			},

			"resource_group_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}
func resourceArmPublicIPAddressCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient)
	publicIPClient := client.publicIPClient

	log.Printf("[INFO] preparing arguments for Azure ARM public ip address creation.")

	name := d.Get("name").(string)
	location := d.Get("location").(string)
	resGroup := d.Get("resource_group_name").(string)

	publicIP := network.PublicIPAddress{
		Name:       &name,
		Location:   &location,
		Properties: getPublicIPAddressProperties(d),
	}

	resp, err := publicIPClient.CreateOrUpdate(resGroup, name, publicIP)
	if err != nil {
		return err
	}

	d.SetId(*resp.ID)

	log.Printf("[DEBUG] Waiting for Public IP Address (%s) to become available", name)
	stateConf := &resource.StateChangeConf{
		Pending: []string{"Accepted", "Updating"},
		Target:  "Succeeded",
		Refresh: publicIPAddressStateRefreshFunc(client, resGroup, name),
		Timeout: 10 * time.Minute,
	}
	if _, err := stateConf.WaitForState(); err != nil {
		return fmt.Errorf("Error waiting for Public IP Address (%s) to become available: %s", name, err)
	}

	return resourceArmPublicIPAddressRead(d, meta)
}
func resourceArmPublicIPAddressRead(d *schema.ResourceData, meta interface{}) error {
	publicIpClient := meta.(*ArmClient).publicIPClient

	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return err
	}
	resGroup := id.ResourceGroup
	name := id.Path["publicIPAddresses"]

	resp, err := publicIpClient.Get(resGroup, name)
	if resp.StatusCode == http.StatusNotFound {
		d.SetId("")
		return nil
	}
	if err != nil {
		return fmt.Errorf("Error making Read request on Azure public ip address %s: %s", name, err)
	}
	publicIp := *resp.Properties

	// update appropriate values
	d.Set("public_ip_allocation_method", publicIp.PublicIPAllocationMethod)

	return nil
}
func resourceArmPublicIPAddressDelete(d *schema.ResourceData, meta interface{}) error {
	publicIpClient := meta.(*ArmClient).publicIPClient

	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return err
	}
	resGroup := id.ResourceGroup
	name := id.Path["publicIPAddresses"]

	_, err = publicIpClient.Delete(resGroup, name)

	return err
}
func getPublicIPAddressProperties(d *schema.ResourceData) *network.PublicIPAddressPropertiesFormat {
	// first; get public ip allocation method:
	publicIPAllocationMethod := d.Get("public_ip_allocation_method").(string)
	var allocationMethod network.IPAllocationMethod

	switch publicIPAllocationMethod {
	case "Dynamic":
		allocationMethod = network.Dynamic
	case "Static":
		allocationMethod = network.Static
	default:
		allocationMethod = network.Dynamic
	}

	// finally; return the struct:
	return &network.PublicIPAddressPropertiesFormat{
		PublicIPAllocationMethod: allocationMethod,
	}
}
func publicIPAddressStateRefreshFunc(client *ArmClient, resourceGroupName string, publicIPAddressName string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		res, err := client.publicIPClient.Get(resourceGroupName, publicIPAddressName)
		if err != nil {
			return nil, "", fmt.Errorf("Error issuing read request in publicIPAddessStateRefreshFunc to Azure ARM for public ip address '%s' (RG: '%s'): %s", publicIPAddressName, resourceGroupName, err)
		}

		return res, *res.Properties.ProvisioningState, nil
	}
}
