# Network

> **Status:** Implemented ✅ (partial — interfaces only)

## Description

Enumerates network interfaces with MAC addresses, IP addresses, and I/O
counters. Wraps
[gopsutil's `net`](https://pkg.go.dev/github.com/shirou/gopsutil/v4/net).

## Scope

**Covered:**

- Interface list with name, MTU, MAC address, flags
- IP addresses per interface
- Per-interface I/O counters (bytes, packets, errors, drops)

**Not covered yet (planned):**

- `default_interface` — the interface reaching the default gateway
- `default_gateway` — the default gateway address
- `routes` — the full routing table

Routing-table access isn't in gopsutil. These are tracked as follow-ups and will
require either parsing `/proc/net/route` + `netstat -rn` ourselves or pulling in
a routing-specific library (e.g., `vishvananda/netlink` on Linux).

## Collected Fields

Top-level: `interfaces []Interface`.

Per-interface:

| Field           | Type      | Description                       |
| --------------- | --------- | --------------------------------- |
| `name`          | string    | Interface name (e.g., `eth0`)     |
| `mtu`           | int       | Maximum transmission unit         |
| `hardware_addr` | string    | MAC address                       |
| `flags`         | []string  | Flags (up, broadcast, loopback)   |
| `addresses`     | []Address | IP addresses                      |
| `counters`      | Counters  | I/O counters (nil if unavailable) |

`addresses[].addr` is the CIDR string (e.g., `10.0.0.5/24`).

`counters` fields: `bytes_sent`, `bytes_recv`, `packets_sent`, `packets_recv`,
`errin`, `errout`, `dropin`, `dropout`.

## Platform Support

| Platform | Source                                          | Supported |
| -------- | ----------------------------------------------- | --------- |
| Linux    | `gopsutil/v4/net.Interfaces` + `net.IOCounters` | ✅        |
| macOS    | `gopsutil/v4/net.Interfaces` + `net.IOCounters` | ✅        |
| Other    | Returns `nil`                                   | —         |

## Example Output

```json
{
  "network": {
    "interfaces": [
      {
        "name": "eth0",
        "mtu": 1500,
        "hardware_addr": "02:42:ac:11:00:02",
        "flags": ["up", "broadcast", "multicast"],
        "addresses": [
          { "addr": "10.0.0.5/24" },
          { "addr": "fe80::42:acff:fe11:2/64" }
        ],
        "counters": {
          "bytes_sent": 1048576,
          "bytes_recv": 10485760,
          "packets_sent": 2048,
          "packets_recv": 8192
        }
      }
    ]
  }
}
```

## SDK Usage

```go
import (
    "context"

    "github.com/osapi-io/gohai/pkg/gohai"
)

g, _ := gohai.New(gohai.WithCollectors("network"))
facts, _ := g.Collect(context.Background())

for _, iface := range facts.Network.Interfaces {
    fmt.Printf("%s (%s): %d addrs\n", iface.Name, iface.HardwareAddr, len(iface.Addresses))
}
```

## Enable/Disable

```bash
gohai --collector.network      # enable (default)
gohai --no-collector.network   # disable
```

## Dependencies

None.

## Backing library

[`github.com/shirou/gopsutil/v4/net`](https://github.com/shirou/gopsutil) —
BSD-3.
