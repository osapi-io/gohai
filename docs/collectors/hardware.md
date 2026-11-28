# Hardware

> **Status:** Implemented ✅

## Description

The collector reports the macOS hardware facts that `system_profiler` gives.
These are the machine identity (model, serial, platform UUID), the CPU labels
(Intel `cpu_type` or Apple Silicon `chip_type`), the memory, the firmware
versions, the attached storage volumes, the battery and the AC charger. The
collector uses the same method as Ohai's `darwin/hardware.rb`. It runs
`system_profiler` three times at the same time, with `SPHardwareDataType`,
`SPStorageDataType` and `SPPowerDataType`. Then it merges the results into a
typed `Info`.

**This collector is for macOS only, by design.** Linux does not need it. On
Linux, the `dmi`, `cpu`, `memory`, `disk`, `filesystem` and `pci` collectors
already report the equivalent identity, CPU, memory and storage facts. macOS
does not have the sources of those collectors (no SMBIOS, no `/sys`). Thus all
facts on macOS come from `system_profiler`. `hardware` is the Darwin equivalent
of the Linux `dmi` collector, with the battery and the charger added. On other
platforms, the collector returns an empty `Info`.

## Collected fields

| Field                     | Type        | Description                                                                                                                                         | Schema mapping                      |
| ------------------------- | ----------- | --------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------- |
| `machine_model`           | `string`    | The model identifier, for example `MacBookPro18,2` or `Macmini9,1`.                                                                                 | No direct schema mapping.           |
| `machine_name`            | `string`    | The marketing name, for example `MacBook Pro` or `Mac mini`.                                                                                        | No direct schema mapping.           |
| `serial_number`           | `string`    | The hardware serial.                                                                                                                                | OCSF `device.hw_info.serial_number` |
| `platform_uuid`           | `string`    | The platform UUID, from IOKit.                                                                                                                      | No direct schema mapping.           |
| `provisioning_udid`       | `string`    | The Apple provisioning UDID.                                                                                                                        | No direct schema mapping.           |
| `cpu_type`                | `string`    | The CPU label on Intel Macs, for example `Intel Core i7`.                                                                                           | OTel `host.cpu.model.name`          |
| `chip_type`               | `string`    | The CPU label on Apple Silicon, for example `Apple M1 Pro`. Empty on Intel Macs. On Apple Silicon, `cpu_type` is empty.                             | OTel `host.cpu.model.name`          |
| `current_processor_speed` | `string`    | The clock speed, as a string. Intel only. Apple Silicon does not have it.                                                                           | No direct schema mapping.           |
| `number_processors`       | `string`    | Intel: `"2"`. Apple Silicon: the string that `system_profiler` gives, for example `"proc 10:8:2:0"` on macOS 26.5. The collector does not parse it. | No direct schema mapping.           |
| `packages`                | `int`       | The number of physical CPU sockets. Intel only.                                                                                                     | No direct schema mapping.           |
| `l2_cache_core`           | `string`    | The L2 cache of each core. Intel only. A string with the unit.                                                                                      | No direct schema mapping.           |
| `l3_cache`                | `string`    | The shared L3 cache. Intel only. A string with the unit.                                                                                            | No direct schema mapping.           |
| `physical_memory`         | `string`    | The RAM size as a string, for example `"16 GB"`. The value is as `system_profiler` gives it.                                                        | No direct schema mapping.           |
| `boot_rom_version`        | `string`    | The Boot ROM or UEFI version.                                                                                                                       | No direct schema mapping.           |
| `os_loader_version`       | `string`    | The OS loader version on Apple Silicon. Not there on Intel.                                                                                         | No direct schema mapping.           |
| `smc_version_system`      | `string`    | The SMC firmware version. Intel only, because Apple Silicon has no SMC.                                                                             | No direct schema mapping.           |
| `storage`                 | `[]Storage` | The attached logical volumes. See below.                                                                                                            | No direct schema mapping.           |
| `battery`                 | `*Battery`  | The battery data. `nil` on desktop Macs that have no battery. See below.                                                                            | No direct schema mapping.           |
| `charger`                 | `*Charger`  | The AC charger data, if a charger is connected. See below.                                                                                          | No direct schema mapping.           |

### Storage

| Field          | Type     | Description                                                                |
| -------------- | -------- | -------------------------------------------------------------------------- |
| `name`         | `string` | The volume name, for example `Macintosh HD`.                               |
| `bsd_name`     | `string` | The device node, for example `disk1s1`.                                    |
| `capacity`     | `int64`  | The size in bytes.                                                         |
| `file_system`  | `string` | For example `APFS` or `HFS+`.                                              |
| `mount_point`  | `string` | The mount path.                                                            |
| `free_space`   | `int64`  | The free space in bytes.                                                   |
| `writable`     | `bool`   | True if the volume is mounted read-write.                                  |
| `drive_type`   | `string` | `ssd` or `hdd`. CoreStorage only. Empty on APFS.                           |
| `smart_status` | `string` | The SMART result, for example `Verified`. CoreStorage only. Empty on APFS. |
| `partitions`   | `int`    | The number of partitions. CoreStorage only. Empty on APFS.                 |

### Battery

| Field                | Type     | Description                                                                                                                                                                            |
| -------------------- | -------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `current_capacity`   | `int`    | The current charge (mAh). Not in the output when `system_profiler` does not give it. macOS 26.5 does not give it.                                                                      |
| `max_capacity`       | `int`    | The design capacity (mAh). Not in the output when `system_profiler` does not give it. macOS 26.5 does not give it.                                                                     |
| `fully_charged`      | `bool`   | True if the charger has fully charged the battery.                                                                                                                                     |
| `is_charging`        | `bool`   | True if the battery receives current from the charger now.                                                                                                                             |
| `charge_cycle_count` | `int`    | The number of charge cycles in the life of the battery.                                                                                                                                |
| `health`             | `string` | The health result from Apple, for example `Good`, `Poor` or `Service Recommended`.                                                                                                     |
| `serial`             | `string` | The battery serial number.                                                                                                                                                             |
| `remaining`          | `int`    | The percent of charge that remains. The collector calculates it as `current / max * 100`, only when `max_capacity` is more than 0. Thus it is not in the output on macOS 26.5.         |
| `amperage`           | `int`    | The current draw or charge rate in milliamps. A positive value shows that the battery charges. Not in the output when `system_profiler` does not give it. macOS 26.5 does not give it. |
| `voltage`            | `int`    | The terminal voltage in millivolts. Not in the output when `system_profiler` does not give it. macOS 26.5 does not give it.                                                            |

### Charger

| Field           | Type     | Description                                                                |
| --------------- | -------- | -------------------------------------------------------------------------- |
| `id`            | `string` | The charger identifier. A hex string that is specific to the manufacturer. |
| `family`        | `string` | The charger family identifier.                                             |
| `revision`      | `string` | The charger firmware revision.                                             |
| `serial_number` | `string` | The charger serial.                                                        |
| `watts`         | `string` | The rated wattage. The string is as `system_profiler` gives it.            |
| `connected`     | `bool`   | True if the charger is connected now.                                      |

## Platform support

| Platform | Supported                                                                                                   |
| -------- | ----------------------------------------------------------------------------------------------------------- |
| Linux    | No fact, with the status `unsupported` (use the `dmi`, `cpu`, `memory`, `disk` and `filesystem` collectors) |
| macOS    | ✅                                                                                                          |

## Example output

This is from a MacBook Pro with macOS 26.5. The identifiers are changed.

```json
{
  "hardware": {
    "machine_model": "MacBookPro18,1",
    "machine_name": "MacBook Pro",
    "serial_number": "F5K123ABC",
    "platform_uuid": "00000000-1111-2222-3333-444444444444",
    "provisioning_udid": "00006000-000000000000001E",
    "chip_type": "Apple M1 Pro",
    "number_processors": "proc 10:8:2:0",
    "physical_memory": "32 GB",
    "boot_rom_version": "18000.121.3",
    "os_loader_version": "18000.121.3",
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
      "fully_charged": true,
      "charge_cycle_count": 207,
      "health": "Good",
      "serial": "F8Y0000000000000A"
    },
    "charger": {
      "id": "0x1656",
      "family": "0xe000400a",
      "watts": "86",
      "connected": true
    }
  }
}
```

## SDK usage

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

## Enable/disable

```bash
gohai collect --collector.hardware      # enable (opt-in)
gohai collect --no-collector.hardware   # disable (default)
gohai collect --category=hardware       # pulls this + all hardware collectors
```

## Dependencies

None.

## Data sources

On macOS:

1. The collector runs `system_profiler ... -json` three times at the same time,
   through the shared `source.Runner`:
   - `SPHardwareDataType` gives the machine identity, the CPU labels, the memory
     and the firmware versions. The collector merges the fields from `_items[0]`
     into the top-level `Info`.
   - `SPStorageDataType` gives one `Storage` entry for each logical volume. The
     fields from the CoreStorage era (`drive_type`, `smart_status`,
     `partitions`) come from the `com.apple.corestorage.pv` array, if it is
     there. Modern APFS volumes do not have that key, so those fields stay
     empty.
   - `SPPowerDataType`. The collector looks at each item for
     `_name == "spbattery_information"`, which sets `battery`, and
     `_name == "sppower_ac_charger_information"`, which sets `charger`. A
     desktop Mac that has no battery has no item that matches. Thus `battery`
     stays `nil`, not zero-valued.
2. `system_profiler` writes booleans as the strings `"TRUE"` and `"FALSE"`. The
   collector changes them to real Go bools. The plist sometimes writes numeric
   fields as strings, for example `cycle_count: "201"`. A helper parses these
   fields. The helper accepts JSON numbers and numeric strings.
3. If `max_capacity > 0`, the collector calculates `battery.remaining` as
   `current_capacity * 100 / max_capacity`. This is the same as Ohai's
   `(current / max * 100).to_i`.
4. A host without `system_profiler` gives an empty `Info`. A call that fails, or
   output that does not parse as JSON, adds its error, and the parts of the
   other calls are kept, so the status is `partial` and a caller can tell a
   failed read from a host that has nothing to report. When no call worked, or
   the context is cancelled, there is no fact. A host without a battery answers
   the power call with no battery, and that is not a failure.

On Linux, the collector gives no fact, with the status `unsupported`. Ohai's
`darwin/hardware.rb` is for Darwin only. On Linux, the `dmi`, `cpu`, `memory`,
`disk` and `filesystem` collectors give equivalent data.

### Deviations from Ohai

- **No `architecture`, `operating_system`, `operating_system_version` or
  `build_version`.** Ohai also merges these into `hardware{}`. But gohai's
  `kernel`, `platform` and `os_release` collectors already report them. A copy
  here breaks the single-source principle.
- **The AC charger data.** Ohai reads `SPPowerDataType` only for the battery
  item. gohai also reports the `sppower_ac_charger_information` item, because
  fleet management must find adapters with the wrong wattage.
- **`amperage` and `voltage`.** Ohai ignores the raw `sppower_current_amperage`
  and `sppower_current_voltage` fields in the battery item. gohai reports them
  for consumers that want charge-rate data.

## Backing library

- [`source.Runner`](../../pkg/gohai/source). The shared command runner. The
  collector uses it to run `system_profiler`. Tests mock it with
  `go.uber.org/mock`.
