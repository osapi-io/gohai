# Interrupts

> **Status:** Implemented ✅

## Description

The collector reports the IRQ statistics that it parses from `/proc/interrupts`
on Linux. It reports one entry for each interrupt line. Each entry has the IRQ
identifier, an optional interrupt controller type, an optional device name, and
the event count for each CPU.

Consumers use this data to:

- Examine how interrupts go to each CPU, for latency and load analysis.
- Find the hardware devices that cause high interrupt rates.
- Audit the IRQ affinity for high-performance tuning.

The collector uses the same method as Ohai's `interrupts` plugin. It reads
`/proc/interrupts` and parses the format that the kernel documents in
[Documentation/filesystems/proc.txt][kernel-proc-txt].

## Collected fields

| Field                   | Type      | Description                                                                                                                        | Schema mapping   |
| ----------------------- | --------- | ---------------------------------------------------------------------------------------------------------------------------------- | ---------------- |
| `irqs`                  | `array`   | The list of interrupt lines from `/proc/interrupts`.                                                                               | gohai convention |
| `irqs[].number`         | `string`  | The IRQ identifier. This is a decimal number for hardware IRQs (`"0"`, `"9"`) or a label for architecture IRQs (`"NMI"`, `"ERR"`). | gohai convention |
| `irqs[].type`           | `string`  | The interrupt controller type, for example `"IO-APIC"` or `"PCI-MSI"`. Empty for non-numeric IRQs that do not have this field.     | gohai convention |
| `irqs[].device`         | `string`  | The driver or device name, for example `"timer"` or `"eth0"`. Empty for non-numeric IRQs, or if the line shows no device.          | gohai convention |
| `irqs[].counts_per_cpu` | `[]int64` | The event count for each CPU, in the order of the CPU index. The slice length is the same as the CPU count in the header line.     | gohai convention |

## Platform support

| Platform | Supported                   |
| -------- | --------------------------- |
| Linux    | ✅                          |
| macOS    | nil (no `/proc/interrupts`) |

macOS does not expose `/proc/interrupts`. The Darwin variant returns `nil`.

## Example output

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

## SDK usage

```go
import (
    "context"

    "github.com/osapi-io/gohai/pkg/gohai"
)

g, _ := gohai.New(gohai.WithCollectors("interrupts"))
facts, _ := g.Collect(context.Background())
// facts.Interrupts.IRQs contains the list
```

## Enable/disable

```bash
gohai collect --collector.interrupts      # enable
gohai collect --no-collector.interrupts   # disable
```

## Dependencies

None.

## Data sources

Ohai's `linux/interrupts.rb` parses `/proc/interrupts` in a similar way. It
reads the count for each CPU and the device and type label of each IRQ line.
gohai uses the same method. But gohai keeps the results in a structured array.
Ohai keeps them in a nested hash with the IRQ number as the key.

On Linux:

1. The collector reads `/proc/interrupts` through the injected `avfs.VFS`. If
   the file is missing, the collector returns an empty IRQ list and no error.
   This occurs in containers that do not bind-mount `/proc`.
2. The collector parses the first line as the CPU header to get the CPU count.
3. For each line after the header, the collector splits the line at `:` to get
   the IRQ number. Then it parses the next _N_ tokens as decimal event counts.
   Whitespace separates the tokens, and _N_ is the CPU count. The collector
   skips lines that have no colon.
4. For numeric IRQs, the token at index _N_ is the interrupt controller type, if
   it is there. The token at index _N+1_ is the interrupt vector, and the
   collector skips it. The collector joins the tokens from index _N+2_ to the
   end to make the device name.
5. For non-numeric IRQs, for example `NMI`, `LOC` and `ERR`, the collector joins
   the tokens after the counts to make the type label. Usually, `ERR` and `MIS`
   have no tokens after the counts.
6. If a count token is not a decimal number, the collector returns a parse
   error.

On macOS, the collector returns `nil`.

## Backing library

- [`github.com/avfs/avfs`](https://github.com/avfs/avfs) to read
  `/proc/interrupts`. Production uses `osfs`. Tests use `memfs`.
- The Go standard library `bufio` and `strconv` to scan the lines and parse the
  counts.

[kernel-proc-txt]: https://www.kernel.org/doc/Documentation/filesystems/proc.txt
