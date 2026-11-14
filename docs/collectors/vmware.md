# VMware

> **Status:** Implemented ✅

## Description

The collector reports the VMware Tools statistics from inside a VMware guest VM.
VMware Tools (`vmware-toolbox-cmd`) must be installed on the guest. If the host
is not a VMware guest, or if VMware Tools is missing, `Collect` returns `nil`
and no error.

The collector tells VMware Workstation/Fusion (desktop) from vSphere with the
raw session JSON query. This method comes directly from Ohai's `vmware.rb`
plugin.

## Signals

The collector reports two related signals about the host type:

- `host_type`. `"vmware_vsphere"` if the raw session JSON returns data. This
  shows that the host hypervisor is vSphere/ESXi. `"vmware_desktop"` for
  Workstation/Fusion. Use this field to start or stop automation that is only
  for vSphere.
- `host_version`. The ESXi host version. It has a value only if `host_type` is
  `"vmware_vsphere"`. It is empty on desktop hypervisors.

## Collected fields

| Field             | Type   | Description                                                    | Schema mapping                  |
| ----------------- | ------ | -------------------------------------------------------------- | ------------------------------- |
| `version`         | string | The VMware Tools version (`12.3.0 build-21581411`).            | No direct OCSF or OTel mapping. |
| `hosttime`        | string | The host wall-clock time, as VMware Tools reports it.          | No direct schema mapping.       |
| `speed`           | string | The CPU speed, as the hypervisor reports it (MHz).             | No direct schema mapping.       |
| `session_id`      | string | The VMware session identifier.                                 | No direct schema mapping.       |
| `balloon`         | string | The memory balloon size (MB).                                  | No direct schema mapping.       |
| `swap`            | string | The size of the swapped memory (MB).                           | No direct schema mapping.       |
| `mem_limit`       | string | The hard memory limit that the hypervisor applies (MB).        | No direct schema mapping.       |
| `mem_res`         | string | The memory reservation (MB).                                   | No direct schema mapping.       |
| `cpu_res`         | string | The CPU reservation (MHz).                                     | No direct schema mapping.       |
| `cpu_limit`       | string | The hard CPU limit that the hypervisor applies (MHz).          | No direct schema mapping.       |
| `upgrade_status`  | string | The status of VMware Tools upgrades.                           | No direct schema mapping.       |
| `timesync_status` | string | The time synchronization status.                               | No direct schema mapping.       |
| `host_type`       | string | The hypervisor type: `"vmware_vsphere"` or `"vmware_desktop"`. | No direct schema mapping.       |
| `host_version`    | string | The ESXi host version. Only on vSphere. Empty on desktop.      | No direct schema mapping.       |

## Platform support

| Platform | Supported                                                     |
| -------- | ------------------------------------------------------------- |
| Linux    | ✅ (checks `/proc/scsi/scsi`, then runs `vmware-toolbox-cmd`) |
| macOS    | ✅ (runs `vmware-toolbox-cmd`. Supports VMware Fusion guests) |

## Example output

### VMware Workstation / Fusion guest

```json
{
  "vmware": {
    "version": "12.3.0 build-21581411",
    "hosttime": "01 Jan 2026 12:00:00",
    "speed": "2600 MHz",
    "session_id": "1",
    "balloon": "0 MB",
    "swap": "0 MB",
    "mem_limit": "4096 MB",
    "mem_res": "0 MB",
    "cpu_res": "0 MHz",
    "cpu_limit": "unlimited",
    "upgrade_status": "VMware Tools are up-to-date",
    "timesync_status": "Timesync is disabled",
    "host_type": "vmware_desktop"
  }
}
```

### vSphere / ESXi guest

```json
{
  "vmware": {
    "version": "12.3.0 build-21581411",
    "host_type": "vmware_vsphere",
    "host_version": "7.0.3"
  }
}
```

### Non-VMware host

```json
{
  "vmware": null
}
```

## SDK usage

```go
g, _ := gohai.New(gohai.WithCollectors("vmware"))
facts, _ := g.Collect(context.Background())
if vm := facts.VMware; vm != nil {
    fmt.Println(vm.HostType, vm.Version)
}
```

## Enable/disable

```bash
gohai collect --collector.vmware    # enable (opt-in)
gohai collect --no-collector.vmware # disable
```

DefaultEnabled: `false`. Not all hosts have VMware Tools. A caller must enable
the collector explicitly.

## Dependencies

None.

## Data sources

On Linux, the collector does two checks in sequence before it collects data:

1. **SCSI controller check.** The collector reads `/proc/scsi/scsi` and looks
   for "VMware" in the output. All VMware guests that have a virtual disk have a
   VMware SCSI controller. This check runs no command.
2. **Tool check.** If `/proc/scsi/scsi` is missing or has no VMware string, the
   collector runs `vmware-toolbox-cmd -v`. A successful response that is not
   empty shows that the guest is VMware. This check finds special cases, for
   example guests that have only NVMe disks and no SCSI controller.

If the two checks do not find VMware, `Collect` returns `nil` immediately.

If the guest is VMware, the collector runs `vmware-toolbox-cmd stat <param>` for
each of these values: `hosttime`, `speed`, `sessionid`, `balloon`, `swap`,
`memlimit`, `memres`, `cpures`, `cpulimit`. Then it runs
`vmware-toolbox-cmd upgrade status` and `vmware-toolbox-cmd timesync status`.
The collector changes each "UpdateInfo failed" response to an empty string.
Ohai's `vmware.rb` uses the same filter.

To tell desktop from vSphere, the collector runs
`vmware-toolbox-cmd stat raw json session`. An empty response shows VMware
Workstation/Fusion (`host_type: "vmware_desktop"`). A JSON object with a
`"version"` key shows vSphere (`host_type: "vmware_vsphere"`). The collector
sets `host_version` from the JSON. This logic is the same as Ohai's.

On macOS (VMware Fusion guest), the collector runs `vmware-toolbox-cmd -v`
directly. If the command is missing, the collector returns `nil`. If it is
there, the collection is the same as on Linux.

gohai does not support Windows. Ohai's `vmware.rb` uses a different path
(`C:/Program Files/VMWare/VMware Tools/VMwareToolboxCmd.exe`), and gohai does
not target Windows.

## Backing library

- [`internal/executor`](../../internal/executor). The shared command runner. The
  collector uses it to run all `vmware-toolbox-cmd` commands.
- [`github.com/avfs/avfs`](https://github.com/avfs/avfs). The virtual filesystem
  to read `/proc/scsi/scsi` on Linux. Tests inject a `memfs` instance.
