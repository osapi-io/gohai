# Shard

> **Status:** Implemented ✅

## Description

Derives a deterministic shard seed from stable host identity. Matches Ohai's
`shard` plugin algorithm: concatenate `machinename + DMI serial + DMI uuid`, MD5
hash, take the first 7 hex characters as a base-16 integer. The same host always
maps to the same bucket; different hosts distribute evenly.

Consumers use this to:

- Stagger scheduled work across a fleet (`seed % 60` for minute-of-hour,
  `seed % 7` for day-of-week, etc.) without a per-host schedule file.
- Distribute work across parallel pipelines without central coordination.
- Pick a canary host deterministically (`seed == 0 mod N`).

## Collected Fields

| Field  | Type  | Description                                                                                                                                     | Schema mapping                                               |
| ------ | ----- | ----------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------ |
| `seed` | `int` | First 7 hex chars of MD5(`machinename + serial + uuid`) interpreted as base-16 integer. Stable across reboots when DMI identity doesn't change. | No direct schema mapping — shard is a gohai/Ohai convention. |

## Platform Support

| Platform | Supported                                                                             |
| -------- | ------------------------------------------------------------------------------------- |
| Linux    | ✅ (inputs: machinename from hostname + serial/uuid from DMI)                         |
| macOS    | ✅ (inputs: machinename from hostname + serial from system_profiler + IOPlatformUUID) |

## Example Output

```json
{
  "shard": {
    "seed": 27767217
  }
}
```

## SDK Usage

```go
g, _ := gohai.New(gohai.WithCollectors("shard"))
facts, _ := g.Collect(context.Background())

bucket := facts.Shard.Seed % 60
fmt.Printf("this host runs at minute %d of every hour\n", bucket)
```

## Enable/Disable

```bash
gohai --collector.shard      # enable (default)
gohai --no-collector.shard   # disable
```

## Dependencies

`hostname`, `dmi`. The shard seed is derived from `hostname.Info.MachineName`
plus DMI serial number and UUID from `dmi.Info`. On macOS where DMI is empty,
serial comes from `system_profiler SPHardwareDataType` and UUID from
IOPlatformUUID via gopsutil.

## Data Sources

On Linux:

1. `machinename` is read from the `hostname` prior result
   (`hostname.Info.MachineName`).
2. `serial` cascades through DMI sections from the `dmi` prior result:
   `Product.SerialNumber` → `Baseboard.SerialNumber` → `Chassis.SerialNumber`,
   taking the first non-blank value. Mirrors Ohai's `get_dmi_property` fallback
   chain (system → base_board → chassis).
3. `uuid` is read from `dmi.Info.Product.UUID`.
4. The three strings are concatenated with no separator and MD5-hashed. The
   first 7 hex characters of the digest are interpreted as a base-16 integer.
   Matches Ohai's `hexdigest(data)[0...7].to_i(16)`.

On macOS:

1. `machinename` is read from the `hostname` prior result (same as Linux).
2. `serial` comes from `system_profiler SPHardwareDataType -json` via the shared
   `internal/executor` runner — matching Ohai's `hardware["serial_number"]`
   source. Error or missing data leaves serial empty.
3. `uuid` comes from gopsutil's `host.InfoWithContext` → `HostID`
   (IOPlatformUUID). Matches Ohai's `hardware["platform_UUID"]`. Error leaves
   uuid empty.
4. Same MD5 → 7-hex-char → integer computation as Linux.

Mirrors Ohai's `shard.rb` algorithm with the same default sources
(`machinename`, `serial`, `uuid`) and the same MD5 → integer output. We do not
implement Ohai's per-plugin `[:sources]` configurability or FIPS-aware digest
selection (SHA-256 under FIPS) — those can be added if a consumer needs them.

## Backing library

- Go stdlib (`crypto/md5`, `fmt`, `strconv`) for the hash computation.
- [`github.com/shirou/gopsutil/v4/host`](https://github.com/shirou/gopsutil) for
  IOPlatformUUID on macOS.
- [`internal/executor`](../../internal/executor) for `system_profiler` on macOS.
