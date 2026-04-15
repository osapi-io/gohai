# GPU

> **Status:** Implemented ✅

## Description

Enumerates graphics / compute devices attached to the host — both integrated
(Apple Silicon M-series, Intel iGPUs) and discrete (NVIDIA, AMD). Consumers use
this to:

- Correlate workloads with available acceleration (ML training, video encoding).
- Verify a driver/vendor pair in fleet inventory.
- Feed CVE databases keyed off GPU firmware/driver versions.

**No Ohai precedent.** Ohai has no GPU plugin — verified against the full
`chef/ohai` tree (no `gpu.rb`, no graphics/video/display plugins anywhere under
`lib/ohai/plugins/`). This collector is therefore native to gohai: there's no
years-of-distro-quirks methodology to mirror, and no consumers migrating from
Ohai have an existing `node['gpu']` shape we'd be breaking. Collection follows
the same shape as every other gohai hardware collector — ghw wraps
`/sys/class/drm` on Linux, and macOS parses `system_profiler SPDisplaysDataType`
via the shared Executor.

## Collected Fields

| Field       | Type   | Description                                                           | Schema mapping            |
| ----------- | ------ | --------------------------------------------------------------------- | ------------------------- |
| `cards[]`   | list   | One entry per graphics card / GPU adapter.                            | No direct schema mapping. |
| `vendor`    | string | Human-readable vendor name (`"Apple"`, `"NVIDIA Corporation"`).       | No direct schema mapping. |
| `model`     | string | Marketing / device model (`"Apple M1 Pro"`, `"GeForce RTX 4090"`).    | No direct schema mapping. |
| `address`   | string | Linux: PCI address (`"0000:03:00.0"`). Darwin: sppci_bus descriptor.  | No direct schema mapping. |
| `vendor_id` | string | Linux: PCI vendor hex (`"10de"`). Empty on macOS.                     | No direct schema mapping. |
| `device_id` | string | Linux: PCI device hex (`"1c82"`). Empty on macOS.                     | No direct schema mapping. |
| `cores`     | int    | Darwin only: GPU core count reported by `sppci_cores`. Zero on Linux. | No direct schema mapping. |
| `bus`       | string | Darwin: `"builtin"` / `"pcie"` (prefix-stripped). Empty on Linux.     | No direct schema mapping. |

Neither OCSF nor OpenTelemetry semantic conventions define a GPU object, so
field naming is gohai-native. Go-idiomatic snake_case; fields chosen to cover
both Linux PCI data and darwin's system_profiler shape.

## Platform Support

| Platform | Supported                                                               |
| -------- | ----------------------------------------------------------------------- |
| Linux    | ✅ (ghw/gpu walks `/sys/class/drm` + ghw's bundled pci.ids database)    |
| macOS    | ✅ (`system_profiler SPDisplaysDataType -json` via the shared Executor) |

On Linux, containers and minimal VMs typically lack `/sys/class/drm` — the
collector returns an empty `cards` slice with no error.

## Example Output

### Linux

```json
{
  "gpu": {
    "cards": [
      {
        "vendor": "NVIDIA Corporation",
        "model": "GP107 [GeForce GTX 1050 Ti]",
        "address": "0000:03:00.0",
        "vendor_id": "10de",
        "device_id": "1c82"
      }
    ]
  }
}
```

### macOS (Apple Silicon)

```json
{
  "gpu": {
    "cards": [
      {
        "vendor": "Apple",
        "model": "Apple M1 Pro",
        "address": "spdisplays_builtin",
        "cores": 16,
        "bus": "builtin"
      }
    ]
  }
}
```

## SDK Usage

```go
g, _ := gohai.New(gohai.WithCollectors("gpu"))
facts, _ := g.Collect(context.Background())

if gpu := facts.GPU; gpu != nil {
    for _, card := range gpu.Cards {
        fmt.Printf("%s %s\n", card.Vendor, card.Model)
    }
}
```

## Enable/Disable

```bash
gohai --collector.gpu        # enable (opt-in)
gohai --no-collector.gpu     # disable (default)
gohai --category=hardware    # pulls in gpu alongside dmi and friends
```

## Dependencies

None.

## Data Sources

On Linux:

1. ghw/gpu reads `/sys/class/drm` for each `cardN` symlink, follows it to the
   PCI device's sysfs directory, and resolves vendor/product IDs against ghw's
   bundled `pci.ids` database. All library-level — no shell-outs.
2. When `/sys/class/drm` is missing (headless server, stripped container, VM
   without a virtual display), ghw's `load` logs a warning and returns; we
   propagate that as an empty `cards` slice with no error.

On macOS:

1. `system_profiler SPDisplaysDataType -json` is run through the shared
   `internal/executor`. Apple returns one object per GPU in the
   `SPDisplaysDataType` array.
2. `spdisplays_vendor` / `sppci_bus` values carry Apple-ish `sppci_vendor_` /
   `spdisplays_` prefixes; we strip those so consumers don't have to know
   Apple's conventions (e.g. `sppci_vendor_Apple` → `Apple`).
3. `sppci_cores` is parsed as an integer (only meaningful on Apple Silicon
   integrated GPUs); non-numeric values leave `cores = 0`.
4. If `sppci_model` is empty the collector falls back to the top-level `_name`
   field, which is always populated.

## Backing library

- [`github.com/jaypipes/ghw/pkg/gpu`](https://pkg.go.dev/github.com/jaypipes/ghw/pkg/gpu)
  — Linux-only backend; resolves PCI IDs via `ghw/pkg/pci`.
- [`internal/executor`](../../internal/executor) — shared command-runner
  abstraction used to invoke `system_profiler` on macOS. Tests mock it with
  `go.uber.org/mock`.
