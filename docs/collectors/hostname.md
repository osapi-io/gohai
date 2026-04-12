# Hostname

> **Status:** Implemented ✅

## Description

Identifies the system hostname, FQDN, and domain. Wraps
[gopsutil's `host.Info`](https://pkg.go.dev/github.com/shirou/gopsutil/v4/host)
for the short hostname and performs a DNS CNAME lookup (via Go's
`net.LookupCNAME`) to resolve the FQDN. The domain is derived from the FQDN.

Consumers use this to:

- Label telemetry and logs with a canonical host identity.
- Tag service registrations with the externally-resolvable name (FQDN) rather
  than just the kernel's `nodename`.
- Detect hostname drift between `/etc/hostname` and DNS records.

## Collected Fields

| Field      | Type   | Description                                              | OCSF mapping                                                                 |
| ---------- | ------ | -------------------------------------------------------- | ---------------------------------------------------------------------------- |
| `hostname` | string | Short hostname (e.g., `web01`).                          | `device.hostname`.                                                           |
| `fqdn`     | string | Fully qualified domain name (e.g., `web01.example.com`). | `device.domain` (combined with hostname) / OCSF has no dedicated FQDN field. |
| `domain`   | string | Domain portion of the FQDN (e.g., `example.com`).        | `device.domain`.                                                             |

If no DNS record exists for the short name, `fqdn` falls back to the short
hostname and `domain` is empty.

## Platform Support

| Platform | Supported                                     |
| -------- | --------------------------------------------- |
| Linux    | ✅ (`gopsutil/host.Info` + `net.LookupCNAME`) |
| macOS    | ✅ (`gopsutil/host.Info` + `net.LookupCNAME`) |

## Example Output

### With DNS CNAME

```json
{
  "hostname": {
    "hostname": "web01",
    "fqdn": "web01.example.com",
    "domain": "example.com"
  }
}
```

### Without DNS CNAME

```json
{
  "hostname": {
    "hostname": "laptop",
    "fqdn": "laptop"
  }
}
```

## SDK Usage

```go
g, _ := gohai.New(gohai.WithCollectors("hostname"))
facts, _ := g.Collect(context.Background())

info := facts.Hostname
fmt.Println(info.Hostname, info.FQDN, info.Domain)
```

## Enable/Disable

```bash
gohai --collector.hostname      # enable (default)
gohai --no-collector.hostname   # disable
```

## Dependencies

None.

## Data Sources

| Platform | What we read                                                                            | Ohai plugin                                                                                                                                    | Alignment                                                                                                                                                                                                                  |
| -------- | --------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Linux    | gopsutil `host.Info` (reads `/proc/sys/kernel/hostname` / `uname`) + `net.LookupCNAME`. | [`hostname.rb`](https://github.com/chef/ohai/blob/main/lib/ohai/plugins/hostname.rb) — `Socket.gethostname` + `Socket.gethostbyname` for FQDN. | **Equivalent sources** (kernel nodename for short; DNS for FQDN). Ohai also surfaces `machinename` (uname `nodename`, which usually matches hostname but can differ on hosts with manual `/etc/hostname` edits) — tracked. |
| macOS    | Same as Linux.                                                                          | Same `hostname.rb`.                                                                                                                            | **Equivalent.**                                                                                                                                                                                                            |

**Known gaps vs. Ohai:**

- `machine_name` — Ohai emits `machinename` (uname `nodename`) separately from
  `hostname`; we fold them. Adding as an optional field is planned (issue #43 in
  the repo's task list).

## Backing library

- [`github.com/shirou/gopsutil/v4/host`](https://github.com/shirou/gopsutil) —
  BSD-3.
- Go stdlib `net.LookupCNAME` for FQDN resolution.
