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
	"reflect"
	"strings"
	"time"

	"github.com/osapi-io/gohai/pkg/gohai/collectors/alibaba"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/azure"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/blockdevice"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/cacertificates"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/command"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/cpu"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/digitalocean"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/disk"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/dmi"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/dns"
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
	"github.com/osapi-io/gohai/pkg/gohai/collectors/initd"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/interrupts"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/ipc"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/kernel"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/kernelmodules"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/languages"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/libvirt"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/linode"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/livepatch"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/load"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/lsb"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/machineid"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/mdadm"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/memory"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/network"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/oci"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/openstack"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/osrelease"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/packagemgr"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/packages"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/pci"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/platform"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/process"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/rootgroup"
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
	"github.com/osapi-io/gohai/pkg/gohai/collectors/systemdpaths"
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
	DNS            *dns.Info            `json:"dns,omitempty"`
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
	CACertificates *cacertificates.Info `json:"ca_certificates,omitempty"`

	CollectTime     time.Time     `json:"collect_time"`
	CollectDuration time.Duration `json:"collect_duration_ns"`

	// Errors is the error of each collector that failed, by its name. A
	// failed collector has no typed field above, the same as one that was
	// not selected, so this is how a caller tells the two apart.
	Errors map[string]string `json:"_errors,omitempty"`

	// Timings is populated only when the Gohai instance was built with
	// WithTimings(). Contains per-collector wall-clock durations, status
	// ("ok" / "err"), and the error message of a failed collector.
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

// countPopulated returns how many collector fields are non-nil. A collector
// field is each pointer field of Facts whose JSON name does not start with
// "_", which marks the run's own data, such as _timings. Reflection keeps
// the count in step with the fields, so a new collector is counted.
func (f *Facts) countPopulated() int {
	v := reflect.ValueOf(f).Elem()
	t := v.Type()
	n := 0
	for i := range t.NumField() {
		field := t.Field(i)
		if field.Type.Kind() != reflect.Pointer ||
			strings.HasPrefix(field.Tag.Get("json"), "_") {
			continue
		}
		if !v.Field(i).IsNil() {
			n++
		}
	}

	return n
}

// set assigns the result of a single collector into the correct typed
// field on f. It is false when the name is unknown or the result is not
// the name's type, so the caller can record the result as an error.
func (f *Facts) set(
	name string,
	result any,
) bool {
	switch name {
	case "platform":
		if v, ok := result.(*platform.Info); ok {
			f.Platform = v
			return true
		}
	case "hostname":
		if v, ok := result.(*hostname.Info); ok {
			f.Hostname = v
			return true
		}
	case "kernel":
		if v, ok := result.(*kernel.Info); ok {
			f.Kernel = v
			return true
		}
	case "kernel_modules":
		if v, ok := result.(*kernelmodules.Info); ok {
			f.KernelModules = v
			return true
		}
	case "uptime":
		if v, ok := result.(*uptime.Info); ok {
			f.Uptime = v
			return true
		}
	case "virtualization":
		if v, ok := result.(*virtualization.Info); ok {
			f.Virtualization = v
			return true
		}
	case "machine_id":
		if v, ok := result.(*machineid.Info); ok {
			f.MachineID = v
			return true
		}
	case "cpu":
		if v, ok := result.(*cpu.Info); ok {
			f.CPU = v
			return true
		}
	case "load":
		if v, ok := result.(*load.Info); ok {
			f.Load = v
			return true
		}
	case "memory":
		if v, ok := result.(*memory.Info); ok {
			f.Memory = v
			return true
		}
	case "filesystem":
		if v, ok := result.(*filesystem.Info); ok {
			f.Filesystem = v
			return true
		}
	case "disk":
		if v, ok := result.(*disk.Info); ok {
			f.Disk = v
			return true
		}
	case "network":
		if v, ok := result.(*network.Info); ok {
			f.Network = v
			return true
		}
	case "process":
		if v, ok := result.(*process.Info); ok {
			f.Process = v
			return true
		}
	case "users":
		if v, ok := result.(*users.Info); ok {
			f.Users = v
			return true
		}
	case "sessions":
		if v, ok := result.(*sessions.Info); ok {
			f.Sessions = v
			return true
		}
	case "timezone":
		if v, ok := result.(*timezone.Info); ok {
			f.Timezone = v
			return true
		}
	case "dns":
		if v, ok := result.(*dns.Info); ok {
			f.DNS = v
			return true
		}
	case "root_group":
		if v, ok := result.(*rootgroup.Info); ok {
			f.RootGroup = v
			return true
		}
	case "shells":
		if v, ok := result.(*shells.Info); ok {
			f.Shells = v
			return true
		}
	case "fips":
		if v, ok := result.(*fips.Info); ok {
			f.Fips = v
			return true
		}
	case "os_release":
		if v, ok := result.(*osrelease.Info); ok {
			f.OSRelease = v
			return true
		}
	case "lsb":
		if v, ok := result.(*lsb.Info); ok {
			f.LSB = v
			return true
		}
	case "init":
		if v, ok := result.(*initd.Info); ok {
			f.Init = v
			return true
		}
	case "shard":
		if v, ok := result.(*shard.Info); ok {
			f.Shard = v
			return true
		}
	case "package_mgr":
		if v, ok := result.(*packagemgr.Info); ok {
			f.PackageMgr = v
			return true
		}
	case "gce":
		if v, ok := result.(*gce.Info); ok {
			f.Gce = v
			return true
		}
	case "ec2":
		if v, ok := result.(*ec2.Info); ok {
			f.Ec2 = v
			return true
		}
	case "azure":
		if v, ok := result.(*azure.Info); ok {
			f.Azure = v
			return true
		}
	case "digital_ocean":
		if v, ok := result.(*digitalocean.Info); ok {
			f.DigitalOcean = v
			return true
		}
	case "oci":
		if v, ok := result.(*oci.Info); ok {
			f.OCI = v
			return true
		}
	case "alibaba":
		if v, ok := result.(*alibaba.Info); ok {
			f.Alibaba = v
			return true
		}
	case "linode":
		if v, ok := result.(*linode.Info); ok {
			f.Linode = v
			return true
		}
	case "openstack":
		if v, ok := result.(*openstack.Info); ok {
			f.OpenStack = v
			return true
		}
	case "scaleway":
		if v, ok := result.(*scaleway.Info); ok {
			f.Scaleway = v
			return true
		}
	case "dmi":
		if v, ok := result.(*dmi.Info); ok {
			f.DMI = v
			return true
		}
	case "gpu":
		if v, ok := result.(*gpu.Info); ok {
			f.GPU = v
			return true
		}
	case "pci":
		if v, ok := result.(*pci.Info); ok {
			f.PCI = v
			return true
		}
	case "scsi":
		if v, ok := result.(*scsi.Info); ok {
			f.SCSI = v
			return true
		}
	case "hardware":
		if v, ok := result.(*hardware.Info); ok {
			f.Hardware = v
			return true
		}
	case "block_device":
		if v, ok := result.(*blockdevice.Info); ok {
			f.BlockDevice = v
			return true
		}
	case "command":
		if v, ok := result.(*command.Info); ok {
			f.Command = v
			return true
		}
	case "docker":
		if v, ok := result.(*docker.Info); ok {
			f.Docker = v
			return true
		}
	case "grub2":
		if v, ok := result.(*grub2.Info); ok {
			f.Grub2 = v
			return true
		}
	case "hostnamectl":
		if v, ok := result.(*hostnamectl.Info); ok {
			f.Hostnamectl = v
			return true
		}
	case "interrupts":
		if v, ok := result.(*interrupts.Info); ok {
			f.Interrupts = v
			return true
		}
	case "ipc":
		if v, ok := result.(*ipc.Info); ok {
			f.IPC = v
			return true
		}
	case "languages":
		if v, ok := result.(*languages.Info); ok {
			f.Languages = v
			return true
		}
	case "libvirt":
		if v, ok := result.(*libvirt.Info); ok {
			f.Libvirt = v
			return true
		}
	case "livepatch":
		if v, ok := result.(*livepatch.Info); ok {
			f.Livepatch = v
			return true
		}
	case "mdadm":
		if v, ok := result.(*mdadm.Info); ok {
			f.Mdadm = v
			return true
		}
	case "packages":
		if v, ok := result.(*packages.Info); ok {
			f.Packages = v
			return true
		}
	case "rpm":
		if v, ok := result.(*rpm.Info); ok {
			f.RPM = v
			return true
		}
	case "selinux":
		if v, ok := result.(*selinux.Info); ok {
			f.SELinux = v
			return true
		}
	case "services":
		if v, ok := result.(*services.Info); ok {
			f.Services = v
			return true
		}
	case "ssh":
		if v, ok := result.(*ssh.Info); ok {
			f.SSH = v
			return true
		}
	case "sysconf":
		if v, ok := result.(*sysconf.Info); ok {
			f.Sysconf = v
			return true
		}
	case "sysctl":
		if v, ok := result.(*sysctl.Info); ok {
			f.Sysctl = v
			return true
		}
	case "systemd_paths":
		if v, ok := result.(*systemdpaths.Info); ok {
			f.SystemdPaths = v
			return true
		}
	case "tc":
		if v, ok := result.(*tc.Info); ok {
			f.TC = v
			return true
		}
	case "virtualbox":
		if v, ok := result.(*virtualbox.Info); ok {
			f.VirtualBox = v
			return true
		}
	case "vmware":
		if v, ok := result.(*vmware.Info); ok {
			f.VMware = v
			return true
		}
	case "zpools":
		if v, ok := result.(*zpools.Info); ok {
			f.Zpools = v
			return true
		}
	case "ca_certificates":
		if v, ok := result.(*cacertificates.Info); ok {
			f.CACertificates = v
			return true
		}
	}
	return false
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
