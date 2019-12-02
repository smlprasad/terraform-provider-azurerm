package compute

import (
	"fmt"

	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/helpers/azure"
)

type VirtualMachineResourceID struct {
	Base azure.ResourceID

	Name string
}

func ParseVirtualMachineID(input string) (*VirtualMachineResourceID, error) {
	id, err := azure.ParseAzureResourceID(input)
	if err != nil {
		return nil, fmt.Errorf("[ERROR] Unable to parse Virtual Machine Scale Set ID %q: %+v", input, err)
	}

	vm := VirtualMachineResourceID{
		Base: *id,
		Name: id.Path["virtualMachines"],
	}

	if vm.Name == "" {
		return nil, fmt.Errorf("ID was missing the `virtualMachineScaleSets` element")
	}

	return &vm, nil
}

func ValidateVirtualMachineID(i interface{}, k string) (warnings []string, errors []error) {
	v, ok := i.(string)
	if !ok {
		return nil, []error{fmt.Errorf("expected type of %q to be string", k)}
	}

	if _, err := ParseVirtualMachineID(v); err != nil {
		return nil, []error{err}
	}

	return nil, nil
}
