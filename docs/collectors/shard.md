# Shard

> **Status:** Implemented ✅

## Description

The collector calculates a deterministic shard seed from the stable identity of
the host. The algorithm is the same as Ohai's `shard` plugin in FIPS mode:

1. Join `machinename + DMI serial + DMI uuid`.
2. Make the SHA-256 hash of the result.
3. Read the first 7 hex characters as a base-16 integer.

The same host always gets the same bucket. Different hosts spread evenly across
the buckets.

Consumers use this data to:

- Spread scheduled work across a fleet without a schedule file for each host.
  For example, use `seed % 60` for the minute of the hour, or `seed % 7` for the
  day of the week.
- Divide work across parallel pipelines without central control.
- Select a canary host in a deterministic way (`seed == 0 mod N`).

## Collected fields

| Field  | Type  | Description                                                                                                                                                             | Schema mapping                                                  |
| ------ | ----- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------- |
| `seed` | `int` | The first 7 hex characters of SHA-256(`machinename + serial + uuid`), read as a base-16 integer. It does not change across reboots if the DMI identity does not change. | No direct schema mapping. Shard is a gohai and Ohai convention. |

## Platform support

| Platform | Supported                                                                           |
| -------- | ----------------------------------------------------------------------------------- |
| Linux    | ✅ (inputs: machinename from hostname, serial and uuid from DMI)                    |
| macOS    | ✅ (inputs: machinename from hostname, serial from system_profiler, IOPlatformUUID) |

## Example output

```json
{
  "shard": {
    "seed": 27767217
  }
}
```

## SDK usage

```go
g, _ := gohai.New(gohai.WithCollectors("shard"))
facts, _ := g.Collect(context.Background())

bucket := facts.Shard.Seed % 60
fmt.Printf("this host runs at minute %d of every hour\n", bucket)
```

## Enable/disable

```bash
gohai collect --collector.shard      # enable (default)
gohai collect --no-collector.shard   # disable
```

## Dependencies

`hostname`, `dmi`. The collector calculates the shard seed from
`hostname.Info.MachineName` and from the DMI serial number and UUID in
`dmi.Info`. On macOS, DMI is empty. There, the serial comes from
`system_profiler SPHardwareDataType`, and the UUID is the IOPlatformUUID from
`ioreg`.

## Data sources

On Linux:

1. The collector reads `machinename` from the earlier result of `hostname`
   (`hostname.Info.MachineName`).
2. The collector reads `serial` from the DMI sections in the earlier result of
   `dmi`, in this order: `Product.SerialNumber` → `Baseboard.SerialNumber` →
   `Chassis.SerialNumber`. It uses the first value that is not blank. This order
   is the same as the fallback chain of Ohai's `get_dmi_property` (system →
   base_board → chassis).
3. The collector reads `uuid` from `dmi.Info.Product.UUID`.
4. The collector joins the three strings with no separator and makes the SHA-256
   hash. It reads the first 7 hex characters of the digest as a base-16 integer.
   This is the same as Ohai's `hexdigest(data)[0...7].to_i(16)`.

On macOS:

1. The collector reads `machinename` from the earlier result of `hostname`, the
   same as on Linux.
2. `serial` comes from `system_profiler SPHardwareDataType -json` through the
   shared `source.Runner`. This is the same source as Ohai's
   `hardware["serial_number"]`. If there is an error or no data, serial stays
   empty.
3. `uuid` is the `IOPlatformUUID` line of
   `ioreg -rd1 -c IOPlatformExpertDevice`. This is the same as Ohai's
   `hardware["platform_UUID"]`. If there is an error, uuid stays empty.
4. The collector does the same SHA-256 → 7 hex characters → integer calculation
   as on Linux.

The algorithm is the same as Ohai's `shard.rb` in FIPS mode. It uses the same
default sources (`machinename`, `serial`, `uuid`) and the same SHA-256 → integer
output. Ohai uses MD5 when FIPS is off, so the seed of a host differs from
Ohai's seed there. gohai always uses SHA-256, because MD5 is not a FIPS 140-3
approved algorithm, and osapi's agent embeds gohai in FIPS 140-3 mode. gohai
cannot configure `[:sources]` for each plugin, as Ohai can. gohai can add it if
a consumer needs it.

## Backing library

- The Go standard library (`crypto/sha256`, `fmt`, `strconv`) to calculate the
  hash.
- [`source.Runner`](../../pkg/gohai/source) to run `system_profiler` and `ioreg`
  on macOS.
