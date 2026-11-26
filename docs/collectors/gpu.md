# GPU

> **Status:** Implemented ✅

## Description

The collector lists the graphics and compute devices of the host. It reports
integrated devices (Apple Silicon M-series, Intel iGPUs) and discrete devices
(NVIDIA, AMD). Consumers use this data to:

- Relate workloads to the available acceleration (ML training, video encoding).
- Verify a driver and vendor pair in the fleet inventory.
- Give data to CVE databases that use GPU firmware and driver versions as keys.

**No Ohai precedent.** Ohai has no GPU plugin. The full `chef/ohai` tree has no
`gpu.rb`, and no graphics, video or display plugins under `lib/ohai/plugins/`.
Thus this collector is native to gohai. There is no Ohai method, with its years
of distribution quirks, to copy. No consumer that moves from Ohai has a
`node['gpu']` shape that this collector can break. The collector has the same
shape as the other gohai hardware collectors. On Linux, the collector reads
`/sys/class/drm`. On macOS, the collector parses
`system_profiler SPDisplaysDataType` through the shared Executor.

## Collected fields

| Field       | Type   | Description                                                                  | Schema mapping            |
| ----------- | ------ | ---------------------------------------------------------------------------- | ------------------------- |
| `cards[]`   | list   | One entry for each graphics card or GPU adapter.                             | No direct schema mapping. |
| `vendor`    | string | The vendor name as text (`"Apple"`, `"NVIDIA Corporation"`).                 | No direct schema mapping. |
| `model`     | string | The marketing or device model (`"Apple M1 Pro"`, `"GeForce RTX 4090"`).      | No direct schema mapping. |
| `address`   | string | Linux: the PCI address (`"0000:03:00.0"`). Darwin: the sppci_bus descriptor. | No direct schema mapping. |
| `vendor_id` | string | Linux: the PCI vendor ID in hex (`"10de"`). Empty on macOS.                  | No direct schema mapping. |
| `device_id` | string | Linux: the PCI device ID in hex (`"1c82"`). Empty on macOS.                  | No direct schema mapping. |
| `cores`     | int    | Darwin only: the GPU core count from `sppci_cores`. Zero on Linux.           | No direct schema mapping. |
| `bus`       | string | Darwin: `"builtin"` or `"pcie"`, without the prefix. Empty on Linux.         | No direct schema mapping. |

OCSF and the OpenTelemetry semantic conventions do not define a GPU object. Thus
the field names are native to gohai. They use snake_case. The fields hold the
Linux PCI data and the data of the Darwin system_profiler shape.

## Platform support

| Platform | Supported                                                                   |
| -------- | --------------------------------------------------------------------------- |
| Linux    | ✅ (`/sys/class/drm`, the PCI device in sysfs, and the host's pci.ids)      |
| macOS    | ✅ (`system_profiler SPDisplaysDataType -json` through the shared Executor) |

On Linux, containers and minimal VMs usually have no `/sys/class/drm`. Then the
collector returns an empty `cards` slice and no error.

## Example output

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

## SDK usage

```go
g, _ := gohai.New(gohai.WithCollectors("gpu"))
facts, _ := g.Collect(context.Background())

if gpu := facts.GPU; gpu != nil {
    for _, card := range gpu.Cards {
        fmt.Printf("%s %s\n", card.Vendor, card.Model)
    }
}
```

## Enable/disable

```bash
gohai collect --collector.gpu        # enable (opt-in)
gohai collect --no-collector.gpu     # disable (default)
gohai collect --category=hardware    # pulls in gpu alongside dmi and friends
```

## Dependencies

None.

## Data sources

Ohai has no equivalent that lists GPUs. Ohai does not have a GPU plugin. This
collector is native to gohai.

On Linux:

1. The collector reads `/sys/class/drm` through its filesystem, and finds each
   `cardN` link. A connector such as `card0-HDMI-A-1` is not a card. The last
   PCI address in the target of the link is the card's `address`. A card with no
   PCI address, such as a platform GPU, is left out.
2. The modalias of that PCI device gives the vendor and device IDs, and the
   host's `pci.ids` gives their names, as the [`pci`](pci.md) collector reads
   them.
3. If `/sys/class/drm` is missing (a headless server, a minimal container, a VM
   without a virtual display), the collector returns an empty `cards` slice and
   no error.

On macOS:

1. The collector runs `system_profiler SPDisplaysDataType -json` through the
   shared `source.Runner`. Apple returns one object for each GPU in the
   `SPDisplaysDataType` array.
2. The `spdisplays_vendor` and `sppci_bus` values have Apple prefixes,
   `sppci_vendor_` and `spdisplays_`. The collector removes them, so consumers
   do not have to know the Apple conventions. For example, `sppci_vendor_Apple`
   becomes `Apple`.
3. The collector parses `sppci_cores` as an integer. This value is useful only
   on integrated Apple Silicon GPUs. If the value is not a number, `cores` is
   `0`.
4. If `sppci_model` is empty, the collector uses the top-level `_name` field.
   `_name` always has a value.

## Backing library

- `internal/pcidev`. The PCI devices of sysfs and the names of `pci.ids`, which
  it reads through the filesystem of the collector. The `pci` collector uses it
  too.
- [`source.Runner`](../../pkg/gohai/source). The shared command runner. The
  collector uses it to run `system_profiler` on macOS. Tests mock it with
  `go.uber.org/mock`.
