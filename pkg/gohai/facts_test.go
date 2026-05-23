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
	"testing"

	"github.com/stretchr/testify/suite"

	blockdevice "github.com/osapi-io/gohai/pkg/gohai/collectors/block_device"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/command"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/cpu"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/disk"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/docker"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/filesystem"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/fips"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/grub2"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/hardware"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/hostname"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/hostnamectl"
	initd "github.com/osapi-io/gohai/pkg/gohai/collectors/init"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/interrupts"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/ipc"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/kernel"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/languages"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/libvirt"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/livepatch"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/load"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/lsb"
	machineid "github.com/osapi-io/gohai/pkg/gohai/collectors/machine_id"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/mdadm"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/memory"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/network"
	osrelease "github.com/osapi-io/gohai/pkg/gohai/collectors/os_release"
	packagemgr "github.com/osapi-io/gohai/pkg/gohai/collectors/package_mgr"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/packages"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/pci"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/platform"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/process"
	rootgroup "github.com/osapi-io/gohai/pkg/gohai/collectors/root_group"
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

func hostnameInfoPtr() *hostname.Info         { return &hostname.Info{} }
func kernelInfoPtr() *kernel.Info             { return &kernel.Info{} }
func uptimeInfoPtr() *uptime.Info             { return &uptime.Info{} }
func virtInfoPtr() *virtualization.Info       { return &virtualization.Info{} }
func machineIDInfoPtr() *machineid.Info       { return &machineid.Info{} }
func cpuInfoPtr() *cpu.Info                   { return &cpu.Info{} }
func memoryInfoPtr() *memory.Info             { return &memory.Info{} }
func filesystemInfoPtr() *filesystem.Info     { return &filesystem.Info{} }
func diskInfoPtr() *disk.Info                 { return &disk.Info{} }
func networkInfoPtr() *network.Info           { return &network.Info{} }
func processInfoPtr() *process.Info           { return &process.Info{} }
func usersInfoPtr() *users.Info               { return &users.Info{} }
func timezoneInfoPtr() *timezone.Info         { return &timezone.Info{} }
func rootGroupInfoPtr() *rootgroup.Info       { return &rootgroup.Info{} }
func shellsInfoPtr() *shells.Info             { return &shells.Info{} }
func fipsInfoPtr() *fips.Info                 { return &fips.Info{} }
func loadInfoPtr() *load.Info                 { return &load.Info{} }
func osReleaseInfoPtr() *osrelease.Info       { return &osrelease.Info{} }
func lsbInfoPtr() *lsb.Info                   { return &lsb.Info{} }
func initInfoPtr() *initd.Info                { return &initd.Info{} }
func shardInfoPtr() *shard.Info               { return &shard.Info{} }
func packageMgrInfoPtr() *packagemgr.Info     { return &packagemgr.Info{} }
func sessionsInfoPtr() *sessions.Info         { return &sessions.Info{} }
func pciInfoPtr() *pci.Info                   { return &pci.Info{} }
func scsiInfoPtr() *scsi.Info                 { return &scsi.Info{} }
func hardwareInfoPtr() *hardware.Info         { return &hardware.Info{} }
func blockDeviceInfoPtr() *blockdevice.Info   { return &blockdevice.Info{} }
func commandInfoPtr() *command.Info           { return &command.Info{} }
func dockerInfoPtr() *docker.Info             { return &docker.Info{} }
func grub2InfoPtr() *grub2.Info               { return &grub2.Info{} }
func hostnamectlInfoPtr() *hostnamectl.Info   { return &hostnamectl.Info{} }
func interruptsInfoPtr() *interrupts.Info     { return &interrupts.Info{} }
func ipcInfoPtr() *ipc.Info                   { return &ipc.Info{} }
func languagesInfoPtr() *languages.Info       { return &languages.Info{} }
func libvirtInfoPtr() *libvirt.Info           { return &libvirt.Info{} }
func livepatchInfoPtr() *livepatch.Info       { return &livepatch.Info{} }
func mdadmInfoPtr() *mdadm.Info               { return &mdadm.Info{} }
func packagesInfoPtr() *packages.Info         { return &packages.Info{} }
func rpmInfoPtr() *rpm.Info                   { return &rpm.Info{} }
func selinuxInfoPtr() *selinux.Info           { return &selinux.Info{} }
func servicesInfoPtr() *services.Info         { return &services.Info{} }
func sshInfoPtr() *ssh.Info                   { return &ssh.Info{} }
func sysconfInfoPtr() *sysconf.Info           { return &sysconf.Info{} }
func sysctlInfoPtr() *sysctl.Info             { return &sysctl.Info{} }
func systemdPathsInfoPtr() *systemdpaths.Info { return &systemdpaths.Info{} }
func tcInfoPtr() *tc.Info                     { return &tc.Info{} }
func virtualboxInfoPtr() *virtualbox.Info     { return &virtualbox.Info{} }
func vmwareInfoPtr() *vmware.Info             { return &vmware.Info{} }
func zpoolsInfoPtr() *zpools.Info             { return &zpools.Info{} }

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
		name string
		in   map[string]any
		want map[string]any
	}{
		{
			name: "scalars",
			in:   map[string]any{"a": 1, "b": "x"},
			want: map[string]any{"a": 1, "b": "x"},
		},
		{
			name: "nested map",
			in:   map[string]any{"cpu": map[string]any{"total": 8, "model": "intel"}},
			want: map[string]any{"cpu.total": 8, "cpu.model": "intel"},
		},
		{
			name: "deep nest",
			in:   map[string]any{"a": map[string]any{"b": map[string]any{"c": 1}}},
			want: map[string]any{"a.b.c": 1},
		},
		{
			name: "empty",
			in:   map[string]any{},
			want: map[string]any{},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			got := flattenMap("", tt.in)
			s.Equal(tt.want, got)
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
			result:   hostnameInfoPtr(),
			check:    func(f *Facts) bool { return f.Hostname != nil },
		},
		{
			name:     "kernel",
			collName: "kernel",
			result:   kernelInfoPtr(),
			check:    func(f *Facts) bool { return f.Kernel != nil },
		},
		{
			name:     "uptime",
			collName: "uptime",
			result:   uptimeInfoPtr(),
			check:    func(f *Facts) bool { return f.Uptime != nil },
		},
		{
			name:     "virtualization",
			collName: "virtualization",
			result:   virtInfoPtr(),
			check:    func(f *Facts) bool { return f.Virtualization != nil },
		},
		{
			name:     "machine_id",
			collName: "machine_id",
			result:   machineIDInfoPtr(),
			check:    func(f *Facts) bool { return f.MachineID != nil },
		},
		{
			name:     "cpu",
			collName: "cpu",
			result:   cpuInfoPtr(),
			check:    func(f *Facts) bool { return f.CPU != nil },
		},
		{
			name:     "load",
			collName: "load",
			result:   loadInfoPtr(),
			check:    func(f *Facts) bool { return f.Load != nil },
		},
		{
			name:     "memory",
			collName: "memory",
			result:   memoryInfoPtr(),
			check:    func(f *Facts) bool { return f.Memory != nil },
		},
		{
			name:     "filesystem",
			collName: "filesystem",
			result:   filesystemInfoPtr(),
			check:    func(f *Facts) bool { return f.Filesystem != nil },
		},
		{
			name:     "disk",
			collName: "disk",
			result:   diskInfoPtr(),
			check:    func(f *Facts) bool { return f.Disk != nil },
		},
		{
			name:     "network",
			collName: "network",
			result:   networkInfoPtr(),
			check:    func(f *Facts) bool { return f.Network != nil },
		},
		{
			name:     "process",
			collName: "process",
			result:   processInfoPtr(),
			check:    func(f *Facts) bool { return f.Process != nil },
		},
		{
			name:     "users",
			collName: "users",
			result:   usersInfoPtr(),
			check:    func(f *Facts) bool { return f.Users != nil },
		},
		{
			name:     "timezone",
			collName: "timezone",
			result:   timezoneInfoPtr(),
			check:    func(f *Facts) bool { return f.Timezone != nil },
		},
		{
			name:     "root_group",
			collName: "root_group",
			result:   rootGroupInfoPtr(),
			check:    func(f *Facts) bool { return f.RootGroup != nil },
		},
		{
			name:     "shells",
			collName: "shells",
			result:   shellsInfoPtr(),
			check:    func(f *Facts) bool { return f.Shells != nil },
		},
		{
			name:     "fips",
			collName: "fips",
			result:   fipsInfoPtr(),
			check:    func(f *Facts) bool { return f.Fips != nil },
		},
		{
			name:     "os_release",
			collName: "os_release",
			result:   osReleaseInfoPtr(),
			check:    func(f *Facts) bool { return f.OSRelease != nil },
		},
		{
			name:     "lsb",
			collName: "lsb",
			result:   lsbInfoPtr(),
			check:    func(f *Facts) bool { return f.LSB != nil },
		},
		{
			name:     "init",
			collName: "init",
			result:   initInfoPtr(),
			check:    func(f *Facts) bool { return f.Init != nil },
		},
		{
			name:     "shard",
			collName: "shard",
			result:   shardInfoPtr(),
			check:    func(f *Facts) bool { return f.Shard != nil },
		},
		{
			name:     "package_mgr",
			collName: "package_mgr",
			result:   packageMgrInfoPtr(),
			check:    func(f *Facts) bool { return f.PackageMgr != nil },
		},
		{
			name:     "sessions",
			collName: "sessions",
			result:   sessionsInfoPtr(),
			check:    func(f *Facts) bool { return f.Sessions != nil },
		},
		{
			name:     "pci",
			collName: "pci",
			result:   pciInfoPtr(),
			check:    func(f *Facts) bool { return f.PCI != nil },
		},
		{
			name:     "scsi",
			collName: "scsi",
			result:   scsiInfoPtr(),
			check:    func(f *Facts) bool { return f.SCSI != nil },
		},
		{
			name:     "hardware",
			collName: "hardware",
			result:   hardwareInfoPtr(),
			check:    func(f *Facts) bool { return f.Hardware != nil },
		},
		{
			name:     "block_device",
			collName: "block_device",
			result:   blockDeviceInfoPtr(),
			check:    func(f *Facts) bool { return f.BlockDevice != nil },
		},
		{
			name:     "command",
			collName: "command",
			result:   commandInfoPtr(),
			check:    func(f *Facts) bool { return f.Command != nil },
		},
		{
			name:     "docker",
			collName: "docker",
			result:   dockerInfoPtr(),
			check:    func(f *Facts) bool { return f.Docker != nil },
		},
		{
			name:     "grub2",
			collName: "grub2",
			result:   grub2InfoPtr(),
			check:    func(f *Facts) bool { return f.Grub2 != nil },
		},
		{
			name:     "hostnamectl",
			collName: "hostnamectl",
			result:   hostnamectlInfoPtr(),
			check:    func(f *Facts) bool { return f.Hostnamectl != nil },
		},
		{
			name:     "interrupts",
			collName: "interrupts",
			result:   interruptsInfoPtr(),
			check:    func(f *Facts) bool { return f.Interrupts != nil },
		},
		{
			name:     "ipc",
			collName: "ipc",
			result:   ipcInfoPtr(),
			check:    func(f *Facts) bool { return f.IPC != nil },
		},
		{
			name:     "languages",
			collName: "languages",
			result:   languagesInfoPtr(),
			check:    func(f *Facts) bool { return f.Languages != nil },
		},
		{
			name:     "libvirt",
			collName: "libvirt",
			result:   libvirtInfoPtr(),
			check:    func(f *Facts) bool { return f.Libvirt != nil },
		},
		{
			name:     "livepatch",
			collName: "livepatch",
			result:   livepatchInfoPtr(),
			check:    func(f *Facts) bool { return f.Livepatch != nil },
		},
		{
			name:     "mdadm",
			collName: "mdadm",
			result:   mdadmInfoPtr(),
			check:    func(f *Facts) bool { return f.Mdadm != nil },
		},
		{
			name:     "packages",
			collName: "packages",
			result:   packagesInfoPtr(),
			check:    func(f *Facts) bool { return f.Packages != nil },
		},
		{
			name:     "rpm",
			collName: "rpm",
			result:   rpmInfoPtr(),
			check:    func(f *Facts) bool { return f.RPM != nil },
		},
		{
			name:     "selinux",
			collName: "selinux",
			result:   selinuxInfoPtr(),
			check:    func(f *Facts) bool { return f.SELinux != nil },
		},
		{
			name:     "services",
			collName: "services",
			result:   servicesInfoPtr(),
			check:    func(f *Facts) bool { return f.Services != nil },
		},
		{
			name:     "ssh",
			collName: "ssh",
			result:   sshInfoPtr(),
			check:    func(f *Facts) bool { return f.SSH != nil },
		},
		{
			name:     "sysconf",
			collName: "sysconf",
			result:   sysconfInfoPtr(),
			check:    func(f *Facts) bool { return f.Sysconf != nil },
		},
		{
			name:     "sysctl",
			collName: "sysctl",
			result:   sysctlInfoPtr(),
			check:    func(f *Facts) bool { return f.Sysctl != nil },
		},
		{
			name:     "systemd_paths",
			collName: "systemd_paths",
			result:   systemdPathsInfoPtr(),
			check:    func(f *Facts) bool { return f.SystemdPaths != nil },
		},
		{
			name:     "tc",
			collName: "tc",
			result:   tcInfoPtr(),
			check:    func(f *Facts) bool { return f.TC != nil },
		},
		{
			name:     "virtualbox",
			collName: "virtualbox",
			result:   virtualboxInfoPtr(),
			check:    func(f *Facts) bool { return f.VirtualBox != nil },
		},
		{
			name:     "vmware",
			collName: "vmware",
			result:   vmwareInfoPtr(),
			check:    func(f *Facts) bool { return f.VMware != nil },
		},
		{
			name:     "zpools",
			collName: "zpools",
			result:   zpoolsInfoPtr(),
			check:    func(f *Facts) bool { return f.Zpools != nil },
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
				Hostname:       hostnameInfoPtr(),
				Kernel:         kernelInfoPtr(),
				Uptime:         uptimeInfoPtr(),
				Virtualization: virtInfoPtr(),
				MachineID:      machineIDInfoPtr(),
				CPU:            cpuInfoPtr(),
				Memory:         memoryInfoPtr(),
				Filesystem:     filesystemInfoPtr(),
				Disk:           diskInfoPtr(),
				Network:        networkInfoPtr(),
				Process:        processInfoPtr(),
				Users:          usersInfoPtr(),
				Timezone:       timezoneInfoPtr(),
				RootGroup:      rootGroupInfoPtr(),
				Shells:         shellsInfoPtr(),
				Fips:           fipsInfoPtr(),
				Load:           loadInfoPtr(),
				OSRelease:      osReleaseInfoPtr(),
				LSB:            lsbInfoPtr(),
				Init:           initInfoPtr(),
				Shard:          shardInfoPtr(),
				PackageMgr:     packageMgrInfoPtr(),
				BlockDevice:    blockDeviceInfoPtr(),
				Command:        commandInfoPtr(),
				Docker:         dockerInfoPtr(),
				Grub2:          grub2InfoPtr(),
				Hostnamectl:    hostnamectlInfoPtr(),
				Interrupts:     interruptsInfoPtr(),
				IPC:            ipcInfoPtr(),
				Languages:      languagesInfoPtr(),
				Libvirt:        libvirtInfoPtr(),
				Livepatch:      livepatchInfoPtr(),
				Mdadm:          mdadmInfoPtr(),
				Packages:       packagesInfoPtr(),
				RPM:            rpmInfoPtr(),
				SELinux:        selinuxInfoPtr(),
				Services:       servicesInfoPtr(),
				SSH:            sshInfoPtr(),
				Sysconf:        sysconfInfoPtr(),
				Sysctl:         sysctlInfoPtr(),
				SystemdPaths:   systemdPathsInfoPtr(),
				TC:             tcInfoPtr(),
				VirtualBox:     virtualboxInfoPtr(),
				VMware:         vmwareInfoPtr(),
				Zpools:         zpoolsInfoPtr(),
			},
			want: 46,
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Equal(tt.want, tt.facts.countPopulated())
		})
	}
}
