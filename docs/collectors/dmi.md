# DMI

> **Status:** Implemented ✅

## Description

The collector reports SMBIOS and DMI data: the BIOS, baseboard, chassis and
product identity. On Linux, the collector reads `/sys/class/dmi/id/*` through
its filesystem. Most fields do not need root. `product_serial` and
`product_uuid` have mode 0400, so they are `"unknown"` for a caller that is not
root. macOS has no equivalent of SMBIOS. On macOS, the collector returns an
empty Info, and the [`hardware`](hardware.md) collector gives the macOS hardware
identity.

The main consumers are the cloud-provider collectors (`gce`, `ec2`, `azure` and
others). They use `product.name`, `product.vendor_name`, `bios.manufacturer` and
`chassis.asset_tag` to find the cloud of a VM. They do this before they send a
request to the metadata endpoint of that provider. Hardware inventory and
compliance tools use all the fields to audit a fleet.

## Collected fields

On Linux, each section is always there. A field that the host does not expose,
such as the chassis data of many virtual machines, reads `unknown`. On macOS,
which has no SMBIOS equivalent, each section is nil. Thus a consumer that runs
on both must check `facts.DMI.Product != nil` before it dereferences the
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
| Linux    | ✅ with `/sys/class/dmi/id/*`                  |
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
gohai reads `/sys/class/dmi/id/*` instead. All users can read these sysfs
entries, so they do not need more privileges. This difference from Ohai is
intentional. On Linux, sysfs is the current way to read SMBIOS data without
privileges. The fields are the same: the BIOS vendor, version and date, the
baseboard, the chassis and the product. But gohai does not parse the free-form
text output of `dmidecode`.

On Linux:

1. The collector reads each `/sys/class/dmi/id/*` file (`bios_vendor`,
   `bios_version`, `bios_date`, `board_*`, `chassis_*`, `product_*`,
   `sys_vendor`) through its filesystem. A file that is not there, or a
   root-only field (`product_serial`, `product_uuid`) for a caller that is not
   root, gives `"unknown"`. A file that fails to read for any other reason is an
   error, so that `"unknown"` does not stand for a read that failed.
   `chassis_type` is an SMBIOS code, and the SMBIOS table gives its
   `type_description`, for example `10` → `Notebook`.
2. A board with no `/sys/class/dmi/id` but a device tree under
   `/sys/firmware/devicetree/base`, such as a Raspberry Pi, gets what the tree
   has: `model`, `serial-number`, the vendor of the first `compatible` entry,
   `chassis-type`, and U-Boot as the BIOS when `chosen/u-boot,version` is there.
   The other fields are `"unknown"`.

macOS does not make SMBIOS or DMI data available. The Linux DMI driver has no
Darwin equivalent. The collector returns an empty Info, so consumers can check
`facts.DMI.Product` without a branch for each OS. The [`hardware`](hardware.md)
collector gives the macOS hardware identity through IOKit, `ioreg` and
`system_profiler`.

Ohai's `dmi` plugin runs `dmidecode`. `dmidecode` reads `/dev/mem` and gives the
full set of SMBIOS records for all 128+ DMI types, but it needs root. The sysfs
method of gohai gives only the four types that sysfs makes available: BIOS,
baseboard, chassis and product. This is an intentional tradeoff, so that gohai
operates without root and in containers. Some useful types are not in this
collector. Other collectors already give some of them: processor details are in
`cpu`, PCI slots are in [`pci`](pci.md), and DIMMs are partially in `memory`. No
consumer has asked for the others (cooling devices, PSU, OEM strings). If a
consumer needs them, an opt-in path through `dmidecode` and `source.Runner` is
possible.

## Backing library

None. The collector reads sysfs through its filesystem.
