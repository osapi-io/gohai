# DMI

> **Status:** Implemented ✅

## Description

Reports SMBIOS / DMI data — BIOS, baseboard, chassis, and product identity. On
Linux the data comes from `/sys/class/dmi/id/*` via [ghw][], which reads the
sysfs entries exposed by the kernel (no root needed for most fields —
`product_serial` and `product_uuid` are 0400 and return empty for non-root
callers). macOS has no SMBIOS equivalent; the collector returns an empty Info
there and the `hardware` collector (planned) covers macOS hardware identity.

Primary consumers are cloud-provider collectors (`gce`, `ec2`, `azure`, ...)
which use `product.Name` / `product.Vendor` / `bios.Vendor` / `chassis.AssetTag`
to detect which cloud a VM is running on before hitting that provider's metadata
endpoint. Hardware inventory and compliance tooling use the full set for fleet
audits.

## Collected Fields

Each section is nil when that part of SMBIOS isn't available — virtual machines
often omit chassis data, minimal containers may have no DMI at all. Consumers
safely check `facts.DMI.Product != nil` before dereferencing.

| Field       | Type         | Description            | Schema mapping                      |
| ----------- | ------------ | ---------------------- | ----------------------------------- |
| `bios`      | `*BIOS`      | Firmware identity.     | OCSF `device.hw_info` (BIOS subset) |
| `baseboard` | `*Baseboard` | Motherboard identity.  | OCSF `device.hw_info`               |
| `chassis`   | `*Chassis`   | Enclosure identity.    | OCSF `device.hw_info`               |
| `product`   | `*Product`   | System-level identity. | OCSF `device.hw_info`               |

### BIOS

| Field     | Type     | Description        | Schema mapping                          |
| --------- | -------- | ------------------ | --------------------------------------- |
| `vendor`  | `string` | BIOS vendor.       | OCSF `device.hw_info.bios_manufacturer` |
| `version` | `string` | BIOS version.      | OCSF `device.hw_info.bios_ver`          |
| `date`    | `string` | BIOS release date. | OCSF `device.hw_info.bios_date`         |

### Baseboard

| Field           | Type     | Description          |
| --------------- | -------- | -------------------- |
| `vendor`        | `string` | Motherboard vendor.  |
| `product`       | `string` | Motherboard model.   |
| `version`       | `string` | Motherboard version. |
| `serial_number` | `string` | Motherboard serial.  |
| `asset_tag`     | `string` | Asset tag.           |

### Chassis

| Field              | Type     | Description                                  |
| ------------------ | -------- | -------------------------------------------- |
| `vendor`           | `string` | Chassis vendor.                              |
| `type`             | `string` | DMI chassis-type numeric code.               |
| `type_description` | `string` | Human-readable chassis type.                 |
| `version`          | `string` | Chassis version.                             |
| `serial_number`    | `string` | Chassis serial.                              |
| `asset_tag`        | `string` | Asset tag (e.g. `"OracleCloud.com"` on OCI). |

### Product

| Field           | Type     | Description                                                      |
| --------------- | -------- | ---------------------------------------------------------------- |
| `vendor`        | `string` | System vendor (`"Google"`, `"Amazon EC2"`, `"Dell Inc."`, etc.). |
| `name`          | `string` | Product name — primary cloud-detection signal.                   |
| `family`        | `string` | Product family.                                                  |
| `version`       | `string` | Product version.                                                 |
| `serial_number` | `string` | Product serial (0400 on Linux — root-only, empty for non-root).  |
| `uuid`          | `string` | System UUID (0400 on Linux — root-only, empty for non-root).     |
| `sku`           | `string` | Product SKU.                                                     |

## Platform Support

| Platform | Supported                                   |
| -------- | ------------------------------------------- |
| Linux    | ✅ via ghw + `/sys/class/dmi/id/*`          |
| macOS    | ✅ returns empty Info (macOS has no SMBIOS) |
| Other    | ✅ returns empty Info                       |

## Example Output

### GCE VM

```json
{
  "dmi": {
    "bios": { "vendor": "Google", "version": "Google", "date": "10/28/2024" },
    "baseboard": { "vendor": "Google", "product": "Google Compute Engine" },
    "chassis": { "vendor": "Google", "type": "1", "type_description": "Other" },
    "product": {
      "vendor": "Google",
      "name": "Google Compute Engine",
      "uuid": ""
    }
  }
}
```

### macOS

```json
{
  "dmi": {}
}
```

## SDK Usage

```go
import (
    "context"
    "strings"

    "github.com/osapi-io/gohai/pkg/gohai"
)

g, _ := gohai.New(gohai.WithCollectors("dmi"))
facts, _ := g.Collect(context.Background())

if facts.DMI != nil && facts.DMI.Product != nil {
    if strings.Contains(facts.DMI.Product.Name, "Google Compute Engine") {
        // on GCE
    }
}
```

Or enable by category:

```go
gohai.New(gohai.WithCategory("hardware"))  // pulls dmi + cpu + memory + disk + filesystem
```

## Enable/Disable

```bash
gohai --collector.dmi      # enable (opt-in)
gohai --no-collector.dmi   # disable (default)
gohai --category=hardware  # pulls dmi + other hardware collectors
```

Opt-in because most consumers only need DMI indirectly — when they enable a
cloud collector, dmi gets pulled in automatically via `Dependencies()`.

## Dependencies

None.

## Data Sources

On Linux:

1. ghw's `bios.New()`, `baseboard.New()`, `chassis.New()`, and `product.New()`
   each read the corresponding `/sys/class/dmi/id/*` files (`bios_vendor`,
   `bios_version`, `bios_date`, `board_*`, `chassis_*`, `product_*`,
   `sys_vendor`). No root required for fields exposed as 0444; root-only fields
   (`product_serial`, `product_uuid`) return empty strings for non-root callers
   rather than erroring.
2. If ghw errors on any individual section (permission denied, sysfs entry
   missing) that section stays nil; other sections still populate. A host
   without `/sys/class/dmi/id/` at all (unprivileged containers) yields an Info
   with all four sub-structs nil.

On macOS SMBIOS/DMI isn't exposed — the Linux DMI driver has no Darwin
counterpart. We return an empty Info so consumers can check `facts.DMI.Product`
without a per-OS branch. macOS hardware identity will be covered by the planned
`hardware` collector via IOKit / `ioreg` / `system_profiler`.

Ohai's `dmi` plugin shells out to `dmidecode`, which reads `/dev/mem` and
exposes the full SMBIOS record set across all 128+ DMI types, but requires root.
Our sysfs approach covers only the four types sysfs exports (BIOS, baseboard,
chassis, product) — intentional tradeoff for rootless, container-friendly
operation. Consumer-relevant types we don't cover are either redundant with
other collectors (processor details → `cpu`, PCI slots → planned `pci`, DIMMs →
partially in `memory`) or niche enough that no consumer has asked (cooling
devices, PSU, OEM strings). A dmidecode-based opt-in path can be added via
`internal/executor` if a consumer needs it.

## Backing library

- [github.com/jaypipes/ghw][ghw] — `pkg/bios`, `pkg/baseboard`, `pkg/chassis`,
  `pkg/product`. ghw reads `/sys/class/dmi/id/*` directly without shelling out.

[ghw]: https://github.com/jaypipes/ghw
