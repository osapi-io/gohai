# VirtualBox

> **Status:** Implemented ✅

## Description

Reports VirtualBox guest additions data from inside a VirtualBox guest VM.
Requires VirtualBox Guest Additions to be installed (`VBoxControl` must be on
PATH). When the host is not a VirtualBox guest, or when Guest Additions are
absent, `Collect` returns `nil` with no error.

Only the guest role is implemented. Host-side inventory (`VBoxManage list vms`,
`VBoxManage showvminfo`, etc.) from Ohai's `virtualbox.rb` host branch requires
`VBoxManage` to be installed alongside the VirtualBox application. That surface
is not yet implemented, consumers who need host inventory should use the
VBoxManage API directly.

## Collected fields

| Field                      | Type   | Description                                          | Schema mapping            |
| -------------------------- | ------ | ---------------------------------------------------- | ------------------------- |
| `version`                  | string | VirtualBox host version (`7.0.14`) from `VBoxVer`.   | No direct schema mapping. |
| `revision`                 | string | VirtualBox host revision (`161095`) from `VBoxRev`.  | No direct schema mapping. |
| `guest_additions_version`  | string | Guest Additions version from `GuestAdd/VersionExt`.  | No direct schema mapping. |
| `guest_additions_revision` | string | Guest Additions revision from `GuestAdd/Revision`.   | No direct schema mapping. |
| `language_id`              | string | Host locale/language ID (`en_US`) from `LanguageID`. | No direct schema mapping. |

## Platform support

| Platform | Supported                                               |
| -------- | ------------------------------------------------------- |
| Linux    | ✅ (`VBoxControl guestproperty enumerate` via executor) |
| macOS    | ✅ (`VBoxControl guestproperty enumerate` via executor) |

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

DefaultEnabled: `false`. VirtualBox Guest Additions are not present on most
hosts; callers must opt in explicitly.

## Dependencies

None.

## Data sources

Ohai's `virtualbox.rb` has two modes: host-side (via `VBoxManage`) and
guest-side (via `VBoxControl`). gohai implements guest-side only, using
`VBoxControl guestproperty enumerate` and extracting the same five properties
(VBoxVer, VBoxRev, GuestAdd/VersionExt, GuestAdd/Revision, LanguageID) via regex
patterns matching Ohai's approach. Host-side `VBoxManage` enumeration is not
implemented.

On both Linux and macOS the collector runs
`VBoxControl guestproperty enumerate`. `VBoxControl` is part of the VirtualBox
Guest Additions package (`virtualbox-guest-utils` on Debian/Ubuntu,
`virtualbox-guest-additions-iso` on RHEL, the Guest Additions disk image on
macOS).

The command enumerates all guest properties exposed by the hypervisor. We parse
five properties using the regex patterns from Ohai's `virtualbox.rb` guest
branch:

- `LanguageID, value: <val>,` → `language_id`
- `VBoxVer, value: <val>,` → `version`
- `VBoxRev, value: <val>,` → `revision`
- `GuestAdd/VersionExt, value: <val>,` → `guest_additions_version`
- `GuestAdd/Revision, value: <val>,` → `guest_additions_revision`

If `VBoxControl` is absent (not a VirtualBox guest, or Guest Additions not
installed) the command fails and `Collect` returns `nil`, not an error. All
other lines in the output are silently ignored.

## Backing library

- [`internal/executor`](../../internal/executor). Shared command-runner
  abstraction for `VBoxControl` invocations. Tests mock it with
  `go.uber.org/mock`.
