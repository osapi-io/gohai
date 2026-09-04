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

package virtualization_test

import (
	"context"
	"errors"
	"testing"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/memfs"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/executor"
	execmocks "github.com/osapi-io/gohai/internal/executor/mocks"
	"github.com/osapi-io/gohai/internal/platform"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/cpu"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/virtualization"
)

var (
	_ collector.Collector = (*virtualization.Linux)(nil)
	_ collector.Collector = (*virtualization.Darwin)(nil)
)

type VirtualizationPublicTestSuite struct {
	suite.Suite
}

func TestVirtualizationPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(VirtualizationPublicTestSuite))
}

// fsWith builds a memfs containing the given (path → contents) map.
func fsWith(
	t require.TestingT,
	files map[string]string,
) avfs.VFS {
	fs := memfs.New()
	for path, content := range files {
		require.NoError(t, fs.MkdirAll(dirOf(path), 0o755))
		require.NoError(t, fs.WriteFile(path, []byte(content), 0o644))
	}
	return fs
}

func dirOf(
	p string,
) string {
	for i := len(p) - 1; i >= 0; i-- {
		if p[i] == '/' {
			if i == 0 {
				return "/"
			}
			return p[:i]
		}
	}
	return "/"
}

// virtExec returns a MockExecutor that maps argv → (output, error).
func virtExec(
	t *testing.T,
	answers map[string][]byte,
) executor.Executor {
	ctrl := gomock.NewController(t)
	m := execmocks.NewMockExecutor(ctrl)
	m.EXPECT().
		Execute(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, name string, args ...string) ([]byte, error) {
			key := name
			for _, a := range args {
				key += " " + a
			}
			if out, ok := answers[key]; ok {
				return out, nil
			}
			return nil, errors.New("not found")
		}).
		AnyTimes()
	return m
}

const sysProfilerVBox = `Hardware:

    Hardware Overview:

      Model Name: VirtualBox
      Boot ROM Version: VirtualBox-1.0.0
      Model Identifier: MacBookPro18,2
`

const sysProfilerVMware = `Hardware Overview:

      Boot ROM Version: VMW1.234
`

const sysProfilerAppleVM = `Hardware Overview:

      Boot ROM Version: 12345.67.89
      Model Identifier: VirtualMac2,1
`

func (s *VirtualizationPublicTestSuite) TestNew() {
	orig := platform.Detect
	defer func() { platform.Detect = orig }()

	tests := []struct {
		name     string
		detect   string
		wantKind string
	}{
		{"darwin dispatches to Darwin", osDarwin, osDarwin},
		{"debian dispatches to Linux", "debian", osLinux},
		{"rhel dispatches to Linux", "rhel", osLinux},
		{"arch dispatches to Linux", "arch", osLinux},
		{"unknown dispatches to Linux", "", osLinux},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			platform.Detect = func() string { return tt.detect }
			c := virtualization.New()
			s.Equal("virtualization", c.Name())
			s.Equal("virtualization", c.Category())
			s.True(c.DefaultEnabled())
			s.Equal([]string{"cpu"}, c.Dependencies())
			switch tt.wantKind {
			case osDarwin:
				_, ok := c.(*virtualization.Darwin)
				s.True(ok)
			case osLinux:
				_, ok := c.(*virtualization.Linux)
				s.True(ok)
			default:
				s.Failf("unhandled case", "%v", tt.wantKind)
			}
		})
	}
}

func (s *VirtualizationPublicTestSuite) TestCollect() {
	tests := []struct {
		name     string
		variant  string
		fs       func() avfs.VFS
		exec     func(*testing.T) executor.Executor
		prior    collector.PriorResults
		validate func(*virtualization.Info)
	}{
		{
			name:     "linux: bare metal empty Systems",
			variant:  osLinux,
			fs:       func() avfs.VFS { return fsWith(s.T(), nil) },
			exec:     func(t *testing.T) executor.Executor { return virtExec(t, nil) },
			validate: func(i *virtualization.Info) { s.Empty(i.Systems); s.Empty(i.System) },
		},
		{
			name:    "linux: systemd-detect-virt --vm reports kvm",
			variant: osLinux,
			fs:      func() avfs.VFS { return fsWith(s.T(), nil) },
			exec: func(t *testing.T) executor.Executor {
				return virtExec(t, map[string][]byte{
					"systemd-detect-virt --vm": []byte("kvm\n"),
				})
			},
			validate: func(i *virtualization.Info) {
				s.Equal(roleGuest, i.Systems["kvm"])
				s.Equal("kvm", i.System)
			},
		},
		{
			name:    "linux: docker host on PATH",
			variant: osLinux,
			fs:      func() avfs.VFS { return fsWith(s.T(), nil) },
			exec: func(t *testing.T) executor.Executor {
				return virtExec(t, map[string][]byte{
					"command -v docker": []byte("/usr/bin/docker\n"),
				})
			},
			validate: func(i *virtualization.Info) { s.Equal(roleHost, i.Systems["docker"]) },
		},
		{
			name:    "linux: podman + nova hosts",
			variant: osLinux,
			fs:      func() avfs.VFS { return fsWith(s.T(), nil) },
			exec: func(t *testing.T) executor.Executor {
				return virtExec(t, map[string][]byte{
					"command -v podman": []byte("/usr/bin/podman\n"),
					"command -v nova":   []byte("/usr/bin/nova\n"),
				})
			},
			validate: func(i *virtualization.Info) {
				s.Equal(roleHost, i.Systems["podman"])
				s.Equal(roleHost, i.Systems["openstack"])
			},
		},
		{
			name:    "linux: xen guest then host (control_d) overrides",
			variant: osLinux,
			fs: func() avfs.VFS {
				return fsWith(s.T(), map[string]string{
					"/proc/xen/capabilities": "control_d\n",
				})
			},
			exec:     func(t *testing.T) executor.Executor { return virtExec(t, nil) },
			validate: func(i *virtualization.Info) { s.Equal(roleHost, i.Systems["xen"]) },
		},
		{
			name:    "linux: vbox host via /proc/modules",
			variant: osLinux,
			fs: func() avfs.VFS {
				return fsWith(s.T(), map[string]string{
					"/proc/modules": "vboxdrv 524288 0 - Live 0x0\n",
				})
			},
			exec:     func(t *testing.T) executor.Executor { return virtExec(t, nil) },
			validate: func(i *virtualization.Info) { s.Equal(roleHost, i.Systems["vbox"]) },
		},
		{
			name:    "linux: vbox guest via /proc/modules",
			variant: osLinux,
			fs: func() avfs.VFS {
				return fsWith(s.T(), map[string]string{
					"/proc/modules": "vboxguest 360448 1 - Live 0x0\n",
				})
			},
			exec:     func(t *testing.T) executor.Executor { return virtExec(t, nil) },
			validate: func(i *virtualization.Info) { s.Equal(roleGuest, i.Systems["vbox"]) },
		},
		{
			// /proc/modules exists on every Linux host; neither
			// VirtualBox module being loaded is the ordinary case.
			name:    "linux: /proc/modules without either vbox module",
			variant: osLinux,
			fs: func() avfs.VFS {
				return fsWith(s.T(), map[string]string{
					"/proc/modules": "ext4 950272 1 - Live 0x0\n",
				})
			},
			exec:     func(t *testing.T) executor.Executor { return virtExec(t, nil) },
			validate: func(i *virtualization.Info) { s.Empty(i.Systems["vbox"]) },
		},
		{
			name:    "linux: kvm guest via /proc/cpuinfo QEMU string",
			variant: osLinux,
			fs: func() avfs.VFS {
				return fsWith(s.T(), map[string]string{
					"/proc/cpuinfo": "model name : QEMU Virtual CPU version 2.5+\n",
				})
			},
			exec:     func(t *testing.T) executor.Executor { return virtExec(t, nil) },
			validate: func(i *virtualization.Info) { s.Equal(roleGuest, i.Systems["kvm"]) },
		},
		{
			name:    "linux: kvm host via /sys/devices/virtual/misc/kvm without hypervisor flag",
			variant: osLinux,
			fs: func() avfs.VFS {
				return fsWith(s.T(), map[string]string{
					"/proc/cpuinfo":                 "flags : vmx\n",
					"/sys/devices/virtual/misc/kvm": "",
				})
			},
			exec:     func(t *testing.T) executor.Executor { return virtExec(t, nil) },
			validate: func(i *virtualization.Info) { s.Equal(roleHost, i.Systems["kvm"]) },
		},
		{
			name:    "linux: kvm guest via hypervisor flag",
			variant: osLinux,
			fs: func() avfs.VFS {
				return fsWith(s.T(), map[string]string{
					"/proc/cpuinfo":                 "flags : vmx hypervisor lm\n",
					"/sys/devices/virtual/misc/kvm": "",
				})
			},
			exec:     func(t *testing.T) executor.Executor { return virtExec(t, nil) },
			validate: func(i *virtualization.Info) { s.Equal(roleGuest, i.Systems["kvm"]) },
		},
		{
			name:    "linux: DMI vmware",
			variant: osLinux,
			fs: func() avfs.VFS {
				return fsWith(s.T(), map[string]string{
					"/sys/class/dmi/id/sys_vendor":   "VMware, Inc.\n",
					"/sys/class/dmi/id/product_name": "VMware Virtual Platform\n",
				})
			},
			exec:     func(t *testing.T) executor.Executor { return virtExec(t, nil) },
			validate: func(i *virtualization.Info) { s.Equal(roleGuest, i.Systems["vmware"]) },
		},
		{
			name:    "linux: DMI hyperv",
			variant: osLinux,
			fs: func() avfs.VFS {
				return fsWith(s.T(), map[string]string{
					"/sys/class/dmi/id/sys_vendor":   "Microsoft Corporation\n",
					"/sys/class/dmi/id/product_name": "Virtual Machine\n",
				})
			},
			exec:     func(t *testing.T) executor.Executor { return virtExec(t, nil) },
			validate: func(i *virtualization.Info) { s.Equal(roleGuest, i.Systems["hyperv"]) },
		},
		{
			// Hyper-V is the one signal that needs both DMI fields.
			// A Microsoft vendor on other hardware is not a guest.
			name:    "linux: Microsoft vendor without a virtual machine product",
			variant: osLinux,
			fs: func() avfs.VFS {
				return fsWith(s.T(), map[string]string{
					"/sys/class/dmi/id/sys_vendor":   "Microsoft Corporation\n",
					"/sys/class/dmi/id/product_name": "Surface Laptop\n",
				})
			},
			exec:     func(t *testing.T) executor.Executor { return virtExec(t, nil) },
			validate: func(i *virtualization.Info) { s.Empty(i.Systems["hyperv"]) },
		},
		{
			name:    "linux: DMI parallels",
			variant: osLinux,
			fs: func() avfs.VFS {
				return fsWith(s.T(), map[string]string{
					"/sys/class/dmi/id/sys_vendor":   "Parallels Software International Inc.\n",
					"/sys/class/dmi/id/product_name": "Parallels Virtual Platform\n",
				})
			},
			exec:     func(t *testing.T) executor.Executor { return virtExec(t, nil) },
			validate: func(i *virtualization.Info) { s.Equal(roleGuest, i.Systems["parallels"]) },
		},
		{
			name:    "linux: DMI xen",
			variant: osLinux,
			fs: func() avfs.VFS {
				return fsWith(s.T(), map[string]string{
					"/sys/class/dmi/id/product_name": "HVM domU\n",
					"/sys/class/dmi/id/sys_vendor":   "Xen\n",
				})
			},
			exec:     func(t *testing.T) executor.Executor { return virtExec(t, nil) },
			validate: func(i *virtualization.Info) { s.Equal(roleGuest, i.Systems["xen"]) },
		},
		{
			name:    "linux: DMI qemu/kvm",
			variant: osLinux,
			fs: func() avfs.VFS {
				return fsWith(s.T(), map[string]string{
					"/sys/class/dmi/id/product_name": "Standard PC (Q35 + ICH9, 2009)\n",
					"/sys/class/dmi/id/sys_vendor":   "QEMU\n",
				})
			},
			exec:     func(t *testing.T) executor.Executor { return virtExec(t, nil) },
			validate: func(i *virtualization.Info) { s.Equal(roleGuest, i.Systems["kvm"]) },
		},
		{
			name:    "linux: DMI openstack via sys_vendor",
			variant: osLinux,
			fs: func() avfs.VFS {
				return fsWith(s.T(), map[string]string{
					"/sys/class/dmi/id/sys_vendor":   "OpenStack Foundation\n",
					"/sys/class/dmi/id/product_name": "OpenStack Nova\n",
				})
			},
			exec:     func(t *testing.T) executor.Executor { return virtExec(t, nil) },
			validate: func(i *virtualization.Info) { s.Equal(roleGuest, i.Systems["openstack"]) },
		},
		{
			name:    "linux: DMI openstack via product_name (Red Hat variant)",
			variant: osLinux,
			fs: func() avfs.VFS {
				return fsWith(s.T(), map[string]string{
					"/sys/class/dmi/id/sys_vendor":   "Red Hat\n",
					"/sys/class/dmi/id/product_name": "OpenStack Compute\n",
				})
			},
			exec:     func(t *testing.T) executor.Executor { return virtExec(t, nil) },
			validate: func(i *virtualization.Info) { s.Equal(roleGuest, i.Systems["openstack"]) },
		},
		{
			name:    "linux: DMI amazonec2",
			variant: osLinux,
			fs: func() avfs.VFS {
				return fsWith(s.T(), map[string]string{
					"/sys/class/dmi/id/sys_vendor": "Amazon EC2\n",
				})
			},
			exec:     func(t *testing.T) executor.Executor { return virtExec(t, nil) },
			validate: func(i *virtualization.Info) { s.Equal(roleGuest, i.Systems["amazonec2"]) },
		},
		{
			name:    "linux: DMI veertu",
			variant: osLinux,
			fs: func() avfs.VFS {
				return fsWith(s.T(), map[string]string{
					"/sys/class/dmi/id/sys_vendor": "Veertu, Inc.\n",
				})
			},
			exec:     func(t *testing.T) executor.Executor { return virtExec(t, nil) },
			validate: func(i *virtualization.Info) { s.Equal(roleGuest, i.Systems["veertu"]) },
		},
		{
			name:    "linux: DMI virtualbox via product_name",
			variant: osLinux,
			fs: func() avfs.VFS {
				return fsWith(s.T(), map[string]string{
					"/sys/class/dmi/id/sys_vendor":   "innotek GmbH\n",
					"/sys/class/dmi/id/product_name": "VirtualBox\n",
				})
			},
			exec:     func(t *testing.T) executor.Executor { return virtExec(t, nil) },
			validate: func(i *virtualization.Info) { s.Equal(roleGuest, i.Systems["vbox"]) },
		},
		{
			name:    "linux: DMI kvm via RHEV product",
			variant: osLinux,
			fs: func() avfs.VFS {
				return fsWith(s.T(), map[string]string{
					"/sys/class/dmi/id/sys_vendor":   "Red Hat\n",
					"/sys/class/dmi/id/product_name": "RHEV Hypervisor\n",
				})
			},
			exec:     func(t *testing.T) executor.Executor { return virtExec(t, nil) },
			validate: func(i *virtualization.Info) { s.Equal(roleGuest, i.Systems["kvm"]) },
		},
		{
			name:    "linux: DMI bhyve via product_name",
			variant: osLinux,
			fs: func() avfs.VFS {
				return fsWith(s.T(), map[string]string{
					"/sys/class/dmi/id/product_name": "BHYVE\n",
				})
			},
			exec:     func(t *testing.T) executor.Executor { return virtExec(t, nil) },
			validate: func(i *virtualization.Info) { s.Equal(roleGuest, i.Systems["bhyve"]) },
		},
		{
			name:    "linux: cpuinfo Common 32-bit KVM processor → kvm guest",
			variant: osLinux,
			fs: func() avfs.VFS {
				return fsWith(s.T(), map[string]string{
					"/proc/cpuinfo": "model name : Common 32-bit KVM processor\n",
				})
			},
			exec:     func(t *testing.T) executor.Executor { return virtExec(t, nil) },
			validate: func(i *virtualization.Info) { s.Equal(roleGuest, i.Systems["kvm"]) },
		},
		{
			name:    "linux: cgroup nested docker (systemd /system.slice/docker-*.scope)",
			variant: osLinux,
			fs: func() avfs.VFS {
				return fsWith(s.T(), map[string]string{
					"/proc/self/cgroup": "0::/system.slice/docker-47341cd3bba14d17d3d67e6b4bd3b46f.scope\n",
				})
			},
			exec:     func(t *testing.T) executor.Executor { return virtExec(t, nil) },
			validate: func(i *virtualization.Info) { s.Equal(roleGuest, i.Systems["docker"]) },
		},
		{
			name:    "linux: cgroup nested docker (docker-ce layout)",
			variant: osLinux,
			fs: func() avfs.VFS {
				return fsWith(s.T(), map[string]string{
					"/proc/self/cgroup": "0::/docker-ce/docker/b15b851234abcdef\n",
				})
			},
			exec:     func(t *testing.T) executor.Executor { return virtExec(t, nil) },
			validate: func(i *virtualization.Info) { s.Equal(roleGuest, i.Systems["docker"]) },
		},
		{
			name:    "linux: kvm via cpu prior (nested VM without /sys/devices/virtual/misc/kvm)",
			variant: osLinux,
			fs: func() avfs.VFS {
				return fsWith(s.T(), map[string]string{})
			},
			exec: func(t *testing.T) executor.Executor { return virtExec(t, nil) },
			prior: collector.PriorResults{
				"cpu": &cpu.Info{HypervisorVendor: "KVM", VirtualizationType: "full"},
			},
			validate: func(i *virtualization.Info) { s.Equal(roleGuest, i.Systems["kvm"]) },
		},
		{
			name:    "linux: lxc host missing cgroup file → no lxc",
			variant: osLinux,
			fs: func() avfs.VFS {
				return fsWith(s.T(), map[string]string{})
			},
			exec: func(t *testing.T) executor.Executor {
				return virtExec(
					t,
					map[string][]byte{"command -v lxc-start": []byte("/usr/bin/lxc-start\n")},
				)
			},
			validate: func(i *virtualization.Info) { s.Empty(i.Systems["lxc"]) },
		},
		{
			name:    "linux: lxc host cgroup root not / → no lxc",
			variant: osLinux,
			fs: func() avfs.VFS {
				return fsWith(s.T(), map[string]string{
					"/proc/self/cgroup": "0::/some/subpath\n",
				})
			},
			exec: func(t *testing.T) executor.Executor {
				return virtExec(
					t,
					map[string][]byte{"command -v lxc-start": []byte("/usr/bin/lxc-start\n")},
				)
			},
			validate: func(i *virtualization.Info) { s.Empty(i.Systems["lxc"]) },
		},
		{
			name:    "linux: lxc host cgroup malformed line → no lxc",
			variant: osLinux,
			fs: func() avfs.VFS {
				return fsWith(s.T(), map[string]string{
					"/proc/self/cgroup": "malformed\n",
				})
			},
			exec: func(t *testing.T) executor.Executor {
				return virtExec(
					t,
					map[string][]byte{"command -v lxc-start": []byte("/usr/bin/lxc-start\n")},
				)
			},
			validate: func(i *virtualization.Info) { s.Empty(i.Systems["lxc"]) },
		},
		{
			name:    "linux: lxc host via lxc-start on PATH + cgroup roots all /",
			variant: osLinux,
			fs: func() avfs.VFS {
				return fsWith(s.T(), map[string]string{
					"/proc/self/cgroup": "0::/\n",
				})
			},
			exec: func(t *testing.T) executor.Executor {
				return virtExec(
					t,
					map[string][]byte{"command -v lxc-start": []byte("/usr/bin/lxc-start\n")},
				)
			},
			validate: func(i *virtualization.Info) { s.Equal(roleHost, i.Systems["lxc"]) },
		},
		{
			name:    "linux: openvz host then guest precedence (host wins)",
			variant: osLinux,
			fs: func() avfs.VFS {
				return fsWith(s.T(), map[string]string{"/proc/bc/0": ""})
			},
			exec:     func(t *testing.T) executor.Executor { return virtExec(t, nil) },
			validate: func(i *virtualization.Info) { s.Equal(roleHost, i.Systems["openvz"]) },
		},
		{
			name:    "linux: openvz guest",
			variant: osLinux,
			fs: func() avfs.VFS {
				return fsWith(s.T(), map[string]string{"/proc/vz": ""})
			},
			exec:     func(t *testing.T) executor.Executor { return virtExec(t, nil) },
			validate: func(i *virtualization.Info) { s.Equal(roleGuest, i.Systems["openvz"]) },
		},
		{
			name:    "linux: hyperv guest via kvp_pool_3 with hypervisor_host extraction",
			variant: osLinux,
			fs: func() avfs.VFS {
				// KVP pool blob — Ohai scans for printable bytes between
				// "HostName" and "HostingSystemEditionId". Embed NULs and
				// mixed case so printable-filter + lowercasing both run.
				blob := "junk\x00HostName\x00\x00HyperV-Host-01\x00\x00HostingSystemEditionId\x00more"
				return fsWith(s.T(), map[string]string{"/var/lib/hyperv/.kvp_pool_3": blob})
			},
			exec: func(t *testing.T) executor.Executor { return virtExec(t, nil) },
			validate: func(i *virtualization.Info) {
				s.Equal(roleGuest, i.Systems["hyperv"])
				s.Equal("hyperv-host-01", i.HypervisorHost)
			},
		},
		{
			name:    "linux: hyperv kvp_pool_3 without HostName leaves hypervisor_host empty",
			variant: osLinux,
			fs: func() avfs.VFS {
				return fsWith(s.T(), map[string]string{"/var/lib/hyperv/.kvp_pool_3": "empty-pool"})
			},
			exec: func(t *testing.T) executor.Executor { return virtExec(t, nil) },
			validate: func(i *virtualization.Info) {
				s.Equal(roleGuest, i.Systems["hyperv"])
				s.Empty(i.HypervisorHost)
			},
		},
		{
			name:    "linux: linux-vserver host via s_context: 0",
			variant: osLinux,
			fs: func() avfs.VFS {
				return fsWith(s.T(), map[string]string{
					"/proc/self/status": "Name: bash\ns_context: 0\n",
				})
			},
			exec:     func(t *testing.T) executor.Executor { return virtExec(t, nil) },
			validate: func(i *virtualization.Info) { s.Equal(roleHost, i.Systems["linux-vserver"]) },
		},
		{
			name:    "linux: linux-vserver guest via VxID",
			variant: osLinux,
			fs: func() avfs.VFS {
				return fsWith(s.T(), map[string]string{
					"/proc/self/status": "Name: bash\nVxID: 42\n",
				})
			},
			exec:     func(t *testing.T) executor.Executor { return virtExec(t, nil) },
			validate: func(i *virtualization.Info) { s.Equal(roleGuest, i.Systems["linux-vserver"]) },
		},
		{
			// /proc/self/status is always present; a host that is not
			// a vserver simply carries no context line.
			name:    "linux: /proc/self/status without a vserver context",
			variant: osLinux,
			fs: func() avfs.VFS {
				return fsWith(s.T(), map[string]string{
					"/proc/self/status": "Name:\tbash\nTgid:\t1\n",
				})
			},
			exec: func(t *testing.T) executor.Executor { return virtExec(t, nil) },
			validate: func(i *virtualization.Info) {
				s.Empty(i.Systems["linux-vserver"])
			},
		},
		{
			// PID 1's environment names a runtime only inside a
			// container.
			name:    "linux: /proc/1/environ names no container runtime",
			variant: osLinux,
			fs: func() avfs.VFS {
				return fsWith(s.T(), map[string]string{
					"/proc/1/environ": "PATH=/usr/bin\x00HOME=/root\x00",
				})
			},
			exec: func(t *testing.T) executor.Executor { return virtExec(t, nil) },
			validate: func(i *virtualization.Info) {
				s.Empty(i.Systems["lxc"])
				s.Empty(i.Systems["nspawn"])
				s.Empty(i.Systems["podman"])
			},
		},
		{
			name:    "linux: cgroup docker container",
			variant: osLinux,
			fs: func() avfs.VFS {
				return fsWith(s.T(), map[string]string{
					"/proc/self/cgroup": "12:devices:/docker/abc123\n",
				})
			},
			exec:     func(t *testing.T) executor.Executor { return virtExec(t, nil) },
			validate: func(i *virtualization.Info) { s.Equal(roleGuest, i.Systems["docker"]) },
		},
		{
			name:    "linux: cgroup containerd remaps to docker",
			variant: osLinux,
			fs: func() avfs.VFS {
				return fsWith(s.T(), map[string]string{
					"/proc/self/cgroup": "12:devices:/containerd/xyz\n",
				})
			},
			exec:     func(t *testing.T) executor.Executor { return virtExec(t, nil) },
			validate: func(i *virtualization.Info) { s.Equal(roleGuest, i.Systems["docker"]) },
		},
		{
			name:    "linux: cgroup lxc",
			variant: osLinux,
			fs: func() avfs.VFS {
				return fsWith(s.T(), map[string]string{
					"/proc/self/cgroup": "12:devices:/lxc/c1\n",
				})
			},
			exec:     func(t *testing.T) executor.Executor { return virtExec(t, nil) },
			validate: func(i *virtualization.Info) { s.Equal(roleGuest, i.Systems["lxc"]) },
		},
		{
			name:    "linux: environ container=lxc",
			variant: osLinux,
			fs: func() avfs.VFS {
				return fsWith(s.T(), map[string]string{
					"/proc/1/environ": "PATH=/usr/bin\x00container=lxc\x00",
				})
			},
			exec:     func(t *testing.T) executor.Executor { return virtExec(t, nil) },
			validate: func(i *virtualization.Info) { s.Equal(roleGuest, i.Systems["lxc"]) },
		},
		{
			name:    "linux: environ container=systemd-nspawn",
			variant: osLinux,
			fs: func() avfs.VFS {
				return fsWith(s.T(), map[string]string{
					"/proc/1/environ": "container=systemd-nspawn\x00",
				})
			},
			exec:     func(t *testing.T) executor.Executor { return virtExec(t, nil) },
			validate: func(i *virtualization.Info) { s.Equal(roleGuest, i.Systems["nspawn"]) },
		},
		{
			name:    "linux: environ container=podman",
			variant: osLinux,
			fs: func() avfs.VFS {
				return fsWith(s.T(), map[string]string{
					"/proc/1/environ": "container=podman\x00",
				})
			},
			exec:     func(t *testing.T) executor.Executor { return virtExec(t, nil) },
			validate: func(i *virtualization.Info) { s.Equal(roleGuest, i.Systems["podman"]) },
		},
		{
			name:    "linux: /.dockerenv override forces docker guest",
			variant: osLinux,
			fs: func() avfs.VFS {
				return fsWith(s.T(), map[string]string{"/.dockerenv": ""})
			},
			exec:     func(t *testing.T) executor.Executor { return virtExec(t, nil) },
			validate: func(i *virtualization.Info) { s.Equal(roleGuest, i.Systems["docker"]) },
		},
		{
			name:    "linux: /.dockerinit alternate",
			variant: osLinux,
			fs: func() avfs.VFS {
				return fsWith(s.T(), map[string]string{"/.dockerinit": ""})
			},
			exec:     func(t *testing.T) executor.Executor { return virtExec(t, nil) },
			validate: func(i *virtualization.Info) { s.Equal(roleGuest, i.Systems["docker"]) },
		},
		{
			name:    "linux: lxd guest via /dev/lxd/sock",
			variant: osLinux,
			fs: func() avfs.VFS {
				return fsWith(s.T(), map[string]string{"/dev/lxd/sock": ""})
			},
			exec:     func(t *testing.T) executor.Executor { return virtExec(t, nil) },
			validate: func(i *virtualization.Info) { s.Equal(roleGuest, i.Systems["lxd"]) },
		},
		{
			name:    "linux: lxd host via /var/lib/lxd/devlxd",
			variant: osLinux,
			fs: func() avfs.VFS {
				return fsWith(s.T(), map[string]string{"/var/lib/lxd/devlxd": ""})
			},
			exec:     func(t *testing.T) executor.Executor { return virtExec(t, nil) },
			validate: func(i *virtualization.Info) { s.Equal(roleHost, i.Systems["lxd"]) },
		},
		{
			name:    "linux: lxd snap host path",
			variant: osLinux,
			fs: func() avfs.VFS {
				return fsWith(s.T(), map[string]string{"/var/snap/lxd/common/lxd/devlxd": ""})
			},
			exec:     func(t *testing.T) executor.Executor { return virtExec(t, nil) },
			validate: func(i *virtualization.Info) { s.Equal(roleHost, i.Systems["lxd"]) },
		},
		{
			name:    "linux: nested kvm guest + docker host",
			variant: osLinux,
			fs: func() avfs.VFS {
				return fsWith(s.T(), map[string]string{
					"/proc/cpuinfo": "model name : QEMU Virtual CPU\n",
				})
			},
			exec: func(t *testing.T) executor.Executor {
				return virtExec(t, map[string][]byte{
					"command -v docker": []byte("/usr/bin/docker\n"),
				})
			},
			validate: func(i *virtualization.Info) {
				s.Equal(roleGuest, i.Systems["kvm"])
				s.Equal(roleHost, i.Systems["docker"])
				s.Len(i.Systems, 2)
			},
		},
		{
			name:    "linux: nil Exec, no exec-based detections file-based still work",
			variant: osLinux,
			fs: func() avfs.VFS {
				return fsWith(s.T(), map[string]string{"/.dockerenv": ""})
			},
			exec:     func(*testing.T) executor.Executor { return nil },
			validate: func(i *virtualization.Info) { s.Equal(roleGuest, i.Systems["docker"]) },
		},
		{
			name:    "linux: xen guest only (no /proc/xen/capabilities)",
			variant: osLinux,
			fs: func() avfs.VFS {
				return fsWith(s.T(), map[string]string{"/proc/xen/other": ""})
			},
			exec:     func(t *testing.T) executor.Executor { return virtExec(t, nil) },
			validate: func(i *virtualization.Info) { s.Equal(roleGuest, i.Systems["xen"]) },
		},
		{
			name:    "linux: systemd-detect-virt empty output skipped",
			variant: osLinux,
			fs:      func() avfs.VFS { return fsWith(s.T(), nil) },
			exec: func(t *testing.T) executor.Executor {
				return virtExec(t, map[string][]byte{"systemd-detect-virt --vm": []byte("\n")})
			},
			validate: func(i *virtualization.Info) { s.Empty(i.Systems) },
		},
		{
			name:     "darwin: bare metal Mac empty",
			variant:  osDarwin,
			fs:       func() avfs.VFS { return fsWith(s.T(), nil) },
			exec:     func(t *testing.T) executor.Executor { return virtExec(t, nil) },
			validate: func(i *virtualization.Info) { s.Empty(i.Systems) },
		},
		{
			name:    "darwin: docker host on PATH",
			variant: osDarwin,
			fs:      func() avfs.VFS { return fsWith(s.T(), nil) },
			exec: func(t *testing.T) executor.Executor {
				return virtExec(t, map[string][]byte{
					"command -v docker": []byte("/usr/local/bin/docker\n"),
				})
			},
			validate: func(i *virtualization.Info) { s.Equal(roleHost, i.Systems["docker"]) },
		},
		{
			name:    "darwin: VBoxManage host on PATH",
			variant: osDarwin,
			fs:      func() avfs.VFS { return fsWith(s.T(), nil) },
			exec: func(t *testing.T) executor.Executor {
				return virtExec(t, map[string][]byte{
					"command -v VBoxManage": []byte("/usr/local/bin/VBoxManage\n"),
				})
			},
			validate: func(i *virtualization.Info) { s.Equal(roleHost, i.Systems["vbox"]) },
		},
		{
			name:    "darwin: prlctl host on PATH",
			variant: osDarwin,
			fs:      func() avfs.VFS { return fsWith(s.T(), nil) },
			exec: func(t *testing.T) executor.Executor {
				return virtExec(t, map[string][]byte{
					"command -v prlctl": []byte("/usr/local/bin/prlctl\n"),
				})
			},
			validate: func(i *virtualization.Info) { s.Equal(roleHost, i.Systems["parallels"]) },
		},
		{
			name:    "darwin: VMware Fusion app present",
			variant: osDarwin,
			fs: func() avfs.VFS {
				return fsWith(s.T(), map[string]string{"/Applications/VMware Fusion.app": ""})
			},
			exec:     func(t *testing.T) executor.Executor { return virtExec(t, nil) },
			validate: func(i *virtualization.Info) { s.Equal(roleHost, i.Systems["vmware"]) },
		},
		{
			name:    "darwin: QEMU/Virtualization.framework guest via sysctl",
			variant: osDarwin,
			fs:      func() avfs.VFS { return fsWith(s.T(), nil) },
			exec: func(t *testing.T) executor.Executor {
				return virtExec(t, map[string][]byte{
					"sysctl -n kern.hv_vmm_present": []byte("1\n"),
				})
			},
			validate: func(i *virtualization.Info) { s.Equal(roleGuest, i.Systems["qemu"]) },
		},
		{
			name:    "darwin: sysctl returns 0, no qemu detection",
			variant: osDarwin,
			fs:      func() avfs.VFS { return fsWith(s.T(), nil) },
			exec: func(t *testing.T) executor.Executor {
				return virtExec(t, map[string][]byte{
					"sysctl -n kern.hv_vmm_present": []byte("0\n"),
				})
			},
			validate: func(i *virtualization.Info) { s.NotContains(i.Systems, "qemu") },
		},
		{
			name:    "darwin: Parallels guest via ioreg",
			variant: osDarwin,
			fs:      func() avfs.VFS { return fsWith(s.T(), nil) },
			exec: func(t *testing.T) executor.Executor {
				return virtExec(t, map[string][]byte{
					"ioreg -n pci1ab8,4000": []byte(
						"    | |   \"compatible\" = <\"pci1ab8,4000\">\n",
					),
				})
			},
			validate: func(i *virtualization.Info) { s.Equal(roleGuest, i.Systems["parallels"]) },
		},
		{
			name:    "darwin: VirtualBox guest via system_profiler Boot ROM",
			variant: osDarwin,
			fs:      func() avfs.VFS { return fsWith(s.T(), nil) },
			exec: func(t *testing.T) executor.Executor {
				return virtExec(t, map[string][]byte{
					"system_profiler SPHardwareDataType": []byte(sysProfilerVBox),
				})
			},
			validate: func(i *virtualization.Info) { s.Equal(roleGuest, i.Systems["vbox"]) },
		},
		{
			name:    "darwin: VMware guest via system_profiler Boot ROM",
			variant: osDarwin,
			fs:      func() avfs.VFS { return fsWith(s.T(), nil) },
			exec: func(t *testing.T) executor.Executor {
				return virtExec(t, map[string][]byte{
					"system_profiler SPHardwareDataType": []byte(sysProfilerVMware),
				})
			},
			validate: func(i *virtualization.Info) { s.Equal(roleGuest, i.Systems["vmware"]) },
		},
		{
			name:    "darwin: Apple VM via Model Identifier",
			variant: osDarwin,
			fs:      func() avfs.VFS { return fsWith(s.T(), nil) },
			exec: func(t *testing.T) executor.Executor {
				return virtExec(t, map[string][]byte{
					"system_profiler SPHardwareDataType": []byte(sysProfilerAppleVM),
				})
			},
			validate: func(i *virtualization.Info) { s.Equal(roleGuest, i.Systems["apple"]) },
		},
		{
			name:    "darwin: nil Exec no exec detections file-based still works",
			variant: osDarwin,
			fs: func() avfs.VFS {
				return fsWith(s.T(), map[string]string{"/Applications/VMware Fusion.app": ""})
			},
			exec:     func(*testing.T) executor.Executor { return nil },
			validate: func(i *virtualization.Info) { s.Equal(roleHost, i.Systems["vmware"]) },
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			var c virtualization.Collector
			switch tt.variant {
			case osLinux:
				c = &virtualization.Linux{FS: tt.fs(), Exec: tt.exec(s.T())}
			case osDarwin:
				c = &virtualization.Darwin{FS: tt.fs(), Exec: tt.exec(s.T())}
			default:
				s.Failf("unhandled case", "%v", tt.variant)
			}
			got, err := c.Collect(context.Background(), tt.prior)
			s.Require().NoError(err)
			info, ok := got.(*virtualization.Info)
			s.Require().True(ok)
			tt.validate(info)
		})
	}
}
