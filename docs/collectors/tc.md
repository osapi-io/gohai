# TC

> **Status:** Implemented ✅

## Description

The collector reports the Linux traffic control (qdisc) configuration. It gets
the data from `tc -s qdisc show`. Each network interface has a list of the
queuing disciplines that are attached to it. macOS does not have the `tc` tool.
On Darwin, the collector returns nil and no error.

The method is the same as Ohai's `linux/tc.rb`. The collector parses the qdisc
entries of each interface from the `tc` command output. It records the kind, the
handle and the parent of each qdisc.

## Collected fields

| Field                          | Type     | Description                                                       | Schema mapping              |
| ------------------------------ | -------- | ----------------------------------------------------------------- | --------------------------- |
| `interfaces`                   | `list`   | The list of interfaces that have a qdisc configuration            | None                        |
| `interfaces[].name`            | `string` | The network interface name, for example `eth0`                    | No direct OCSF/OTel mapping |
| `interfaces[].qdiscs`          | `list`   | The qdiscs that are attached to this interface                    | None                        |
| `interfaces[].qdiscs[].kind`   | `string` | The qdisc type, for example `fq_codel`, `pfifo_fast` or `noqueue` | No direct OCSF/OTel mapping |
| `interfaces[].qdiscs[].handle` | `string` | The qdisc handle, for example `1:` or `0:`                        | No direct OCSF/OTel mapping |
| `interfaces[].qdiscs[].parent` | `string` | The parent handle. It is empty for root qdiscs                    | No direct OCSF/OTel mapping |

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
        "qdiscs": [{ "kind": "noqueue", "handle": "0" }]
      },
      {
        "name": "eth0",
        "qdiscs": [{ "kind": "fq_codel", "handle": "0" }]
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
   qdisc that is attached to a network interface. The collector skips the other
   lines (statistics, blank lines, legend).
3. The line format is:
   `qdisc <kind> <handle>: dev <iface> [parent <parent>] [root] ...`
4. The collector gets `kind`, `handle`, `iface` and the optional `parent`. It
   removes the trailing `:` from `handle`. This parse is the same as in Ohai's
   `tc.rb`.
5. It skips lines that have fewer than 5 fields or no `dev` keyword.
6. If `tc` is missing, the collector returns an empty list, not an error.
   Minimal container images and non-Linux kernels without iproute2 often have no
   `tc`.

The interfaces are in the order of their first line in the `tc` output. This
order is the same as the internal order of the kernel.

On macOS, the collector returns `nil`.

## Backing library

`source.Runner` to run commands.
