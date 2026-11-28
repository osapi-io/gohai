# SELinux

> **Status:** Implemented ✅

## Description

The collector reports the SELinux enforcement status and the policy data on
Linux hosts. The running kernel gives the status and the runtime mode, through
selinuxfs's `/sys/fs/selinux/enforce`. `/etc/selinux/config` gives the
configured mode and the policy type, and `sestatus` gives the loaded policy
name, the policy version and the maximum kernel policy version.

On macOS, the collector returns `nil`. SELinux is a Linux kernel security
module, and macOS does not have it.

Consumers use this data to:

- Make sure that SELinux is enforcing on production hosts, for compliance checks
  and CIS benchmarks.
- Find a difference between the configured mode (`config_mode`) and the runtime
  mode (`current_mode`). A difference shows that the system started with a
  different kernel argument, or that an admin ran `setenforce 0`.
- Audit which policy module is loaded (`loaded_policy_name`: `targeted`, `mls`,
  `minimum`).
- Check that the installed policy is compatible with the running kernel. Compare
  `policy_version` with `max_kernel_policy_version`.

## Signals

- `status`. This field tells if SELinux is on in the running kernel. It is
  `"enabled"` when `/sys/fs/selinux/enforce` exists, whatever the config file
  says, and `"disabled"` when it does not. Check this field first. If it is
  `"disabled"`, `current_mode` and the versions do not apply.
- `current_mode`. The runtime mode from `/sys/fs/selinux/enforce`: `"enforcing"`
  for 1, `"permissive"` for 0. It can be different from `config_mode` when an
  admin ran `setenforce` and did not reboot.
- `config_mode`. The SELINUX= value from `/etc/selinux/config`. This is the mode
  that the system uses when it boots. Compare it with `current_mode` to find
  runtime overrides.

## Collected fields

| Field                       | Type     | Description                                                                          | Schema mapping                                                                  |
| --------------------------- | -------- | ------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------- |
| `status`                    | `string` | Tells if SELinux is available: `"enabled"` or `"disabled"`.                          | No direct OCSF mapping. OTel has no SELinux object. gohai convention: `status`. |
| `current_mode`              | `string` | The runtime enforcement mode from `sestatus` (`enforcing`/`permissive`).             | No direct OCSF or OTel mapping. gohai convention: `current_mode`.               |
| `config_mode`               | `string` | The SELINUX= value from `/etc/selinux/config` (`enforcing`/`permissive`/`disabled`). | No direct OCSF or OTel mapping. gohai convention: `config_mode`.                |
| `policy_version`            | `string` | The version number of the running policy, for example `"33"`.                        | No direct OCSF or OTel mapping. gohai convention: `policy_version`.             |
| `max_kernel_policy_version` | `string` | The maximum policy version that the kernel supports, for example `"33"`.             | No direct OCSF or OTel mapping. gohai convention: `max_kernel_policy_version`.  |
| `loaded_policy_name`        | `string` | The name of the loaded SELinux policy module (`targeted`, `mls`, `minimum`).         | No direct OCSF or OTel mapping. gohai convention: `loaded_policy_name`.         |

## Platform support

| Platform | Supported                                        |
| -------- | ------------------------------------------------ |
| Linux    | ✅                                               |
| macOS    | Returns `nil`. SELinux is not available on macOS |

## Example output

### Enforcing host with targeted policy

```json
{
  "selinux": {
    "status": "enabled",
    "current_mode": "enforcing",
    "config_mode": "enforcing",
    "policy_version": "33",
    "max_kernel_policy_version": "33",
    "loaded_policy_name": "targeted"
  }
}
```

### SELinux disabled

```json
{
  "selinux": {
    "status": "disabled"
  }
}
```

### SELinux not installed (no /etc/selinux/config)

```json
{
  "selinux": {
    "status": "disabled"
  }
}
```

## SDK usage

```go
import (
    "context"
    "fmt"

    "github.com/osapi-io/gohai/pkg/gohai"
)

g, _ := gohai.New(gohai.WithCollectors("selinux"))
facts, _ := g.Collect(context.Background())

if facts.SELinux != nil {
    fmt.Printf("SELinux: %s (%s)\n", facts.SELinux.Status, facts.SELinux.CurrentMode)
}
```

## Enable/disable

```bash
gohai collect --collector.selinux       # enable (opt-in)
gohai collect --no-collector.selinux    # disable
```

This collector is opt-in (`DefaultEnabled: false`). On some kernel
configurations, `sestatus` must run as root.

## Dependencies

None.

## Data sources

On Linux:

1. The collector reads `/etc/selinux/config` through the injected `avfs.VFS`. It
   parses `SELINUX=` to get the configured mode and `SELINUXTYPE=` to get the
   policy type. It skips lines that start with `#` and blank lines. A missing
   file gives an empty `config_mode`.

2. It reads `/sys/fs/selinux/enforce`. When the file does not exist, the running
   kernel has SELinux off: the collector returns `status: "disabled"` with the
   config fields, and does not run `sestatus`. A config file that says
   `enforcing` does not change that. A file that exists and cannot be read is an
   error.

3. When the file exists, `status` is `"enabled"`, and `current_mode` is
   `"enforcing"` for 1 and `"permissive"` for 0.

4. The collector runs `sestatus` through the injected `source.Runner`, and takes
   these lines of the output:

   - `Loaded policy name:` → `loaded_policy_name`.
   - `Max kernel policy version:` → `max_kernel_policy_version`.
   - `Policy version:` → `policy_version`.

   It does not take the status or the mode from `sestatus`: the kernel gives
   them. A missing `sestatus` leaves the versions empty. A `sestatus` that is
   installed and fails is an error.

Note: Ohai uses `sestatus -v -b`, which also collects policy booleans and
process contexts. gohai uses plain `sestatus` and collects only the status, mode
and version fields. Policy booleans and process contexts are not in the current
scope of gohai.

## Backing library

- [`github.com/avfs/avfs`](https://github.com/avfs/avfs) to read
  `/etc/selinux/config`. Production uses `osfs`. Tests use `memfs`.
- `source.Runner` to run `sestatus`. Tests use a gomock mock.
