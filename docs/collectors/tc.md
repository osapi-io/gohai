# tc

## Description

Reports Linux traffic control (qdisc) configuration via `tc -s qdisc show`. Each
network interface gets a list of queuing disciplines attached to it. macOS does
not ship the `tc` tool; Darwin returns nil gracefully.

Mirrors Ohai's `linux/tc.rb` methodology: parse per-interface qdisc entries from
the `tc` command output, recording each qdisc's kind, handle, and parent.

## Collected Fields

| Field                          | Type   | Schema mapping             | Notes                                                 |
| ------------------------------ | ------ | -------------------------- | ----------------------------------------------------- |
| `interfaces`                   | list   | —                          | List of interfaces with qdisc configuration           |
| `interfaces[].name`            | string | gohai convention: `name`   | Network interface name, e.g. `eth0`                   |
| `interfaces[].qdiscs`          | list   | —                          | Qdiscs attached to this interface                     |
| `interfaces[].qdiscs[].kind`   | string | gohai convention: `kind`   | Qdisc type: `fq_codel`, `pfifo_fast`, `noqueue`, etc. |
| `interfaces[].qdiscs[].handle` | string | gohai convention: `handle` | Qdisc handle (e.g. `1:`, `0:`)                        |
| `interfaces[].qdiscs[].parent` | string | gohai convention: `parent` | Parent handle; empty for root qdiscs                  |

## Platform Support

| Platform | Supported | Backing source               |
| -------- | --------- | ---------------------------- |
| Linux    | Yes       | `tc -s qdisc show`           |
| macOS    | No        | Returns nil (tc not present) |

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
g := gohai.New(gohai.WithEnabled("tc"))
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

## Backing Library

`internal/executor.Executor` for command execution.
