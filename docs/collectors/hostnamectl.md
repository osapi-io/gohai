# Hostnamectl

> **Status:** Implemented ✅

## Description

Reports system identity fields from `hostnamectl` on Linux — static hostname,
chassis type, deployment environment, kernel identity, hardware vendor and
model, firmware version, and virtualization type. Matches Ohai's
`linux/hostnamectl` plugin. Darwin returns `nil` — `hostnamectl` is a Linux
systemd tool.

DefaultEnabled is `false` — the data is Linux-only and requires systemd.

## Collected Fields

| Field                          | Type   | Description                                                | Schema mapping                                                                   |
| ------------------------------ | ------ | ---------------------------------------------------------- | -------------------------------------------------------------------------------- |
| `static_hostname`              | string | Static hostname configured via `hostnamectl`.              | `device.hostname` (OCSF) — leaf stripped per redundant-prefix rule → `hostname`. |
| `icon_name`                    | string | Freedesktop icon name for the chassis class.               | gohai convention.                                                                |
| `chassis`                      | string | Chassis type (`vm`, `container`, `desktop`, `server`).     | `device_hw_info.chassis` (OCSF).                                                 |
| `deployment`                   | string | Deployment environment (`production`, `staging`, etc.).    | gohai convention.                                                                |
| `location`                     | string | Physical or logical location string.                       | gohai convention.                                                                |
| `kernel_name`                  | string | Kernel name (`Linux`).                                     | gohai convention (OCSF `os.kernel_release` covers release; name is separate).    |
| `kernel_release`               | string | Kernel release string (`5.15.0-91-generic`).               | `os.kernel_release` (OCSF) — leaf stripped → `release` in kernel collector.      |
| `operating_system_pretty_name` | string | Human-readable OS name (`Ubuntu 22.04.3 LTS`).             | `os.name` (OCSF).                                                                |
| `operating_system_cpe_name`    | string | CPE OS identifier (`cpe:/o:ubuntu:ubuntu:22.04`).          | gohai convention.                                                                |
| `virtualization`               | string | Virtualization type detected by systemd (`kvm`, `docker`). | `device_hw_info.type` (OCSF) — closest match.                                    |
| `hardware_vendor`              | string | Hardware vendor string.                                    | `device_hw_info.vendor_name` (OCSF).                                             |
| `hardware_model`               | string | Hardware model string.                                     | `device_hw_info.model` (OCSF).                                                   |
| `firmware_version`             | string | Firmware/BIOS version string.                              | `device_hw_info.bios_ver` (OCSF) — closest match.                                |

## Platform Support

| Platform | Supported                       |
| -------- | ------------------------------- |
| Linux    | ✅ (`hostnamectl` via executor) |
| macOS    | `nil` (systemd is Linux-only)   |

## Example Output

### Linux with systemd

```json
{
  "hostnamectl": {
    "static_hostname": "myhost",
    "icon_name": "computer-vm",
    "chassis": "vm",
    "deployment": "production",
    "location": "rack-42",
    "kernel_name": "Linux",
    "kernel_release": "5.15.0-91-generic",
    "operating_system_pretty_name": "Ubuntu 22.04.3 LTS",
    "operating_system_cpe_name": "cpe:/o:ubuntu:ubuntu:22.04",
    "virtualization": "kvm",
    "hardware_vendor": "QEMU",
    "hardware_model": "Standard PC (Q35 + ICH9, 2009)",
    "firmware_version": "2.5+dfsg-4"
  }
}
```

### System without `hostnamectl`

```json
{
  "hostnamectl": {}
}
```

## SDK Usage

```go
g, _ := gohai.New(gohai.WithCollectors("hostnamectl"))
facts, _ := g.Collect(context.Background())
if h := facts.Hostnamectl; h != nil {
    fmt.Println(h.Chassis, h.HardwareVendor)
}
```

## Enable/Disable

```bash
gohai --collector.hostnamectl    # enable (opt-in)
gohai --no-collector.hostnamectl # disable
```

## Dependencies

None.

## Data Sources

On Linux the collector runs `hostnamectl` (no subcommand) through the shared
`internal/executor` runner and parses its `Key: value` output line by line:

1. Each line is split on the first `": "` separator. Lines without this
   separator are skipped.
2. The key portion is trimmed of leading/trailing whitespace, lowercased, and
   spaces replaced with underscores — matching Ohai's
   `key.downcase.tr(" ", "_")` transform in `linux/hostnamectl.rb`.
3. Non-ASCII characters (Unicode decorators/emoji introduced in systemd ≥ v250)
   are stripped from values via regex, and any resulting double spaces are
   collapsed to single spaces. This mirrors Ohai's `/[^\p{ASCII}]/u` regex.
4. The `Kernel` key is split on the first space to separate the kernel name
   (`Linux`) from the release string (`5.15.0-91-generic`).
5. When `hostnamectl` is absent or returns a non-zero exit code, an empty `Info`
   is returned without error — matches Ohai's no-panic behaviour.

macOS is not covered — `hostnamectl` is a Linux systemd utility.

## Backing library

- [`internal/executor`](../../internal/executor) — shared command-runner
  abstraction used to invoke `hostnamectl` on Linux. Tests mock it with
  `go.uber.org/mock`.
