# GRUB2

> **Status:** Implemented ✅

## Description

The collector reports the variables of the GRUB2 environment block from the
`grubenv` file. The GRUB2 environment block is a 1024-byte file of fixed size.
It keeps boot-time variables after a reboot. Bootloaders and `grub2-editenv` use
it to record the boot state.

On macOS, the collector returns `nil`. GRUB2 is a Linux and BSD bootloader. It
is not on macOS.

Consumers use this data to:

- Read `saved_entry` to find the boot entry that GRUB2 selects on the next boot.
  This helps to verify a kernel upgrade or a rollback that is not complete.
- Examine `boot_success` and `boot_indeterminate` for the upgrade state machines
  of Atomic Host, ostree and rpm-ostree.
- Read `kernelopts` or custom environment variables that provisioning tools set.

## Collected fields

| Field         | Type                | Description                                                                                            | Schema mapping                                                                                  |
| ------------- | ------------------- | ------------------------------------------------------------------------------------------------------ | ----------------------------------------------------------------------------------------------- |
| `environment` | `map[string]string` | The key=value pairs from the grubenv file. With no grubenv file on any path, there is no `grub2` fact. | No direct OCSF or OTel mapping. gohai convention: `environment`. Ohai uses `grub2.environment`. |

## Platform support

| Platform | Supported                            |
| -------- | ------------------------------------ |
| Linux    | ✅                                   |
| macOS    | Returns `nil`. GRUB2 is not on macOS |

## Example output

### Typical Fedora/RHEL host

```json
{
  "grub2": {
    "environment": {
      "saved_entry": "0",
      "boot_success": "1",
      "boot_indeterminate": "0",
      "kernelopts": "root=/dev/mapper/fedora-root ro rhgb quiet"
    }
  }
}
```

### Host without GRUB2 installed

The facts have no `grub2` key.

## SDK usage

```go
import (
    "context"
    "fmt"

    "github.com/osapi-io/gohai/pkg/gohai"
)

g, _ := gohai.New(gohai.WithCollectors("grub2"))
facts, _ := g.Collect(context.Background())

if facts.Grub2 != nil {
    fmt.Println("Next boot entry:", facts.Grub2.Environment["saved_entry"])
}
```

## Enable/disable

```bash
gohai collect --collector.grub2       # enable (opt-in)
gohai collect --no-collector.grub2    # disable
```

This collector is opt-in (`DefaultEnabled: false`) because not all Linux hosts
have GRUB2. For example, containers and UEFI-only systems that use systemd-boot
do not have it.

## Dependencies

None.

## Data sources

On Linux:

1. The collector first reads `/boot/grub2/grubenv` through the injected
   `avfs.VFS`. RHEL, Fedora, CentOS, Rocky and AlmaLinux use this path.
2. If that file does not exist, the collector reads `/boot/grub/grubenv`. Debian
   and Ubuntu use this path.
3. If neither file exists, the collector gives no fact. The missing `grub2` key
   tells consumers that GRUB2 is not installed.
4. The collector parses the file one line at a time:
   - It skips lines that start with `#`, for example the standard
     `# GRUB Environment Block` header.
   - It splits each line at the first `=`. The left side is the key. The right
     side is the value.
   - It skips lines that have no `=`.
5. The collector returns the map.

Note: Ohai's `grub2.rb` runs `grub2-editenv list` to read the environment. It
does not parse the file. gohai reads the file directly, so it does not start a
subprocess for a simple key=value parse. The grubenv format is documented and
stable. The file is human-readable on purpose. For standard deployments, the two
methods give the same output.

## Backing library

- [`github.com/avfs/avfs`](https://github.com/avfs/avfs) to read the grubenv
  file. Production uses `osfs`. Tests use `memfs`.
