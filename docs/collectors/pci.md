# PCI

> **Status:** Implemented ✅

## Description

Enumerates PCI devices attached to the host. Reports vendor / product
identifiers + human names, device class + subclass, subsystem IDs, kernel driver
binding, revision, IOMMU group, and the parent (bridge) PCI address.

Mirrors Ohai's `linux/lspci.rb` output shape — a map keyed by PCI address — but
sources the data from `/sys/bus/pci/devices` via ghw rather than shelling out to
`lspci`. ghw resolves vendor / product / class names against its bundled pci.ids
database.

## Collected Fields

| Field     | Type                | Description                   | Schema mapping            |
| --------- | ------------------- | ----------------------------- | ------------------------- |
| `devices` | `map[string]Device` | PCI devices keyed by address. | No direct schema mapping. |

### Device

| Field            | Type     | Description                                                       |
| ---------------- | -------- | ----------------------------------------------------------------- |
| `vendor_id`      | `string` | 4-hex PCI vendor ID (e.g. `8086`).                                |
| `vendor_name`    | `string` | Human vendor name from pci.ids (e.g. `Intel Corporation`).        |
| `device_id`      | `string` | 4-hex PCI device ID (e.g. `24fd`).                                |
| `device_name`    | `string` | Human device name (e.g. `Wireless 8265 / 8275`).                  |
| `class_id`       | `string` | 2-hex PCI class code (e.g. `02` — Network controller).            |
| `class_name`     | `string` | Human class name.                                                 |
| `subclass_id`    | `string` | 2-hex subclass code.                                              |
| `subclass_name`  | `string` | Human subclass name.                                              |
| `sdevice_id`     | `string` | 4-hex subsystem device ID.                                        |
| `sdevice_name`   | `string` | Human subsystem name.                                             |
| `revision`       | `string` | PCI revision register (e.g. `0x01`).                              |
| `driver`         | `string` | Bound kernel driver (e.g. `iwlwifi`). Empty when no driver bound. |
| `iommu_group`    | `string` | IOMMU group identifier (when virtualization-ready hardware).      |
| `parent_address` | `string` | PCI address of the upstream bridge (e.g. `0000:00:1c.0`).         |

## Platform Support

| Platform | Supported                                                   |
| -------- | ----------------------------------------------------------- |
| Linux    | ✅                                                          |
| macOS    | ❌ (sysfs PCI tree is Linux-specific; returns empty `Info`) |

## Example Output

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

## SDK Usage

```go
g, _ := gohai.New(gohai.WithCollectors("pci"))
facts, _ := g.Collect(context.Background())

for addr, d := range facts.PCI.Devices {
    fmt.Printf("%s  %s  %s\n", addr, d.VendorName, d.DeviceName)
}
```

## Enable/Disable

```bash
gohai --collector.pci      # enable (opt-in)
gohai --no-collector.pci   # disable (default)
gohai --category=hardware  # pulls this + all hardware collectors
```

## Dependencies

None.

## Data Sources

On Linux:

1. ghw's `pci.New()` walks `/sys/bus/pci/devices` and reads each device's sysfs
   files (`vendor`, `device`, `class`, `revision`, `subsystem_vendor`,
   `subsystem_device`, `iommu_group`, `driver` symlink).
2. Vendor / product / class / subclass names are resolved against ghw's bundled
   pci.ids database (no network lookup). When a pcidb entry is missing the ID is
   still populated but the corresponding name is empty.
3. ghw's `Driver: "unknown"` — used when the device has no kernel driver bound —
   is normalized to an empty string for a cleaner consumer contract.
4. A ghw load error (missing sysfs, container without `/sys/bus/pci`) yields an
   empty `devices` map with no error — matches the "fails quietly" behavior of
   the other hardware collectors.

On macOS the collector returns an empty `Info{Devices: {}}`. macOS exposes PCI
topology through IOKit (`ioreg -p IODeviceTree -c IOPCIDevice`) but Ohai's pci
plugin is Linux-only and the output shape it emits is built around Linux's sysfs
/ lspci model; we match that scope.

Mirrors Ohai's `linux/lspci.rb` field surface (`vendor_id` / `vendor_name`,
`device_id` / `device_name`, `class_id` / `class_name`, `sdevice_id` /
`sdevice_name`, `driver`, `revision`). Fields Ohai doesn't emit but ghw exposes
— `subclass_id` / `subclass_name`, `iommu_group`, `parent_address` — are added
on top. Ohai's `module` array (kernel modules that _can_ bind) is not surfaced;
`driver` reports only the currently bound kernel module, which is what inventory
consumers typically want.

## Backing library

- [`github.com/jaypipes/ghw/pkg/pci`](https://github.com/jaypipes/ghw) —
  canonical sysfs PCI reader with bundled pci.ids name resolution.
