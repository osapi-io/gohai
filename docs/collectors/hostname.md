# Hostname

> **Status:** Implemented ✅

## Description

The collector reports the short name of the host, its fully qualified domain
name (FQDN) and its DNS domain. A DNS round trip makes the FQDN canonical. If
the resolver has a temporary failure, the collector tries again, up to three
times. If all tries fail, it uses the short hostname. Consumers use this data as
a stable identity for telemetry, inventory and correlation.

On macOS, the collector also reports the friendly machine name. This name comes
from `ComputerName`, for example "John's MacBook Pro". The collector uses
`hostname -s`. Thus the short name is always the same as the value that
`$(hostname -s)` gives in other places on the host. This is important on Macs
that MDM manages, because there `scutil --get HostName` can be different.

## Collected fields

| Field          | Type   | Description                                                                                          | Schema mapping                                                                            |
| -------------- | ------ | ---------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------- |
| `name`         | string | The short hostname, for example `web01`.                                                             | OCSF `device.hostname`. The leaf is removed because it is the same as the collector name. |
| `fqdn`         | string | The canonical FQDN. If DNS canonicalization fails on all tries, this is the short hostname.          | `device.domain` + `.hostname`.                                                            |
| `domain`       | string | The DNS domain: all text after the first `.` of the FQDN. Empty if the FQDN is the same as `name`.   | `device.domain`.                                                                          |
| `machine_name` | string | The human-friendly name. On macOS, it comes from `ComputerName`. On Linux, it is the same as `name`. | No direct schema mapping.                                                                 |

## Platform support

| Platform | Supported                                                                    |
| -------- | ---------------------------------------------------------------------------- |
| Linux    | ✅ (`hostname -s` and `hostname` through the executor, DNS canonicalization) |
| macOS    | ✅ (`hostname -s` and `hostname` through the executor, DNS canonicalization) |

## Example output

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

## SDK usage

```go
g, _ := gohai.New(gohai.WithCollectors("hostname"))
facts, _ := g.Collect(context.Background())

info := facts.Hostname
fmt.Println(info.Name, info.FQDN, info.Domain, info.MachineName)
```

## Enable/disable

```bash
gohai collect --collector.hostname      # enable (default)
gohai collect --no-collector.hostname   # disable
```

## Dependencies

None.

## Data sources

On Linux and macOS, the steps are the same. They are the same as the linux and
darwin branches of Ohai's `hostname.rb`:

1. `hostname -s` gives `name`. The collector runs it through the shared
   `source.Runner`. Thus the short name is the same as the value that
   `$(hostname -s)` gives in other tools on the host. This is important on Macs
   that MDM manages, where `scutil --get HostName` can be different. It is also
   important on Linux hosts where a person changed `/etc/hostname` to an FQDN.
2. `hostname` with no arguments gives `machine_name`. On stock macOS, this is
   the friendly name from `ComputerName`. On Linux, it is usually the same as
   the short name.
3. The collector makes the FQDN canonical with `LookupHost` and then
   `LookupAddr` of the `Resolver` of its sources: the system resolver, which
   reads `/etc/hosts` and then DNS, when the caller gives none. On a temporary
   error, it tries again, up to 3 times, with a 100ms backoff. This is the same
   as Ohai's `canonicalize_hostname_with_retries`. If the last try fails, the
   collector uses the short hostname as the FQDN.
4. The collector splits the FQDN at the first `.` to get `domain`. `domain` is
   empty if the FQDN is the same as `name`.
5. If the exec runner is not available, or `hostname -s` or `hostname` fails,
   the collector uses fallbacks. On Linux, the short name is the kernel's name
   in `/proc/sys/kernel/hostname`, up to the first `.`. `machine_name` is the
   short name. Thus the collector also works in minimal containers that do not
   have `util-linux-hostname`. On macOS, there is no fallback for the short
   name, and the collector gives an error.

The collector runs both commands through the shared `source.Runner`. Tests use
`go.uber.org/mock` to mock the cases where the command is there, is missing, or
gives empty output.

## Backing library

- [`source.Runner`](../../pkg/gohai/source). The shared command runner. The
  collector uses it to run `hostname -s` and `hostname` on Linux and macOS.
  Tests mock it with `go.uber.org/mock`.
- The Go standard library `net` package for forward and reverse DNS
  canonicalization.
