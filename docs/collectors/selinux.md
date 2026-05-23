# SELinux

> **Status:** Implemented ✅

## Description

Reports SELinux enforcement status and policy information on Linux hosts. The
collector reads `/etc/selinux/config` for the configured mode and policy type,
and runs `sestatus` for the runtime enforcement mode, loaded policy name, policy
version, and maximum kernel policy version.

On macOS the collector returns nil — SELinux is a Linux kernel security module
and is not present on macOS.

Consumers use this to:

- Verify that SELinux is enforcing on production hosts (compliance checks, CIS
  benchmark alignment).
- Detect drift between the configured mode (`config_mode`) and the runtime mode
  (`current_mode`) — a mismatch indicates the system was rebooted with a
  different kernel argument or an admin ran `setenforce 0`.
- Audit which policy module is loaded (`loaded_policy_name`: `targeted`, `mls`,
  `minimum`).
- Check whether the installed policy is compatible with the running kernel
  (`policy_version` vs `max_kernel_policy_version`).

## Signals

- `status` — overall SELinux availability: `"enabled"` when SELinux is compiled
  in and the config mode is not `disabled`; `"disabled"` otherwise. This is the
  first field to check — if `"disabled"`, all other fields are irrelevant.
- `current_mode` — runtime mode from `sestatus`: `"enforcing"`, `"permissive"`,
  or `"disabled"`. This can differ from `config_mode` when an admin has called
  `setenforce` without rebooting.
- `config_mode` — the SELINUX= value from `/etc/selinux/config`, the mode the
  system will boot into. Compare with `current_mode` to detect runtime
  overrides.

## Collected Fields

| Field                       | Type     | Description                                                                      | Schema mapping                                                                  |
| --------------------------- | -------- | -------------------------------------------------------------------------------- | ------------------------------------------------------------------------------- |
| `status`                    | `string` | SELinux availability: `"enabled"` or `"disabled"`.                               | No direct OCSF mapping. OTel has no SELinux object. gohai convention: `status`. |
| `current_mode`              | `string` | Runtime enforcement mode from `sestatus` (`enforcing`/`permissive`).             | No direct OCSF or OTel mapping. gohai convention: `current_mode`.               |
| `config_mode`               | `string` | SELINUX= value from `/etc/selinux/config` (`enforcing`/`permissive`/`disabled`). | No direct OCSF or OTel mapping. gohai convention: `config_mode`.                |
| `policy_version`            | `string` | Running policy version number (e.g. `"33"`).                                     | No direct OCSF or OTel mapping. gohai convention: `policy_version`.             |
| `max_kernel_policy_version` | `string` | Maximum policy version the kernel supports (e.g. `"33"`).                        | No direct OCSF or OTel mapping. gohai convention: `max_kernel_policy_version`.  |
| `loaded_policy_name`        | `string` | Name of the loaded SELinux policy module (`targeted`, `mls`, `minimum`).         | No direct OCSF or OTel mapping. gohai convention: `loaded_policy_name`.         |

## Platform Support

| Platform | Supported                                       |
| -------- | ----------------------------------------------- |
| Linux    | ✅                                              |
| macOS    | Returns nil — SELinux is not available on macOS |

## Example Output

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

## SDK Usage

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

## Enable/Disable

```bash
gohai --collector.selinux       # enable (opt-in)
gohai --no-collector.selinux    # disable
```

This collector is opt-in (`DefaultEnabled: false`) because `sestatus` requires
root on some kernel configurations.

## Dependencies

None.

## Data Sources

On Linux:

1. Read `/etc/selinux/config` through the injected `avfs.VFS`. Parse `SELINUX=`
   for the configured mode and `SELINUXTYPE=` for the policy type. Lines
   beginning with `#` and blank lines are skipped. If the file does not exist,
   SELinux is not installed — return `{status: "disabled"}` immediately without
   calling `sestatus`.
2. If the configured mode is `disabled`, return
   `{status: "disabled", config_mode: "disabled", loaded_policy_name: <SELINUXTYPE>}`
   without calling `sestatus` — sestatus exits non-zero when SELinux is
   disabled.
3. If the configured mode is `enforcing` or `permissive`, run `sestatus` via the
   injected `executor.Executor`. Parse the output for:
   - `SELinux status:` → overrides `status` field.
   - `Current mode:` → `current_mode`.
   - `Loaded policy name:` → `loaded_policy_name` (may override the SELINUXTYPE
     from config if they differ).
   - `Max kernel policy version:` → `max_kernel_policy_version`.
   - `Policy version:` → `policy_version`.
4. If `sestatus` fails (binary not found, permission denied), fall back to
   deriving `status: "enabled"` from the non-disabled config mode. The
   `current_mode`, `policy_version`, and `max_kernel_policy_version` fields will
   be absent.

Note: Ohai uses `sestatus -v -b` which also collects policy booleans and process
contexts. gohai uses plain `sestatus` and collects only the status/mode/version
fields — policy booleans and process contexts are outside our current scope.

## Backing library

- [`github.com/avfs/avfs`](https://github.com/avfs/avfs) (`osfs` in production,
  `memfs` in tests) for the `/etc/selinux/config` read.
- `internal/executor` for `sestatus` execution (gomock-backed in tests).
