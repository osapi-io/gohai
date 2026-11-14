# Disk

> **Status:** Implemented ✅

## Description

The collector reports the disk I/O counters of each device: the read and write
counts, the bytes, and the latency. The data is the same as what `iostat` shows
at the block-device layer. Use it to find saturated disks and unbalanced I/O
across NVMe devices, and to see the storage workload over time.

Note: this collector reports **I/O counters only**. The physical device metadata
(model, vendor, serial, rotational, block sizes) comes from the
[`block_device`](block_device.md) collector, which reads sysfs. Ohai splits
these the same way.

## Collected fields

| Field per device | Type   | Description                                               | Schema mapping            |
| ---------------- | ------ | --------------------------------------------------------- | ------------------------- |
| `name`           | string | The device name, for example `sda`, `nvme0n1` or `disk0`. | No direct schema mapping. |
| `read_count`     | uint64 | The number of completed reads.                            | No direct schema mapping. |
| `write_count`    | uint64 | The number of completed writes.                           | No direct schema mapping. |
| `read_bytes`     | uint64 | The bytes read.                                           | No direct schema mapping. |
| `write_bytes`    | uint64 | The bytes written.                                        | No direct schema mapping. |
| `read_time`      | uint64 | The time of the reads (ms).                               | No direct schema mapping. |
| `write_time`     | uint64 | The time of the writes (ms).                              | No direct schema mapping. |
| `io_time`        | uint64 | The time when I/O was in progress (ms).                   | No direct schema mapping. |

The top level is `devices: []Device`.

## Platform support

| Platform | Supported                    |
| -------- | ---------------------------- |
| Linux    | ✅ (reads `/proc/diskstats`) |
| macOS    | ✅ (IOKit through gopsutil)  |

## Example output

```json
{
  "disk": {
    "devices": [
      {
        "name": "sda",
        "read_count": 152341,
        "write_count": 98234,
        "read_bytes": 9823746048,
        "write_bytes": 4123498752,
        "read_time": 482013,
        "write_time": 182947,
        "io_time": 623481
      }
    ]
  }
}
```

## SDK usage

```go
g, _ := gohai.New(gohai.WithCollectors("disk"))
facts, _ := g.Collect(context.Background())
for _, d := range facts.Disk.Devices {
    fmt.Printf("%s: %d reads / %d writes\n", d.Name, d.ReadCount, d.WriteCount)
}
```

## Enable/disable

```bash
gohai --collector.disk      # enable (default)
gohai --no-collector.disk   # disable
```

## Dependencies

None.

## Data sources

On Linux:

1. gopsutil's `disk.IOCountersWithContext` reads `/proc/diskstats`. For each
   device, it returns the read and write counts, the bytes and the time. The
   collector gives each device as a `Device` row and does not change it.

On macOS:

1. gopsutil's `disk.IOCountersWithContext` calls IOKit and returns the same
   shape. The device names are the same as the names that `iostat` reports, for
   example `disk0`.

This collector does not report the physical device metadata (model, vendor,
serial, rotational, block sizes). The [`block_device`](block_device.md)
collector reports it. Ohai splits the two topics the same way. Ohai has no
`diskstats` data, and `linux/block_device.rb` reads only the sysfs device
metadata.

## Backing library

- [`github.com/shirou/gopsutil/v4/disk`](https://github.com/shirou/gopsutil),
  BSD-3.
