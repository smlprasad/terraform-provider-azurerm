package compute

import (
	"testing"

	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/helpers/azure"
)

func TestParseVirtualMachineScaleSetExtensionID(t *testing.T) {
	testData := []struct {
		Name     string
		Input    string
		Expected *VirtualMachineScaleSetExtensionResourceID
	}{
		{
			Name:     "Empty",
			Input:    "",
			Expected: nil,
		},
		{
			Name:     "No Virtual Machine Scale Set Segment",
			Input:    "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/foo",
			Expected: nil,
		},
		{
			Name:     "No Virtual Machine Scale Set Value",
			Input:    "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/foo/virtualMachineScaleSets/",
			Expected: nil,
		},
		{
			Name:     "No Virtual Machine Scale Set Extensions Segment",
			Input:    "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/foo/virtualMachineScaleSets/example1",
			Expected: nil,
		},
		{
			Name:     "No Virtual Machine Scale Set Extensions Value",
			Input:    "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/foo/virtualMachineScaleSets/example1/extensions/",
			Expected: nil,
		},
		{
			Name:  "Completed",
			Input: "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/foo/virtualMachineScaleSets/example1/extensions/example2",
			Expected: &VirtualMachineScaleSetExtensionResourceID{
				Name:               "example2",
				VirtualMachineName: "example1",
				Base: azure.ResourceID{
					ResourceGroup: "foo",
				},
			},
		},
	}

	for _, v := range testData {
		t.Logf("[DEBUG] Testing %q", v.Name)

		actual, err := ParseVirtualMachineScaleSetExtensionResourceID(v.Input)
		if err != nil {
			if v.Expected == nil {
				continue
			}

			t.Fatalf("Expected a value but got an error: %s", err)
		}

		if actual.Name != v.Expected.Name {
			t.Fatalf("Expected %q but got %q for Name", v.Expected.Name, actual.Name)
		}

		if actual.VirtualMachineName != v.Expected.VirtualMachineName {
			t.Fatalf("Expected %q but got %q for VirtualMachineName", v.Expected.VirtualMachineName, actual.VirtualMachineName)
		}

		if actual.Base.ResourceGroup != v.Expected.Base.ResourceGroup {
			t.Fatalf("Expected %q but got %q for ResourceGroup", v.Expected.Base.ResourceGroup, actual.Base.ResourceGroup)
		}
	}
}
