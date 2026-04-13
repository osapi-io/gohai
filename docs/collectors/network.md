# Network

> **Status:** Implemented ⚠️ (interfaces + counters; routes/gateway/DNS planned)

## Description

Enumerates network interfaces — name, MTU, hardware address, flags, addresses,
and per-interface I/O counters. Matches what `ip addr` + `ip -s link` show on
Linux and `ifconfig` + `netstat -i` show on macOS.

Consumers use this to:

- Discover the public / private addresses for a host (feed a service registry).
- Correlate traffic spikes to specific interfaces.
- Detect MTU mismatches or flapping interfaces (high `errin`/`errout`).

**Scope note:** this collector covers interfaces + counters only. Default
gateway, routing table, and DNS configuration are tracked as follow-ups (see
Known gaps below) — Ohai groups those under the same plugin, we split them for
selectability.

## Collected Fields

Top level: `interfaces: []Interface`.

| Field per interface | Type     | Description                                          | Schema mapping |
| ------------------- | -------- | ---------------------------------------------------- | ------------------------- |
| `name`              | string   | Interface name (`eth0`, `en0`, `lo0`).               | `network_interface.name`. |
| `mtu`               | int      | Maximum transmission unit (bytes).                   | No direct OCSF.           |
| `hardware_addr`     | string   | MAC address (`"aa:bb:cc:dd:ee:ff"`). Empty for `lo`. | `network_interface.mac`.  |
| `flags`             | []string | Interface flags (`up`, `broadcast`, `multicast`).    | No direct OCSF.           |
| `addresses[].addr`  | string   | CIDR (`"192.168.1.5/24"`, `"fe80::abcd/64"`).        | `network_interface.ip`.   |
| `counters.*`        | Counters | I/O counters — see below.                            | No direct OCSF.           |

### `Counters` entry (per interface)

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

| Platform | Supported                                                  |
| -------- | ---------------------------------------------------------- |
| Linux    | ✅ (rtnetlink via gopsutil + `/proc/net/dev` for counters) |
| macOS    | ✅ (`getifaddrs` + `netstat -i` counters via gopsutil)     |

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
          { "addr": "172.17.0.2/16" },
          { "addr": "fe80::42:acff:fe11:2/64" }
        ],
        "counters": {
          "bytes_sent": 123456789,
          "bytes_recv": 987654321,
          "packets_sent": 123456,
          "packets_recv": 987654
        }
      }
    ]
  }
}
```

## SDK Usage

```go
g, _ := gohai.New(gohai.WithCollectors("network"))
facts, _ := g.Collect(context.Background())

for _, iface := range facts.Network.Interfaces {
    for _, a := range iface.Addresses {
        fmt.Printf("%s: %s\n", iface.Name, a.Addr)
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

| Platform | What we read                                                                       | Ohai plugin                                                                                                                                               | Alignment                                                                                                                                                                                                                                                                                   |
| -------- | ---------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Linux    | gopsutil `net.Interfaces` (rtnetlink) + `net.IOCounters` (per-interface counters). | [`linux/network.rb`](https://github.com/chef/ohai/blob/main/lib/ohai/plugins/linux/network.rb) — `ip addr`, `ip route`, `ip -s link`, `/etc/resolv.conf`. | **Same primary source (kernel netlink).** Ohai also surfaces `default_interface`, `default_gateway`, full routes, per-address family/prefixlen/netmask/scope, `arp` cache, and DNS config — we currently expose interfaces-only. Tracked as follow-ups (in-progress issue #54 in the plan). |
| macOS    | gopsutil `net.Interfaces` + `net.IOCounters` (`getifaddrs` + `netstat -i`).        | [`darwin/network.rb`](https://github.com/chef/ohai/blob/main/lib/ohai/plugins/darwin/network.rb) — `ifconfig`, `netstat`, `scutil --dns`.                 | **Equivalent on interfaces + counters.** Ohai also runs `netstat -rn` for routes and `scutil --dns` for resolver config — deferred.                                                                                                                                                         |

**Known gaps vs. Ohai:**

- `default_interface` / `default_gateway` (v4 and v6).
- Full routing table.
- Per-address `family` / `prefixlen` / `netmask` / `scope` fields.
- ARP / neighbor cache.
- `/etc/resolv.conf` nameservers + search domains.

These are tracked as the planned `routes`, `arp`, and `dns` collectors (see
[README collector index](README.md#-network)).

## Backing library

- [`github.com/shirou/gopsutil/v4/net`](https://github.com/shirou/gopsutil) —
  BSD-3.
