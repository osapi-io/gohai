# Network

> **Status:** Implemented ✅ (interfaces, structured addresses, routes, default
> gateway, link details, ARP/ND)

## Description

Enumerates network interfaces with structured per-address data, the routing
table, and top-level default-gateway facts (v4 + v6). On Linux we additionally
derive the canonical encapsulation name from sysfs ARPHRD types and merge OpenVZ
`venet0:N` aliases under the primary `venet0` interface so consumers querying
the canonical interface name find the addresses.

Consumers use this to:

- Discover the public / private addresses for a host.
- Identify the egress interface and default next-hop.
- Correlate traffic spikes to specific interfaces.
- Filter addresses by `scope` (`Global` vs `Link` vs `Host`) without
  string-parsing CIDRs.
- Detect MTU mismatches or flapping interfaces (high `errin`/`errout`).

DNS resolver configuration (`/etc/resolv.conf` nameservers + search domains)
will live in its own `dns` collector — out of scope here.

## Collected Fields

Top level:

| Field                     | Type        | Description                                           | Schema mapping            |
| ------------------------- | ----------- | ----------------------------------------------------- | ------------------------- |
| `interfaces`              | []Interface | Interfaces enumerated from gopsutil + extensions.     | —                         |
| `routes`                  | []Route     | All routes from `ip route show table main` (v4 + v6). | No direct schema mapping. |
| `default_interface`       | string      | Interface the default IPv4 route exits through.       | No direct schema mapping. |
| `default_gateway`         | string      | Next-hop IPv4 address for the default route.          | No direct schema mapping. |
| `default_inet6_interface` | string      | Interface the default IPv6 route exits through.       | No direct schema mapping. |
| `default_inet6_gateway`   | string      | Next-hop IPv6 address for the default route.          | No direct schema mapping. |
| `neighbours`              | []Neighbour | Kernel ARP + NDP cache (Linux only — netlink).        | No direct schema mapping. |

Per interface:

| Field                     | Type              | Description                                                                                                                                                                                                                                                                                                               | Schema mapping            |
| ------------------------- | ----------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------- |
| `name`                    | string            | Interface name (`eth0`, `en0`, `lo`).                                                                                                                                                                                                                                                                                     | `network_interface.name`. |
| `number`                  | int               | Kernel interface index (`ip link show` column 1). Mirrors Ohai `iface[:number]`.                                                                                                                                                                                                                                          | No direct schema mapping. |
| `state`                   | string            | Admin state (`"up"` / `"down"`) derived from the `up` flag. Mirrors Ohai `iface["state"]`.                                                                                                                                                                                                                                | No direct schema mapping. |
| `mtu`                     | int               | Maximum transmission unit (bytes).                                                                                                                                                                                                                                                                                        | No direct schema mapping. |
| `hardware_addr`           | string            | MAC address (`"aa:bb:cc:dd:ee:ff"`). Empty for loopback.                                                                                                                                                                                                                                                                  | `network_interface.mac`.  |
| `encapsulation`           | string            | Canonical encapsulation: `Ethernet` / `Loopback` / `PPP` / `SLIP` / `IPIP` / `6to4` / `VJSLIP`. Linux only.                                                                                                                                                                                                               | No direct schema mapping. |
| `driver`                  | string            | Kernel driver bound to the NIC (`e1000e`, `virtio_net`, `ixgbe`). Linux only — sysfs-derived.                                                                                                                                                                                                                             | No direct schema mapping. |
| `speed`                   | string            | Negotiated link speed (`"1000Mb/s"`, `"10Gb/s"`). Linux only — ghw-derived.                                                                                                                                                                                                                                               | No direct schema mapping. |
| `duplex`                  | string            | Link duplex (`Full`, `Half`). Linux only — ghw-derived.                                                                                                                                                                                                                                                                   | No direct schema mapping. |
| `flags`                   | []string          | Interface flags (`up`, `broadcast`, `multicast`).                                                                                                                                                                                                                                                                         | No direct schema mapping. |
| `addresses[]`             | []Address         | See below.                                                                                                                                                                                                                                                                                                                | —                         |
| `routes[]`                | []Route           | Routes whose `dev` matches this interface.                                                                                                                                                                                                                                                                                | No direct schema mapping. |
| `counters.*`              | Counters          | I/O counters — see below.                                                                                                                                                                                                                                                                                                 | No direct schema mapping. |
| `ethtool.driver_info`     | map[string]string | `ethtool -i <iface>` output as a map. Common keys: `driver`, `version`, `firmware_version`, `bus_info`, `supports_statistics`, `supports_test`, `supports_eeprom_access`, `supports_register_dump`, `supports_priv_flags`. Linux only; populated only when the `ethtool` binary is on PATH and the interface is Ethernet. | No direct schema mapping. |
| `ethtool.ring_params`     | map[string]int    | `ethtool -g <iface>` — keys are prefixed `max_` (from "Pre-set maximums") or `current_` (from "Current hardware settings") then the field name snake_cased: `max_rx`, `current_rx`, `current_rx_jumbo`, etc. Values are buffer descriptor counts.                                                                         | No direct schema mapping. |
| `ethtool.channel_params`  | map[string]int    | `ethtool -l <iface>` — same `max_` / `current_` prefix convention as `ring_params`. Keys: `rx`, `tx`, `other`, `combined`. Values are queue counts.                                                                                                                                                                       | No direct schema mapping. |
| `ethtool.coalesce_params` | map[string]any    | `ethtool -c <iface>` — most values are integers (microseconds, frame counts). The exception is the `Adaptive RX: on  TX: off` line which becomes two string entries `adaptive_rx` / `adaptive_tx` carrying `"on"`/`"off"`.                                                                                                | No direct schema mapping. |
| `ethtool.offload_params`  | map[string]string | `ethtool -k <iface>` — keys are feature names snake_cased (`rx_checksumming`, `generic_receive_offload`, ...). Values are `"on"` / `"off"`; trailing `[fixed]` / `[requested ...]` annotations are stripped.                                                                                                              | No direct schema mapping. |
| `ethtool.pause_params`    | map[string]bool   | `ethtool -a <iface>` — keys: `autonegotiate`, `rx`, `tx`. Values are booleans (`"on"` → true).                                                                                                                                                                                                                            | No direct schema mapping. |

Per address:

| Field       | Type   | Description                                                       | Schema mapping            |
| ----------- | ------ | ----------------------------------------------------------------- | ------------------------- |
| `addr`      | string | Address literal, no CIDR (`"10.0.0.5"`, `"fe80::1"`).             | `network_interface.ip`.   |
| `family`    | string | `inet` or `inet6`.                                                | No direct schema mapping. |
| `prefixlen` | int    | Prefix length (24, 64, …).                                        | No direct schema mapping. |
| `netmask`   | string | IPv4 netmask derived from prefix (`"255.255.255.0"`); IPv6 omits. | No direct schema mapping. |
| `broadcast` | string | IPv4 broadcast address derived from address + prefix; IPv6 omits. | No direct schema mapping. |
| `scope`     | string | `Global`, `Link`, or `Host` — derived from stdlib classification. | No direct schema mapping. |

Per route:

| Field         | Type   | Description                                                   | Schema mapping            |
| ------------- | ------ | ------------------------------------------------------------- | ------------------------- |
| `destination` | string | Route destination (`"default"`, `"10.0.0.0/24"`, `"::/0"`).   | No direct schema mapping. |
| `family`      | string | `inet` or `inet6`.                                            | No direct schema mapping. |
| `gateway`     | string | Next-hop address.                                             | No direct schema mapping. |
| `interface`   | string | Egress interface (matches an entry in `interfaces[].name`).   | No direct schema mapping. |
| `source`      | string | Preferred source address.                                     | No direct schema mapping. |
| `scope`       | string | `global` / `link` / `host` (kernel-reported, lowercase).      | No direct schema mapping. |
| `proto`       | string | Routing protocol / source (`kernel`, `dhcp`, `static`, `ra`). | No direct schema mapping. |
| `metric`      | int    | Route metric.                                                 | No direct schema mapping. |

Per neighbour:

| Field       | Type   | Description                                                                                             | Schema mapping            |
| ----------- | ------ | ------------------------------------------------------------------------------------------------------- | ------------------------- |
| `address`   | string | IPv4 / IPv6 address of the neighbour.                                                                   | No direct schema mapping. |
| `family`    | string | `inet` or `inet6`.                                                                                      | No direct schema mapping. |
| `mac`       | string | Hardware address (MAC) when known.                                                                      | No direct schema mapping. |
| `interface` | string | Egress interface (resolved from netlink LinkIndex via `net.InterfaceByIndex`).                          | No direct schema mapping. |
| `state`     | string | NUD state: `INCOMPLETE` / `REACHABLE` / `STALE` / `DELAY` / `PROBE` / `FAILED` / `NOARP` / `PERMANENT`. | No direct schema mapping. |

Per `Counters`:

| Field          | Type   | Description          |
| -------------- | ------ | -------------------- |
| `bytes_sent`   | uint64 | Bytes transmitted.   |
| `bytes_recv`   | uint64 | Bytes received.      |
| `packets_sent` | uint64 | Packets transmitted. |
| `packets_recv` | uint64 | Packets received.    |
| `errin`        | uint64 | Receive errors.      |
| `errout`       | uint64 | Transmit errors.     |
| `dropin`       | uint64 | Receive drops.       |
| `dropout`      | uint64 | Transmit drops.      |

## Platform Support

| Platform | Supported                                                                                  |
| -------- | ------------------------------------------------------------------------------------------ |
| Linux    | ✅ (gopsutil interfaces + counters, sysfs ARPHRD, `ip route` v4/v6, OpenVZ alias merge)    |
| macOS    | ✅ (gopsutil interfaces + counters; routes/encap/OpenVZ are Linux-only — see Data Sources) |

## Example Output

```json
{
  "network": {
    "default_interface": "eth0",
    "default_gateway": "10.0.0.1",
    "interfaces": [
      {
        "name": "eth0",
        "mtu": 1500,
        "hardware_addr": "02:42:ac:11:00:02",
        "encapsulation": "Ethernet",
        "flags": ["up", "broadcast", "multicast"],
        "addresses": [
          {
            "addr": "10.0.0.5",
            "family": "inet",
            "prefixlen": 24,
            "netmask": "255.255.255.0",
            "broadcast": "10.0.0.255",
            "scope": "Global"
          },
          {
            "addr": "fe80::42:acff:fe11:2",
            "family": "inet6",
            "prefixlen": 64,
            "scope": "Link"
          }
        ],
        "routes": [
          {
            "destination": "default",
            "family": "inet",
            "gateway": "10.0.0.1",
            "interface": "eth0",
            "proto": "dhcp",
            "metric": 100
          }
        ]
      }
    ],
    "routes": [
      {
        "destination": "default",
        "family": "inet",
        "gateway": "10.0.0.1",
        "interface": "eth0",
        "proto": "dhcp",
        "metric": 100
      }
    ]
  }
}
```

## SDK Usage

```go
g, _ := gohai.New(gohai.WithCollectors("network"))
facts, _ := g.Collect(context.Background())

n := facts.Network
fmt.Printf("egress interface: %s via %s\n", n.DefaultInterface, n.DefaultGateway)

for _, iface := range n.Interfaces {
    for _, a := range iface.Addresses {
        if a.Scope == "Global" {
            fmt.Printf("%s: %s/%d\n", iface.Name, a.Addr, a.Prefixlen)
        }
    }
}
```

## Enable/Disable

```bash
gohai --collector.network      # enable (default)
gohai --no-collector.network   # disable
```

## Dependencies

None.

## Data Sources

On Linux:

1. **gopsutil `net.Interfaces`** enumerates interfaces — name, MTU, hardware
   address, flags, raw address CIDRs.
2. **gopsutil `net.IOCounters`** per-interface bytes/packets/errors/drops.
3. **Address structuring**: each gopsutil CIDR is parsed via `net.ParseCIDR`. We
   populate `family` from address kind, `prefixlen` from the mask, `netmask`
   (IPv4) from the prefix, `broadcast` (IPv4) by OR-ing host bits with the
   inverse mask, and `scope` from stdlib classification (`IsLoopback` → `Host`,
   `IsLinkLocalUnicast` / `IsLinkLocalMulticast` → `Link`, otherwise `Global`).
4. **Encapsulation**: read `/sys/class/net/<iface>/type` (an ARPHRD\_\* integer)
   through the injected `avfs.VFS` and map via a static table to Ohai's
   canonical name (`Ethernet` / `Loopback` / `PPP` / `SLIP` / `VJSLIP` / `IPIP`
   / `6to4`). Unknown ARPHRD values leave the field empty.
5. **Routes**: run `ip -o -4 route show table main` and
   `ip -o -6 route show table main` through the shared `internal/executor`
   runner. Each line is parsed into a `Route` (`destination`, `gateway`,
   `interface`, `source`, `scope`, `proto`, `metric`); multipath entries
   (containing `\`) are collapsed onto a single line via space-rewrite. Routes
   whose destination is `default`, `0.0.0.0/0`, or `::/0` populate the
   corresponding top-level `default_*` fields. Each route is also appended to
   the matching interface's `routes` slice. When `ip` is unavailable (minimal
   containers without iproute2), routing fields stay empty.
6. **Link details**: per-interface `Speed` and `Duplex` come from
   `ghw.Network()` keyed by interface name (ghw reads
   `/sys/class/net/<iface>/speed` and `/duplex`). `Driver` is the basename of
   `/sys/class/net/<iface>/device/driver`'s symlink target, read through the
   injected `avfs.VFS`. Virtual / loopback interfaces typically lack the driver
   symlink and stay empty.
7. **Neighbours**: `vishvananda/netlink.NeighList(0, 0)` returns the kernel
   ARP + NDP cache. Each entry is mapped to a `Neighbour` with address / family
   / MAC / interface (resolved from LinkIndex) / state (NUD bitmask → ip-neigh
   canonical string).
8. **OpenVZ alias merge**: detect OpenVZ guest via `/proc/vz` present AND
   `/proc/bc/0` absent. When detected, any `<base>:<n>` interface (typically
   `venet0:0`) has its addresses appended to the primary interface (`venet0`)
   and the alias is removed from the output. Mirrors Ohai's behaviour so
   `interfaces[venet0]` is queryable for the container's actual IPs.

On macOS we use gopsutil's `getifaddrs` + `netstat -i` only — encapsulation,
routing, and OpenVZ handling are Linux-specific. A future enhancement may add
`route -n get default` + `netstat -nr` parsing for macOS routes.

## Backing library

- [`github.com/shirou/gopsutil/v4/net`](https://github.com/shirou/gopsutil) —
  BSD-3. Primary source for interfaces + counters on both platforms.
- [`github.com/avfs/avfs`](https://github.com/avfs/avfs) — virtual filesystem
  for `/sys/class/net/<iface>/type` and the OpenVZ detection probes (`/proc/vz`,
  `/proc/bc/0`).
- [`internal/executor`](../../internal/executor) — shared command-runner
  abstraction used to invoke `ip route show` on Linux. Tests mock with
  `go.uber.org/mock`.
- [`github.com/jaypipes/ghw`](https://github.com/jaypipes/ghw) — Apache-2.
  Source for per-interface link `Speed` + `Duplex` on Linux.
- [`github.com/vishvananda/netlink`](https://github.com/vishvananda/netlink) —
  Apache-2. Source for the kernel ARP + NDP cache on Linux. Both libraries
  compile cross-platform; their darwin paths return errors at runtime, leaving
  the relevant fields blank.
