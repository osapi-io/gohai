# Hardware

> **Status:** Implemented ✅

## Description

Collects the macOS hardware facts `system_profiler` surfaces — machine identity
(model, serial, platform UUID), CPU labels (Intel `cpu_type` or Apple Silicon
`chip_type`), memory, firmware versions, attached storage volumes, battery, and
AC charger. Mirrors Ohai's `darwin/hardware.rb` methodology: three
`system_profiler` invocations — `SPHardwareDataType`, `SPStorageDataType`,
`SPPowerDataType` — run concurrently and merged into a typed Info.

**This collector is macOS-only by design.** Linux doesn't need it — the
equivalent identity / CPU / memory / storage facts are already split across
`dmi`, `cpu`, `memory`, `disk`, `filesystem`, and `pci`. On macOS those
abstractions don't exist (no SMBIOS, no `/sys`), so every fact funnels through
`system_profiler`. `hardware` is the Darwin equivalent of Linux's `dmi` plus
battery / charger. Non-Darwin platforms return an empty Info.

## Collected Fields

| Field                     | Type        | Description                                                                                                      | Schema mapping                      |
| ------------------------- | ----------- | ---------------------------------------------------------------------------------------------------------------- | ----------------------------------- |
| `machine_model`           | `string`    | Model identifier (`MacBookPro18,2`, `Macmini9,1`).                                                               | No direct schema mapping.           |
| `machine_name`            | `string`    | Marketing name (`MacBook Pro`, `Mac mini`).                                                                      | No direct schema mapping.           |
| `serial_number`           | `string`    | Hardware serial.                                                                                                 | OCSF `device.hw_info.serial_number` |
| `platform_uuid`           | `string`    | Platform UUID from IOKit.                                                                                        | No direct schema mapping.           |
| `provisioning_udid`       | `string`    | Apple provisioning UDID.                                                                                         | No direct schema mapping.           |
| `cpu_type`                | `string`    | CPU label on Intel Macs (`Intel Core i7`).                                                                       | OTel `host.cpu.model.name`          |
| `chip_type`               | `string`    | CPU label on Apple Silicon (`Apple M1 Pro`). Intel Macs leave this empty; Apple Silicon leaves `cpu_type` empty. | OTel `host.cpu.model.name`          |
| `current_processor_speed` | `string`    | Clock speed string (Intel only; Apple Silicon omits).                                                            | No direct schema mapping.           |
| `number_processors`       | `string`    | Intel: `"2"`. Apple Silicon: `"proc 10:8:2"` (total:performance:efficiency).                                     | No direct schema mapping.           |
| `packages`                | `int`       | Physical CPU sockets (Intel only).                                                                               | No direct schema mapping.           |
| `l2_cache_core`           | `string`    | Per-core L2 cache (Intel, string with unit).                                                                     | No direct schema mapping.           |
| `l3_cache`                | `string`    | Shared L3 cache (Intel, string with unit).                                                                       | No direct schema mapping.           |
| `physical_memory`         | `string`    | RAM size string (`"16 GB"`). Verbatim from `system_profiler`.                                                    | No direct schema mapping.           |
| `boot_rom_version`        | `string`    | Boot ROM / UEFI version.                                                                                         | No direct schema mapping.           |
| `os_loader_version`       | `string`    | Apple Silicon OS loader version (absent on Intel).                                                               | No direct schema mapping.           |
| `smc_version_system`      | `string`    | SMC firmware version (Intel; Apple Silicon has no SMC).                                                          | No direct schema mapping.           |
| `storage`                 | `[]Storage` | Attached logical volumes. See below.                                                                             | No direct schema mapping.           |
| `battery`                 | `*Battery`  | Battery details (nil on desktop Macs without a battery). See below.                                              | No direct schema mapping.           |
| `charger`                 | `*Charger`  | AC charger info when a charger is connected. See below.                                                          | No direct schema mapping.           |

### Storage

| Field          | Type     | Description                                                   |
| -------------- | -------- | ------------------------------------------------------------- |
| `name`         | `string` | Volume name (`Macintosh HD`).                                 |
| `bsd_name`     | `string` | Device node (`disk1s1`).                                      |
| `capacity`     | `int64`  | Size in bytes.                                                |
| `file_system`  | `string` | `APFS`, `HFS+`, etc.                                          |
| `mount_point`  | `string` | Mount path.                                                   |
| `free_space`   | `int64`  | Free bytes.                                                   |
| `writable`     | `bool`   | Whether the volume is mounted read-write.                     |
| `drive_type`   | `string` | `ssd` / `hdd` (CoreStorage only — empty on APFS).             |
| `smart_status` | `string` | SMART verdict (`Verified`, CoreStorage only — empty on APFS). |
| `partitions`   | `int`    | Partition count (CoreStorage only — empty on APFS).           |

### Battery

| Field                | Type     | Description                                                     |
| -------------------- | -------- | --------------------------------------------------------------- |
| `current_capacity`   | `int`    | Current charge (mAh).                                           |
| `max_capacity`       | `int`    | Design capacity (mAh).                                          |
| `fully_charged`      | `bool`   | True when charger has topped off the battery.                   |
| `is_charging`        | `bool`   | True when currently drawing from the charger.                   |
| `charge_cycle_count` | `int`    | Lifetime charge cycles.                                         |
| `health`             | `string` | Apple's health verdict (`Good`, `Poor`, `Service Recommended`). |
| `serial`             | `string` | Battery serial number.                                          |
| `remaining`          | `int`    | Percent remaining, computed as `current / max * 100`.           |
| `amperage`           | `int`    | Current draw / charge rate in milliamps (positive = charging).  |
| `voltage`            | `int`    | Terminal voltage in millivolts.                                 |

### Charger

| Field           | Type     | Description                                             |
| --------------- | -------- | ------------------------------------------------------- |
| `id`            | `string` | Charger identifier (manufacturer-specific hex string).  |
| `family`        | `string` | Charger family identifier.                              |
| `revision`      | `string` | Charger firmware revision.                              |
| `serial_number` | `string` | Charger serial.                                         |
| `watts`         | `string` | Rated wattage (verbatim string from `system_profiler`). |
| `connected`     | `bool`   | Whether the charger is currently plugged in.            |

## Platform Support

| Platform | Supported                                                                |
| -------- | ------------------------------------------------------------------------ |
| Linux    | ❌ (use `dmi`, `cpu`, `memory`, `disk`, `filesystem` collectors instead) |
| macOS    | ✅                                                                       |

## Example Output

```json
{
  "hardware": {
    "machine_model": "MacBookPro18,2",
    "machine_name": "MacBook Pro",
    "serial_number": "F5K123ABC",
    "platform_uuid": "00000000-1111-2222-3333-444444444444",
    "chip_type": "Apple M1 Pro",
    "number_processors": "proc 10:8:2",
    "physical_memory": "32 GB",
    "boot_rom_version": "10151.101.3",
    "storage": [
      {
        "name": "Macintosh HD",
        "bsd_name": "disk1s1",
        "capacity": 249661751296,
        "file_system": "APFS",
        "mount_point": "/",
        "writable": true
      }
    ],
    "battery": {
      "current_capacity": 5841,
      "max_capacity": 5841,
      "fully_charged": true,
      "charge_cycle_count": 201,
      "health": "Good",
      "remaining": 100
    },
    "charger": {
      "watts": "96",
      "connected": true
    }
  }
}
```

## SDK Usage

```go
g, _ := gohai.New(gohai.WithCollectors("hardware"))
facts, _ := g.Collect(context.Background())

if h := facts.Hardware; h != nil {
    fmt.Println(h.MachineModel, h.SerialNumber)
    if h.Battery != nil {
        fmt.Printf("battery: %d%%\n", h.Battery.Remaining)
    }
}
```

## Enable/Disable

```bash
gohai --collector.hardware      # enable (opt-in)
gohai --no-collector.hardware   # disable (default)
gohai --category=hardware       # pulls this + all hardware collectors
```

## Dependencies

None.

## Data Sources

On macOS:

1. Three `system_profiler ... -json` invocations run concurrently through the
   shared `internal/executor` runner:
   - `SPHardwareDataType` → machine identity, CPU labels, memory, firmware
     versions. Fields are merged from `_items[0]` into the top-level Info.
   - `SPStorageDataType` → one `Storage` entry per logical volume.
     CoreStorage-era fields (`drive_type`, `smart_status`, `partitions`) come
     from the `com.apple.corestorage.pv` array when present; on modern APFS
     volumes that key is absent so those fields stay empty.
   - `SPPowerDataType` → iterated for `_name == "spbattery_information"`
     (populates `battery`) and `_name == "sppower_ac_charger_information"`
     (populates `charger`). Desktop Macs without a battery simply have no
     matching item, leaving `battery` nil rather than zero-valued.
2. System_profiler's string-boolean convention (`"TRUE"` / `"FALSE"`) is
   normalized to real Go bools. Numeric fields that the plist sometimes encodes
   as strings (e.g. `cycle_count: "201"`) are parsed via a helper that accepts
   both JSON numbers and numeric strings.
3. `battery.remaining` is computed as `current_capacity * 100 / max_capacity`
   when `max_capacity > 0` — matches Ohai's `(current / max * 100).to_i`.
4. Each of the three calls is tolerated independently: missing binary, exec
   error, or malformed JSON in one invocation leaves only that section empty.
   This keeps the collector useful on hosts where one `SP*DataType` mode fails
   but others succeed.

On Linux the collector returns an empty Info. Ohai's `darwin/hardware.rb` is
Darwin-only and Linux gets equivalent coverage from the `dmi`, `cpu`, `memory`,
`disk`, and `filesystem` collectors.

### Deviations from Ohai

- **Skip `architecture`, `operating_system`, `operating_system_version`,
  `build_version`** — Ohai merges these into `hardware{}` too, but they're
  already collected by gohai's `kernel`, `platform`, and `os_release`
  collectors. Duplicating them would violate the single-source principle.
- **Surface AC charger info** — Ohai reads `SPPowerDataType` only for the
  battery item. We additionally surface the `sppower_ac_charger_information`
  item because fleet management cares about adapter wattage mismatches.
- **Surface `amperage` / `voltage`** — Ohai ignores the raw
  `sppower_current_amperage` and `sppower_current_voltage` fields in the battery
  item; we expose them for consumers that want charge-rate signals.

## Backing library

- [`internal/executor`](../../internal/executor) — shared command-runner
  abstraction used to invoke `system_profiler`. Tests mock it with
  `go.uber.org/mock`.
