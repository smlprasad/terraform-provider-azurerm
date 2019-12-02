package azurerm

import (
	"fmt"
	"log"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-07-01/compute"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/structure"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/helpers/azure"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/helpers/tf"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/features"
	computeSvc "github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/services/compute"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/tags"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/timeouts"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/utils"
)

// TODO: confirm that `publisher` and `type` can be changed without recreation

func resourceArmVirtualMachineExtension() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmVirtualMachineExtensionCreateUpdate,
		Read:   resourceArmVirtualMachineExtensionRead,
		Update: resourceArmVirtualMachineExtensionCreateUpdate,
		Delete: resourceArmVirtualMachineExtensionDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(30 * time.Minute),
			Read:   schema.DefaultTimeout(5 * time.Minute),
			Update: schema.DefaultTimeout(30 * time.Minute),
			Delete: schema.DefaultTimeout(30 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				// TODO: validation
			},

			"virtual_machine_id": {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				ValidateFunc: computeSvc.ValidateVirtualMachineID,

				// TODO: remove in 2.0
				Computed: true,
			},

			"publisher": {
				Type:     schema.TypeString,
				Required: true,
			},

			"type": {
				Type:     schema.TypeString,
				Required: true,
			},

			"type_handler_version": {
				Type:     schema.TypeString,
				Required: true,
			},

			"auto_upgrade_minor_version": {
				Type:     schema.TypeBool,
				Optional: true,
			},

			"force_update_tag": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"settings": {
				Type:             schema.TypeString,
				Optional:         true,
				ValidateFunc:     validation.ValidateJsonString,
				DiffSuppressFunc: structure.SuppressJsonDiff,
			},

			// due to the sensitive nature, these are not returned by the API
			"protected_settings": {
				Type:             schema.TypeString,
				Optional:         true,
				Sensitive:        true,
				ValidateFunc:     validation.ValidateJsonString,
				DiffSuppressFunc: structure.SuppressJsonDiff,
			},

			"tags": tags.Schema(),

			// TODO: deprecate these in the docs & remove in 2.0
			"location":            azure.SchemaLocationDeprecated(),
			"resource_group_name": azure.SchemaResourceGroupNameDeprecated(),
			"virtual_machine_name": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},
		},
	}
}

func resourceArmVirtualMachineExtensionCreateUpdate(d *schema.ResourceData, meta interface{}) error {
	vmsClient := meta.(*ArmClient).Compute.VMClient
	extensionsClient := meta.(*ArmClient).Compute.VMExtensionClient
	ctx, cancel := timeouts.ForCreateUpdate(meta.(*ArmClient).StopContext, d)
	defer cancel()

	name := d.Get("name").(string)

	location := d.Get("location").(string)
	resourceGroup := d.Get("resource_group_name").(string)
	virtualMachineName := d.Get("virtual_machine_name").(string)
	virtualMachineId := d.Get("virtual_machine_id").(string)
	if virtualMachineId != "" {
		vmId, err := computeSvc.ParseVirtualMachineID(virtualMachineId)
		if err != nil {
			return err
		}

		resourceGroup = vmId.Base.ResourceGroup
		virtualMachineName = vmId.Name

		virtualMachine, err := vmsClient.Get(ctx, resourceGroup, virtualMachineName, "")
		if err != nil {
			return fmt.Errorf("Error retrieving Virtual Machine %q (Resource Group %q): %+v", name, resourceGroup, err)
		}
		if virtualMachine.Location == nil {
			return fmt.Errorf("Error retrieving Virtual Machine %q (Resource Group %q): `location` was nil", name, resourceGroup)
		}

		location = *virtualMachine.Location
	} else {
		if location == "" {
			return fmt.Errorf("`location` must be specified if `virtual_machine_id` isn't configured!")
		}

		if resourceGroup == "" {
			return fmt.Errorf("`resource_group_name` must be specified if `virtual_machine_id` isn't configured!")
		}

		if virtualMachineName == "" {
			return fmt.Errorf("`virtual_machine_name` must be specified if `virtual_machine_id` isn't configured!")
		}
	}

	if features.ShouldResourcesBeImported() && d.IsNewResource() {
		existing, err := extensionsClient.Get(ctx, resourceGroup, virtualMachineName, name, "")
		if err != nil {
			if !utils.ResponseWasNotFound(existing.Response) {
				return fmt.Errorf("Error checking for presence of existing Extension %q (Virtual Machine %q / Resource Group %q): %s", name, virtualMachineName, resourceGroup, err)
			}
		}

		if existing.ID != nil && *existing.ID != "" {
			return tf.ImportAsExistsError("azurerm_virtual_machine_extension", *existing.ID)
		}
	}

	location = azure.NormalizeLocation(location)
	autoUpgradeMinor := d.Get("auto_upgrade_minor_version").(bool)
	publisher := d.Get("publisher").(string)
	extensionType := d.Get("type").(string)
	forceUpdateTag := d.Get("force_update_tag").(string)
	typeHandlerVersion := d.Get("type_handler_version").(string)
	t := d.Get("tags").(map[string]interface{})

	extension := compute.VirtualMachineExtension{
		Location: utils.String(location),
		VirtualMachineExtensionProperties: &compute.VirtualMachineExtensionProperties{
			AutoUpgradeMinorVersion: utils.Bool(autoUpgradeMinor),
			ForceUpdateTag:          utils.String(forceUpdateTag),
			Publisher:               utils.String(publisher),
			Type:                    utils.String(extensionType),
			TypeHandlerVersion:      utils.String(typeHandlerVersion),
		},
		Tags: tags.Expand(t),
	}

	if settingsString := d.Get("settings").(string); settingsString != "" {
		settings, err := structure.ExpandJsonFromString(settingsString)
		if err != nil {
			return fmt.Errorf("unable to parse settings: %s", err)
		}
		extension.VirtualMachineExtensionProperties.Settings = &settings
	}

	if protectedSettingsString := d.Get("protected_settings").(string); protectedSettingsString != "" {
		protectedSettings, err := structure.ExpandJsonFromString(protectedSettingsString)
		if err != nil {
			return fmt.Errorf("unable to parse protected_settings: %s", err)
		}
		extension.VirtualMachineExtensionProperties.ProtectedSettings = &protectedSettings
	}

	future, err := extensionsClient.CreateOrUpdate(ctx, resourceGroup, virtualMachineName, name, extension)
	if err != nil {
		return fmt.Errorf("Error creating Extension %q (Virtual Machine %q / Resource Group %q): %+v", name, virtualMachineName, resourceGroup, err)
	}

	if err = future.WaitForCompletionRef(ctx, extensionsClient.Client); err != nil {
		return fmt.Errorf("Error waiting for creation of Extension %q (Virtual Machine %q / Resource Group %q): %+v", name, virtualMachineName, resourceGroup, err)
	}

	read, err := extensionsClient.Get(ctx, resourceGroup, virtualMachineName, name, "")
	if err != nil {
		return fmt.Errorf("Error retrieving Extension %q (Virtual Machine %q / Resource Group %q): %+v", name, virtualMachineName, resourceGroup, err)
	}

	if read.ID == nil {
		return fmt.Errorf("Error retrieving Extension %q (Virtual Machine %q / Resource Group %q): `id` was nil", name, virtualMachineName, resourceGroup)
	}

	d.SetId(*read.ID)

	return resourceArmVirtualMachineExtensionRead(d, meta)
}

func resourceArmVirtualMachineExtensionRead(d *schema.ResourceData, meta interface{}) error {
	vmsClient := meta.(*ArmClient).Compute.VMClient
	extensionsClient := meta.(*ArmClient).Compute.VMExtensionClient
	ctx, cancel := timeouts.ForRead(meta.(*ArmClient).StopContext, d)
	defer cancel()

	id, err := computeSvc.ParseVirtualMachineExtensionID(d.Id())
	if err != nil {
		return err
	}

	virtualMachine, err := vmsClient.Get(ctx, id.Base.ResourceGroup, id.VirtualMachineName, "")
	if err != nil {
		if utils.ResponseWasNotFound(virtualMachine.Response) {
			log.Printf("[DEBUG] Virtual Machine %q was not found in Resource Group %q - removing from state!", id.VirtualMachineName, id.Base.ResourceGroup)
			d.SetId("")
			return nil
		}

		return fmt.Errorf("Error retrieving Virtual Machine %q (Resource Group %q): %+v", id.VirtualMachineName, id.Base.ResourceGroup, err)
	}

	resp, err := extensionsClient.Get(ctx, id.Base.ResourceGroup, id.VirtualMachineName, id.Name, "")
	if err != nil {
		if utils.ResponseWasNotFound(resp.Response) {
			log.Printf("[DEBUG] Extension %q was not found for Virtual Machine %q (Resource Group %q) - removing from state!", id.Name, id.VirtualMachineName, id.Base.ResourceGroup)
			d.SetId("")
			return nil
		}

		return fmt.Errorf("Error retrieving Extension %q (Virtual Machine %q / Resource Group %q): %+v", id.Name, id.VirtualMachineName, id.Base.ResourceGroup, err)
	}

	d.Set("name", id.Name)
	if location := resp.Location; location != nil {
		d.Set("location", azure.NormalizeLocation(*location))
	}
	d.Set("resource_group_name", id.Base.ResourceGroup)
	d.Set("virtual_machine_name", id.VirtualMachineName)
	d.Set("virtual_machine_id", virtualMachine.ID)

	if props := resp.VirtualMachineExtensionProperties; props != nil {
		d.Set("auto_upgrade_minor_version", props.AutoUpgradeMinorVersion)
		d.Set("force_update_tag", props.ForceUpdateTag)
		d.Set("publisher", props.Publisher)
		d.Set("type", props.Type)
		d.Set("type_handler_version", props.TypeHandlerVersion)

		settingsJson := ""
		if settings := props.Settings; settings != nil {
			settingsVal := settings.(map[string]interface{})
			settingsJson, err = structure.FlattenJsonToString(settingsVal)
			if err != nil {
				return fmt.Errorf("unable to parse settings from response: %s", err)
			}
		}
		d.Set("settings", settingsJson)
	}

	return tags.FlattenAndSet(d, resp.Tags)
}

func resourceArmVirtualMachineExtensionDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient).Compute.VMExtensionClient
	ctx, cancel := timeouts.ForDelete(meta.(*ArmClient).StopContext, d)
	defer cancel()

	id, err := computeSvc.ParseVirtualMachineExtensionID(d.Id())
	if err != nil {
		return err
	}

	future, err := client.Delete(ctx, id.Base.ResourceGroup, id.VirtualMachineName, id.Name)
	if err != nil {
		return fmt.Errorf("Error deleting Extension %q (Virtual Machine %q / Resource Group %q): %+v", id.Name, id.VirtualMachineName, id.Base.ResourceGroup, err)
	}

	if err := future.WaitForCompletionRef(ctx, client.Client); err != nil {
		return fmt.Errorf("Error waiting for deletion of Extension %q (Virtual Machine %q / Resource Group %q): %+v", id.Name, id.VirtualMachineName, id.Base.ResourceGroup, err)
	}

	return nil
}
