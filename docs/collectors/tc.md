# TC

> **Status:** Implemented ✅

## Description

Reports Linux traffic control (qdisc) configuration via `tc -s qdisc show`. Each
network interface gets a list of queuing disciplines attached to it. macOS does
not ship the `tc` tool; Darwin returns nil gracefully.

Mirrors Ohai's `linux/tc.rb` methodology: parse per-interface qdisc entries from
the `tc` command output, recording each qdisc's kind, handle, and parent.

## Collected Fields

| Field                          | Type     | Description                                           | Schema mapping              |
| ------------------------------ | -------- | ----------------------------------------------------- | --------------------------- |
| `interfaces`                   | `list`   | List of interfaces with qdisc configuration            | —                           |
| `interfaces[].name`            | `string` | Network interface name, e.g. `eth0`                    | No direct OCSF/OTel mapping |
| `interfaces[].qdiscs`          | `list`   | Qdiscs attached to this interface                      | —                           |
| `interfaces[].qdiscs[].kind`   | `string` | Qdisc type: `fq_codel`, `pfifo_fast`, `noqueue`, etc. | No direct OCSF/OTel mapping |
| `interfaces[].qdiscs[].handle` | `string` | Qdisc handle (e.g. `1:`, `0:`)                        | No direct OCSF/OTel mapping |
| `interfaces[].qdiscs[].parent` | `string` | Parent handle; empty for root qdiscs                   | No direct OCSF/OTel mapping |

## Platform Support

| Platform | Supported |
| -------- | --------- |
| Linux    | ✅        |
| macOS    | ❌        |

## Example Output

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

## SDK Usage

```go
g, _ := gohai.New(gohai.WithCollectors("tc"))
facts, _ := g.Collect(ctx)
```

## Enable/Disable

Default: **disabled** (opt-in). Requires iproute2 (`tc`) to be installed, which
is not universal across container images and embedded distributions.

```bash
gohai --collector.tc              # enable
gohai --no-collector.tc           # disable
```

## Dependencies

None.

## Data Sources

On Linux:

1. Runs `tc -s qdisc show`.
2. Parses lines that begin with `qdisc` — each such line describes one qdisc
   attached to a network interface. Other lines (statistics, blank lines,
   legend) are skipped.
3. Line format:
   `qdisc <kind> <handle>: dev <iface> [parent <parent>] [root] ...`
4. Extracts `kind`, `handle` (trailing `:` stripped), `iface`, and optional
   `parent`. Mirrors Ohai's `tc.rb` parsing strategy.
5. Skips lines with fewer than 5 fields or missing the `dev` keyword.
6. Returns an empty list (not an error) when `tc` is absent — common on minimal
   container images and non-Linux kernels without iproute2.

Interface ordering follows the order of first appearance in `tc` output, which
is consistent with the kernel's internal ordering.

On macOS: returns nil.

## Backing library

`internal/executor.Executor` for command execution.
