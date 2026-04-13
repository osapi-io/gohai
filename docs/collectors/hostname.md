# Hostname

> **Status:** Implemented ✅

## Description

Reports the host's short name, fully qualified domain name, and DNS domain. The
FQDN is canonicalized via a DNS round-trip; the collector tolerates transient
resolver failures by retrying up to three times before falling back to the short
hostname. Consumers use this as a stable identity for telemetry, inventory, and
correlation.

On macOS we additionally capture the friendly machine name
(ComputerName-derived, e.g. "John's MacBook Pro") and prefer `hostname -s` over
gopsutil so the short name always matches what `$(hostname -s)` reports
elsewhere on the host — important on MDM-managed Macs where
`scutil --get HostName` can differ.

## Collected Fields

| Field          | Type   | Description                                                                              | Schema mapping                                                       |
| -------------- | ------ | ---------------------------------------------------------------------------------------- | -------------------------------------------------------------------- |
| `name`         | string | Short hostname (e.g., `web01`).                                                          | OCSF `device.hostname` (leaf stripped: leaf matches collector name). |
| `fqdn`         | string | Canonical FQDN; falls back to short hostname when DNS canonicalization ultimately fails. | `device.domain` + `.hostname`.                                       |
| `domain`       | string | DNS domain — everything after the first `.` of the FQDN. Empty when FQDN equals `name`.  | `device.domain`.                                                     |
| `machine_name` | string | Human-friendly name (macOS `ComputerName`-derived). Linux: same as `name`.               | No direct schema mapping.                                            |

## Platform Support

| Platform | Supported                                                          |
| -------- | ------------------------------------------------------------------ |
| Linux    | ✅ (`hostname -s` + `hostname` via executor, DNS canonicalization) |
| macOS    | ✅ (`hostname -s` + `hostname` via executor, DNS canonicalization) |

## Example Output

```json
{
  "hostname": {
    "name": "web01",
    "fqdn": "web01.example.com",
    "domain": "example.com",
    "machine_name": "web01"
  }
}
```

## SDK Usage

```go
g, _ := gohai.New(gohai.WithCollectors("hostname"))
facts, _ := g.Collect(context.Background())

info := facts.Hostname
fmt.Println(info.Name, info.FQDN, info.Domain, info.MachineName)
```

## Enable/Disable

```bash
gohai --collector.hostname      # enable (default)
gohai --no-collector.hostname   # disable
```

## Dependencies

None.

## Data Sources

On Linux and macOS (identical — mirrors Ohai's `hostname.rb` linux and darwin
branches):

1. `hostname -s` → `name`. Run through the shared `internal/executor` runner. We
   prefer this over gopsutil so our short name matches what `$(hostname -s)`
   reports in any other tool on the host — important on MDM-managed Macs where
   `scutil --get HostName` can differ, and on Linux hosts where `/etc/hostname`
   has been manually edited to an FQDN.
2. `hostname` (no args) → `machine_name`. On stock macOS this is the
   `ComputerName`-derived friendly name; on Linux it normally matches the short
   name.
3. FQDN is canonicalized via `net.LookupHost` followed by `net.LookupAddr`,
   retried up to 3 times with a 100ms backoff on transient errors — matches
   Ohai's `canonicalize_hostname_with_retries`. On final failure we use the
   short hostname as FQDN.
4. Derive `domain` by splitting FQDN on the first `.`. Empty when FQDN equals
   `name`.
5. Fallback chain when the exec runner is unavailable or `hostname -s` /
   `hostname` fail: short name from gopsutil's `host.Info`, `machine_name` from
   `os.Hostname()`. Keeps minimal containers without `util-linux-hostname`
   working.

Both commands are invoked through the shared `internal/executor` runner; tests
mock present / absent / empty-output cases via `go.uber.org/mock`.

## Backing library

- [`internal/executor`](../../internal/executor) — shared command-runner
  abstraction used to invoke `hostname -s` and `hostname` on both Linux and
  macOS. Tests mock it with `go.uber.org/mock`.
- Go stdlib `net` package for forward/reverse DNS canonicalization.
- [`github.com/shirou/gopsutil/v4/host`](https://github.com/shirou/gopsutil) —
  BSD-3. Fallback source for the short hostname when the exec runner is
  unavailable.
