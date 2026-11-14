# DMI

> **Status:** Implemented ✅

## Description

The collector reports SMBIOS and DMI data: the BIOS, baseboard, chassis and
product identity. On Linux, the data comes from `/sys/class/dmi/id/*` through
[ghw]. ghw reads the sysfs entries that the kernel makes available. Most fields
do not need root. `product_serial` and `product_uuid` have mode 0400, so they
are empty for a caller that is not root. macOS has no equivalent of SMBIOS. On
macOS, the collector returns an empty Info, and the `hardware` collector
(planned) gives the macOS hardware identity.

The main consumers are the cloud-provider collectors (`gce`, `ec2`, `azure` and
others). They use `product.name`, `product.vendor_name`, `bios.manufacturer` and
`chassis.asset_tag` to find the cloud of a VM. They do this before they send a
request to the metadata endpoint of that provider. Hardware inventory and
compliance tools use all the fields to audit a fleet.

## Collected fields

Each section is nil when that part of SMBIOS is not available. Virtual machines
frequently have no chassis data. Minimal containers can have no DMI data. A
consumer must check `facts.DMI.Product != nil` before it dereferences the
section.

| Field       | Type         | Description               | Schema mapping                      |
| ----------- | ------------ | ------------------------- | ----------------------------------- |
| `bios`      | `*BIOS`      | The firmware identity.    | OCSF `device.hw_info` (BIOS subset) |
| `baseboard` | `*Baseboard` | The motherboard identity. | OCSF `device.hw_info`               |
| `chassis`   | `*Chassis`   | The enclosure identity.   | OCSF `device.hw_info`               |
| `product`   | `*Product`   | The system identity.      | OCSF `device.hw_info`               |

### BIOS

| Field          | Type     | Description            | Schema mapping                          |
| -------------- | -------- | ---------------------- | --------------------------------------- |
| `manufacturer` | `string` | The BIOS vendor.       | OCSF `device.hw_info.bios_manufacturer` |
| `ver`          | `string` | The BIOS version.      | OCSF `device.hw_info.bios_ver`          |
| `date`         | `string` | The BIOS release date. | OCSF `device.hw_info.bios_date`         |

### Baseboard

| Field           | Type     | Description                    |
| --------------- | -------- | ------------------------------ |
| `vendor`        | `string` | The motherboard vendor.        |
| `product`       | `string` | The motherboard model.         |
| `version`       | `string` | The motherboard version.       |
| `serial_number` | `string` | The motherboard serial number. |
| `asset_tag`     | `string` | The asset tag.                 |

### Chassis

| Field              | Type     | Description                                            |
| ------------------ | -------- | ------------------------------------------------------ |
| `vendor`           | `string` | The chassis vendor.                                    |
| `type`             | `string` | The numeric DMI code of the chassis type.              |
| `type_description` | `string` | The chassis type as text.                              |
| `version`          | `string` | The chassis version.                                   |
| `serial_number`    | `string` | The chassis serial number.                             |
| `asset_tag`        | `string` | The asset tag, for example `"OracleCloud.com"` on OCI. |

### Product

| Field           | Type     | Description                                                                                                  |
| --------------- | -------- | ------------------------------------------------------------------------------------------------------------ |
| `vendor_name`   | `string` | The system vendor, for example `"Google"`, `"Amazon EC2"` or `"Dell Inc."`.                                  |
| `name`          | `string` | The product name. This is the main signal to find the cloud.                                                 |
| `family`        | `string` | The product family.                                                                                          |
| `version`       | `string` | The product version.                                                                                         |
| `serial_number` | `string` | The product serial number. On Linux it has mode 0400. Only root can read it, so it is empty for other users. |
| `uuid`          | `string` | The system UUID. On Linux it has mode 0400. Only root can read it, so it is empty for other users.           |
| `sku`           | `string` | The product SKU.                                                                                             |

## Platform support

| Platform | Supported                                      |
| -------- | ---------------------------------------------- |
| Linux    | ✅ with ghw and `/sys/class/dmi/id/*`          |
| macOS    | ✅ returns an empty Info (macOS has no SMBIOS) |
| Other    | ✅ returns an empty Info                       |

## Example output

### GCE VM

```json
{
  "dmi": {
    "bios": { "manufacturer": "Google", "ver": "Google", "date": "10/28/2024" },
    "baseboard": { "vendor": "Google", "product": "Google Compute Engine" },
    "chassis": { "vendor": "Google", "type": "1", "type_description": "Other" },
    "product": {
      "vendor_name": "Google",
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

## SDK usage

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

You can also enable the collector by category:

```go
gohai.New(gohai.WithCategory("hardware"))  // pulls dmi + cpu + memory + disk + filesystem
```

## Enable/disable

```bash
gohai collect --collector.dmi      # enable (opt-in)
gohai collect --no-collector.dmi   # disable (default)
gohai collect --category=hardware  # pulls dmi + other hardware collectors
```

The collector is opt-in because most consumers need DMI only indirectly. When a
consumer enables a cloud collector, `Dependencies()` enables dmi automatically.

## Dependencies

None.

## Data sources

Ohai's `dmi.rb` runs `dmidecode`. `dmidecode` reads `/dev/mem` and needs root.
gohai uses `ghw` instead. ghw reads `/sys/class/dmi/id/*`. All users can read
these sysfs entries, so they do not need more privileges. This difference from
Ohai is intentional. On Linux, sysfs is the current way to read SMBIOS data
without privileges. The fields are the same: the BIOS vendor, version and date,
the baseboard, the chassis and the product. But gohai does not parse the
free-form text output of `dmidecode`.

On Linux:

1. ghw's `bios.New()`, `baseboard.New()`, `chassis.New()` and `product.New()`
   each read their `/sys/class/dmi/id/*` files (`bios_vendor`, `bios_version`,
   `bios_date`, `board_*`, `chassis_*`, `product_*`, `sys_vendor`). Fields with
   mode 0444 do not need root. For a caller that is not root, the root-only
   fields (`product_serial`, `product_uuid`) are empty strings, not errors.
2. If ghw returns an error for one section, that section stays nil. Examples of
   errors are "permission denied" and a missing sysfs entry. The other sections
   still get their data. If a host has no `/sys/class/dmi/id/` (unprivileged
   containers), the collector returns an Info with all four sub-structs nil.

macOS does not make SMBIOS or DMI data available. The Linux DMI driver has no
Darwin equivalent. The collector returns an empty Info, so consumers can check
`facts.DMI.Product` without a branch for each OS. The planned `hardware`
collector will give the macOS hardware identity through IOKit, `ioreg` and
`system_profiler`.

Ohai's `dmi` plugin runs `dmidecode`. `dmidecode` reads `/dev/mem` and gives the
full set of SMBIOS records for all 128+ DMI types, but it needs root. The sysfs
method of gohai gives only the four types that sysfs makes available: BIOS,
baseboard, chassis and product. This is an intentional tradeoff, so that gohai
operates without root and in containers. Some useful types are not in this
collector. Other collectors already give some of them: processor details are in
`cpu`, PCI slots are in the planned `pci`, and DIMMs are partially in `memory`.
No consumer has asked for the others (cooling devices, PSU, OEM strings). If a
consumer needs them, an opt-in path through `dmidecode` and `internal/executor`
is possible.

## Backing library

- [github.com/jaypipes/ghw][ghw]. `pkg/bios`, `pkg/baseboard`, `pkg/chassis`,
  `pkg/product`. ghw reads `/sys/class/dmi/id/*` directly. It does not run a
  command.

[ghw]: https://github.com/jaypipes/ghw
