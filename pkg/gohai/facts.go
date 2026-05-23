// Copyright (c) 2026 John Dewey

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to
// deal in the Software without restriction, including without limitation the
// rights to use, copy, modify, merge, publish, distribute, sublicense, and/or
// sell copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
// FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER
// DEALINGS IN THE SOFTWARE.

// Package gohai is the public SDK for collecting system facts.
package gohai

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/osapi-io/gohai/pkg/gohai/collectors/alibaba"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/azure"
	blockdevice "github.com/osapi-io/gohai/pkg/gohai/collectors/block_device"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/command"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/cpu"
	digitalocean "github.com/osapi-io/gohai/pkg/gohai/collectors/digital_ocean"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/disk"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/dmi"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/docker"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/ec2"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/filesystem"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/fips"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/gce"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/gpu"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/grub2"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/hardware"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/hostname"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/hostnamectl"
	initd "github.com/osapi-io/gohai/pkg/gohai/collectors/init"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/interrupts"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/ipc"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/kernel"
	kernelmodules "github.com/osapi-io/gohai/pkg/gohai/collectors/kernel_modules"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/languages"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/libvirt"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/linode"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/livepatch"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/load"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/lsb"
	machineid "github.com/osapi-io/gohai/pkg/gohai/collectors/machine_id"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/mdadm"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/memory"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/network"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/oci"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/openstack"
	osrelease "github.com/osapi-io/gohai/pkg/gohai/collectors/os_release"
	packagemgr "github.com/osapi-io/gohai/pkg/gohai/collectors/package_mgr"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/packages"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/pci"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/platform"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/process"
	rootgroup "github.com/osapi-io/gohai/pkg/gohai/collectors/root_group"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/rpm"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/scaleway"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/scsi"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/selinux"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/services"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/sessions"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/shard"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/shells"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/ssh"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/sysconf"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/sysctl"
	systemdpaths "github.com/osapi-io/gohai/pkg/gohai/collectors/systemd_paths"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/tc"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/timezone"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/uptime"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/users"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/virtualbox"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/virtualization"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/vmware"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/zpools"
)

// Facts holds the result of a collection run. Each collector populates its
// own typed field; disabled or failed collectors leave their field nil.
// Facts round-trips through JSON cleanly — marshaled output can be
// unmarshaled back into a Facts value without losing type information.
type Facts struct {
	Platform       *platform.Info       `json:"platform,omitempty"`
	Hostname       *hostname.Info       `json:"hostname,omitempty"`
	Kernel         *kernel.Info         `json:"kernel,omitempty"`
	KernelModules  *kernelmodules.Info  `json:"kernel_modules,omitempty"`
	Uptime         *uptime.Info         `json:"uptime,omitempty"`
	Virtualization *virtualization.Info `json:"virtualization,omitempty"`
	MachineID      *machineid.Info      `json:"machine_id,omitempty"`
	CPU            *cpu.Info            `json:"cpu,omitempty"`
	Load           *load.Info           `json:"load,omitempty"`
	Memory         *memory.Info         `json:"memory,omitempty"`
	Filesystem     *filesystem.Info     `json:"filesystem,omitempty"`
	Disk           *disk.Info           `json:"disk,omitempty"`
	Network        *network.Info        `json:"network,omitempty"`
	Process        *process.Info        `json:"process,omitempty"`
	Users          *users.Info          `json:"users,omitempty"`
	Sessions       *sessions.Info       `json:"sessions,omitempty"`
	Timezone       *timezone.Info       `json:"timezone,omitempty"`
	RootGroup      *rootgroup.Info      `json:"root_group,omitempty"`
	Shells         *shells.Info         `json:"shells,omitempty"`
	Fips           *fips.Info           `json:"fips,omitempty"`
	OSRelease      *osrelease.Info      `json:"os_release,omitempty"`
	LSB            *lsb.Info            `json:"lsb,omitempty"`
	Init           *initd.Info          `json:"init,omitempty"`
	Shard          *shard.Info          `json:"shard,omitempty"`
	PackageMgr     *packagemgr.Info     `json:"package_mgr,omitempty"`
	Gce            *gce.Info            `json:"gce,omitempty"`
	Ec2            *ec2.Info            `json:"ec2,omitempty"`
	Azure          *azure.Info          `json:"azure,omitempty"`
	DigitalOcean   *digitalocean.Info   `json:"digital_ocean,omitempty"`
	OCI            *oci.Info            `json:"oci,omitempty"`
	Alibaba        *alibaba.Info        `json:"alibaba,omitempty"`
	Linode         *linode.Info         `json:"linode,omitempty"`
	OpenStack      *openstack.Info      `json:"openstack,omitempty"`
	Scaleway       *scaleway.Info       `json:"scaleway,omitempty"`
	DMI            *dmi.Info            `json:"dmi,omitempty"`
	GPU            *gpu.Info            `json:"gpu,omitempty"`
	PCI            *pci.Info            `json:"pci,omitempty"`
	SCSI           *scsi.Info           `json:"scsi,omitempty"`
	Hardware       *hardware.Info       `json:"hardware,omitempty"`
	BlockDevice    *blockdevice.Info    `json:"block_device,omitempty"`
	Command        *command.Info        `json:"command,omitempty"`
	Docker         *docker.Info         `json:"docker,omitempty"`
	Grub2          *grub2.Info          `json:"grub2,omitempty"`
	Hostnamectl    *hostnamectl.Info    `json:"hostnamectl,omitempty"`
	Interrupts     *interrupts.Info     `json:"interrupts,omitempty"`
	IPC            *ipc.Info            `json:"ipc,omitempty"`
	Languages      *languages.Info      `json:"languages,omitempty"`
	Libvirt        *libvirt.Info        `json:"libvirt,omitempty"`
	Livepatch      *livepatch.Info      `json:"livepatch,omitempty"`
	Mdadm          *mdadm.Info          `json:"mdadm,omitempty"`
	Packages       *packages.Info       `json:"packages,omitempty"`
	RPM            *rpm.Info            `json:"rpm,omitempty"`
	SELinux        *selinux.Info        `json:"selinux,omitempty"`
	Services       *services.Info       `json:"services,omitempty"`
	SSH            *ssh.Info            `json:"ssh,omitempty"`
	Sysconf        *sysconf.Info        `json:"sysconf,omitempty"`
	Sysctl         *sysctl.Info         `json:"sysctl,omitempty"`
	SystemdPaths   *systemdpaths.Info   `json:"systemd_paths,omitempty"`
	TC             *tc.Info             `json:"tc,omitempty"`
	VirtualBox     *virtualbox.Info     `json:"virtualbox,omitempty"`
	VMware         *vmware.Info         `json:"vmware,omitempty"`
	Zpools         *zpools.Info         `json:"zpools,omitempty"`

	CollectTime     time.Time     `json:"collect_time"`
	CollectDuration time.Duration `json:"collect_duration_ns"`

	// Timings is populated only when the Gohai instance was built with
	// WithTimings(). Contains per-collector wall-clock durations, status
	// ("ok" / "err"), and — for failed collectors — the error message.
	// Failed collectors are dropped from the typed fields above; their
	// entry here is how the failure surfaces.
	Timings *Timings `json:"_timings,omitempty"`
}

// Timings captures the runtime observability data surfaced into Facts
// when WithTimings() is passed to gohai.New. Total wall-clock time
// for the run lives on Facts.CollectDuration — this struct is purely
// the per-collector breakdown.
type Timings struct {
	Collectors map[string]CollectorTiming `json:"collectors"`
}

// CollectorTiming is one collector's per-run observability entry.
type CollectorTiming struct {
	DurationNs int64  `json:"duration_ns"`
	Status     string `json:"status"` // "ok" | "err"
	Error      string `json:"error,omitempty"`
}

// JSON returns the compact JSON representation of the facts.
func (f *Facts) JSON() ([]byte, error) {
	return json.Marshal(f)
}

// PrettyJSON returns the indented JSON representation of the facts.
func (f *Facts) PrettyJSON() ([]byte, error) {
	return json.MarshalIndent(f, "", "  ")
}

// Flat returns a flat dot-separated key map of all facts. Marshal and
// unmarshal are guaranteed to succeed because every field on Facts is
// JSON-safe (Info structs with JSON tags, time.Time, time.Duration).
func (f *Facts) Flat() map[string]any {
	b, _ := json.Marshal(f)
	var generic map[string]any
	_ = json.Unmarshal(b, &generic)
	return flattenMap("", generic)
}

// Get returns the value at a dot-separated key path, or nil if absent.
func (f *Facts) Get(
	path string,
) any {
	return f.Flat()[path]
}

// String returns a printable summary.
func (f *Facts) String() string {
	return fmt.Sprintf("Facts{%d collectors, took %s}", f.countPopulated(), f.CollectDuration)
}

// countPopulated returns how many collector fields are non-nil.
func (f *Facts) countPopulated() int {
	n := 0
	if f.Platform != nil {
		n++
	}
	if f.Hostname != nil {
		n++
	}
	if f.Kernel != nil {
		n++
	}
	if f.KernelModules != nil {
		n++
	}
	if f.Uptime != nil {
		n++
	}
	if f.Virtualization != nil {
		n++
	}
	if f.MachineID != nil {
		n++
	}
	if f.CPU != nil {
		n++
	}
	if f.Load != nil {
		n++
	}
	if f.Memory != nil {
		n++
	}
	if f.Filesystem != nil {
		n++
	}
	if f.Disk != nil {
		n++
	}
	if f.Network != nil {
		n++
	}
	if f.Process != nil {
		n++
	}
	if f.Users != nil {
		n++
	}
	if f.Timezone != nil {
		n++
	}
	if f.RootGroup != nil {
		n++
	}
	if f.Shells != nil {
		n++
	}
	if f.Fips != nil {
		n++
	}
	if f.OSRelease != nil {
		n++
	}
	if f.LSB != nil {
		n++
	}
	if f.Init != nil {
		n++
	}
	if f.Shard != nil {
		n++
	}
	if f.PackageMgr != nil {
		n++
	}
	if f.Gce != nil {
		n++
	}
	if f.Ec2 != nil {
		n++
	}
	if f.Azure != nil {
		n++
	}
	if f.DigitalOcean != nil {
		n++
	}
	if f.OCI != nil {
		n++
	}
	if f.Alibaba != nil {
		n++
	}
	if f.Linode != nil {
		n++
	}
	if f.OpenStack != nil {
		n++
	}
	if f.Scaleway != nil {
		n++
	}
	if f.DMI != nil {
		n++
	}
	if f.GPU != nil {
		n++
	}
	if f.BlockDevice != nil {
		n++
	}
	if f.Command != nil {
		n++
	}
	if f.Docker != nil {
		n++
	}
	if f.Grub2 != nil {
		n++
	}
	if f.Hostnamectl != nil {
		n++
	}
	if f.Interrupts != nil {
		n++
	}
	if f.IPC != nil {
		n++
	}
	if f.Languages != nil {
		n++
	}
	if f.Libvirt != nil {
		n++
	}
	if f.Livepatch != nil {
		n++
	}
	if f.Mdadm != nil {
		n++
	}
	if f.Packages != nil {
		n++
	}
	if f.RPM != nil {
		n++
	}
	if f.SELinux != nil {
		n++
	}
	if f.Services != nil {
		n++
	}
	if f.SSH != nil {
		n++
	}
	if f.Sysconf != nil {
		n++
	}
	if f.Sysctl != nil {
		n++
	}
	if f.SystemdPaths != nil {
		n++
	}
	if f.TC != nil {
		n++
	}
	if f.VirtualBox != nil {
		n++
	}
	if f.VMware != nil {
		n++
	}
	if f.Zpools != nil {
		n++
	}
	return n
}

// set assigns the result of a single collector into the correct typed
// field on f. Unknown names are silently ignored (shouldn't happen for
// registered collectors).
func (f *Facts) set(
	name string,
	result any,
) {
	switch name {
	case "platform":
		if v, ok := result.(*platform.Info); ok {
			f.Platform = v
		}
	case "hostname":
		if v, ok := result.(*hostname.Info); ok {
			f.Hostname = v
		}
	case "kernel":
		if v, ok := result.(*kernel.Info); ok {
			f.Kernel = v
		}
	case "kernel_modules":
		if v, ok := result.(*kernelmodules.Info); ok {
			f.KernelModules = v
		}
	case "uptime":
		if v, ok := result.(*uptime.Info); ok {
			f.Uptime = v
		}
	case "virtualization":
		if v, ok := result.(*virtualization.Info); ok {
			f.Virtualization = v
		}
	case "machine_id":
		if v, ok := result.(*machineid.Info); ok {
			f.MachineID = v
		}
	case "cpu":
		if v, ok := result.(*cpu.Info); ok {
			f.CPU = v
		}
	case "load":
		if v, ok := result.(*load.Info); ok {
			f.Load = v
		}
	case "memory":
		if v, ok := result.(*memory.Info); ok {
			f.Memory = v
		}
	case "filesystem":
		if v, ok := result.(*filesystem.Info); ok {
			f.Filesystem = v
		}
	case "disk":
		if v, ok := result.(*disk.Info); ok {
			f.Disk = v
		}
	case "network":
		if v, ok := result.(*network.Info); ok {
			f.Network = v
		}
	case "process":
		if v, ok := result.(*process.Info); ok {
			f.Process = v
		}
	case "users":
		if v, ok := result.(*users.Info); ok {
			f.Users = v
		}
	case "sessions":
		if v, ok := result.(*sessions.Info); ok {
			f.Sessions = v
		}
	case "timezone":
		if v, ok := result.(*timezone.Info); ok {
			f.Timezone = v
		}
	case "root_group":
		if v, ok := result.(*rootgroup.Info); ok {
			f.RootGroup = v
		}
	case "shells":
		if v, ok := result.(*shells.Info); ok {
			f.Shells = v
		}
	case "fips":
		if v, ok := result.(*fips.Info); ok {
			f.Fips = v
		}
	case "os_release":
		if v, ok := result.(*osrelease.Info); ok {
			f.OSRelease = v
		}
	case "lsb":
		if v, ok := result.(*lsb.Info); ok {
			f.LSB = v
		}
	case "init":
		if v, ok := result.(*initd.Info); ok {
			f.Init = v
		}
	case "shard":
		if v, ok := result.(*shard.Info); ok {
			f.Shard = v
		}
	case "package_mgr":
		if v, ok := result.(*packagemgr.Info); ok {
			f.PackageMgr = v
		}
	case "gce":
		if v, ok := result.(*gce.Info); ok {
			f.Gce = v
		}
	case "ec2":
		if v, ok := result.(*ec2.Info); ok {
			f.Ec2 = v
		}
	case "azure":
		if v, ok := result.(*azure.Info); ok {
			f.Azure = v
		}
	case "digital_ocean":
		if v, ok := result.(*digitalocean.Info); ok {
			f.DigitalOcean = v
		}
	case "oci":
		if v, ok := result.(*oci.Info); ok {
			f.OCI = v
		}
	case "alibaba":
		if v, ok := result.(*alibaba.Info); ok {
			f.Alibaba = v
		}
	case "linode":
		if v, ok := result.(*linode.Info); ok {
			f.Linode = v
		}
	case "openstack":
		if v, ok := result.(*openstack.Info); ok {
			f.OpenStack = v
		}
	case "scaleway":
		if v, ok := result.(*scaleway.Info); ok {
			f.Scaleway = v
		}
	case "dmi":
		if v, ok := result.(*dmi.Info); ok {
			f.DMI = v
		}
	case "gpu":
		if v, ok := result.(*gpu.Info); ok {
			f.GPU = v
		}
	case "pci":
		if v, ok := result.(*pci.Info); ok {
			f.PCI = v
		}
	case "scsi":
		if v, ok := result.(*scsi.Info); ok {
			f.SCSI = v
		}
	case "hardware":
		if v, ok := result.(*hardware.Info); ok {
			f.Hardware = v
		}
	case "block_device":
		if v, ok := result.(*blockdevice.Info); ok {
			f.BlockDevice = v
		}
	case "command":
		if v, ok := result.(*command.Info); ok {
			f.Command = v
		}
	case "docker":
		if v, ok := result.(*docker.Info); ok {
			f.Docker = v
		}
	case "grub2":
		if v, ok := result.(*grub2.Info); ok {
			f.Grub2 = v
		}
	case "hostnamectl":
		if v, ok := result.(*hostnamectl.Info); ok {
			f.Hostnamectl = v
		}
	case "interrupts":
		if v, ok := result.(*interrupts.Info); ok {
			f.Interrupts = v
		}
	case "ipc":
		if v, ok := result.(*ipc.Info); ok {
			f.IPC = v
		}
	case "languages":
		if v, ok := result.(*languages.Info); ok {
			f.Languages = v
		}
	case "libvirt":
		if v, ok := result.(*libvirt.Info); ok {
			f.Libvirt = v
		}
	case "livepatch":
		if v, ok := result.(*livepatch.Info); ok {
			f.Livepatch = v
		}
	case "mdadm":
		if v, ok := result.(*mdadm.Info); ok {
			f.Mdadm = v
		}
	case "packages":
		if v, ok := result.(*packages.Info); ok {
			f.Packages = v
		}
	case "rpm":
		if v, ok := result.(*rpm.Info); ok {
			f.RPM = v
		}
	case "selinux":
		if v, ok := result.(*selinux.Info); ok {
			f.SELinux = v
		}
	case "services":
		if v, ok := result.(*services.Info); ok {
			f.Services = v
		}
	case "ssh":
		if v, ok := result.(*ssh.Info); ok {
			f.SSH = v
		}
	case "sysconf":
		if v, ok := result.(*sysconf.Info); ok {
			f.Sysconf = v
		}
	case "sysctl":
		if v, ok := result.(*sysctl.Info); ok {
			f.Sysctl = v
		}
	case "systemd_paths":
		if v, ok := result.(*systemdpaths.Info); ok {
			f.SystemdPaths = v
		}
	case "tc":
		if v, ok := result.(*tc.Info); ok {
			f.TC = v
		}
	case "virtualbox":
		if v, ok := result.(*virtualbox.Info); ok {
			f.VirtualBox = v
		}
	case "vmware":
		if v, ok := result.(*vmware.Info); ok {
			f.VMware = v
		}
	case "zpools":
		if v, ok := result.(*zpools.Info); ok {
			f.Zpools = v
		}
	}
}

func flattenMap(
	prefix string,
	in map[string]any,
) map[string]any {
	out := make(map[string]any)
	for k, v := range in {
		key := k
		if prefix != "" {
			key = prefix + "." + k
		}
		if sub, ok := v.(map[string]any); ok {
			for sk, sv := range flattenMap(key, sub) {
				out[sk] = sv
			}
			continue
		}
		out[key] = v
	}
	return out
}
