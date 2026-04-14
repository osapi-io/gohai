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

1. **Linux:** ghw's `bios.New()`, `baseboard.New()`, `chassis.New()`, and
   `product.New()` — each reads the corresponding `/sys/class/dmi/id/*` files
   (`bios_vendor`, `bios_version`, `bios_date`, `board_*`, `chassis_*`,
   `product_*`, `sys_vendor`). No root required for fields exposed as 0444;
   root-only fields (`product_serial`, `product_uuid`) return empty strings for
   non-root callers rather than erroring.
2. **macOS:** SMBIOS/DMI is Linux-specific (exposed through the kernel's DMI
   driver). macOS hardware identity lives in IOKit / `ioreg` / `system_profiler`
   and is covered by the `hardware` collector (planned). The dmi collector
   returns an empty Info so consumers can safely check `facts.DMI.Product`
   without a per-OS branch.
3. **Failure handling:** if ghw errors on any individual section (permission
   denied, sysfs entry missing), that section is nil in the result — other
   sections still populate. A completely empty host (container without
   `/sys/class/dmi/id/`) yields an Info with all four sub-structs nil.

## Methodology gap vs. Ohai

Ohai's `dmi` plugin shells out to `dmidecode` and parses its text output,
exposing the full SMBIOS record set across all DMI types (128+ types defined).
gohai reads `/sys/class/dmi/id/*` via ghw, which only covers four DMI types. The
tradeoff is deliberate — sysfs works in unprivileged containers, rootless
daemons, and locked-down environments where `dmidecode` would fail — but the
coverage gap is real.

**What we cover** (same as Ohai's output for these types):

- DMI type 0 — BIOS Information (`bios` field)
- DMI type 1 — System Information (`product` field)
- DMI type 2 — Base Board Information (`baseboard` field)
- DMI type 3 — Chassis Information (`chassis` field)

**What we don't cover** (`dmidecode` exposes these; sysfs doesn't):

| DMI type                  | Content Ohai surfaces                                                      | Consumer impact                                                                                   |
| ------------------------- | -------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------- |
| 4 — Processor Information | Socket type, voltage, max/current speed, family, manufacturer, part number | Covered by our `cpu` collector via ghw (reads `/sys/devices/system/cpu/`, richer than dmidecode). |
| 9 — System Slots          | PCI slot names and usage                                                   | Will be covered by the `pci` collector (planned, ghw-backed).                                     |
| 16 — Memory Array         | DIMM slot count, max capacity, ECC capability                              | No current consumer. Partial overlap with `memory` (gopsutil/ghw).                                |
| 17 — Memory Device        | DIMM slot location, size, speed, manufacturer, part number, serial         | No current consumer. ghw's memory package exposes some of this.                                   |
| 11 — OEM Strings          | Vendor-specific strings                                                    | Occasionally used by cloud providers as out-of-band tags. Low-priority gap.                       |
| 13 — BIOS Language        | Installed BIOS language codes                                              | No known consumer.                                                                                |
| 22 — Portable Battery     | Battery info on laptops                                                    | Laptops rarely run gohai; macOS has this via IOKit in the planned `hardware` collector.           |
| 41 — Onboard Devices Ext. | Ethernet / video controller locations                                      | Overlap with `network` (ghw) and `gpu` (planned).                                                 |
| 27 — Cooling Device       | Fan info                                                                   | Niche; no consumer.                                                                               |
| 39 — System Power Supply  | PSU info                                                                   | Niche; no consumer.                                                                               |

**Access-level differences:**

- Ohai requires root + `/dev/mem` to run `dmidecode`.
- gohai runs without root for most fields. `product_serial` and `product_uuid`
  are 0400 on Linux and return empty strings for non-root callers; everything
  else is 0444 and readable by anyone.

**If you need dmidecode's extra coverage** (DIMM part numbers, full processor
sockets, OEM strings), file an issue and we'll add a second code path that
shells out via `internal/executor` — same pattern other collectors use for
optional, root-requiring data sources.

## Backing library

- [github.com/jaypipes/ghw][ghw] — `pkg/bios`, `pkg/baseboard`, `pkg/chassis`,
  `pkg/product`. ghw reads `/sys/class/dmi/id/*` directly without shelling out.

[ghw]: https://github.com/jaypipes/ghw
