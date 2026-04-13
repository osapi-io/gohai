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

| Platform | What we read                                                                                                                                                      | Ohai plugin                                                                                                                                                                                                  | Alignment                                                                                                                                                              |
| -------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Linux    | gopsutil `host.InfoWithContext` (reads `/etc/machine-id` → DMI `product_uuid` → `boot_id`) + our `/var/lib/dbus/machine-id` fallback when gopsutil returns empty. | [`linux/machineid.rb`](https://github.com/chef/ohai/blob/main/lib/ohai/plugins/linux/machineid.rb) — reads `/etc/machine-id` primarily, falls back to `/var/lib/dbus/machine-id`. No DMI / boot_id fallback. | **Superset of Ohai.** We inherit gopsutil's DMI fallback (useful on minimal hosts without systemd) and extend with the dbus path to match Ohai's pre-systemd coverage. |
| macOS    | gopsutil `host.InfoWithContext` (reads `IOPlatformUUID` via IOKit).                                                                                               | No Ohai `:darwin` handler for `machine_id`; Ohai surfaces the hardware UUID via [`darwin/hardware.rb`](https://github.com/chef/ohai/blob/main/lib/ohai/plugins/darwin/hardware.rb) instead.                  | **Richer than Ohai.** gopsutil gives us `IOPlatformUUID` directly under `machine_id.id` for consistency with the Linux shape.                                          |

**Known gaps:** None. gopsutil's `boot_id` tail — which is reboot-unstable — is
documented in the Description so consumers know to treat mismatched reboots as
unknown rather than two different hosts.

## Backing library

- [`github.com/shirou/gopsutil/v4/host`](https://github.com/shirou/gopsutil) for
  the primary machine-id/DMI read. Linux `/var/lib/dbus/machine-id` fallback is
  our own `os.ReadFile`.
