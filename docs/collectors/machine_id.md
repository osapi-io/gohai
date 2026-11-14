# Machine ID

> **Status:** Implemented ✅

## Description

The collector reports a stable host identifier. The identifier does not change
after a reboot. On Linux, the main source is `/etc/machine-id` (systemd). The
fallback is `/var/lib/dbus/machine-id`, for Debian and Ubuntu hosts that are
older than systemd. On macOS, the source is `IOPlatformUUID` from IOKit. Apple
makes this value the authoritative hardware identifier.

Consumers use this data to:

- Make a stable index of host identities. The index does not change after
  reboots, IP changes or hostname changes.
- Compare the asset inventory with events from the operating system.
- Remove duplicate hosts that have more than one hostname.

Limits:

- Some hosts have **none** of the expected sources, for example very minimal
  container images or systems that boot from initramfs. On these hosts, gopsutil
  can fall back to `/proc/sys/kernel/random/boot_id`. **That value changes at
  each reboot. Do not use it as a stable identifier.** If the ID is different
  after two reboots in sequence, treat it as unknown.
- On Linux, if `/etc/machine-id` and DMI product_uuid are both missing, the
  collector also reads `/var/lib/dbus/machine-id`. gopsutil does not read it. If
  that file is also missing, the collector returns an empty value.

## Collected fields

| Field | Type   | Description                                    | Schema mapping                                                                                                                                                              |
| ----- | ------ | ---------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `id`  | string | The stable machine identifier, usually a UUID. | OCSF `device.uid`, the canonical OCSF host-identifier field (dictionary: "unique device identifier. Typically BIOS hardware identifier, machine id, or agent identifier."). |

## Platform support

| Platform | Supported |
| -------- | --------- |
| Linux    | ✅        |
| macOS    | ✅        |
| Other    | None      |

## Example output

### Linux (systemd host)

```json
{
  "machine_id": {
    "id": "abc123def4567890abcdef1234567890"
  }
}
```

### macOS

```json
{
  "machine_id": {
    "id": "12345678-ABCD-EF01-2345-6789ABCDEF01"
  }
}
```

## SDK usage

```go
import (
    "context"

    "github.com/osapi-io/gohai/pkg/gohai"
)

g, _ := gohai.New(gohai.WithCollectors("machine_id"))
facts, _ := g.Collect(context.Background())

fmt.Println(facts.MachineID.ID)
```

## Enable/disable

```bash
gohai collect --collector.machine_id      # enable (default)
gohai collect --no-collector.machine_id   # disable
```

## Dependencies

None.

## Data sources

On Linux:

1. gopsutil's `host.InfoWithContext` sets `HostID` from a sequence of sources.
   It tries `/etc/machine-id` first, then DMI `product_uuid`, then
   `/proc/sys/kernel/random/boot_id`. The collector reports that value as `id`.
2. If gopsutil returns an empty value, the collector reads
   `/var/lib/dbus/machine-id` through the injected `avfs.VFS`. This is the last
   fallback, for minimal hosts and hosts older than systemd. It is the same as
   the dbus fallback in Ohai's `linux/machineid.rb`.

On macOS:

1. gopsutil's `host.InfoWithContext` reads `IOPlatformUUID` through IOKit and
   returns it as `HostID`. The collector reports it as `id` with no change, so
   the shape is the same as on Linux. Ohai has no `:darwin` `machineid` handler.
   Its darwin plugin reports the same UUID under `hardware.uuid`.

The last source of gopsutil, `boot_id`, changes at each reboot. If a consumer
sees a new value after a reboot, it must treat the value as "unknown", not as a
different host. The Description section also tells this.

## Backing library

- [`github.com/shirou/gopsutil/v4/host`](https://github.com/shirou/gopsutil) for
  the primary read of the machine ID and DMI. On Linux, the
  `/var/lib/dbus/machine-id` fallback is a gohai `os.ReadFile` call.
