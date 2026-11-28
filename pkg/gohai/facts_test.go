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

package gohai

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/pkg/gohai/collectors/blockdevice"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/cacertificates"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/command"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/cpu"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/disk"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/dns"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/docker"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/filesystem"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/fips"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/grub2"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/hardware"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/hostname"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/hostnamectl"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/initd"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/interrupts"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/ipc"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/kernel"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/languages"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/libvirt"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/livepatch"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/load"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/lsb"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/machineid"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/mdadm"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/memory"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/network"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/osrelease"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/packagemgr"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/packages"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/pci"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/platform"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/process"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/rootgroup"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/rpm"
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

type FactsTestSuite struct {
	suite.Suite
}

func TestFactsTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(FactsTestSuite))
}

func (s *FactsTestSuite) TestFlattenMap() {
	tests := []struct {
		name         string
		in           map[string]any
		validateFunc func(map[string]any)
	}{
		{
			name: "scalars",
			in:   map[string]any{"a": 1, "b": "x"},
			validateFunc: func(got map[string]any) {
				s.Equal(map[string]any{"a": 1, "b": "x"}, got)
			},
		},
		{
			name: "nested map",
			in:   map[string]any{"cpu": map[string]any{"total": 8, "model": "intel"}},
			validateFunc: func(got map[string]any) {
				s.Equal(map[string]any{"cpu.total": 8, "cpu.model": "intel"}, got)
			},
		},
		{
			name: "deep nest",
			in:   map[string]any{"a": map[string]any{"b": map[string]any{"c": 1}}},
			validateFunc: func(got map[string]any) {
				s.Equal(map[string]any{"a.b.c": 1}, got)
			},
		},
		{
			name: "a list's elements get an index key",
			in:   map[string]any{"cpu": map[string]any{"flags": []any{"fpu", "sse"}}},
			validateFunc: func(got map[string]any) {
				s.Equal(map[string]any{"cpu.flags.0": "fpu", "cpu.flags.1": "sse"}, got)
			},
		},
		{
			name: "a list of maps flattens each map under its index",
			in: map[string]any{"disks": []any{
				map[string]any{"name": "vda", "size": 10},
				map[string]any{"name": "vdb", "size": 20},
			}},
			validateFunc: func(got map[string]any) {
				s.Equal(map[string]any{
					"disks.0.name": "vda", "disks.0.size": 10,
					"disks.1.name": "vdb", "disks.1.size": 20,
				}, got)
			},
		},
		{
			name: "an empty list or map has no key",
			in:   map[string]any{"flags": []any{}, "labels": map[string]any{}},
			validateFunc: func(got map[string]any) {
				s.Empty(got)
			},
		},
		{
			name: "empty",
			in:   map[string]any{},
			validateFunc: func(got map[string]any) {
				s.Empty(got)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.validateFunc(flattenMap(tt.in))
		})
	}
}

func (s *FactsTestSuite) TestSet() {
	tests := []struct {
		name     string
		collName string
		result   any
		check    func(*Facts) bool
	}{
		{
			name:     "platform populated",
			collName: "platform",
			result:   &platform.Info{Name: "ubuntu"},
			check:    func(f *Facts) bool { return f.Platform != nil && f.Platform.Name == "ubuntu" },
		},
		{
			name:     "wrong type ignored",
			collName: "platform",
			result:   "not an Info",
			check:    func(f *Facts) bool { return f.Platform == nil },
		},
		{
			name:     "unknown name ignored",
			collName: "nope",
			result:   &platform.Info{Name: "x"},
			check:    func(f *Facts) bool { return f.Platform == nil },
		},
		{
			name:     "hostname",
			collName: "hostname",
			result:   new(hostname.Info),
			check:    func(f *Facts) bool { return f.Hostname != nil },
		},
		{
			name:     "kernel",
			collName: "kernel",
			result:   new(kernel.Info),
			check:    func(f *Facts) bool { return f.Kernel != nil },
		},
		{
			name:     "uptime",
			collName: "uptime",
			result:   new(uptime.Info),
			check:    func(f *Facts) bool { return f.Uptime != nil },
		},
		{
			name:     "virtualization",
			collName: "virtualization",
			result:   new(virtualization.Info),
			check:    func(f *Facts) bool { return f.Virtualization != nil },
		},
		{
			name:     "machine_id",
			collName: "machine_id",
			result:   new(machineid.Info),
			check:    func(f *Facts) bool { return f.MachineID != nil },
		},
		{
			name:     "cpu",
			collName: "cpu",
			result:   new(cpu.Info),
			check:    func(f *Facts) bool { return f.CPU != nil },
		},
		{
			name:     "load",
			collName: "load",
			result:   new(load.Info),
			check:    func(f *Facts) bool { return f.Load != nil },
		},
		{
			name:     "memory",
			collName: "memory",
			result:   new(memory.Info),
			check:    func(f *Facts) bool { return f.Memory != nil },
		},
		{
			name:     "filesystem",
			collName: "filesystem",
			result:   new(filesystem.Info),
			check:    func(f *Facts) bool { return f.Filesystem != nil },
		},
		{
			name:     "disk",
			collName: "disk",
			result:   new(disk.Info),
			check:    func(f *Facts) bool { return f.Disk != nil },
		},
		{
			name:     "network",
			collName: "network",
			result:   new(network.Info),
			check:    func(f *Facts) bool { return f.Network != nil },
		},
		{
			name:     "process",
			collName: "process",
			result:   new(process.Info),
			check:    func(f *Facts) bool { return f.Process != nil },
		},
		{
			name:     "users",
			collName: "users",
			result:   new(users.Info),
			check:    func(f *Facts) bool { return f.Users != nil },
		},
		{
			name:     "timezone",
			collName: "timezone",
			result:   new(timezone.Info),
			check:    func(f *Facts) bool { return f.Timezone != nil },
		},
		{
			name:     "dns",
			collName: "dns",
			result:   new(dns.Info),
			check:    func(f *Facts) bool { return f.DNS != nil },
		},
		{
			name:     "root_group",
			collName: "root_group",
			result:   new(rootgroup.Info),
			check:    func(f *Facts) bool { return f.RootGroup != nil },
		},
		{
			name:     "shells",
			collName: "shells",
			result:   new(shells.Info),
			check:    func(f *Facts) bool { return f.Shells != nil },
		},
		{
			name:     "fips",
			collName: "fips",
			result:   new(fips.Info),
			check:    func(f *Facts) bool { return f.Fips != nil },
		},
		{
			name:     "os_release",
			collName: "os_release",
			result:   new(osrelease.Info),
			check:    func(f *Facts) bool { return f.OSRelease != nil },
		},
		{
			name:     "lsb",
			collName: "lsb",
			result:   new(lsb.Info),
			check:    func(f *Facts) bool { return f.LSB != nil },
		},
		{
			name:     "init",
			collName: "init",
			result:   new(initd.Info),
			check:    func(f *Facts) bool { return f.Init != nil },
		},
		{
			name:     "shard",
			collName: "shard",
			result:   new(shard.Info),
			check:    func(f *Facts) bool { return f.Shard != nil },
		},
		{
			name:     "package_mgr",
			collName: "package_mgr",
			result:   new(packagemgr.Info),
			check:    func(f *Facts) bool { return f.PackageMgr != nil },
		},
		{
			name:     "sessions",
			collName: "sessions",
			result:   new(sessions.Info),
			check:    func(f *Facts) bool { return f.Sessions != nil },
		},
		{
			name:     "pci",
			collName: "pci",
			result:   new(pci.Info),
			check:    func(f *Facts) bool { return f.PCI != nil },
		},
		{
			name:     "scsi",
			collName: "scsi",
			result:   new(scsi.Info),
			check:    func(f *Facts) bool { return f.SCSI != nil },
		},
		{
			name:     "hardware",
			collName: "hardware",
			result:   new(hardware.Info),
			check:    func(f *Facts) bool { return f.Hardware != nil },
		},
		{
			name:     "block_device",
			collName: "block_device",
			result:   new(blockdevice.Info),
			check:    func(f *Facts) bool { return f.BlockDevice != nil },
		},
		{
			name:     "command",
			collName: "command",
			result:   new(command.Info),
			check:    func(f *Facts) bool { return f.Command != nil },
		},
		{
			name:     "docker",
			collName: "docker",
			result:   new(docker.Info),
			check:    func(f *Facts) bool { return f.Docker != nil },
		},
		{
			name:     "grub2",
			collName: "grub2",
			result:   new(grub2.Info),
			check:    func(f *Facts) bool { return f.Grub2 != nil },
		},
		{
			name:     "hostnamectl",
			collName: "hostnamectl",
			result:   new(hostnamectl.Info),
			check:    func(f *Facts) bool { return f.Hostnamectl != nil },
		},
		{
			name:     "interrupts",
			collName: "interrupts",
			result:   new(interrupts.Info),
			check:    func(f *Facts) bool { return f.Interrupts != nil },
		},
		{
			name:     "ipc",
			collName: "ipc",
			result:   new(ipc.Info),
			check:    func(f *Facts) bool { return f.IPC != nil },
		},
		{
			name:     "languages",
			collName: "languages",
			result:   new(languages.Info),
			check:    func(f *Facts) bool { return f.Languages != nil },
		},
		{
			name:     "libvirt",
			collName: "libvirt",
			result:   new(libvirt.Info),
			check:    func(f *Facts) bool { return f.Libvirt != nil },
		},
		{
			name:     "livepatch",
			collName: "livepatch",
			result:   new(livepatch.Info),
			check:    func(f *Facts) bool { return f.Livepatch != nil },
		},
		{
			name:     "mdadm",
			collName: "mdadm",
			result:   new(mdadm.Info),
			check:    func(f *Facts) bool { return f.Mdadm != nil },
		},
		{
			name:     "packages",
			collName: "packages",
			result:   new(packages.Info),
			check:    func(f *Facts) bool { return f.Packages != nil },
		},
		{
			name:     "rpm",
			collName: "rpm",
			result:   new(rpm.Info),
			check:    func(f *Facts) bool { return f.RPM != nil },
		},
		{
			name:     "selinux",
			collName: "selinux",
			result:   new(selinux.Info),
			check:    func(f *Facts) bool { return f.SELinux != nil },
		},
		{
			name:     "services",
			collName: "services",
			result:   new(services.Info),
			check:    func(f *Facts) bool { return f.Services != nil },
		},
		{
			name:     "ssh",
			collName: "ssh",
			result:   new(ssh.Info),
			check:    func(f *Facts) bool { return f.SSH != nil },
		},
		{
			name:     "sysconf",
			collName: "sysconf",
			result:   new(sysconf.Info),
			check:    func(f *Facts) bool { return f.Sysconf != nil },
		},
		{
			name:     "sysctl",
			collName: "sysctl",
			result:   new(sysctl.Info),
			check:    func(f *Facts) bool { return f.Sysctl != nil },
		},
		{
			name:     "systemd_paths",
			collName: "systemd_paths",
			result:   new(systemdpaths.Info),
			check:    func(f *Facts) bool { return f.SystemdPaths != nil },
		},
		{
			name:     "tc",
			collName: "tc",
			result:   new(tc.Info),
			check:    func(f *Facts) bool { return f.TC != nil },
		},
		{
			name:     "virtualbox",
			collName: "virtualbox",
			result:   new(virtualbox.Info),
			check:    func(f *Facts) bool { return f.VirtualBox != nil },
		},
		{
			name:     "vmware",
			collName: "vmware",
			result:   new(vmware.Info),
			check:    func(f *Facts) bool { return f.VMware != nil },
		},
		{
			name:     "zpools",
			collName: "zpools",
			result:   new(zpools.Info),
			check:    func(f *Facts) bool { return f.Zpools != nil },
		},
		{
			name:     "ca_certificates",
			collName: "ca_certificates",
			result:   new(cacertificates.Info),
			check:    func(f *Facts) bool { return f.CACertificates != nil },
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			f := &Facts{}
			f.set(tt.collName, tt.result)
			s.True(tt.check(f))
		})
	}
}

func (s *FactsTestSuite) TestCountPopulated() {
	tests := []struct {
		name  string
		facts *Facts
		want  int
	}{
		{"empty", &Facts{}, 0},
		{"platform only", &Facts{Platform: &platform.Info{}}, 1},
		{
			name: "all fields populated",
			facts: &Facts{
				Platform:       &platform.Info{},
				Hostname:       new(hostname.Info),
				Kernel:         new(kernel.Info),
				Uptime:         new(uptime.Info),
				Virtualization: new(virtualization.Info),
				MachineID:      new(machineid.Info),
				CPU:            new(cpu.Info),
				Memory:         new(memory.Info),
				Filesystem:     new(filesystem.Info),
				Disk:           new(disk.Info),
				Network:        new(network.Info),
				Process:        new(process.Info),
				Users:          new(users.Info),
				Timezone:       new(timezone.Info),
				DNS:            new(dns.Info),
				RootGroup:      new(rootgroup.Info),
				Shells:         new(shells.Info),
				Fips:           new(fips.Info),
				Load:           new(load.Info),
				OSRelease:      new(osrelease.Info),
				LSB:            new(lsb.Info),
				Init:           new(initd.Info),
				Shard:          new(shard.Info),
				PackageMgr:     new(packagemgr.Info),
				BlockDevice:    new(blockdevice.Info),
				Command:        new(command.Info),
				Docker:         new(docker.Info),
				Grub2:          new(grub2.Info),
				Hostnamectl:    new(hostnamectl.Info),
				Interrupts:     new(interrupts.Info),
				IPC:            new(ipc.Info),
				Languages:      new(languages.Info),
				Libvirt:        new(libvirt.Info),
				Livepatch:      new(livepatch.Info),
				Mdadm:          new(mdadm.Info),
				Packages:       new(packages.Info),
				RPM:            new(rpm.Info),
				SELinux:        new(selinux.Info),
				Services:       new(services.Info),
				SSH:            new(ssh.Info),
				Sysconf:        new(sysconf.Info),
				Sysctl:         new(sysctl.Info),
				SystemdPaths:   new(systemdpaths.Info),
				TC:             new(tc.Info),
				VirtualBox:     new(virtualbox.Info),
				VMware:         new(vmware.Info),
				Zpools:         new(zpools.Info),
				CACertificates: new(cacertificates.Info),
			},
			want: 48,
		},
		{
			// Every collector field set, by reflection, so a field that
			// countPopulated does not count fails here.
			name:  "every collector field populated",
			facts: everyCollectorField(),
			want:  64,
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Equal(tt.want, tt.facts.countPopulated())
		})
	}
}

// everyCollectorField is a Facts with each collector field set, and
// _timings, which is not a collector, set too.
func everyCollectorField() *Facts {
	f := &Facts{Timings: &Timings{}}
	v := reflect.ValueOf(f).Elem()
	for i := range v.NumField() {
		field := v.Type().Field(i)
		if field.Type.Kind() == reflect.Pointer && field.Name != "Timings" {
			v.Field(i).Set(reflect.New(field.Type.Elem()))
		}
	}

	return f
}
