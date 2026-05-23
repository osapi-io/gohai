# GRUB2

> **Status:** Implemented ✅

## Description

Reports the GRUB2 environment block variables from the `grubenv` file. The GRUB2
environment block is a fixed-size 1024-byte file that persists boot-time
variables across reboots. It is commonly used by bootloaders and `grub2-editenv`
to track boot state.

On macOS the collector returns nil — GRUB2 is a Linux/BSD bootloader not present
on macOS.

Consumers use this to:

- Read `saved_entry` to determine which boot entry will be selected on the next
  boot (useful for verifying a pending kernel upgrade or rollback).
- Check `boot_success` and `boot_indeterminate` for Atomic Host / ostree /
  rpm-ostree upgrade state machines.
- Read `kernelopts` or custom environment variables set by provisioning tooling.

## Collected Fields

| Field         | Type                | Description                                                                      | Schema mapping                                                                                   |
| ------------- | ------------------- | -------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------ |
| `environment` | `map[string]string` | Key=value pairs from the grubenv file. Nil if no grubenv file found on any path. | No direct OCSF or OTel mapping. gohai convention: `environment` (Ohai uses `grub2.environment`). |

## Platform Support

| Platform | Supported                                     |
| -------- | --------------------------------------------- |
| Linux    | ✅                                            |
| macOS    | Returns nil — GRUB2 is not available on macOS |

## Example Output

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

```json
{
  "grub2": {
    "environment": null
  }
}
```

## SDK Usage

```go
import (
    "context"
    "fmt"

    "github.com/osapi-io/gohai/pkg/gohai"
)

g, _ := gohai.New(gohai.WithCollectors("grub2"))
facts, _ := g.Collect(context.Background())

if facts.GRUB2 != nil && facts.GRUB2.Environment != nil {
    fmt.Println("Next boot entry:", facts.GRUB2.Environment["saved_entry"])
}
```

## Enable/Disable

```bash
gohai --collector.grub2       # enable (opt-in)
gohai --no-collector.grub2    # disable
```

This collector is opt-in (`DefaultEnabled: false`) because GRUB2 is not present
on all Linux hosts (e.g. containers, UEFI-only systems using systemd-boot).

## Dependencies

None.

## Data Sources

On Linux:

1. Try `/boot/grub2/grubenv` first (RHEL, Fedora, CentOS, Rocky, AlmaLinux) via
   the injected `avfs.VFS`.
2. If that file does not exist, try `/boot/grub/grubenv` (Debian, Ubuntu).
3. If neither file exists, return `{environment: nil}` — nil signals "GRUB2 not
   installed" to consumers.
4. Parse the file line by line:
   - Skip lines beginning with `#` (the standard `# GRUB Environment Block`
     header).
   - Split on the first `=`; left side is the key, right side is the value.
   - Lines without `=` are skipped.
5. Return the populated map.

Note: Ohai's `grub2.rb` uses `grub2-editenv list` to read the environment rather
than parsing the file directly. gohai reads the file directly to avoid forking a
subprocess for a simple key=value parse. The grubenv format is documented and
stable (the file is intentionally human-readable). Both approaches yield
identical output for standard deployments.

## Backing library

- [`github.com/avfs/avfs`](https://github.com/avfs/avfs) (`osfs` in production,
  `memfs` in tests) for the grubenv file reads.
