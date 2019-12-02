---
subcategory: "Compute"
layout: "azurerm"
page_title: "Azure Resource Manager: azurerm_virtual_machine_extension"
sidebar_current: "docs-azurerm-resource-compute-virtualmachine-extension"
description: |-
    Manages an Extension within a Virtual Machine.
---

# azurerm_virtual_machine_extension

Manages an Extension within a Virtual Machine.

~> **NOTE:** Custom Script Extensions for Linux & Windows require that the `commandToExecute` returns a `0` exit code to be classified as successfully deployed. You can achieve this by appending `exit 0` to the end of your `commandToExecute`.

-> **NOTE:** Custom Script Extensions require that the Azure Virtual Machine Guest Agent is running on the Virtual Machine.

## Example Usage

```hcl
data "azurerm_virtual_machine" "example" {
  name                = "example-machine"
  resource_group_name = "example-resources"
}

resource "azurerm_virtual_machine_extension" "example" {
  name                 = "hostname"
  virtual_machine_name = data.azurerm_virtual_machine.example.id
  publisher            = "Microsoft.Azure.Extensions"
  type                 = "CustomScript"
  type_handler_version = "2.0"

  settings = jsonencode({
    "commandToExecute" = "hostname && uptime"
  })
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of this Virtual Machine Extension. Changing this forces a new resource to be created.
    
* `virtual_machine_id` - (Optional) The ID of the Virtual Machine where this Extension should be created. Changing this forces a new resource to be created.

-> **NOTE:** This will become Required in 2.0.

* `publisher` - (Required) The Publisher of the Virtual Machine Extension.

* `type` - (Required) The Type of Virtual Machine Extension.

~> **Note:** The `Publisher` and `Type` of Virtual Machine Extensions can be found using the Azure CLI using:

```shell
$ az vm extension image list --location westus -o table
```

* `type_handler_version` - (Required) Specifies the version of the extension to use, available versions can be found using the Azure CLI.

---

* `auto_upgrade_minor_version` - (Optional) Specifies if the platform deploys the latest minor version update to the `type_handler_version` specified.

* `force_update_tag` - (Optional) A value which can be used to trigger a re-run of this Virtual Machine Extension, even if the settings haven't changed.

* `settings` - (Optional) A JSON Object representing the settings for the Virtual Machine Extension.

~> **Please Note:** Certain VM Extensions require that the keys in the `settings` block are case sensitive. If you're seeing unhelpful errors, please ensure the keys are consistent with how Azure is expecting them (for instance, for the `JsonADDomainExtension` extension, the keys are expected to be in `TitleCase`.)

* `settings` - (Optional) A JSON Object representing the protected settings for the Virtual Machine Extension.

~> **Please Note:** Certain VM Extensions require that the keys in the `protected_settings` block are case sensitive. If you're seeing unhelpful errors, please ensure the keys are consistent with how Azure is expecting them (for instance, for the `JsonADDomainExtension` extension, the keys are expected to be in `TitleCase`.)

* `tags` - (Optional) A mapping of tags to assign to the resource.

---

These fields have been replaced by the `virtual_machine_id` field and will be removed in 2.0 - but are still available for compatibility reasons:

* `location` - (Optional / **Deprecated**) The location where the Extension is created. Changing this forces a new resource to be created.

* `resource_group_name` - (Optional / **Deprecated**) The name of the Resource Group where the Virtual Machine exists. Changing this forces a new resource to be created.

* `virtual_machine_name` - (Optional / **Deprecated**) The name of the Virtual Machine where this should. Changing this forces a new resource to be created.

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the Virtual Machine Extension.

## Import

Virtual Machine Extensions can be imported using the `resource id`, e.g.

```shell
terraform import azurerm_virtual_machine_extension.example /subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/mygroup1/providers/Microsoft.Compute/virtualMachines/myVM/extensions/hostname
```
