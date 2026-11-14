# Network

> **Status:** Implemented ✅ (interfaces, structured addresses, routes, default
> gateway, link details, ARP and ND)

## Description

The collector lists the network interfaces with structured data for each
address. It also reports the routing table and the default-gateway facts at the
top level (IPv4 and IPv6). On Linux, the collector also gets the canonical
encapsulation name from the sysfs ARPHRD types. It also merges the OpenVZ
`venet0:N` aliases into the primary `venet0` interface. Thus a consumer that
queries the canonical interface name finds the addresses.

Consumers use this data to:

- Find the public and private addresses of a host.
- Find the egress interface and the default next hop.
- Relate traffic spikes to specific interfaces.
- Filter addresses by `scope` (`Global`, `Link` or `Host`) without parsing CIDR
  strings.
- Find MTU mismatches or interfaces that go up and down often (high `errin` and
  `errout`).

The DNS resolver configuration (the nameservers and search domains of
`/etc/resolv.conf`) will be in its own `dns` collector. This collector does not
report it.

## Collected fields

Top level:

| Field                     | Type        | Description                                                 | Schema mapping            |
| ------------------------- | ----------- | ----------------------------------------------------------- | ------------------------- |
| `interfaces`              | []Interface | The interfaces from gopsutil, with the gohai extensions.    | None                      |
| `routes`                  | []Route     | All routes from `ip route show table main` (IPv4 and IPv6). | No direct schema mapping. |
| `default_interface`       | string      | The interface that the default IPv4 route goes out through. | No direct schema mapping. |
| `default_gateway`         | string      | The next-hop IPv4 address of the default route.             | No direct schema mapping. |
| `default_inet6_interface` | string      | The interface that the default IPv6 route goes out through. | No direct schema mapping. |
| `default_inet6_gateway`   | string      | The next-hop IPv6 address of the default route.             | No direct schema mapping. |
| `neighbours`              | []Neighbour | The kernel ARP and NDP cache (Linux only, netlink).         | No direct schema mapping. |

Per interface:

| Field                     | Type              | Description                                                                                                                                                                                                                                                                                                                                 | Schema mapping            |
| ------------------------- | ----------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------- |
| `name`                    | string            | The interface name (`eth0`, `en0`, `lo`).                                                                                                                                                                                                                                                                                                   | `network_interface.name`. |
| `number`                  | int               | The kernel interface index (column 1 of `ip link show`). The same as Ohai `iface[:number]`.                                                                                                                                                                                                                                                 | No direct schema mapping. |
| `state`                   | string            | The admin state (`"up"` or `"down"`), from the `up` flag. The same as Ohai `iface["state"]`.                                                                                                                                                                                                                                                | No direct schema mapping. |
| `mtu`                     | int               | The maximum transmission unit, in bytes.                                                                                                                                                                                                                                                                                                    | No direct schema mapping. |
| `mac`                     | string            | The MAC address (`"aa:bb:cc:dd:ee:ff"`). Empty for loopback.                                                                                                                                                                                                                                                                                | `network_interface.mac`.  |
| `encapsulation`           | string            | The canonical encapsulation: `Ethernet`, `Loopback`, `PPP`, `SLIP`, `IPIP`, `6to4` or `VJSLIP`. Linux only.                                                                                                                                                                                                                                 | No direct schema mapping. |
| `driver`                  | string            | The kernel driver of the NIC (`e1000e`, `virtio_net`, `ixgbe`). Linux only, from sysfs.                                                                                                                                                                                                                                                     | No direct schema mapping. |
| `speed`                   | string            | The negotiated link speed (`"1000Mb/s"`, `"10Gb/s"`). Linux only, from ghw.                                                                                                                                                                                                                                                                 | No direct schema mapping. |
| `duplex`                  | string            | The link duplex (`Full`, `Half`). Linux only, from ghw.                                                                                                                                                                                                                                                                                     | No direct schema mapping. |
| `flags`                   | []string          | The interface flags (`up`, `broadcast`, `multicast`).                                                                                                                                                                                                                                                                                       | No direct schema mapping. |
| `addresses[]`             | []Address         | Refer to the table below.                                                                                                                                                                                                                                                                                                                   | None                      |
| `routes[]`                | []Route           | The routes whose `dev` is this interface.                                                                                                                                                                                                                                                                                                   | No direct schema mapping. |
| `counters.*`              | Counters          | The I/O counters. Refer to the table below.                                                                                                                                                                                                                                                                                                 | No direct schema mapping. |
| `ethtool.driver_info`     | map[string]string | The output of `ethtool -i <iface>` as a map. Usual keys: `driver`, `version`, `firmware_version`, `bus_info`, `supports_statistics`, `supports_test`, `supports_eeprom_access`, `supports_register_dump`, `supports_priv_flags`. Linux only. The collector sets it only when the `ethtool` binary is on PATH and the interface is Ethernet. | No direct schema mapping. |
| `ethtool.ring_params`     | map[string]int    | From `ethtool -g <iface>`. Each key has the prefix `max_` (from "Pre-set maximums") or `current_` (from "Current hardware settings"), then the field name in snake case: `max_rx`, `current_rx`, `current_rx_jumbo`, and more. The values are buffer descriptor counts.                                                                     | No direct schema mapping. |
| `ethtool.channel_params`  | map[string]int    | From `ethtool -l <iface>`. The `max_` and `current_` prefixes are the same as in `ring_params`. Keys: `rx`, `tx`, `other`, `combined`. The values are queue counts.                                                                                                                                                                         | No direct schema mapping. |
| `ethtool.coalesce_params` | map[string]any    | From `ethtool -c <iface>`. Most values are integers (microseconds, frame counts). The exception is the `Adaptive RX: on  TX: off` line. It becomes two string entries, `adaptive_rx` and `adaptive_tx`, with the value `"on"` or `"off"`.                                                                                                   | No direct schema mapping. |
| `ethtool.offload_params`  | map[string]string | From `ethtool -k <iface>`. The keys are feature names in snake case (`rx_checksumming`, `generic_receive_offload`, ...). The values are `"on"` or `"off"`. The collector removes the `[fixed]` and `[requested ...]` notes at the end.                                                                                                      | No direct schema mapping. |
| `ethtool.pause_params`    | map[string]bool   | From `ethtool -a <iface>`. Keys: `autonegotiate`, `rx`, `tx`. The values are booleans (`"on"` gives true).                                                                                                                                                                                                                                  | No direct schema mapping. |
| `vlan.id`                 | string            | The 802.1Q VLAN tag (decimal). Linux only, parsed from `ip -d link`.                                                                                                                                                                                                                                                                        | No direct schema mapping. |
| `vlan.protocol`           | string            | `802.1Q` or `802.1ad`. Only in the `ip` form that shows the protocol.                                                                                                                                                                                                                                                                       | No direct schema mapping. |
| `vlan.flags`              | []string          | `REORDER_HDR`, `GVRP` or `LOOSE_BINDING`, when set.                                                                                                                                                                                                                                                                                         | No direct schema mapping. |
| `tunnel_info.*`           | TunnelInfo        | The ip6tnl and ipip tunnel metadata: `proto`, `external`, `remote`, `local`, `encaplimit`, `hoplimit`, `tclass`, `flowlabel`, `addrgenmode`, `numtxqueues`, `numrxqueues`, `gso_max_size`, `gso_max_segs`. Linux only, parsed from `ip -d link`.                                                                                            | No direct schema mapping. |
| `xdp.attached[]`          | []XDPProgram      | The attached XDP (eBPF) programs. Each entry has `mode` (`xdpdrv`, `xdpgeneric`, `xdpoffload` or `xdpmulti`), `id` and `tag`. Linux only, parsed from `ip -d link`.                                                                                                                                                                         | No direct schema mapping. |

Per address:

| Field       | Type   | Description                                                                     | Schema mapping            |
| ----------- | ------ | ------------------------------------------------------------------------------- | ------------------------- |
| `addr`      | string | The address as text, with no CIDR (`"10.0.0.5"`, `"fe80::1"`).                  | `network_interface.ip`.   |
| `family`    | string | `inet` or `inet6`.                                                              | No direct schema mapping. |
| `prefixlen` | int    | The prefix length (24, 64, …).                                                  | No direct schema mapping. |
| `netmask`   | string | The IPv4 netmask, from the prefix (`"255.255.255.0"`). Not set for IPv6.        | No direct schema mapping. |
| `broadcast` | string | The IPv4 broadcast address, from the address and the prefix. Not set for IPv6.  | No direct schema mapping. |
| `scope`     | string | `Global`, `Link` or `Host`, from the classification of the Go standard library. | No direct schema mapping. |

Per route:

| Field         | Type   | Description                                                              | Schema mapping            |
| ------------- | ------ | ------------------------------------------------------------------------ | ------------------------- |
| `destination` | string | The route destination (`"default"`, `"10.0.0.0/24"`, `"::/0"`).          | No direct schema mapping. |
| `family`      | string | `inet` or `inet6`.                                                       | No direct schema mapping. |
| `gateway`     | string | The next-hop address.                                                    | No direct schema mapping. |
| `interface`   | string | The egress interface. It is the same as an entry in `interfaces[].name`. | No direct schema mapping. |
| `source`      | string | The preferred source address.                                            | No direct schema mapping. |
| `scope`       | string | `global`, `link` or `host`, as the kernel reports it, in lowercase.      | No direct schema mapping. |
| `proto`       | string | The routing protocol or source (`kernel`, `dhcp`, `static`, `ra`).       | No direct schema mapping. |
| `metric`      | int    | The route metric.                                                        | No direct schema mapping. |

Per neighbour:

| Field       | Type   | Description                                                                                            | Schema mapping            |
| ----------- | ------ | ------------------------------------------------------------------------------------------------------ | ------------------------- |
| `address`   | string | The IPv4 or IPv6 address of the neighbour.                                                             | No direct schema mapping. |
| `family`    | string | `inet` or `inet6`.                                                                                     | No direct schema mapping. |
| `mac`       | string | The hardware address (MAC), when it is known.                                                          | No direct schema mapping. |
| `interface` | string | The egress interface, from the netlink LinkIndex through `net.InterfaceByIndex`.                       | No direct schema mapping. |
| `state`     | string | The NUD state: `INCOMPLETE`, `REACHABLE`, `STALE`, `DELAY`, `PROBE`, `FAILED`, `NOARP` or `PERMANENT`. | No direct schema mapping. |

Per `Counters`:

| Field          | Type   | Description           |
| -------------- | ------ | --------------------- |
| `bytes_sent`   | uint64 | The bytes sent.       |
| `bytes_recv`   | uint64 | The bytes received.   |
| `packets_sent` | uint64 | The packets sent.     |
| `packets_recv` | uint64 | The packets received. |
| `errin`        | uint64 | The receive errors.   |
| `errout`       | uint64 | The send errors.      |
| `dropin`       | uint64 | The receive drops.    |
| `dropout`      | uint64 | The send drops.       |

## Platform support

| Platform | Supported                                                                                                      |
| -------- | -------------------------------------------------------------------------------------------------------------- |
| Linux    | ✅ (gopsutil interfaces and counters, sysfs ARPHRD, `ip route` IPv4 and IPv6, OpenVZ alias merge)              |
| macOS    | ✅ (gopsutil interfaces and counters. Routes, encapsulation and OpenVZ are Linux only. Refer to Data sources.) |

## Example output

```json
{
  "network": {
    "default_interface": "eth0",
    "default_gateway": "10.0.0.1",
    "interfaces": [
      {
        "name": "eth0",
        "mtu": 1500,
        "mac": "02:42:ac:11:00:02",
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

## SDK usage

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

## Enable/disable

```bash
gohai --collector.network      # enable (default)
gohai --no-collector.network   # disable
```

## Dependencies

None.

## Data sources

On Linux:

01. **gopsutil `net.Interfaces`** lists the interfaces. For each interface, it
    gives the name, the MTU, the hardware address, the flags, the raw address
    CIDRs and `Index`. The collector reports `Index` as the `number` field of
    the interface, which is the same as Ohai's `iface[:number]`. The collector
    gets the admin `state` (`"up"` or `"down"`) from the `up` flag of gopsutil.
    This is the same as Ohai's `iface[:state]`, which reads `ip link show`.

02. **gopsutil `net.IOCounters`** gives the bytes, packets, errors and drops of
    each interface.

03. **Address structure.** The collector parses each gopsutil CIDR with
    `net.ParseCIDR`. It sets these fields:

    - `family`, from the address kind.
    - `prefixlen`, from the mask.
    - `netmask` (IPv4), from the prefix.
    - `broadcast` (IPv4). The collector sets the host bits with an OR of the
      inverse mask.
    - `scope`, from the classification of the Go standard library. `IsLoopback`
      gives `Host`. `IsLinkLocalUnicast` or `IsLinkLocalMulticast` gives `Link`.
      All other addresses give `Global`.

04. **Encapsulation.** The collector reads `/sys/class/net/<iface>/type` (an
    ARPHRD\_\* integer) through the injected `avfs.VFS`. A static table maps the
    value to Ohai's canonical name (`Ethernet`, `Loopback`, `PPP`, `SLIP`,
    `VJSLIP`, `IPIP` or `6to4`). If the ARPHRD value is unknown, the field stays
    empty.

05. **Routes.** The collector runs `ip -o -4 route show table main` and
    `ip -o -6 route show table main` through the shared `internal/executor`
    runner. It parses each line into a `Route` (`destination`, `gateway`,
    `interface`, `source`, `scope`, `proto`, `metric`). A multipath entry
    contains `\`. The collector replaces it with a space, so the entry is on one
    line. A route with the destination `default`, `0.0.0.0/0` or `::/0` sets the
    related top-level `default_*` fields. The collector also adds each route to
    the `routes` slice of its interface. If `ip` is not available, the routing
    fields stay empty. Minimal containers without iproute2 do not have `ip`.

06. **Link details.** The `Speed` and `Duplex` of each interface come from
    `ghw.Network()`, with the interface name as the key. ghw reads
    `/sys/class/net/<iface>/speed` and `/duplex`. `Driver` is the basename of
    the symlink target of `/sys/class/net/<iface>/device/driver`. The collector
    reads it through the injected `avfs.VFS`. Virtual and loopback interfaces
    usually have no driver symlink, so the field stays empty.

07. **Ethtool driver info.** For each interface that is similar to Ethernet, the
    collector runs `ethtool -i <iface>` through the shared `internal/executor`
    runner. It parses the key and value output into
    `interfaces[].ethtool.driver_info` (`driver`, `version`, `firmware_version`,
    `bus_info`, `expansion_rom_version`, `supports_statistics`, `supports_test`,
    `supports_eeprom_access`, `supports_register_dump`, `supports_priv_flags`).
    The collector skips interfaces that are not Ethernet (loopback, tunnels). If
    the binary is missing or a call fails, the field stays empty.

08. **Ethtool tuning set.** For the same interfaces, the collector runs five
    more ethtool probes through the executor. It parses each probe into a typed
    sub-record under `interfaces[].ethtool`:

    - `ethtool -g <iface>` gives `ring_params` (the current and max values of
      rx, rx-mini, rx-jumbo and tx).
    - `ethtool -l <iface>` gives `channel_params` (the current and max values of
      combined, rx, tx and other).
    - `ethtool -c <iface>` gives `coalesce_params` (adaptive-rx, adaptive-tx,
      `rx_usecs`, `tx_usecs`, ...).
    - `ethtool -k <iface>` gives `offload_params` (checksumming, TSO, GSO, GRO
      and LRO on or off, with `[fixed]` when the kernel does not allow changes).
    - `ethtool -a <iface>` gives `pause_params` (autoneg, rx, tx).

    This is the same as a new audit of the method of Ohai's `linux/network.rb`.
    If one probe fails, the collector continues. Only the sub-record of that
    probe stays empty.

09. **Neighbours.** `vishvananda/netlink.NeighList(0, 0)` returns the kernel ARP
    and NDP cache. The collector maps each entry to a `Neighbour` with these
    values: address, family, MAC, interface (from LinkIndex), and state. The
    state is the NUD bitmask as the canonical string of ip-neigh.

10. **OpenVZ alias merge.** The host is an OpenVZ guest if `/proc/vz` is there
    AND `/proc/bc/0` is missing. On an OpenVZ guest, the collector adds the
    addresses of each `<base>:<n>` interface (usually `venet0:0`) to the primary
    interface (`venet0`). Then it removes the alias from the output. This is the
    same as Ohai, so a query for `interfaces[venet0]` finds the real IPs of the
    container.

On macOS, the collector uses only gopsutil's `getifaddrs` and `netstat -i`.
Encapsulation, routing and the OpenVZ merge are Linux only. A future change can
parse `route -n get default` and `netstat -nr` to get the macOS routes.

## Backing library

- [`github.com/shirou/gopsutil/v4/net`](https://github.com/shirou/gopsutil),
  BSD-3. The primary source for interfaces and counters on the two platforms.
- [`github.com/avfs/avfs`](https://github.com/avfs/avfs). The virtual file
  system for `/sys/class/net/<iface>/type` and the OpenVZ probes (`/proc/vz`,
  `/proc/bc/0`).
- [`internal/executor`](../../internal/executor). The shared command runner. The
  collector uses it to run `ip route show` on Linux. Tests mock it with
  `go.uber.org/mock`.
- [`github.com/jaypipes/ghw`](https://github.com/jaypipes/ghw). Apache-2. The
  source for the link `Speed` and `Duplex` of each interface on Linux.
- [`github.com/vishvananda/netlink`](https://github.com/vishvananda/netlink),
  Apache-2. The source for the kernel ARP and NDP cache on Linux. You can
  compile the two libraries on all platforms. On darwin, they return errors at
  runtime, and the related fields stay empty.
