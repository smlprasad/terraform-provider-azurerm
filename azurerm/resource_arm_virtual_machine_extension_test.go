package azurerm

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/helpers/tf"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/features"
	computeSvc "github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/services/compute"
)

func TestAccAzureRMVirtualMachineExtension_basicLinux(t *testing.T) {
	resourceName := "azurerm_virtual_machine_extension.test"
	ri := tf.AccRandTimeInt()
	location := testLocation()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMVirtualMachineExtensionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAzureRMVirtualMachineExtension_basicLinux(ri, location),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMVirtualMachineExtensionExists(resourceName),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"protected_settings"},
			},
		},
	})
}

func TestAccAzureRMVirtualMachineExtension_basicWindows(t *testing.T) {
	resourceName := "azurerm_virtual_machine_extension.test"
	ri := tf.AccRandTimeInt()
	location := testLocation()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMVirtualMachineExtensionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAzureRMVirtualMachineExtension_basicWindows(ri, location),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMVirtualMachineExtensionExists(resourceName),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"protected_settings"},
			},
		},
	})
}

func TestAccAzureRMVirtualMachineExtension_requiresImport(t *testing.T) {
	if !features.ShouldResourcesBeImported() {
		t.Skip("Skipping since resources aren't required to be imported")
		return
	}

	resourceName := "azurerm_virtual_machine_extension.test"
	ri := tf.AccRandTimeInt()
	location := testLocation()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMVirtualMachineExtensionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAzureRMVirtualMachineExtension_basicLinux(ri, location),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMVirtualMachineExtensionExists(resourceName),
				),
			},
			{
				Config:      testAccAzureRMVirtualMachineExtension_requiresImport(ri, location),
				ExpectError: testRequiresImportError("azurerm_virtual_machine_extension"),
			},
		},
	})
}

func TestAccAzureRMVirtualMachineExtension_autoUpgradeMinorVersion(t *testing.T) {
	resourceName := "azurerm_virtual_machine_extension.test"
	ri := tf.AccRandTimeInt()
	location := testLocation()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMVirtualMachineExtensionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAzureRMVirtualMachineExtension_autoUpgradeMinorVersion(ri, location, false),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMVirtualMachineExtensionExists(resourceName),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"protected_settings"},
			},
			{
				Config: testAccAzureRMVirtualMachineExtension_autoUpgradeMinorVersion(ri, location, true),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMVirtualMachineExtensionExists(resourceName),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"protected_settings"},
			},
			{
				Config: testAccAzureRMVirtualMachineExtension_autoUpgradeMinorVersion(ri, location, false),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMVirtualMachineExtensionExists(resourceName),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"protected_settings"},
			},
		},
	})
}

func TestAccAzureRMVirtualMachineExtension_concurrent(t *testing.T) {
	firstResourceName := "azurerm_virtual_machine_extension.first"
	secondResourceName := "azurerm_virtual_machine_extension.second"
	ri := tf.AccRandTimeInt()
	location := testLocation()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMVirtualMachineExtensionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAzureRMVirtualMachineExtension_concurrent(ri, location),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMVirtualMachineExtensionExists(firstResourceName),
					testCheckAzureRMVirtualMachineExtensionExists(secondResourceName),
				),
			},
			{
				ResourceName:            firstResourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"protected_settings"},
			},
			{
				ResourceName:            secondResourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"protected_settings"},
			},
		},
	})
}

func TestAccAzureRMVirtualMachineExtension_deprecated(t *testing.T) {
	// TODO: this can be removed in 2.0
	resourceName := "azurerm_virtual_machine_extension.test"
	ri := tf.AccRandTimeInt()
	location := testLocation()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMVirtualMachineExtensionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAzureRMVirtualMachineExtension_deprecated(ri, location),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMVirtualMachineExtensionExists(resourceName),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"protected_settings"},
			},
		},
	})
}

func TestAccAzureRMVirtualMachineExtension_forceUpdateTag(t *testing.T) {
	resourceName := "azurerm_virtual_machine_extension.test"
	ri := tf.AccRandTimeInt()
	location := testLocation()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMVirtualMachineExtensionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAzureRMVirtualMachineExtension_forceUpdateTag(ri, location, "first"),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMVirtualMachineExtensionExists(resourceName),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"force_update_tag",
					"protected_settings",
				},
			},
			{
				Config: testAccAzureRMVirtualMachineExtension_forceUpdateTag(ri, location, "second"),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMVirtualMachineExtensionExists(resourceName),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"force_update_tag",
					"protected_settings",
				},
			},
		},
	})
}

func TestAccAzureRMVirtualMachineExtension_linuxDiagnostics(t *testing.T) {
	resourceName := "azurerm_virtual_machine_extension.test"
	ri := tf.AccRandTimeInt()
	location := testLocation()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMVirtualMachineExtensionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAzureRMVirtualMachineExtension_linuxDiagnostics(ri, location),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMVirtualMachineExtensionExists(resourceName),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"protected_settings"},
			},
		},
	})
}

func TestAccAzureRMVirtualMachineExtension_updated(t *testing.T) {
	resourceName := "azurerm_virtual_machine_extension.test"
	ri := tf.AccRandTimeInt()
	location := testLocation()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMVirtualMachineExtensionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAzureRMVirtualMachineExtension_basicLinux(ri, location),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMVirtualMachineExtensionExists(resourceName),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"protected_settings"},
			},
			{
				Config: testAccAzureRMVirtualMachineExtension_updated(ri, location),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMVirtualMachineExtensionExists(resourceName),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"protected_settings"},
			},
		},
	})
}

func testCheckAzureRMVirtualMachineExtensionExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// Ensure we have enough information in state to look up in API
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("Not found: %s", resourceName)
		}

		id, err := computeSvc.ParseVirtualMachineExtensionID(rs.Primary.ID)
		if err != nil {
			return err
		}

		client := testAccProvider.Meta().(*ArmClient).Compute.VMExtensionClient
		ctx := testAccProvider.Meta().(*ArmClient).StopContext
		resp, err := client.Get(ctx, id.Base.ResourceGroup, id.VirtualMachineName, id.Name, "")
		if err != nil {
			return fmt.Errorf("Bad: Get on vmExtensionClient: %s", err)
		}

		if resp.StatusCode == http.StatusNotFound {
			return fmt.Errorf("Bad: Extension %q (Virtual Machine %q / Resource Group: %q) does not exist", id.Name, id.VirtualMachineName, id.Base.ResourceGroup)
		}

		return nil
	}
}

func testCheckAzureRMVirtualMachineExtensionDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "azurerm_virtual_machine_extension" {
			continue
		}

		id, err := computeSvc.ParseVirtualMachineExtensionID(rs.Primary.ID)
		if err != nil {
			return err
		}

		client := testAccProvider.Meta().(*ArmClient).Compute.VMExtensionClient
		ctx := testAccProvider.Meta().(*ArmClient).StopContext
		resp, err := client.Get(ctx, id.Base.ResourceGroup, id.VirtualMachineName, id.Name, "")
		if err != nil {
			return nil
		}

		if resp.StatusCode != http.StatusNotFound {
			return fmt.Errorf("Virtual Machine Extension still exists:\n%#v", resp.VirtualMachineExtensionProperties)
		}
	}

	return nil
}

func testAccAzureRMVirtualMachineExtension_basicLinux(rInt int, location string) string {
	template := testAccAzureRMVirtualMachineExtension_templateLinux(rInt, location)
	return fmt.Sprintf(`
%s

resource "azurerm_virtual_machine_extension" "test" {
  name                 = "acctvme-%d"
  virtual_machine_id   = azurerm_virtual_machine.test.id
  publisher            = "Microsoft.Azure.Extensions"
  type                 = "CustomScript"
  type_handler_version = "2.0"

  settings = jsonencode({
    "commandToExecute" = "hostname"
  })
}
`, template, rInt)
}

func testAccAzureRMVirtualMachineExtension_basicWindows(rInt int, location string) string {
	template := testAccAzureRMVirtualMachineExtension_templateWindows(rInt, location)
	return fmt.Sprintf(`
%s

resource "azurerm_virtual_machine_extension" "test" {
  name                 = "acctvme-%d"
  virtual_machine_id   = azurerm_virtual_machine.test.id
  publisher            = "Microsoft.Azure.Extensions"
  type                 = "CustomScript"
  type_handler_version = "2.0"

  settings = jsonencode({
    "commandToExecute" = "hostname"
  })
}
`, template, rInt)
}

func testAccAzureRMVirtualMachineExtension_requiresImport(rInt int, location string) string {
	template := testAccAzureRMVirtualMachineExtension_basicLinux(rInt, location)
	return fmt.Sprintf(`
%s

resource "azurerm_virtual_machine_extension" "import" {
  name                 = azurerm_virtual_machine_extension.test.name
  virtual_machine_id   = azurerm_virtual_machine_extension.test.virtual_machine_id
  publisher            = azurerm_virtual_machine_extension.test.publisher
  type                 = azurerm_virtual_machine_extension.test.type
  type_handler_version = azurerm_virtual_machine_extension.test.type_handler_version
  settings             = azurerm_virtual_machine_extension.test.settings
  tags                 = azurerm_virtual_machine_extension.test.tags
}
`, template)
}

func testAccAzureRMVirtualMachineExtension_autoUpgradeMinorVersion(rInt int, location string, enabled bool) string {
	template := testAccAzureRMVirtualMachineExtension_templateLinux(rInt, location)
	return fmt.Sprintf(`
%s

resource "azurerm_virtual_machine_extension" "test" {
  name                       = "acctvme-%d"
  virtual_machine_id         = azurerm_virtual_machine.test.id
  publisher                  = "Microsoft.Azure.Extensions"
  type                       = "CustomScript"
  type_handler_version       = "2.0"
  auto_upgrade_minor_version = %t

  settings = jsonencode({
    "commandToExecute" = "hostname"
  })
}
`, template, rInt, enabled)
}

func testAccAzureRMVirtualMachineExtension_deprecated(rInt int, location string) string {
	template := testAccAzureRMVirtualMachineExtension_templateLinux(rInt, location)
	return fmt.Sprintf(`
%s

resource "azurerm_virtual_machine_extension" "test" {
  name                 = "acctvme-%d"
  location             = azurerm_resource_group.test.location
  resource_group_name  = azurerm_resource_group.test.name
  virtual_machine_name = azurerm_virtual_machine.test.name
  publisher            = "Microsoft.Azure.Extensions"
  type                 = "CustomScript"
  type_handler_version = "2.0"

  settings = jsonencode({
    "commandToExecute" = "hostname"
  })
}
`, template, rInt)
}

func testAccAzureRMVirtualMachineExtension_concurrent(rInt int, location string) string {
	template := testAccAzureRMVirtualMachineExtension_templateLinux(rInt, location)
	return fmt.Sprintf(`
%s

resource "azurerm_virtual_machine_extension" "first" {
  name                 = "acctvme-%d-1"
  virtual_machine_id   = azurerm_virtual_machine.test.id
  publisher            = "Microsoft.Azure.Extensions"
  type                 = "CustomScript"
  type_handler_version = "2.0"

  settings = jsonencode({
    "commandToExecute" = "hostname"
  })
}

resource "azurerm_virtual_machine_extension" "second" {
  name                 = "acctvme-%d-2"
  virtual_machine_id   = azurerm_virtual_machine.test.id
  publisher            = "Microsoft.OSTCExtensions"
  type                 = "CustomScriptForLinux"
  type_handler_version = "1.5"

  settings = jsonencode({
    "commandToExecute" = "whoami"
  })
}
`, template, rInt, rInt)
}

func testAccAzureRMVirtualMachineExtension_forceUpdateTag(rInt int, location, forceUpdateTag string) string {
	template := testAccAzureRMVirtualMachineExtension_templateLinux(rInt, location)
	return fmt.Sprintf(`
%s

resource "azurerm_virtual_machine_extension" "test" {
  name                 = "acctvme-%d"
  virtual_machine_id   = azurerm_virtual_machine.test.id
  publisher            = "Microsoft.Azure.Extensions"
  type                 = "CustomScript"
  type_handler_version = "2.0"
  force_update_tag     = %q

  settings = jsonencode({
    "commandToExecute" = "hostname"
  })
}
`, template, rInt, forceUpdateTag)
}

func testAccAzureRMVirtualMachineExtension_linuxDiagnostics(rInt int, location string) string {
	template := testAccAzureRMVirtualMachineExtension_templateLinux(rInt, location)
	return fmt.Sprintf(`
%s

resource "azurerm_storage_account" "test" {
  name                     = "accsa%d"
  resource_group_name      = "${azurerm_resource_group.test.name}"
  location                 = "${azurerm_resource_group.test.location}"
  account_tier             = "Standard"
  account_replication_type = "LRS"

  tags = {
    environment = "staging"
  }
}

resource "azurerm_storage_container" "test" {
  name                  = "vhds"
  resource_group_name   = "${azurerm_resource_group.test.name}"
  storage_account_name  = "${azurerm_storage_account.test.name}"
  container_access_type = "private"
}

resource "azurerm_virtual_machine_extension" "test" {
  name                 = "acctvme-%d"
  virtual_machine_id   = azurerm_virtual_machine.test.id
  publisher            = "Microsoft.OSTCExtensions"
  type                 = "LinuxDiagnostic"
  type_handler_version = "2.3"

  protected_settings = jsonencode({
    "storageAccountName" = azurerm_storage_account.test.name
    "storageAccountKey"  = azurerm_storage_account.test.primary_access_key
  })
}
`, template, rInt, rInt)
}

func testAccAzureRMVirtualMachineExtension_updated(rInt int, location string) string {
	template := testAccAzureRMVirtualMachineExtension_templateLinux(rInt, location)
	return fmt.Sprintf(`
%s

resource "azurerm_virtual_machine_extension" "test" {
  name                 = "acctvme-%d"
  virtual_machine_id   = azurerm_virtual_machine.test.id
  publisher            = "Microsoft.Azure.Extensions"
  type                 = "CustomScript"
  type_handler_version = "2.0"

  settings = jsonencode({
    "commandToExecute": "whoami"
  })

  tags = {
    environment = "Production"
    cost_center = "MSFT"
  }
}
`, template, rInt)
}

func testAccAzureRMVirtualMachineExtension_templateLinux(rInt int, location string) string {
	// TODO: use the new resource when it becomes available
	return fmt.Sprintf(`
resource "azurerm_resource_group" "test" {
  name     = "acctestrg-%d"
  location = "%s"
}

resource "azurerm_virtual_network" "test" {
  name                = "acctestvn-%d"
  address_space       = ["10.0.0.0/16"]
  location            = azurerm_resource_group.test.location
  resource_group_name = azurerm_resource_group.test.name
}

resource "azurerm_subnet" "test" {
  name                 = "internal"
  resource_group_name  = azurerm_resource_group.test.name
  virtual_network_name = azurerm_virtual_network.test.name
  address_prefix       = "10.0.2.0/24"
}

resource "azurerm_network_interface" "test" {
  name                = "acctestnic-%d"
  location            = azurerm_resource_group.test.location
  resource_group_name = azurerm_resource_group.test.name

  ip_configuration {
    name                          = "testconfiguration1"
    subnet_id                     = azurerm_subnet.test.id
    private_ip_address_allocation = "Dynamic"
  }
}

resource "azurerm_virtual_machine" "test" {
  name                  = "acctestvm-%d"
  location              = azurerm_resource_group.test.location
  resource_group_name   = azurerm_resource_group.test.name
  network_interface_ids = [ azurerm_network_interface.test.id ]
  vm_size               = "Standard_F2"

  storage_image_reference {
    publisher = "Canonical"
    offer     = "UbuntuServer"
    sku       = "16.04-LTS"
    version   = "latest"
  }

  storage_os_disk {
    name              = "myosdisk1"
    caching           = "ReadWrite"
    create_option     = "FromImage"
    managed_disk_type = "Standard_LRS"
  }

  os_profile {
    computer_name  = "hostname"
    admin_username = "testadmin"
    admin_password = "Password1234!"
  }
  
  os_profile_linux_config {
    disable_password_authentication = false
  }
}
`, rInt, location, rInt, rInt, rInt)
}

func testAccAzureRMVirtualMachineExtension_templateWindows(rInt int, location string) string {
	// TODO: use the new resource when it becomes available
	return fmt.Sprintf(`
resource "azurerm_resource_group" "test" {
  name     = "acctestrg-%d"
  location = "%s"
}

resource "azurerm_virtual_network" "test" {
  name                = "acctestvn-%d"
  address_space       = ["10.0.0.0/16"]
  location            = azurerm_resource_group.test.location
  resource_group_name = azurerm_resource_group.test.name
}

resource "azurerm_subnet" "test" {
  name                 = "internal"
  resource_group_name  = azurerm_resource_group.test.name
  virtual_network_name = azurerm_virtual_network.test.name
  address_prefix       = "10.0.2.0/24"
}

resource "azurerm_network_interface" "test" {
  name                = "acctestnic-%d"
  location            = azurerm_resource_group.test.location
  resource_group_name = azurerm_resource_group.test.name

  ip_configuration {
    name                          = "testconfiguration1"
    subnet_id                     = azurerm_subnet.test.id
    private_ip_address_allocation = "Dynamic"
  }
}

resource "azurerm_virtual_machine" "test" {
  name                  = "acctestvm-%d"
  location              = azurerm_resource_group.test.location
  resource_group_name   = azurerm_resource_group.test.name
  network_interface_ids = [ azurerm_network_interface.test.id ]
  vm_size               = "Standard_F2"

  storage_image_reference {
    publisher = "MicrosoftWindowsServer"
    offer     = "WindowsServer"
    sku       = "2016-Datacenter"
    version   = "latest"
  }

  storage_os_disk {
    name              = "myosdisk1"
    caching           = "ReadWrite"
    create_option     = "FromImage"
    managed_disk_type = "Standard_LRS"
  }

  os_profile {
    computer_name  = "hostname"
    admin_username = "testadmin"
    admin_password = "Password1234!"
  }
  
  os_profile_windows_config {
    provision_vm_agent = true
  }
}
`, rInt, location, rInt, rInt, rInt)
}
