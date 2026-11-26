# SELinux

> **Status:** Implemented ✅

## Description

The collector reports the SELinux enforcement status and the policy data on
Linux hosts. It reads `/etc/selinux/config` to get the configured mode and the
policy type. It runs `sestatus` to get the runtime enforcement mode, the loaded
policy name, the policy version and the maximum kernel policy version.

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

- `status`. This field tells if SELinux is available. It is `"enabled"` when
  SELinux is compiled in and the config mode is not `disabled`. Otherwise, it is
  `"disabled"`. Check this field first. If it is `"disabled"`, the other fields
  do not apply.
- `current_mode`. The runtime mode from `sestatus`: `"enforcing"`,
  `"permissive"` or `"disabled"`. It can be different from `config_mode` when an
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
   policy type. It skips lines that start with `#` and blank lines. If the file
   does not exist, SELinux is not installed. The collector then returns
   `{status: "disabled"}` immediately and does not run `sestatus`.
2. If the configured mode is `disabled`, the collector returns
   `{status: "disabled", config_mode: "disabled", loaded_policy_name: <SELINUXTYPE>}`
   and does not run `sestatus`. When SELinux is disabled, sestatus exits with a
   status that is not zero.
3. If the configured mode is `enforcing` or `permissive`, the collector runs
   `sestatus` through the injected `source.Runner`. It parses these lines of the
   output:
   - `SELinux status:` → overrides the `status` field.
   - `Current mode:` → `current_mode`.
   - `Loaded policy name:` → `loaded_policy_name`. If this value is different
     from the SELINUXTYPE in the config, it can override it.
   - `Max kernel policy version:` → `max_kernel_policy_version`.
   - `Policy version:` → `policy_version`.
4. If `sestatus` fails, for example because the binary is not found or the
   permission is denied, the collector uses the config mode. Because that mode
   is not `disabled`, it sets `status: "enabled"`. The `current_mode`,
   `policy_version` and `max_kernel_policy_version` fields are then missing.

Note: Ohai uses `sestatus -v -b`, which also collects policy booleans and
process contexts. gohai uses plain `sestatus` and collects only the status, mode
and version fields. Policy booleans and process contexts are not in the current
scope of gohai.

## Backing library

- [`github.com/avfs/avfs`](https://github.com/avfs/avfs) to read
  `/etc/selinux/config`. Production uses `osfs`. Tests use `memfs`.
- `source.Runner` to run `sestatus`. Tests use a gomock mock.
