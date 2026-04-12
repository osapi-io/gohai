# Hostname

> **Status:** Implemented ✅

## Description

Identifies the system hostname, FQDN, and domain. Wraps
[gopsutil's `host.Info`](https://pkg.go.dev/github.com/shirou/gopsutil/v4/host)
for the short hostname and performs a DNS CNAME lookup (via Go's
`net.LookupCNAME`) to resolve the FQDN. The domain is derived from the FQDN.

## Collected Fields

| Field      | Type   | Description                                             |
| ---------- | ------ | ------------------------------------------------------- |
| `hostname` | string | Short hostname (e.g., `web01`)                          |
| `fqdn`     | string | Fully qualified domain name (e.g., `web01.example.com`) |
| `domain`   | string | Domain portion of the FQDN (e.g., `example.com`)        |

If no DNS record exists for the short name, `fqdn` falls back to the short
hostname and `domain` is empty.

## Platform Support

| Platform | Source                                                 | Supported |
| -------- | ------------------------------------------------------ | --------- |
| Linux    | `gopsutil/v4/host.InfoWithContext` + `net.LookupCNAME` | ✅        |
| macOS    | `gopsutil/v4/host.InfoWithContext` + `net.LookupCNAME` | ✅        |
| Other    | Returns `nil` (no hostname data)                       | —         |

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
import (
    "context"

    "github.com/osapi-io/gohai/pkg/gohai"
    "github.com/osapi-io/gohai/pkg/gohai/collectors/hostname"
)

g, _ := gohai.New(gohai.WithCollectors("hostname"))
facts, _ := g.Collect(context.Background())

info := facts.Data["hostname"].(*hostname.Info)
fmt.Println(info.Hostname, info.FQDN, info.Domain)
```

## Enable/Disable

```bash
gohai --collector.hostname      # enable (default)
gohai --no-collector.hostname   # disable
```

## Dependencies

None — Tier 1 core collector with no upstream collector dependencies.

## Backing library

- [`github.com/shirou/gopsutil/v4/host`](https://github.com/shirou/gopsutil) —
  BSD-3
- Go stdlib `net.LookupCNAME` for FQDN resolution
