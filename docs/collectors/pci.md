# PCI

> **Status:** Implemented ✅

## Description

The collector lists the PCI devices of the host. For each device, it reports:

- The vendor and product identifiers, and their names.
- The device class and subclass.
- The subsystem IDs.
- The kernel driver that the device is bound to.
- The revision.
- The IOMMU group.
- The PCI address of the parent bridge.

The output shape is the same as Ohai's `linux/lspci.rb`: a map with the PCI
address as the key. But gohai reads the data from `/sys/bus/pci/devices` through
ghw. It does not run `lspci`. ghw finds the vendor, product and class names in
its bundled pci.ids database.

## Collected fields

| Field     | Type                | Description                                   | Schema mapping            |
| --------- | ------------------- | --------------------------------------------- | ------------------------- |
| `devices` | `map[string]Device` | The PCI devices, with the address as the key. | No direct schema mapping. |

### Device

| Field            | Type     | Description                                                                        |
| ---------------- | -------- | ---------------------------------------------------------------------------------- |
| `vendor_id`      | `string` | The PCI vendor ID in 4 hex digits, for example `8086`.                             |
| `vendor_name`    | `string` | The vendor name from pci.ids, for example `Intel Corporation`.                     |
| `device_id`      | `string` | The PCI device ID in 4 hex digits, for example `24fd`.                             |
| `device_name`    | `string` | The device name, for example `Wireless 8265 / 8275`.                               |
| `class_id`       | `string` | The PCI class code in 2 hex digits, for example `02` (Network controller).         |
| `class_name`     | `string` | The class name.                                                                    |
| `subclass_id`    | `string` | The subclass code in 2 hex digits.                                                 |
| `subclass_name`  | `string` | The subclass name.                                                                 |
| `sdevice_id`     | `string` | The subsystem device ID in 4 hex digits.                                           |
| `sdevice_name`   | `string` | The subsystem name.                                                                |
| `revision`       | `string` | The PCI revision register, for example `0x01`.                                     |
| `driver`         | `string` | The bound kernel driver, for example `iwlwifi`. It is empty if no driver is bound. |
| `iommu_group`    | `string` | The IOMMU group identifier, on hardware that supports virtualization.              |
| `parent_address` | `string` | The PCI address of the upstream bridge, for example `0000:00:1c.0`.                |

## Platform support

| Platform | Supported                                                                       |
| -------- | ------------------------------------------------------------------------------- |
| Linux    | ✅                                                                              |
| macOS    | ❌ (the sysfs PCI tree is only on Linux; the collector returns an empty `Info`) |

## Example output

```json
{
  "pci": {
    "devices": {
      "0000:03:00.0": {
        "vendor_id": "8086",
        "vendor_name": "Intel Corporation",
        "device_id": "24fd",
        "device_name": "Wireless 8265 / 8275",
        "class_id": "02",
        "class_name": "Network controller",
        "driver": "iwlwifi",
        "iommu_group": "12"
      }
    }
  }
}
```

## SDK usage

```go
g, _ := gohai.New(gohai.WithCollectors("pci"))
facts, _ := g.Collect(context.Background())

for addr, d := range facts.PCI.Devices {
    fmt.Printf("%s  %s  %s\n", addr, d.VendorName, d.DeviceName)
}
```

## Enable/disable

```bash
gohai collect --collector.pci      # enable (opt-in)
gohai collect --no-collector.pci   # disable (default)
gohai collect --category=hardware  # pulls this + all hardware collectors
```

## Dependencies

None.

## Data sources

On Linux:

1. ghw's `pci.New()` reads `/sys/bus/pci/devices`. For each device, it reads the
   sysfs files `vendor`, `device`, `class`, `revision`, `subsystem_vendor`,
   `subsystem_device`, `iommu_group` and the `driver` symlink.
2. ghw finds the vendor, product, class and subclass names in its bundled
   pci.ids database. It does no network lookup. If a pcidb entry is missing, the
   ID has a value but the name is empty.
3. ghw sets `Driver: "unknown"` if no kernel driver is bound to the device. The
   collector changes this value to an empty string, so that consumers get a
   clearer contract.
4. If ghw cannot load the data (sysfs is missing, or a container has no
   `/sys/bus/pci`), the collector returns an empty `devices` map and no error.
   The other hardware collectors also fail quietly in this way.

On macOS, the collector returns an empty `Info{Devices: {}}`. macOS gives the
PCI topology through IOKit (`ioreg -p IODeviceTree -c IOPCIDevice`). But Ohai's
pci plugin is only for Linux, and its output shape follows the Linux sysfs and
lspci model. gohai has the same scope.

gohai reports the same fields as Ohai's `linux/lspci.rb`: `vendor_id`,
`vendor_name`, `device_id`, `device_name`, `class_id`, `class_name`,
`sdevice_id`, `sdevice_name`, `driver` and `revision`. gohai also reports some
fields that ghw gives and Ohai does not: `subclass_id`, `subclass_name`,
`iommu_group` and `parent_address`. gohai does not report Ohai's `module` array,
which lists the kernel modules that _can_ bind. `driver` reports only the kernel
module that is bound now. Inventory consumers usually need only that value.

## Backing library

- [`github.com/jaypipes/ghw/pkg/pci`](https://github.com/jaypipes/ghw). The
  canonical sysfs PCI reader. It finds names in its bundled pci.ids database.
