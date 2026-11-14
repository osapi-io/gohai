# Hostnamectl

> **Status:** Implemented ✅

## Description

The collector reports the system identity fields from `hostnamectl` on Linux.
These are the static hostname, the chassis type, the deployment environment, the
kernel identity, the hardware vendor and model, the firmware version, and the
virtualization type. The collector is the same as Ohai's `linux/hostnamectl`
plugin. On Darwin, the collector returns `nil`, because `hostnamectl` is a
systemd tool for Linux.

`DefaultEnabled` is `false`. The data is for Linux only, and it needs systemd.

## Collected fields

| Field                          | Type   | Description                                                                | Schema mapping                                                                                  |
| ------------------------------ | ------ | -------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------- |
| `static_hostname`              | string | The static hostname that `hostnamectl` sets.                               | `device.hostname` (OCSF). The redundant-prefix rule removes the leaf, which gives `hostname`.   |
| `icon_name`                    | string | The Freedesktop icon name for the chassis class.                           | gohai convention.                                                                               |
| `chassis`                      | string | The chassis type, for example `vm`, `container`, `desktop` or `server`.    | `device_hw_info.chassis` (OCSF).                                                                |
| `deployment`                   | string | The deployment environment, for example `production` or `staging`.         | gohai convention.                                                                               |
| `location`                     | string | The physical or logical location.                                          | gohai convention.                                                                               |
| `kernel_name`                  | string | The kernel name, for example `Linux`.                                      | gohai convention. OCSF `os.kernel_release` has the release. The name is a different value.      |
| `kernel_release`               | string | The kernel release, for example `5.15.0-91-generic`.                       | `os.kernel_release` (OCSF). The leaf is removed, which gives `release` in the kernel collector. |
| `operating_system_pretty_name` | string | The human-readable OS name, for example `Ubuntu 22.04.3 LTS`.              | `os.name` (OCSF).                                                                               |
| `operating_system_cpe_name`    | string | The CPE OS identifier, for example `cpe:/o:ubuntu:ubuntu:22.04`.           | gohai convention.                                                                               |
| `virtualization`               | string | The virtualization type that systemd finds, for example `kvm` or `docker`. | `device_hw_info.type` (OCSF). This is the nearest match.                                        |
| `hardware_vendor`              | string | The hardware vendor.                                                       | `device_hw_info.vendor_name` (OCSF).                                                            |
| `hardware_model`               | string | The hardware model.                                                        | `device_hw_info.model` (OCSF).                                                                  |
| `firmware_version`             | string | The firmware or BIOS version.                                              | `device_hw_info.bios_ver` (OCSF). This is the nearest match.                                    |

## Platform support

| Platform | Supported                               |
| -------- | --------------------------------------- |
| Linux    | ✅ (`hostnamectl` through the executor) |
| macOS    | `nil` (systemd is for Linux only)       |

## Example output

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

## SDK usage

```go
g, _ := gohai.New(gohai.WithCollectors("hostnamectl"))
facts, _ := g.Collect(context.Background())
if h := facts.Hostnamectl; h != nil {
    fmt.Println(h.Chassis, h.HardwareVendor)
}
```

## Enable/disable

```bash
gohai collect --collector.hostnamectl    # enable (opt-in)
gohai collect --no-collector.hostnamectl # disable
```

## Dependencies

None.

## Data sources

On Linux, the collector runs `hostnamectl` with no subcommand through the shared
`internal/executor` runner. It parses the `Key: value` output one line at a
time:

1. The collector splits each line at the first `": "` separator. It skips lines
   that do not have this separator.
2. The collector removes the leading and trailing whitespace from the key. It
   changes the key to lowercase and replaces spaces with underscores. This is
   the same as Ohai's `key.downcase.tr(" ", "_")` transform in
   `linux/hostnamectl.rb`.
3. A regex removes non-ASCII characters from the values. These are the Unicode
   decorators and emoji that systemd v250 and later add. Then the collector
   changes each double space that remains to one space. This is the same as
   Ohai's `/[^\p{ASCII}]/u` regex.
4. The collector splits the `Kernel` value at the first space. The first part is
   the kernel name (`Linux`). The second part is the release
   (`5.15.0-91-generic`).
5. If `hostnamectl` is missing or gives a non-zero exit code, the collector
   returns an empty `Info` and no error. This is the same as Ohai, which does
   not panic in this case.

The collector does not support macOS, because `hostnamectl` is a systemd tool
for Linux.

## Backing library

- [`internal/executor`](../../internal/executor). The shared command runner. The
  collector uses it to run `hostnamectl` on Linux. Tests mock it with
  `go.uber.org/mock`.
