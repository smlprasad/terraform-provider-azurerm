package compute

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-07-01/compute"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/helpers/azure"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/helpers/suppress"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/helpers/tf"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/helpers/validate"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/clients"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/features"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/locks"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/tags"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/tf/base64"
	azSchema "github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/tf/schema"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/timeouts"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/utils"
)

// TODO: locking as appropriate

func resourceLinuxVirtualMachine() *schema.Resource {
	return &schema.Resource{
		Create: resourceLinuxVirtualMachineCreate,
		Read:   resourceLinuxVirtualMachineRead,
		Update: resourceLinuxVirtualMachineUpdate,
		Delete: resourceLinuxVirtualMachineDelete,
		Importer: azSchema.ValidateResourceIDPriorToImport(func(id string) error {
			_, err := ParseVirtualMachineID(id)
			// TODO: confirm prior to the Beta that this is a Linux VM
			// TODO: confirm that the OS Disk isn't "attach"
			return err
		}),

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(45 * time.Minute),
			Read:   schema.DefaultTimeout(5 * time.Minute),
			Update: schema.DefaultTimeout(45 * time.Minute),
			Delete: schema.DefaultTimeout(45 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: ValidateLinuxName,
			},

			"resource_group_name": azure.SchemaResourceGroupName(),

			"location": azure.SchemaLocation(),

			// Required
			"admin_username": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validate.NoEmptyStrings,
			},

			"network_interface_ids": {
				Type:     schema.TypeList,
				Required: true,
				MinItems: 1,
				Elem: &schema.Schema{
					Type: schema.TypeString,
					// TODO: validate is a NIC Resource ID
				},
			},

			"os_disk": virtualMachineOSDiskSchema(),

			"size": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validate.NoEmptyStrings,
			},

			// Optional
			"additional_capabilities": virtualMachineAdditionalCapabilitiesSchema(),

			"admin_password": {
				Type:      schema.TypeString,
				Optional:  true,
				ForceNew:  true,
				Sensitive: true,
			},

			"admin_ssh_key": SSHKeysSchema(),

			"allow_extension_operations": {
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true, // TODO: confirm behaviour
				Default:  true,
			},

			"availability_set_id": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				// TODO: it'd be nice to add more granular validation here
				ValidateFunc: azure.ValidateResourceID,
				// the Compute/VM API is broken and returns the Resource Group name in UPPERCASE :shrug:
				DiffSuppressFunc: suppress.CaseDifference,
				// TODO: raise a GH issue for the broken API
				//           availability_set_id:                 "/subscriptions/1a6092a6-137e-4025-9a7c-ef77f76f2c02/resourceGroups/acctestRG-200122113424880096/providers/Microsoft.Compute/availabilitySets/ACCTESTAVSET-200122113424880096" => "/subscriptions/1a6092a6-137e-4025-9a7c-ef77f76f2c02/resourceGroups/acctestRG-200122113424880096/providers/Microsoft.Compute/availabilitySets/acctestavset-200122113424880096" (forces new resource)
				ConflictsWith: []string{
					// TODO: "virtual_machine_scale_set_id"
					"zone",
				},
			},

			"boot_diagnostics": bootDiagnosticsSchema(),

			"computer_name": {
				Type:     schema.TypeString,
				Optional: true,

				// Computed since we reuse the VM name if one's not specified
				Computed: true,
				ForceNew: true,
				// note: whilst the portal says 1-15 characters it seems to mirror the rules for the vm name
				// (e.g. 1-15 for Windows, 1-63 for Linux)
				ValidateFunc: ValidateLinuxName,
			},

			"custom_data": base64.OptionalSchema(),

			"dedicated_host_id": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				// TODO: validation for the ID once it's merged
				// the Compute/VM API is broken and returns the Resource Group name in UPPERCASE :shrug:
				DiffSuppressFunc: suppress.CaseDifference,
				// TODO: raise a GH issue for the broken API
				// /subscriptions/88720cb0-d9d7-4d5f-917b-7ba469118dbc/resourceGroups/TOM-MANUAL/providers/Microsoft.Compute/hostGroups/tom-hostgroup/hosts/tom-manual-host
			},

			"disable_password_authentication": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},

			"eviction_policy": {
				// only applicable when `priority` is set to `Spot`
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				ValidateFunc: validation.StringInSlice([]string{
					// NOTE: whilst Delete is an option here, it's only applicable for VMSS
					string(compute.Deallocate),
				}, false),
			},

			"identity": virtualMachineIdentitySchema(),

			"max_bid_price": {
				Type:     schema.TypeFloat,
				Optional: true,
				Default:  -1,
			},

			"plan": planSchema(),

			"priority": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true, // TODO: confirm is this ForceNew for VM's
				ValidateFunc: validation.StringInSlice([]string{
					string(compute.Regular),
					string(compute.Spot),
				}, false),
			},

			"provision_vm_agent": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
				ForceNew: true,
			},

			"proximity_placement_group_id": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				// TODO: it'd be nice to add more granular validation here
				ValidateFunc: azure.ValidateResourceID,
				// the Compute/VM API is broken and returns the Resource Group name in UPPERCASE :shrug:
				DiffSuppressFunc: suppress.CaseDifference,
			},

			"secret": linuxSecretSchema(),

			"source_image_id": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: azure.ValidateResourceID,
				// TODO: does this want to be ForceNew for a VM?
			},

			// TODO: does this want to be ForceNew for a VM?
			"source_image_reference": sourceImageReferenceSchema(),

			"tags": tags.Schema(),

			"zone": {
				Type:     schema.TypeString,
				Optional: true,
				// TODO: does this want to be ForceNew for a VM?
				ForceNew: true,
				ConflictsWith: []string{
					// TODO: confirm is this right?
					"availability_set_id",
					// TODO: "virtual_machine_scale_set_id"
				},
			},

			// Computed
			"private_ip_address": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"private_ip_addresses": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"public_ip_address": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"public_ip_addresses": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"virtual_machine_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceLinuxVirtualMachineCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).Compute.VMClient
	ctx, cancel := timeouts.ForCreate(meta.(*clients.Client).StopContext, d)
	defer cancel()

	name := d.Get("name").(string)
	resourceGroup := d.Get("resource_group_name").(string)

	locks.ByName(name, virtualMachineResourceName)
	defer locks.UnlockByName(name, virtualMachineResourceName)

	if features.ShouldResourcesBeImported() {
		resp, err := client.Get(ctx, resourceGroup, name, "")
		if err != nil {
			if !utils.ResponseWasNotFound(resp.Response) {
				return fmt.Errorf("Error checking for existing Linux Virtual Machine %q (Resource Group %q): %+v", name, resourceGroup, err)
			}
		}

		if !utils.ResponseWasNotFound(resp.Response) {
			return tf.ImportAsExistsError("azurerm_linux_virtual_machine", *resp.ID)
		}
	}

	additionalCapabilitiesRaw := d.Get("additional_capabilities").([]interface{})
	additionalCapabilities := expandVirtualMachineAdditionalCapabilities(additionalCapabilitiesRaw)

	adminUsername := d.Get("admin_username").(string)
	allowExtensionOperations := d.Get("allow_extension_operations").(bool)
	bootDiagnosticsRaw := d.Get("boot_diagnostics").([]interface{})
	bootDiagnostics := expandBootDiagnostics(bootDiagnosticsRaw)
	var computerName string
	if v, ok := d.GetOk("computer_name"); ok && len(v.(string)) > 0 {
		computerName = v.(string)
	} else {
		computerName = name
	}
	disablePasswordAuthentication := d.Get("disable_password_authentication").(bool)
	location := azure.NormalizeLocation(d.Get("location").(string))
	identityRaw := d.Get("identity").([]interface{})
	identity, err := expandVirtualMachineIdentity(identityRaw)
	if err != nil {
		return fmt.Errorf("Error expanding `identity`: %+v", err)
	}
	planRaw := d.Get("plan").([]interface{})
	plan := expandPlan(planRaw)
	priority := compute.VirtualMachinePriorityTypes(d.Get("priority").(string))
	provisionVMAgent := d.Get("provision_vm_agent").(bool)
	size := d.Get("size").(string)
	t := d.Get("tags").(map[string]interface{})

	networkInterfaceIdsRaw := d.Get("network_interface_ids").([]interface{})
	networkInterfaceIds := expandVirtualMachineNetworkInterfaceIDs(networkInterfaceIdsRaw)

	osDiskRaw := d.Get("os_disk").([]interface{})
	osDisk := expandVirtualMachineOSDisk(osDiskRaw, compute.Linux)

	secretsRaw := d.Get("secret").([]interface{})
	secrets := expandLinuxSecrets(secretsRaw)

	sourceImageReferenceRaw := d.Get("source_image_reference").([]interface{})
	sourceImageId := d.Get("source_image_id").(string)
	sourceImageReference, err := expandSourceImageReference(sourceImageReferenceRaw, sourceImageId)
	if err != nil {
		return err
	}

	sshKeysRaw := d.Get("admin_ssh_key").(*schema.Set).List()
	sshKeys := ExpandSSHKeys(sshKeysRaw)

	params := compute.VirtualMachine{
		Name:     utils.String(name),
		Location: utils.String(location),
		Identity: identity,
		Plan:     plan,
		VirtualMachineProperties: &compute.VirtualMachineProperties{
			HardwareProfile: &compute.HardwareProfile{
				VMSize: compute.VirtualMachineSizeTypes(size),
			},
			OsProfile: &compute.OSProfile{
				AdminUsername:            utils.String(adminUsername),
				ComputerName:             utils.String(computerName),
				AllowExtensionOperations: utils.Bool(allowExtensionOperations),
				LinuxConfiguration: &compute.LinuxConfiguration{
					DisablePasswordAuthentication: utils.Bool(disablePasswordAuthentication),
					ProvisionVMAgent:              utils.Bool(provisionVMAgent),
					SSH: &compute.SSHConfiguration{
						PublicKeys: &sshKeys,
					},
				},
				Secrets: secrets,
			},
			NetworkProfile: &compute.NetworkProfile{
				NetworkInterfaces: &networkInterfaceIds,
			},
			Priority: priority,
			StorageProfile: &compute.StorageProfile{
				ImageReference: sourceImageReference,
				OsDisk:         osDisk,

				// Data Disks are instead handled via the Association resource - as such we can send an empty value here
				// but for Updates this'll need to be nil, else any associations will be overwritten
				DataDisks: &[]compute.DataDisk{},
			},

			// Optional
			AdditionalCapabilities: additionalCapabilities,
			DiagnosticsProfile:     bootDiagnostics,

			// @tombuildsstuff: passing in a VMSS ID returns:
			// > Code="InvalidParameter" Message="The value of parameter virtualMachineScaleSet is invalid." Target="virtualMachineScaleSet"
			// presuming this isn't finished yet; note: this'll conflict with availability set id
			VirtualMachineScaleSet: nil,

			// only applicable to Windows
			//LicenseType:             utils.String(licenseType),
		},
		Tags: tags.Expand(t),
	}

	if !provisionVMAgent && allowExtensionOperations {
		return fmt.Errorf("`allow_extension_operations` cannot be set to `true` when `provision_vm_agent` is set to `false`")
	}

	if v, ok := d.GetOk("availability_set_id"); ok {
		params.AvailabilitySet = &compute.SubResource{
			ID: utils.String(v.(string)),
		}
	}

	if v, ok := d.GetOk("custom_data"); ok {
		params.OsProfile.CustomData = utils.String(v.(string))
	}

	if v, ok := d.GetOk("dedicated_host_id"); ok {
		params.Host = &compute.SubResource{
			ID: utils.String(v.(string)),
		}
	}

	if evictionPolicyRaw, ok := d.GetOk("eviction_policy"); ok {
		if params.Priority != compute.Spot {
			return fmt.Errorf("An `eviction_policy` can only be specified when `priority` is set to `Spot`")
		}

		params.EvictionPolicy = compute.VirtualMachineEvictionPolicyTypes(evictionPolicyRaw.(string))
	} else if priority == compute.Spot {
		return fmt.Errorf("An `eviction_policy` must be specified when `priority` is set to `Spot`")
	}

	if v, ok := d.Get("max_bid_price").(float64); ok && v > 0 {
		if priority != compute.Spot {
			return fmt.Errorf("`max_bid_price` can only be configured when `priority` is set to `Spot`")
		}

		params.BillingProfile = &compute.BillingProfile{
			MaxPrice: utils.Float(v),
		}
	}

	if v, ok := d.GetOk("proximity_placement_group_id"); ok {
		params.ProximityPlacementGroup = &compute.SubResource{
			ID: utils.String(v.(string)),
		}
	}

	if v, ok := d.GetOk("zone"); ok {
		params.Zones = &[]string{
			v.(string),
		}
	}

	// "Authentication using either SSH or by user name and password must be enabled in Linux profile." Target="linuxConfiguration"
	adminPassword := d.Get("admin_password").(string)
	if disablePasswordAuthentication && len(sshKeys) == 0 {
		return fmt.Errorf("At least one `admin_ssh_key` must be specified when `disable_password_authentication` is set to `true`")
	} else if !disablePasswordAuthentication {
		if adminPassword == "" {
			return fmt.Errorf("An `admin_password` must be specified if `disable_password_authentication` is set to `false`")
		}

		params.OsProfile.AdminPassword = utils.String(adminPassword)
	}

	future, err := client.CreateOrUpdate(ctx, resourceGroup, name, params)
	if err != nil {
		return fmt.Errorf("Error creating Linux Virtual Machine %q (Resource Group %q): %+v", name, resourceGroup, err)
	}

	if err := future.WaitForCompletionRef(ctx, client.Client); err != nil {
		return fmt.Errorf("Error waiting for creation of Linux Virtual Machine %q (Resource Group %q): %+v", name, resourceGroup, err)
	}

	read, err := client.Get(ctx, resourceGroup, name, "")
	if err != nil {
		return fmt.Errorf("Error retrieving Linux Virtual Machine %q (Resource Group %q): %+v", name, resourceGroup, err)
	}

	if read.ID == nil {
		return fmt.Errorf("Error retrieving Linux Virtual Machine %q (Resource Group %q): `id` was nil", name, resourceGroup)
	}

	d.SetId(*read.ID)
	return resourceLinuxVirtualMachineRead(d, meta)
}

func resourceLinuxVirtualMachineRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).Compute.VMClient
	networkInterfacesClient := meta.(*clients.Client).Network.InterfacesClient
	publicIPAddressesClient := meta.(*clients.Client).Network.PublicIPsClient
	ctx, cancel := timeouts.ForRead(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id, err := ParseVirtualMachineID(d.Id())
	if err != nil {
		return err
	}

	resp, err := client.Get(ctx, id.ResourceGroup, id.Name, "")
	if err != nil {
		if utils.ResponseWasNotFound(resp.Response) {
			log.Printf("[DEBUG] Linux Virtual Machine %q was not found in Resource Group %q - removing from state!", id.Name, id.ResourceGroup)
			d.SetId("")
			return nil
		}

		return fmt.Errorf("Error retrieving Linux Virtual Machine %q (Resource Group %q): %+v", id.Name, id.ResourceGroup, err)
	}

	d.Set("name", id.Name)
	d.Set("resource_group_name", id.ResourceGroup)
	if location := resp.Location; location != nil {
		d.Set("location", azure.NormalizeLocation(*resp.Location))
	}

	if err := d.Set("identity", flattenVirtualMachineIdentity(resp.Identity)); err != nil {
		return fmt.Errorf("Error setting `identity`: %+v", err)
	}

	if err := d.Set("plan", flattenPlan(resp.Plan)); err != nil {
		return fmt.Errorf("Error setting `plan`: %+v", err)
	}

	if resp.VirtualMachineProperties == nil {
		return fmt.Errorf("Error retrieving Linux Virtual Machine %q (Resource Group %q): `properties` was nil", id.Name, id.ResourceGroup)
	}

	props := *resp.VirtualMachineProperties
	if err := d.Set("additional_capabilities", flattenVirtualMachineAdditionalCapabilities(props.AdditionalCapabilities)); err != nil {
		return fmt.Errorf("Error setting `additional_capabilities`: %+v", err)
	}

	availabilitySetId := ""
	if props.AvailabilitySet != nil && props.AvailabilitySet.ID != nil {
		availabilitySetId = *props.AvailabilitySet.ID
	}
	d.Set("availability_set_id", availabilitySetId)

	if err := d.Set("boot_diagnostics", flattenBootDiagnostics(props.DiagnosticsProfile)); err != nil {
		return fmt.Errorf("Error setting `boot_diagnostics`: %+v", err)
	}

	d.Set("eviction_policy", string(props.EvictionPolicy))
	if profile := props.HardwareProfile; profile != nil {
		d.Set("size", string(profile.VMSize))
	}

	// defaulted since BillingProfile isn't returned if it's unset
	maxBidPrice := float64(-1.0)
	if props.BillingProfile != nil && props.BillingProfile.MaxPrice != nil {
		maxBidPrice = *props.BillingProfile.MaxPrice
	}
	d.Set("max_bid_price", maxBidPrice)

	if profile := props.NetworkProfile; profile != nil {
		if err := d.Set("network_interface_ids", flattenVirtualMachineNetworkInterfaceIDs(props.NetworkProfile.NetworkInterfaces)); err != nil {
			return fmt.Errorf("Error setting `network_interface_ids`: %+v", err)
		}
	}

	dedicatedHostId := ""
	if props.Host != nil && props.Host.ID != nil {
		dedicatedHostId = *props.Host.ID
	}
	d.Set("dedicated_host_id", dedicatedHostId)

	if profile := props.OsProfile; profile != nil {
		d.Set("admin_username", profile.AdminUsername)
		d.Set("allow_extension_operations", profile.AllowExtensionOperations)
		d.Set("computer_name", profile.ComputerName)

		if config := profile.LinuxConfiguration; config != nil {
			d.Set("disable_password_authentication", config.DisablePasswordAuthentication)
			d.Set("provision_vm_agent", config.ProvisionVMAgent)

			flattenedSSHKeys, err := FlattenSSHKeys(config.SSH)
			if err != nil {
				return fmt.Errorf("Error flattening `admin_ssh_key`: %+v", err)
			}
			if err := d.Set("admin_ssh_key", flattenedSSHKeys); err != nil {
				return fmt.Errorf("Error setting `admin_ssh_key`: %+v", err)
			}
		}

		if err := d.Set("secret", flattenLinuxSecrets(profile.Secrets)); err != nil {
			return fmt.Errorf("Error setting `secret`: %+v", err)
		}
	}

	d.Set("priority", string(props.Priority))
	proximityPlacementGroupId := ""
	if props.ProximityPlacementGroup != nil && props.ProximityPlacementGroup.ID != nil {
		proximityPlacementGroupId = *props.ProximityPlacementGroup.ID
	}
	d.Set("proximity_placement_group_id", proximityPlacementGroupId)

	if profile := props.StorageProfile; profile != nil {
		if err := d.Set("os_disk", flattenVirtualMachineOSDisk(profile.OsDisk)); err != nil {
			return fmt.Errorf("Error settings `os_disk`: %+v", err)
		}

		var storageImageId string
		if profile.ImageReference != nil && profile.ImageReference.ID != nil {
			storageImageId = *profile.ImageReference.ID
		}
		d.Set("source_image_id", storageImageId)

		if err := d.Set("source_image_reference", flattenSourceImageReference(profile.ImageReference)); err != nil {
			return fmt.Errorf("Error setting `source_image_reference`: %+v", err)
		}
	}

	d.Set("virtual_machine_id", props.VMID)

	zone := ""
	if resp.Zones != nil {
		if zones := *resp.Zones; len(zones) > 0 {
			zone = zones[0]
		}
	}
	d.Set("zone", zone)

	connectionInfo := retrieveConnectionInformation(ctx, networkInterfacesClient, publicIPAddressesClient, resp.VirtualMachineProperties)
	d.Set("private_ip_address", connectionInfo.primaryPrivateAddress)
	d.Set("private_ip_addresses", connectionInfo.privateAddresses)
	d.Set("public_ip_address", connectionInfo.primaryPublicAddress)
	d.Set("public_ip_addresses", connectionInfo.publicAddresses)
	isWindows := false
	setConnectionInformation(d, connectionInfo, isWindows)

	return tags.FlattenAndSet(d, resp.Tags)
}

func resourceLinuxVirtualMachineUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).Compute.VMClient
	ctx, cancel := timeouts.ForRead(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id, err := ParseVirtualMachineID(d.Id())
	if err != nil {
		return err
	}

	locks.ByName(id.Name, virtualMachineResourceName)
	defer locks.UnlockByName(id.Name, virtualMachineResourceName)

	existing, err := client.InstanceView(ctx, id.ResourceGroup, id.Name)
	if err != nil {
		return fmt.Errorf("Error retrieving Linux Virtual Machine %q (Resource Group %q): %+v", id.Name, id.ResourceGroup, err)
	}

	shouldTurnBackOn := true
	if existing.Statuses != nil {
		for _, status := range *existing.Statuses {
			if status.Code == nil {
				continue
			}

			// could also be the provisioning state which we're not bothered with here
			state := strings.ToLower(*status.Code)
			if !strings.HasPrefix(state, "PowerState/") {
				continue
			}

			state = strings.TrimPrefix(state, "powerstate/")
			switch strings.ToLower(state) {
			case "deallocating":
			case "deallocated":
			case "stopped":
				shouldTurnBackOn = false
			}
		}
	}

	shouldUpdate := false
	shouldShutDown := false
	update := compute.VirtualMachineUpdate{
		VirtualMachineProperties: &compute.VirtualMachineProperties{},
	}

	if d.HasChange("boot_diagnostics") {
		shouldUpdate = true

		bootDiagnosticsRaw := d.Get("boot_diagnostics").([]interface{})
		update.VirtualMachineProperties.DiagnosticsProfile = expandBootDiagnostics(bootDiagnosticsRaw)
	}

	// TODO: can allow_extension_operation be updated too?
	if d.HasChange("admin_ssh_key") || d.HasChange("custom_data") || d.HasChange("disable_password_authentication") || d.HasChange("secret") {
		shouldUpdate = true

		profile := compute.OSProfile{}

		if d.HasChange("admin_ssh_key") || d.HasChange("disable_password_authentication") {
			config := compute.LinuxConfiguration{}

			if d.HasChange("admin_ssh_key") {
				sshKeysRaw := d.Get("admin_ssh_key").(*schema.Set).List()
				sshKeys := ExpandSSHKeys(sshKeysRaw)
				config.SSH = &compute.SSHConfiguration{
					PublicKeys: &sshKeys,
				}
			}

			if d.HasChange("disable_password_authentication") {
				config.DisablePasswordAuthentication = utils.Bool(d.Get("disable_password_authentication").(bool))
			}

			// TODO: should we also support updating "provision_vm_agent" here?

			profile.LinuxConfiguration = &config
		}

		if d.HasChange("custom_data") {
			profile.CustomData = utils.String(d.Get("custom_data").(string))
		}

		if d.HasChange("secret") {
			secretsRaw := d.Get("secret").([]interface{})
			profile.Secrets = expandLinuxSecrets(secretsRaw)
		}

		update.VirtualMachineProperties.OsProfile = &profile
	}

	if d.HasChange("identity") {
		shouldUpdate = true

		identityRaw := d.Get("identity").([]interface{})
		identity, err := expandVirtualMachineIdentity(identityRaw)
		if err != nil {
			return fmt.Errorf("Error expanding `identity`: %+v", err)
		}
		update.Identity = identity
	}

	if d.HasChange("max_bid_price") {
		shouldUpdate = true

		// Code="OperationNotAllowed" Message="Max price change is not allowed. For more information, see http://aka.ms/AzureSpot/errormessages"
		shouldShutDown = true

		maxBidPrice := d.Get("max_bid_price").(float64)
		update.VirtualMachineProperties.BillingProfile = &compute.BillingProfile{
			MaxPrice: utils.Float(maxBidPrice),
		}
	}

	if d.HasChange("network_interface_ids") {
		shouldUpdate = true
		//shouldShutDown = true // TODO: confirm if this is needed or not

		networkInterfaceIdsRaw := d.Get("network_interface_ids").([]interface{})
		networkInterfaceIds := expandVirtualMachineNetworkInterfaceIDs(networkInterfaceIdsRaw)

		update.VirtualMachineProperties.NetworkProfile = &compute.NetworkProfile{
			NetworkInterfaces: &networkInterfaceIds,
		}
	}

	if d.HasChange("os_disk") {
		shouldUpdate = true
		//shouldShutDown = true // TODO: confirm if needed

		// TODO: we'll likely want to expand each individual field here
		// Code="Conflict" Message="Disk resizing is allowed only when creating a VM or when the VM is deallocated." Target="disk.diskSizeGB"

		osDiskRaw := d.Get("os_disk").([]interface{})
		osDisk := expandVirtualMachineOSDisk(osDiskRaw, compute.Linux)
		update.VirtualMachineProperties.StorageProfile = &compute.StorageProfile{
			OsDisk: osDisk,
		}
	}

	if d.HasChange("size") {
		shouldUpdate = true
		vmSize := d.Get("size").(string)

		// Azure will auto-reboot this for us, providing this machine will fit on this host
		// otherwise we need to shut down the VM to move it to another host to be able to use this size
		availableOnThisHost := false
		sizes, err := client.ListAvailableSizes(ctx, id.ResourceGroup, id.Name)
		if err != nil {
			return fmt.Errorf("Error retrieving available sizes for Linux Virtual Machine %q (Resource Group %q): %+v", id.Name, id.ResourceGroup, err)
		}

		if sizes.Value != nil {
			for _, size := range *sizes.Value {
				if size.Name == nil {
					continue
				}

				if strings.EqualFold(*size.Name, vmSize) {
					availableOnThisHost = true
					break
				}
			}
		}

		if !availableOnThisHost {
			log.Printf("[DEBUG] Requested VM Size isn't available on the Host - VM must be shut down to resize..")
			shouldShutDown = true
		}

		update.VirtualMachineProperties.HardwareProfile = &compute.HardwareProfile{
			VMSize: compute.VirtualMachineSizeTypes(vmSize),
		}
	}

	if d.HasChange("tags") {
		shouldUpdate = true

		tagsRaw := d.Get("tags").(map[string]interface{})
		update.Tags = tags.Expand(tagsRaw)
	}

	if shouldShutDown {
		log.Printf("[DEBUG] Shutting Down Linux Virtual Machine %q (Resource Group %q)..", id.Name, id.ResourceGroup)
		forceShutdown := false
		future, err := client.PowerOff(ctx, id.ResourceGroup, id.Name, utils.Bool(forceShutdown))
		if err != nil {
			return fmt.Errorf("Error sending Power Off to Linux Virtual Machine %q (Resource Group %q): %+v", id.Name, id.ResourceGroup, err)
		}

		if err := future.WaitForCompletionRef(ctx, client.Client); err != nil {
			return fmt.Errorf("Error waiting for Power Off of Linux Virtual Machine %q (Resource Group %q): %+v", id.Name, id.ResourceGroup, err)
		}

		log.Printf("[DEBUG] Shut Down Linux Virtual Machine %q (Resource Group %q)..", id.Name, id.ResourceGroup)
	}

	if shouldUpdate {
		log.Printf("[DEBUG] Updating Linux Virtual Machine %q (Resource Group %q)..", id.Name, id.ResourceGroup)
		future, err := client.Update(ctx, id.ResourceGroup, id.Name, update)
		if err != nil {
			return fmt.Errorf("Error updating Linux Virtual Machine %q (Resource Group %q): %+v", id.Name, id.ResourceGroup, err)
		}

		if err := future.WaitForCompletionRef(ctx, client.Client); err != nil {
			return fmt.Errorf("Error waiting for update of Linux Virtual Machine %q (Resource Group %q): %+v", id.Name, id.ResourceGroup, err)
		}

		log.Printf("[DEBUG] Updated Linux Virtual Machine %q (Resource Group %q).", id.Name, id.ResourceGroup)
	}

	// if we've shut it down and it was turned off, let's boot it back up
	if shouldTurnBackOn && shouldShutDown {
		log.Printf("[DEBUG] Starting Linux Virtual Machine %q (Resource Group %q)..", id.Name, id.ResourceGroup)
		future, err := client.Start(ctx, id.ResourceGroup, id.Name)
		if err != nil {
			return fmt.Errorf("Error starting Linux Virtual Machine %q (Resource Group %q): %+v", id.Name, id.ResourceGroup, err)
		}

		if err := future.WaitForCompletionRef(ctx, client.Client); err != nil {
			return fmt.Errorf("Error waiting for start of Linux Virtual Machine %q (Resource Group %q): %+v", id.Name, id.ResourceGroup, err)
		}

		log.Printf("[DEBUG] Started Linux Virtual Machine %q (Resource Group %q)..", id.Name, id.ResourceGroup)
	}

	return resourceLinuxVirtualMachineRead(d, meta)
}

func resourceLinuxVirtualMachineDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).Compute.VMClient
	ctx, cancel := timeouts.ForRead(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id, err := ParseVirtualMachineID(d.Id())
	if err != nil {
		return err
	}

	locks.ByName(id.Name, virtualMachineResourceName)
	defer locks.UnlockByName(id.Name, virtualMachineResourceName)

	log.Printf("[DEBUG] Retrieving Linux Virtual Machine %q (Resource Group %q)..", id.Name, id.ResourceGroup)
	existing, err := client.Get(ctx, id.ResourceGroup, id.Name, "")
	if err != nil {
		if utils.ResponseWasNotFound(existing.Response) {
			return nil
		}

		return fmt.Errorf("Error retrieving Linux Virtual Machine %q (Resource Group %q): %+v", id.Name, id.ResourceGroup, err)
	}

	// ISSUE: XXX
	// shutting down the Virtual Machine prior to removing it means users are no longer charged for the compute
	// thus this can be a large cost-saving when deleting larger instances
	// in addition - since we're shutting down the machine to remove it, forcing a power-off is fine (as opposed
	// to waiting for a graceful shut down)
	log.Printf("[DEBUG] Powering Off Linux Virtual Machine %q (Resource Group %q)..", id.Name, id.ResourceGroup)
	skipShutdown := true
	powerOffFuture, err := client.PowerOff(ctx, id.ResourceGroup, id.Name, utils.Bool(skipShutdown))
	if err != nil {
		return fmt.Errorf("Error powering off Linux Virtual Machine %q (Resource Group %q): %+v", id.Name, id.ResourceGroup, err)
	}
	if err := powerOffFuture.WaitForCompletionRef(ctx, client.Client); err != nil {
		return fmt.Errorf("Error waiting for power off of Linux Virtual Machine %q (Resource Group %q): %+v", id.Name, id.ResourceGroup, err)
	}
	log.Printf("[DEBUG] Powered Off Linux Virtual Machine %q (Resource Group %q).", id.Name, id.ResourceGroup)

	log.Printf("[DEBUG] Deleting Linux Virtual Machine %q (Resource Group %q)..", id.Name, id.ResourceGroup)
	deleteFuture, err := client.Delete(ctx, id.ResourceGroup, id.Name)
	if err != nil {
		return fmt.Errorf("Error deleting Linux Virtual Machine %q (Resource Group %q): %+v", id.Name, id.ResourceGroup, err)
	}
	if err := deleteFuture.WaitForCompletionRef(ctx, client.Client); err != nil {
		return fmt.Errorf("Error waiting for deletion of Linux Virtual Machine %q (Resource Group %q): %+v", id.Name, id.ResourceGroup, err)
	}
	log.Printf("[DEBUG] Deleted Linux Virtual Machine %q (Resource Group %q).", id.Name, id.ResourceGroup)

	return nil
}
