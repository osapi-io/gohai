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
	"maps"
	"reflect"
	"strings"
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
	v := reflect.ValueOf(f).Elem()

	n := 0

	for _, i := range factFields {
		if !v.Field(i).IsNil() {
			n++
		}
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
	i, ok := factFields[name]
	if !ok {
		return
	}

	field := reflect.ValueOf(f).Elem().Field(i)

	v := reflect.ValueOf(result)
	if !v.IsValid() || !v.Type().AssignableTo(field.Type()) {
		return
	}

	field.Set(v)
}

// factFields maps a collector's name to the Facts field that carries its
// result, read off the struct's own json tags. Before this the mapping
// was written out twice — a 62-case switch in set and 58 nil checks in
// countPopulated — and the two had already drifted: sessions, hardware,
// scsi and pci were stored but never counted.
var factFields = buildFactFields()

func buildFactFields() map[string]int {
	m := make(map[string]int)

	t := reflect.TypeOf(Facts{})
	for i := range t.NumField() {
		field := t.Field(i)
		if field.Type.Kind() != reflect.Pointer {
			continue
		}

		tag, _, _ := strings.Cut(field.Tag.Get("json"), ",")
		// A leading underscore marks a field that is not a collector.
		if tag == "" || tag == "-" || strings.HasPrefix(tag, "_") {
			continue
		}

		m[tag] = i
	}

	return m
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

		sub, nested := v.(map[string]any)
		if !nested {
			out[key] = v

			continue
		}

		maps.Copy(out, flattenMap(key, sub))
	}

	return out
}
