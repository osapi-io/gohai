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
	execmocks "github.com/osapi-io/gohai/internal/executor/gen"
	"github.com/osapi-io/gohai/internal/platform"
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
		{"darwin dispatches to Darwin", "darwin", "darwin"},
		{"debian dispatches to Linux", "debian", "linux"},
		{"rhel dispatches to Linux", "rhel", "linux"},
		{"arch dispatches to Linux", "arch", "linux"},
		{"unknown dispatches to Linux", "", "linux"},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			platform.Detect = func() string { return tt.detect }
			c := virtualization.New()
			s.Equal("virtualization", c.Name())
			s.Equal("virtualization", c.Category())
			s.True(c.DefaultEnabled())
			s.Empty(c.Dependencies())
			switch tt.wantKind {
			case "darwin":
				_, ok := c.(*virtualization.Darwin)
				s.True(ok)
			case "linux":
				_, ok := c.(*virtualization.Linux)
				s.True(ok)
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
		validate func(*virtualization.Info)
	}{
		{
			name:     "linux: bare metal empty Systems",
			variant:  "linux",
			fs:       func() avfs.VFS { return fsWith(s.T(), nil) },
			exec:     func(t *testing.T) executor.Executor { return virtExec(t, nil) },
			validate: func(i *virtualization.Info) { s.Empty(i.Systems); s.Empty(i.System) },
		},
		{
			name:    "linux: systemd-detect-virt --vm reports kvm",
			variant: "linux",
			fs:      func() avfs.VFS { return fsWith(s.T(), nil) },
			exec: func(t *testing.T) executor.Executor {
				return virtExec(t, map[string][]byte{
					"systemd-detect-virt --vm": []byte("kvm\n"),
				})
			},
			validate: func(i *virtualization.Info) {
				s.Equal("guest", i.Systems["kvm"])
				s.Equal("kvm", i.System)
			},
		},
		{
			name:    "linux: docker host on PATH",
			variant: "linux",
			fs:      func() avfs.VFS { return fsWith(s.T(), nil) },
			exec: func(t *testing.T) executor.Executor {
				return virtExec(t, map[string][]byte{
					"command -v docker": []byte("/usr/bin/docker\n"),
				})
			},
			validate: func(i *virtualization.Info) { s.Equal("host", i.Systems["docker"]) },
		},
		{
			name:    "linux: podman + nova hosts",
			variant: "linux",
			fs:      func() avfs.VFS { return fsWith(s.T(), nil) },
			exec: func(t *testing.T) executor.Executor {
				return virtExec(t, map[string][]byte{
					"command -v podman": []byte("/usr/bin/podman\n"),
					"command -v nova":   []byte("/usr/bin/nova\n"),
				})
			},
			validate: func(i *virtualization.Info) {
				s.Equal("host", i.Systems["podman"])
				s.Equal("host", i.Systems["openstack"])
			},
		},
		{
			name:    "linux: xen guest then host (control_d) overrides",
			variant: "linux",
			fs: func() avfs.VFS {
				return fsWith(s.T(), map[string]string{
					"/proc/xen/capabilities": "control_d\n",
				})
			},
			exec:     func(t *testing.T) executor.Executor { return virtExec(t, nil) },
			validate: func(i *virtualization.Info) { s.Equal("host", i.Systems["xen"]) },
		},
		{
			name:    "linux: vbox host via /proc/modules",
			variant: "linux",
			fs: func() avfs.VFS {
				return fsWith(s.T(), map[string]string{
					"/proc/modules": "vboxdrv 524288 0 - Live 0x0\n",
				})
			},
			exec:     func(t *testing.T) executor.Executor { return virtExec(t, nil) },
			validate: func(i *virtualization.Info) { s.Equal("host", i.Systems["vbox"]) },
		},
		{
			name:    "linux: vbox guest via /proc/modules",
			variant: "linux",
			fs: func() avfs.VFS {
				return fsWith(s.T(), map[string]string{
					"/proc/modules": "vboxguest 360448 1 - Live 0x0\n",
				})
			},
			exec:     func(t *testing.T) executor.Executor { return virtExec(t, nil) },
			validate: func(i *virtualization.Info) { s.Equal("guest", i.Systems["vbox"]) },
		},
		{
			name:    "linux: kvm guest via /proc/cpuinfo QEMU string",
			variant: "linux",
			fs: func() avfs.VFS {
				return fsWith(s.T(), map[string]string{
					"/proc/cpuinfo": "model name : QEMU Virtual CPU version 2.5+\n",
				})
			},
			exec:     func(t *testing.T) executor.Executor { return virtExec(t, nil) },
			validate: func(i *virtualization.Info) { s.Equal("guest", i.Systems["kvm"]) },
		},
		{
			name:    "linux: kvm host via /sys/devices/virtual/misc/kvm without hypervisor flag",
			variant: "linux",
			fs: func() avfs.VFS {
				return fsWith(s.T(), map[string]string{
					"/proc/cpuinfo":                 "flags : vmx\n",
					"/sys/devices/virtual/misc/kvm": "",
				})
			},
			exec:     func(t *testing.T) executor.Executor { return virtExec(t, nil) },
			validate: func(i *virtualization.Info) { s.Equal("host", i.Systems["kvm"]) },
		},
		{
			name:    "linux: kvm guest via hypervisor flag",
			variant: "linux",
			fs: func() avfs.VFS {
				return fsWith(s.T(), map[string]string{
					"/proc/cpuinfo":                 "flags : vmx hypervisor lm\n",
					"/sys/devices/virtual/misc/kvm": "",
				})
			},
			exec:     func(t *testing.T) executor.Executor { return virtExec(t, nil) },
			validate: func(i *virtualization.Info) { s.Equal("guest", i.Systems["kvm"]) },
		},
		{
			name:    "linux: DMI vmware",
			variant: "linux",
			fs: func() avfs.VFS {
				return fsWith(s.T(), map[string]string{
					"/sys/class/dmi/id/product_name": "VMware Virtual Platform\n",
				})
			},
			exec:     func(t *testing.T) executor.Executor { return virtExec(t, nil) },
			validate: func(i *virtualization.Info) { s.Equal("guest", i.Systems["vmware"]) },
		},
		{
			name:    "linux: DMI hyperv",
			variant: "linux",
			fs: func() avfs.VFS {
				return fsWith(s.T(), map[string]string{
					"/sys/class/dmi/id/sys_vendor":   "Microsoft Corporation\n",
					"/sys/class/dmi/id/product_name": "Virtual Machine\n",
				})
			},
			exec:     func(t *testing.T) executor.Executor { return virtExec(t, nil) },
			validate: func(i *virtualization.Info) { s.Equal("guest", i.Systems["hyperv"]) },
		},
		{
			name:    "linux: DMI parallels",
			variant: "linux",
			fs: func() avfs.VFS {
				return fsWith(s.T(), map[string]string{
					"/sys/class/dmi/id/product_name": "Parallels Virtual Platform\n",
				})
			},
			exec:     func(t *testing.T) executor.Executor { return virtExec(t, nil) },
			validate: func(i *virtualization.Info) { s.Equal("guest", i.Systems["parallels"]) },
		},
		{
			name:    "linux: DMI xen",
			variant: "linux",
			fs: func() avfs.VFS {
				return fsWith(s.T(), map[string]string{
					"/sys/class/dmi/id/product_name": "HVM domU\n",
					"/sys/class/dmi/id/sys_vendor":   "Xen\n",
				})
			},
			exec:     func(t *testing.T) executor.Executor { return virtExec(t, nil) },
			validate: func(i *virtualization.Info) { s.Equal("guest", i.Systems["xen"]) },
		},
		{
			name:    "linux: DMI qemu/kvm",
			variant: "linux",
			fs: func() avfs.VFS {
				return fsWith(s.T(), map[string]string{
					"/sys/class/dmi/id/product_name": "Standard PC (Q35 + ICH9, 2009)\n",
					"/sys/class/dmi/id/sys_vendor":   "QEMU\n",
				})
			},
			exec:     func(t *testing.T) executor.Executor { return virtExec(t, nil) },
			validate: func(i *virtualization.Info) { s.Equal("guest", i.Systems["kvm"]) },
		},
		{
			name:    "linux: openvz host then guest precedence (host wins)",
			variant: "linux",
			fs: func() avfs.VFS {
				return fsWith(s.T(), map[string]string{"/proc/bc/0": ""})
			},
			exec:     func(t *testing.T) executor.Executor { return virtExec(t, nil) },
			validate: func(i *virtualization.Info) { s.Equal("host", i.Systems["openvz"]) },
		},
		{
			name:    "linux: openvz guest",
			variant: "linux",
			fs: func() avfs.VFS {
				return fsWith(s.T(), map[string]string{"/proc/vz": ""})
			},
			exec:     func(t *testing.T) executor.Executor { return virtExec(t, nil) },
			validate: func(i *virtualization.Info) { s.Equal("guest", i.Systems["openvz"]) },
		},
		{
			name:    "linux: hyperv guest via kvp_pool_3",
			variant: "linux",
			fs: func() avfs.VFS {
				return fsWith(s.T(), map[string]string{"/var/lib/hyperv/.kvp_pool_3": ""})
			},
			exec:     func(t *testing.T) executor.Executor { return virtExec(t, nil) },
			validate: func(i *virtualization.Info) { s.Equal("guest", i.Systems["hyperv"]) },
		},
		{
			name:    "linux: linux-vserver host via s_context: 0",
			variant: "linux",
			fs: func() avfs.VFS {
				return fsWith(s.T(), map[string]string{
					"/proc/self/status": "Name: bash\ns_context: 0\n",
				})
			},
			exec:     func(t *testing.T) executor.Executor { return virtExec(t, nil) },
			validate: func(i *virtualization.Info) { s.Equal("host", i.Systems["linux-vserver"]) },
		},
		{
			name:    "linux: linux-vserver guest via VxID",
			variant: "linux",
			fs: func() avfs.VFS {
				return fsWith(s.T(), map[string]string{
					"/proc/self/status": "Name: bash\nVxID: 42\n",
				})
			},
			exec:     func(t *testing.T) executor.Executor { return virtExec(t, nil) },
			validate: func(i *virtualization.Info) { s.Equal("guest", i.Systems["linux-vserver"]) },
		},
		{
			name:    "linux: cgroup docker container",
			variant: "linux",
			fs: func() avfs.VFS {
				return fsWith(s.T(), map[string]string{
					"/proc/self/cgroup": "12:devices:/docker/abc123\n",
				})
			},
			exec:     func(t *testing.T) executor.Executor { return virtExec(t, nil) },
			validate: func(i *virtualization.Info) { s.Equal("guest", i.Systems["docker"]) },
		},
		{
			name:    "linux: cgroup containerd remaps to docker",
			variant: "linux",
			fs: func() avfs.VFS {
				return fsWith(s.T(), map[string]string{
					"/proc/self/cgroup": "12:devices:/containerd/xyz\n",
				})
			},
			exec:     func(t *testing.T) executor.Executor { return virtExec(t, nil) },
			validate: func(i *virtualization.Info) { s.Equal("guest", i.Systems["docker"]) },
		},
		{
			name:    "linux: cgroup lxc",
			variant: "linux",
			fs: func() avfs.VFS {
				return fsWith(s.T(), map[string]string{
					"/proc/self/cgroup": "12:devices:/lxc/c1\n",
				})
			},
			exec:     func(t *testing.T) executor.Executor { return virtExec(t, nil) },
			validate: func(i *virtualization.Info) { s.Equal("guest", i.Systems["lxc"]) },
		},
		{
			name:    "linux: environ container=lxc",
			variant: "linux",
			fs: func() avfs.VFS {
				return fsWith(s.T(), map[string]string{
					"/proc/1/environ": "PATH=/usr/bin\x00container=lxc\x00",
				})
			},
			exec:     func(t *testing.T) executor.Executor { return virtExec(t, nil) },
			validate: func(i *virtualization.Info) { s.Equal("guest", i.Systems["lxc"]) },
		},
		{
			name:    "linux: environ container=systemd-nspawn",
			variant: "linux",
			fs: func() avfs.VFS {
				return fsWith(s.T(), map[string]string{
					"/proc/1/environ": "container=systemd-nspawn\x00",
				})
			},
			exec:     func(t *testing.T) executor.Executor { return virtExec(t, nil) },
			validate: func(i *virtualization.Info) { s.Equal("guest", i.Systems["nspawn"]) },
		},
		{
			name:    "linux: environ container=podman",
			variant: "linux",
			fs: func() avfs.VFS {
				return fsWith(s.T(), map[string]string{
					"/proc/1/environ": "container=podman\x00",
				})
			},
			exec:     func(t *testing.T) executor.Executor { return virtExec(t, nil) },
			validate: func(i *virtualization.Info) { s.Equal("guest", i.Systems["podman"]) },
		},
		{
			name:    "linux: /.dockerenv override forces docker guest",
			variant: "linux",
			fs: func() avfs.VFS {
				return fsWith(s.T(), map[string]string{"/.dockerenv": ""})
			},
			exec:     func(t *testing.T) executor.Executor { return virtExec(t, nil) },
			validate: func(i *virtualization.Info) { s.Equal("guest", i.Systems["docker"]) },
		},
		{
			name:    "linux: /.dockerinit alternate",
			variant: "linux",
			fs: func() avfs.VFS {
				return fsWith(s.T(), map[string]string{"/.dockerinit": ""})
			},
			exec:     func(t *testing.T) executor.Executor { return virtExec(t, nil) },
			validate: func(i *virtualization.Info) { s.Equal("guest", i.Systems["docker"]) },
		},
		{
			name:    "linux: lxd guest via /dev/lxd/sock",
			variant: "linux",
			fs: func() avfs.VFS {
				return fsWith(s.T(), map[string]string{"/dev/lxd/sock": ""})
			},
			exec:     func(t *testing.T) executor.Executor { return virtExec(t, nil) },
			validate: func(i *virtualization.Info) { s.Equal("guest", i.Systems["lxd"]) },
		},
		{
			name:    "linux: lxd host via /var/lib/lxd/devlxd",
			variant: "linux",
			fs: func() avfs.VFS {
				return fsWith(s.T(), map[string]string{"/var/lib/lxd/devlxd": ""})
			},
			exec:     func(t *testing.T) executor.Executor { return virtExec(t, nil) },
			validate: func(i *virtualization.Info) { s.Equal("host", i.Systems["lxd"]) },
		},
		{
			name:    "linux: lxd snap host path",
			variant: "linux",
			fs: func() avfs.VFS {
				return fsWith(s.T(), map[string]string{"/var/snap/lxd/common/lxd/devlxd": ""})
			},
			exec:     func(t *testing.T) executor.Executor { return virtExec(t, nil) },
			validate: func(i *virtualization.Info) { s.Equal("host", i.Systems["lxd"]) },
		},
		{
			name:    "linux: nested kvm guest + docker host",
			variant: "linux",
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
				s.Equal("guest", i.Systems["kvm"])
				s.Equal("host", i.Systems["docker"])
				s.Len(i.Systems, 2)
			},
		},
		{
			name:    "linux: nil Exec, no exec-based detections file-based still work",
			variant: "linux",
			fs: func() avfs.VFS {
				return fsWith(s.T(), map[string]string{"/.dockerenv": ""})
			},
			exec:     func(*testing.T) executor.Executor { return nil },
			validate: func(i *virtualization.Info) { s.Equal("guest", i.Systems["docker"]) },
		},
		{
			name:    "linux: xen guest only (no /proc/xen/capabilities)",
			variant: "linux",
			fs: func() avfs.VFS {
				return fsWith(s.T(), map[string]string{"/proc/xen/other": ""})
			},
			exec:     func(t *testing.T) executor.Executor { return virtExec(t, nil) },
			validate: func(i *virtualization.Info) { s.Equal("guest", i.Systems["xen"]) },
		},
		{
			name:    "linux: systemd-detect-virt empty output skipped",
			variant: "linux",
			fs:      func() avfs.VFS { return fsWith(s.T(), nil) },
			exec: func(t *testing.T) executor.Executor {
				return virtExec(t, map[string][]byte{"systemd-detect-virt --vm": []byte("\n")})
			},
			validate: func(i *virtualization.Info) { s.Empty(i.Systems) },
		},
		{
			name:     "darwin: bare metal Mac empty",
			variant:  "darwin",
			fs:       func() avfs.VFS { return fsWith(s.T(), nil) },
			exec:     func(t *testing.T) executor.Executor { return virtExec(t, nil) },
			validate: func(i *virtualization.Info) { s.Empty(i.Systems) },
		},
		{
			name:    "darwin: docker host on PATH",
			variant: "darwin",
			fs:      func() avfs.VFS { return fsWith(s.T(), nil) },
			exec: func(t *testing.T) executor.Executor {
				return virtExec(t, map[string][]byte{
					"command -v docker": []byte("/usr/local/bin/docker\n"),
				})
			},
			validate: func(i *virtualization.Info) { s.Equal("host", i.Systems["docker"]) },
		},
		{
			name:    "darwin: VBoxManage host on PATH",
			variant: "darwin",
			fs:      func() avfs.VFS { return fsWith(s.T(), nil) },
			exec: func(t *testing.T) executor.Executor {
				return virtExec(t, map[string][]byte{
					"command -v VBoxManage": []byte("/usr/local/bin/VBoxManage\n"),
				})
			},
			validate: func(i *virtualization.Info) { s.Equal("host", i.Systems["vbox"]) },
		},
		{
			name:    "darwin: prlctl host on PATH",
			variant: "darwin",
			fs:      func() avfs.VFS { return fsWith(s.T(), nil) },
			exec: func(t *testing.T) executor.Executor {
				return virtExec(t, map[string][]byte{
					"command -v prlctl": []byte("/usr/local/bin/prlctl\n"),
				})
			},
			validate: func(i *virtualization.Info) { s.Equal("host", i.Systems["parallels"]) },
		},
		{
			name:    "darwin: VMware Fusion app present",
			variant: "darwin",
			fs: func() avfs.VFS {
				return fsWith(s.T(), map[string]string{"/Applications/VMware Fusion.app": ""})
			},
			exec:     func(t *testing.T) executor.Executor { return virtExec(t, nil) },
			validate: func(i *virtualization.Info) { s.Equal("host", i.Systems["vmware"]) },
		},
		{
			name:    "darwin: QEMU/Virtualization.framework guest via sysctl",
			variant: "darwin",
			fs:      func() avfs.VFS { return fsWith(s.T(), nil) },
			exec: func(t *testing.T) executor.Executor {
				return virtExec(t, map[string][]byte{
					"sysctl -n kern.hv_vmm_present": []byte("1\n"),
				})
			},
			validate: func(i *virtualization.Info) { s.Equal("guest", i.Systems["qemu"]) },
		},
		{
			name:    "darwin: sysctl returns 0, no qemu detection",
			variant: "darwin",
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
			variant: "darwin",
			fs:      func() avfs.VFS { return fsWith(s.T(), nil) },
			exec: func(t *testing.T) executor.Executor {
				return virtExec(t, map[string][]byte{
					"ioreg -n pci1ab8,4000": []byte(
						"    | |   \"compatible\" = <\"pci1ab8,4000\">\n",
					),
				})
			},
			validate: func(i *virtualization.Info) { s.Equal("guest", i.Systems["parallels"]) },
		},
		{
			name:    "darwin: VirtualBox guest via system_profiler Boot ROM",
			variant: "darwin",
			fs:      func() avfs.VFS { return fsWith(s.T(), nil) },
			exec: func(t *testing.T) executor.Executor {
				return virtExec(t, map[string][]byte{
					"system_profiler SPHardwareDataType": []byte(sysProfilerVBox),
				})
			},
			validate: func(i *virtualization.Info) { s.Equal("guest", i.Systems["vbox"]) },
		},
		{
			name:    "darwin: VMware guest via system_profiler Boot ROM",
			variant: "darwin",
			fs:      func() avfs.VFS { return fsWith(s.T(), nil) },
			exec: func(t *testing.T) executor.Executor {
				return virtExec(t, map[string][]byte{
					"system_profiler SPHardwareDataType": []byte(sysProfilerVMware),
				})
			},
			validate: func(i *virtualization.Info) { s.Equal("guest", i.Systems["vmware"]) },
		},
		{
			name:    "darwin: Apple VM via Model Identifier",
			variant: "darwin",
			fs:      func() avfs.VFS { return fsWith(s.T(), nil) },
			exec: func(t *testing.T) executor.Executor {
				return virtExec(t, map[string][]byte{
					"system_profiler SPHardwareDataType": []byte(sysProfilerAppleVM),
				})
			},
			validate: func(i *virtualization.Info) { s.Equal("guest", i.Systems["apple"]) },
		},
		{
			name:    "darwin: nil Exec no exec detections file-based still works",
			variant: "darwin",
			fs: func() avfs.VFS {
				return fsWith(s.T(), map[string]string{"/Applications/VMware Fusion.app": ""})
			},
			exec:     func(*testing.T) executor.Executor { return nil },
			validate: func(i *virtualization.Info) { s.Equal("host", i.Systems["vmware"]) },
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			var c virtualization.Collector
			switch tt.variant {
			case "linux":
				c = &virtualization.Linux{FS: tt.fs(), Exec: tt.exec(s.T())}
			case "darwin":
				c = &virtualization.Darwin{FS: tt.fs(), Exec: tt.exec(s.T())}
			}
			got, err := c.Collect(context.Background(), nil)
			s.Require().NoError(err)
			info, ok := got.(*virtualization.Info)
			s.Require().True(ok)
			tt.validate(info)
		})
	}
}
