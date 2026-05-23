# VMware

> **Status:** Implemented ✅

## Description

Reports VMware Tools statistics from inside a VMware guest VM. Requires VMware
Tools (`vmware-toolbox-cmd`) to be installed on the guest. When the host is not
a VMware guest, or when VMware Tools is absent, `Collect` returns `nil` with no
error.

The collector distinguishes VMware Workstation/Fusion (desktop) from vSphere
using the raw session JSON query — a technique taken directly from Ohai's
`vmware.rb` plugin.

## Signals

The collector reports two related signals about host type:

- `host_type` — `"vmware_vsphere"` when the raw session JSON returns data
  (indicates the host hypervisor is vSphere/ESXi), or `"vmware_desktop"` for
  Workstation/Fusion. Use this to gate vSphere-specific automation.
- `host_version` — populated only when `host_type` is `"vmware_vsphere"`;
  contains the ESXi host version string. Empty on desktop hypervisors.

## Collected Fields

| Field             | Type   | Description                                                | Schema mapping                  |
| ----------------- | ------ | ---------------------------------------------------------- | ------------------------------- |
| `version`         | string | VMware Tools version (`12.3.0 build-21581411`).            | No direct OCSF or OTel mapping. |
| `hosttime`        | string | Host wall-clock time reported by VMware Tools.             | No direct schema mapping.       |
| `speed`           | string | CPU speed as reported by the hypervisor (MHz).             | No direct schema mapping.       |
| `session_id`      | string | VMware session identifier.                                 | No direct schema mapping.       |
| `balloon`         | string | Memory balloon size (MB).                                  | No direct schema mapping.       |
| `swap`            | string | Swapped memory size (MB).                                  | No direct schema mapping.       |
| `mem_limit`       | string | Memory hard limit enforced by the hypervisor (MB).         | No direct schema mapping.       |
| `mem_res`         | string | Memory reservation (MB).                                   | No direct schema mapping.       |
| `cpu_res`         | string | CPU reservation (MHz).                                     | No direct schema mapping.       |
| `cpu_limit`       | string | CPU hard limit enforced by the hypervisor (MHz).           | No direct schema mapping.       |
| `upgrade_status`  | string | VMware Tools upgrade availability status.                  | No direct schema mapping.       |
| `timesync_status` | string | Time synchronization status.                               | No direct schema mapping.       |
| `host_type`       | string | Hypervisor type: `"vmware_vsphere"` or `"vmware_desktop"`. | No direct schema mapping.       |
| `host_version`    | string | ESXi host version (vSphere only; empty on desktop).        | No direct schema mapping.       |

## Platform Support

| Platform | Supported                                                       |
| -------- | --------------------------------------------------------------- |
| Linux    | ✅ (checks `/proc/scsi/scsi` then `vmware-toolbox-cmd` probe)   |
| macOS    | ✅ (`vmware-toolbox-cmd` probe; VMware Fusion guests supported) |

## Example Output

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

## SDK Usage

```go
g, _ := gohai.New(gohai.WithCollectors("vmware"))
facts, _ := g.Collect(context.Background())
if vm := facts.VMware; vm != nil {
    fmt.Println(vm.HostType, vm.Version)
}
```

## Enable/Disable

```bash
gohai --collector.vmware    # enable (opt-in)
gohai --no-collector.vmware # disable
```

DefaultEnabled: `false` — VMware Tools is not universally installed; callers
must opt in explicitly.

## Dependencies

None.

## Data Sources

On Linux the collector cascades through two detection signals before collecting:

1. **SCSI controller check:** read `/proc/scsi/scsi` and look for "VMware" in
   the output. A VMware SCSI controller is present on all VMware guests that
   have a virtual disk — this is a zero-exec fast path.
2. **Tool probe:** if `/proc/scsi/scsi` is absent or contains no VMware string,
   run `vmware-toolbox-cmd -v`. A successful non-empty response confirms the
   guest is VMware. This handles edge cases such as NVMe-only guests where no
   SCSI controller is attached.

If neither signal detects VMware, `Collect` returns `nil` immediately.

When VMware is confirmed, the collector runs `vmware-toolbox-cmd stat <param>`
for each of: `hosttime`, `speed`, `sessionid`, `balloon`, `swap`, `memlimit`,
`memres`, `cpures`, `cpulimit`. It then runs `vmware-toolbox-cmd upgrade status`
and `vmware-toolbox-cmd timesync status`. "UpdateInfo failed" responses are
filtered to empty strings — Ohai's `vmware.rb` applies the same filter.

To distinguish desktop vs vSphere, the collector runs
`vmware-toolbox-cmd stat raw json session`. An empty response indicates VMware
Workstation/Fusion (`host_type: "vmware_desktop"`); a JSON object with a
`"version"` key indicates vSphere (`host_type: "vmware_vsphere"`, `host_version`
set from the JSON). This mirrors Ohai's detection logic.

On macOS (VMware Fusion guest) the collector probes `vmware-toolbox-cmd -v`
directly; if absent, returns `nil`. Collection proceeds identically when
present.

Windows is not implemented — Ohai's `vmware.rb` uses a different path
(`C:/Program Files/VMWare/VMware Tools/VMwareToolboxCmd.exe`) and gohai does not
target Windows.

## Backing library

- [`internal/executor`](../../internal/executor) — shared command-runner
  abstraction for all `vmware-toolbox-cmd` invocations.
- [`github.com/avfs/avfs`](https://github.com/avfs/avfs) — virtual filesystem
  for reading `/proc/scsi/scsi` on Linux. Tests inject a `memfs` instance.
