# Interrupts

> **Status:** Implemented ✅

## Description

Reports IRQ statistics parsed from `/proc/interrupts` on Linux. One entry is
emitted per interrupt line. Each entry includes the IRQ identifier, an optional
interrupt-controller type, an optional device name, and the per-CPU event
counts.

Consumers use this to:

- Profile interrupt distribution across CPUs for latency and load analysis.
- Identify hardware devices generating high interrupt rates.
- Audit IRQ affinity for high-performance tuning.

The collector mirrors Ohai's `interrupts` plugin methodology: it reads
`/proc/interrupts` and parses the kernel-documented format described in
[Documentation/filesystems/proc.txt][kernel-proc-txt].

## Collected Fields

| Field                   | Type      | Description                                                                                                             | Schema mapping   |
| ----------------------- | --------- | ----------------------------------------------------------------------------------------------------------------------- | ---------------- |
| `irqs`                  | `array`   | List of interrupt lines from `/proc/interrupts`.                                                                        | gohai convention |
| `irqs[].number`         | `string`  | IRQ identifier — a decimal number for hardware IRQs (`"0"`, `"9"`) or a label for architecture IRQs (`"NMI"`, `"ERR"`). | gohai convention |
| `irqs[].type`           | `string`  | Interrupt controller type (e.g. `"IO-APIC"`, `"PCI-MSI"`). Empty for non-numeric IRQs that lack this field.             | gohai convention |
| `irqs[].device`         | `string`  | Driver or device name (e.g. `"timer"`, `"eth0"`). Empty for non-numeric IRQs or when no device is listed.               | gohai convention |
| `irqs[].counts_per_cpu` | `[]int64` | Per-CPU event count in CPU-index order. Slice length equals the CPU count in the header line.                           | gohai convention |

## Platform Support

| Platform | Supported                   |
| -------- | --------------------------- |
| Linux    | ✅                          |
| macOS    | nil (no `/proc/interrupts`) |

macOS does not expose `/proc/interrupts`. The Darwin variant returns `nil`.

## Example Output

```json
{
  "interrupts": {
    "irqs": [
      {
        "number": "0",
        "type": "IO-APIC",
        "device": "timer",
        "counts_per_cpu": [46, 0]
      },
      {
        "number": "9",
        "type": "ACPI",
        "device": "acpi",
        "counts_per_cpu": [0, 0]
      },
      {
        "number": "NMI",
        "type": "Non-maskable interrupts",
        "counts_per_cpu": [0, 0]
      },
      {
        "number": "ERR",
        "counts_per_cpu": [0]
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

g, _ := gohai.New(gohai.WithCollectors("interrupts"))
facts, _ := g.Collect(context.Background())
// facts.Interrupts.IRQs contains the list
```

## Enable/Disable

```bash
gohai --collector.interrupts      # enable
gohai --no-collector.interrupts   # disable
```

## Dependencies

None.

## Data Sources

On Linux:

1. Read `/proc/interrupts` via the injected `avfs.VFS`. If the file is absent
   (containers without `/proc` bind-mounted), return an empty IRQ list without
   error.
2. Parse the first line as the CPU header to determine the CPU count.
3. For each subsequent line: split on `:` to extract the IRQ number, then parse
   the next _N_ whitespace-delimited tokens (where _N_ = CPU count) as decimal
   event counts. Lines without a colon are skipped.
4. For numeric IRQs, the token at index _N_ (if present) is the interrupt
   controller type, index _N+1_ is the interrupt vector (skipped), and index
   _N+2_ onward is joined as the device name.
5. For non-numeric IRQs (e.g. `NMI`, `LOC`, `ERR`), any trailing tokens after
   the counts are joined as the type label. `ERR` and `MIS` typically have no
   trailing tokens at all.
6. Invalid count tokens (non-decimal) propagate as a parse error.

On macOS: returns `nil`.

## Backing Library

- [`github.com/avfs/avfs`](https://github.com/avfs/avfs) (`osfs` in production,
  `memfs` in tests) for the `/proc/interrupts` read.
- Go stdlib `bufio` and `strconv` for line scanning and count parsing.

[kernel-proc-txt]: https://www.kernel.org/doc/Documentation/filesystems/proc.txt
