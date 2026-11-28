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
  container images or systems that boot from initramfs. On these hosts, the
  collector returns an empty value. It does not use
  `/proc/sys/kernel/random/boot_id`, because that value changes at each reboot.

## Collected fields

| Field    | Type   | Description                                                                | Schema mapping                                                                                                                                                              |
| -------- | ------ | -------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `id`     | string | The stable machine identifier, usually a UUID.                             | OCSF `device.uid`, the canonical OCSF host-identifier field (dictionary: "unique device identifier. Typically BIOS hardware identifier, machine id, or agent identifier."). |
| `source` | string | Where the ID came from: the file path on Linux, `IOPlatformUUID` on macOS. | None.                                                                                                                                                                       |

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
    "id": "abc123def4567890abcdef1234567890",
    "source": "/etc/machine-id"
  }
}
```

### macOS

```json
{
  "machine_id": {
    "id": "12345678-ABCD-EF01-2345-6789ABCDEF01",
    "source": "IOPlatformUUID"
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

1. The collector reads `/etc/machine-id`. If that file is missing or doesn't
   hold an ID, it reads `/var/lib/dbus/machine-id`, the fallback in Ohai's
   `linux/machineid.rb` for hosts older than systemd. Then it reads DMI
   `/sys/class/dmi/id/product_uuid`, in lower case. The first valid value is
   `id`, and its file path is `source`.
2. A value from the two machine-id files is valid only when it has 32 hex
   digits, as machine-id(5) says. Thus the `uninitialized` that systemd writes
   on a first-boot image is not an ID, and the collector reads the next file.
3. A missing file is skipped. Only root can read `product_uuid`, so for a user
   that isn't root a permission error there is skipped too, and the result has
   no ID. Any other file that is there but can't be read is an error.

On macOS:

1. The collector runs `ioreg -rd1 -c IOPlatformExpertDevice` and reports its
   `IOPlatformUUID` as `id` with no change, so the shape is the same as on
   Linux. Ohai has no `:darwin` `machineid` handler. Its darwin plugin reports
   the same UUID under `hardware.uuid`.

## Backing library

None. The collector reads the files through its filesystem, and runs `ioreg`
through its command runner.
