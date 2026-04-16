# Machine ID

> **Status:** Implemented ✅

## Description

Reports a stable host identifier — one that survives reboots. On Linux the
source of truth is `/etc/machine-id` (systemd) with a `/var/lib/dbus/machine-id`
fallback for pre-systemd Debian/Ubuntu hosts. On macOS the source is
`IOPlatformUUID` from IOKit, which Apple intends as the authoritative hardware
identifier.

Consumers use this to:

- Build a stable host-identity index that survives reboots, IP changes, and
  hostname renames.
- Correlate asset inventory with OS-level events.
- Dedupe hosts that appear under multiple hostnames.

Caveats:

- On a host where **none** of the expected sources exist (very minimal container
  images, boot-from-initramfs systems), gopsutil may fall back to
  `/proc/sys/kernel/random/boot_id` — **that value changes every reboot and is
  not safe to use as a stable identifier.** Treat an ID that differs across two
  consecutive reboots as unknown.
- On Linux, if both `/etc/machine-id` and DMI product_uuid are missing, our
  collector also consults `/var/lib/dbus/machine-id` (which gopsutil doesn't)
  before giving up and returning empty.

## Collected Fields

| Field | Type   | Description                                 | Schema mapping                                                                                                                                                               |
| ----- | ------ | ------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `id`  | string | Stable machine identifier (typically UUID). | OCSF `device.uid` — the canonical OCSF host-identifier field (dictionary: "unique device identifier. Typically BIOS hardware identifier, machine id, or agent identifier."). |

## Platform Support

| Platform | Supported |
| -------- | --------- |
| Linux    | ✅        |
| macOS    | ✅        |
| Other    | —         |

## Example Output

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

## SDK Usage

```go
import (
    "context"

    "github.com/osapi-io/gohai/pkg/gohai"
)

g, _ := gohai.New(gohai.WithCollectors("machine_id"))
facts, _ := g.Collect(context.Background())

fmt.Println(facts.MachineID.ID)
```

## Enable/Disable

```bash
gohai --collector.machine_id      # enable (default)
gohai --no-collector.machine_id   # disable
```

## Dependencies

None.

## Data Sources

On Linux:

1. gopsutil's `host.InfoWithContext` populates `HostID` from a cascade —
   `/etc/machine-id` first, DMI `product_uuid` next, then
   `/proc/sys/kernel/random/boot_id` last. We forward that value as `id`.
2. When gopsutil returns empty (minimal / pre-systemd hosts) we read
   `/var/lib/dbus/machine-id` through the injected `avfs.VFS` as a final
   fallback. Matches Ohai's `linux/machineid.rb` dbus fallback.

On macOS:

1. gopsutil's `host.InfoWithContext` reads `IOPlatformUUID` via IOKit and
   returns it as `HostID`. Forwarded verbatim as `id` so the shape matches
   Linux. Ohai has no `:darwin` `machineid` handler — its darwin plugin surfaces
   the same UUID under `hardware.uuid` instead.

gopsutil's `boot_id` tail is reboot-unstable; consumers seeing a new value after
a reboot should treat that as "unknown", not as a different host. The
Description section calls this out.

## Backing library

- [`github.com/shirou/gopsutil/v4/host`](https://github.com/shirou/gopsutil) for
  the primary machine-id/DMI read. Linux `/var/lib/dbus/machine-id` fallback is
  our own `os.ReadFile`.
