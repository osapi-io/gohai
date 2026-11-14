# VirtualBox

> **Status:** Implemented ✅

## Description

The collector reports the VirtualBox Guest Additions data from inside a
VirtualBox guest VM. VirtualBox Guest Additions must be installed, and
`VBoxControl` must be on PATH. If the host is not a VirtualBox guest, or if
Guest Additions are missing, `Collect` returns `nil` and no error.

gohai has only the guest role. Ohai's `virtualbox.rb` has a host branch that
gets the host inventory (`VBoxManage list vms`, `VBoxManage showvminfo` and
others). This branch needs `VBoxManage`, which installs with the VirtualBox
application. gohai does not have this branch yet. If a consumer needs the host
inventory, use the VBoxManage API directly.

## Collected fields

| Field                      | Type   | Description                                                  | Schema mapping            |
| -------------------------- | ------ | ------------------------------------------------------------ | ------------------------- |
| `version`                  | string | The VirtualBox host version (`7.0.14`), from `VBoxVer`.      | No direct schema mapping. |
| `revision`                 | string | The VirtualBox host revision (`161095`), from `VBoxRev`.     | No direct schema mapping. |
| `guest_additions_version`  | string | The Guest Additions version, from `GuestAdd/VersionExt`.     | No direct schema mapping. |
| `guest_additions_revision` | string | The Guest Additions revision, from `GuestAdd/Revision`.      | No direct schema mapping. |
| `language_id`              | string | The host locale or language ID (`en_US`), from `LanguageID`. | No direct schema mapping. |

## Platform support

| Platform | Supported                                                       |
| -------- | --------------------------------------------------------------- |
| Linux    | ✅ (`VBoxControl guestproperty enumerate` through the executor) |
| macOS    | ✅ (`VBoxControl guestproperty enumerate` through the executor) |

## Example output

### VirtualBox guest with guest additions installed

```json
{
  "virtualbox": {
    "version": "7.0.14",
    "revision": "161095",
    "guest_additions_version": "7.0.14",
    "guest_additions_revision": "161095",
    "language_id": "en_US"
  }
}
```

### Non-VirtualBox host or guest additions absent

```json
{
  "virtualbox": null
}
```

## SDK usage

```go
g, _ := gohai.New(gohai.WithCollectors("virtualbox"))
facts, _ := g.Collect(context.Background())
if vb := facts.VirtualBox; vb != nil {
    fmt.Println(vb.HostVersion, vb.GuestAdditionsVersion)
}
```

## Enable/disable

```bash
gohai --collector.virtualbox    # enable (opt-in)
gohai --no-collector.virtualbox # disable
```

DefaultEnabled: `false`. Most hosts do not have VirtualBox Guest Additions. A
caller must enable the collector explicitly.

## Dependencies

None.

## Data sources

Ohai's `virtualbox.rb` has two modes: the host side (through `VBoxManage`) and
the guest side (through `VBoxControl`). gohai has only the guest side. It runs
`VBoxControl guestproperty enumerate` and gets the same five properties as Ohai
(VBoxVer, VBoxRev, GuestAdd/VersionExt, GuestAdd/Revision, LanguageID). It uses
the same regex patterns as Ohai. gohai does not have the host-side `VBoxManage`
list.

On Linux and on macOS, the collector runs `VBoxControl guestproperty enumerate`.
`VBoxControl` is part of the VirtualBox Guest Additions package. The package is
`virtualbox-guest-utils` on Debian/Ubuntu and `virtualbox-guest-additions-iso`
on RHEL. On macOS, it is the Guest Additions disk image.

The command lists all guest properties that the hypervisor gives. gohai parses
five properties with the regex patterns from the guest branch of Ohai's
`virtualbox.rb`:

- `LanguageID, value: <val>,` → `language_id`
- `VBoxVer, value: <val>,` → `version`
- `VBoxRev, value: <val>,` → `revision`
- `GuestAdd/VersionExt, value: <val>,` → `guest_additions_version`
- `GuestAdd/Revision, value: <val>,` → `guest_additions_revision`

If `VBoxControl` is missing, the command fails and `Collect` returns `nil`, not
an error. `VBoxControl` is missing if the host is not a VirtualBox guest, or if
Guest Additions are not installed. The collector ignores all other lines in the
output and gives no message about them.

## Backing library

- [`internal/executor`](../../internal/executor). The shared command runner. The
  collector uses it to run `VBoxControl`. Tests mock it with `go.uber.org/mock`.
