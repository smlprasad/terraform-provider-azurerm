package compute

import (
	"fmt"

	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/helpers/azure"
)

type VirtualMachineExtensionResourceID struct {
	Base azure.ResourceID

	VirtualMachineName string
	Name               string
}

func ParseVirtualMachineExtensionID(input string) (*VirtualMachineExtensionResourceID, error) {
	id, err := azure.ParseAzureResourceID(input)
	if err != nil {
		return nil, fmt.Errorf("[ERROR] Unable to parse Virtual Machine Extension ID %q: %+v", input, err)
	}

	extension := VirtualMachineExtensionResourceID{
		Base:               *id,
		VirtualMachineName: id.Path["virtualMachines"],
		Name:               id.Path["extensions"],
	}

	if extension.VirtualMachineName == "" {
		return nil, fmt.Errorf("ID was missing the `virtualMachines` element")
	}

	if extension.Name == "" {
		return nil, fmt.Errorf("ID was missing the `extensions` element")
	}

	return &extension, nil
}
