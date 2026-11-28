# TC

> **Status:** Implemented ✅

## Description

The collector reports the Linux traffic control (qdisc) configuration. It gets
the data from `tc -s qdisc show`. Each network interface has a list of the
queuing disciplines that are attached to it. macOS does not have the `tc` tool.
On Darwin, the collector gives no fact, and its status is `unsupported`.

The method is the same as Ohai's `linux/tc.rb`. The collector parses the qdisc
entries of each interface from the `tc` command output. It records the kind, the
handle and the parent of each qdisc. gohai also records the statistics that
`-s` prints under each qdisc, which Ohai does not.

## Collected fields

| Field                          | Type     | Description                                                                                                                  | Schema mapping              |
| ------------------------------ | -------- | ---------------------------------------------------------------------------------------------------------------------------- | --------------------------- |
| `interfaces`                   | `list`   | The list of interfaces that have a qdisc configuration                                                                       | None                        |
| `interfaces[].name`            | `string` | The network interface name, for example `eth0`                                                                               | No direct OCSF/OTel mapping |
| `interfaces[].qdiscs`          | `list`   | The qdiscs that are attached to this interface                                                                               | None                        |
| `interfaces[].qdiscs[].kind`   | `string` | The qdisc type, for example `fq_codel`, `pfifo_fast` or `noqueue`                                                            | No direct OCSF/OTel mapping |
| `interfaces[].qdiscs[].handle` | `string` | The qdisc handle without its colon, for example `1` or `0`                                                                   | No direct OCSF/OTel mapping |
| `interfaces[].qdiscs[].parent` | `string` | The parent handle, or `root` for a root qdisc, which `tc` prints with the word `root`. It is empty for a qdisc with neither. | No direct OCSF/OTel mapping |
| `interfaces[].qdiscs[].stats` | `object` | The counters that `tc -s` prints under the qdisc. It is absent when `tc` prints no statistics line for the qdisc. | None |
| `interfaces[].qdiscs[].stats.sent_bytes` | `integer` | The bytes that the qdisc sent | No direct OCSF/OTel mapping |
| `interfaces[].qdiscs[].stats.sent_packets` | `integer` | The packets that the qdisc sent | No direct OCSF/OTel mapping |
| `interfaces[].qdiscs[].stats.dropped` | `integer` | The packets that the qdisc dropped | No direct OCSF/OTel mapping |
| `interfaces[].qdiscs[].stats.overlimits` | `integer` | The times that the qdisc was over its limit | No direct OCSF/OTel mapping |
| `interfaces[].qdiscs[].stats.requeues` | `integer` | The packets that the qdisc put back in its queue | No direct OCSF/OTel mapping |
| `interfaces[].qdiscs[].stats.backlog_bytes` | `integer` | The bytes in the queue now | No direct OCSF/OTel mapping |
| `interfaces[].qdiscs[].stats.backlog_packets` | `integer` | The packets in the queue now | No direct OCSF/OTel mapping |

## Platform support

| Platform | Supported |
| -------- | --------- |
| Linux    | ✅        |
| macOS    | ❌        |

## Example output

```json
{
  "tc": {
    "interfaces": [
      {
        "name": "lo",
        "qdiscs": [{ "kind": "noqueue", "handle": "0", "parent": "root" }]
      },
      {
        "name": "eth0",
        "qdiscs": [
          {
            "kind": "fq_codel",
            "handle": "0",
            "parent": "root",
            "stats": {
              "sent_bytes": 98765,
              "sent_packets": 432,
              "dropped": 7,
              "overlimits": 0,
              "requeues": 3,
              "backlog_bytes": 1514,
              "backlog_packets": 1
            }
          }
        ]
      },
      {
        "name": "eth1",
        "qdiscs": [{ "kind": "pfifo_fast", "handle": "0", "parent": "1:1" }]
      }
    ]
  }
}
```

## SDK usage

```go
g, _ := gohai.New(gohai.WithCollectors("tc"))
facts, _ := g.Collect(ctx)
```

## Enable/disable

Default: **disabled** (opt-in). The collector needs iproute2 (`tc`). Not all
container images and embedded distributions have iproute2.

```bash
gohai collect --collector.tc              # enable
gohai collect --no-collector.tc           # disable
```

## Dependencies

None.

## Data sources

Ohai's `linux/tc.rb` runs `tc -s qdisc show` and parses the qdisc entries of
each interface. It records the kind, the handle and the parent. gohai parses the
output in the same way. It gets the fields from the `qdisc` lines and groups
them by the `dev` keyword. The method is the same as Ohai's.

On Linux:

1. The collector runs `tc -s qdisc show`.
2. It parses the lines that start with `qdisc`. Each of these lines gives one
   qdisc that is attached to a network interface.
3. The line format is:
   `qdisc <kind> <handle>: dev <iface> [parent <parent>] [root] ...`
4. The collector gets `kind`, `handle`, `iface` and the optional `parent`. It
   removes the trailing `:` from `handle`. This parse is the same as in Ohai's
   `tc.rb`.
5. It skips lines that have fewer than 5 fields or no `dev` keyword.
6. A line that starts with `Sent` or `backlog` after a qdisc line gives the
   statistics of that qdisc. The `Sent` line has the form
   `Sent <bytes> bytes <packets> pkt (dropped <n>, overlimits <n> requeues <n>)`.
   The `backlog` line has the form `backlog <size> <packets>p`. `tc` prints the
   size with a `b`, `Kb`, `Mb` or `Gb` suffix in powers of 1024, and the
   collector converts it to bytes. A value that does not parse is 0. The
   collector skips the other lines, such as blank lines and the per-kind lines
   that `tc` prints after `backlog`.
7. If `tc` is missing, the collector returns an empty list, not an error. If
   `tc` fails, the collector returns the error. Minimal container images and
   non-Linux kernels without iproute2 often have no `tc`.

The interfaces are in the order of their first line in the `tc` output. This
order is the same as the internal order of the kernel.

On macOS, the collector gives no fact, with the status `unsupported`.

## Backing library

`source.Runner` to run commands.
