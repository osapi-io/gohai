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

	"github.com/osapi-io/gohai/internal/executor"
	execmocks "github.com/osapi-io/gohai/internal/executor/gen"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/virtualization"
)

type VirtualizationLinuxPublicTestSuite struct {
	suite.Suite
}

func TestVirtualizationLinuxPublicTestSuite(t *testing.T) {
	suite.Run(t, new(VirtualizationLinuxPublicTestSuite))
}

// fsWith builds a memfs containing the given (path → contents) map.
// Empty content means the file is created empty (presence-only check).
// Parent directories are auto-created.
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

func dirOf(p string) string {
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
// Argv is joined with spaces ("command -v docker"). Unmatched calls
// return ("", error) so a missing binary acts like a not-found.
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

func (s *VirtualizationLinuxPublicTestSuite) TestCollect() {
	tests := []struct {
		name     string
		fs       avfs.VFS
		exec     executor.Executor
		validate func(*virtualization.Info)
	}{
		{
			name:     "bare metal: empty info, empty Systems",
			fs:       fsWith(s.T(), nil),
			exec:     virtExec(s.T(), nil),
			validate: func(i *virtualization.Info) { s.Empty(i.Systems); s.Empty(i.System) },
		},
		{
			name: "systemd-detect-virt --vm reports kvm",
			fs:   fsWith(s.T(), nil),
			exec: virtExec(s.T(), map[string][]byte{
				"systemd-detect-virt --vm": []byte("kvm\n"),
			}),
			validate: func(i *virtualization.Info) {
				s.Equal("guest", i.Systems["kvm"])
				s.Equal("kvm", i.System)
			},
		},
		{
			name: "docker host on PATH",
			fs:   fsWith(s.T(), nil),
			exec: virtExec(s.T(), map[string][]byte{
				"command -v docker": []byte("/usr/bin/docker\n"),
			}),
			validate: func(i *virtualization.Info) { s.Equal("host", i.Systems["docker"]) },
		},
		{
			name: "podman + nova hosts",
			fs:   fsWith(s.T(), nil),
			exec: virtExec(s.T(), map[string][]byte{
				"command -v podman": []byte("/usr/bin/podman\n"),
				"command -v nova":   []byte("/usr/bin/nova\n"),
			}),
			validate: func(i *virtualization.Info) {
				s.Equal("host", i.Systems["podman"])
				s.Equal("host", i.Systems["openstack"])
			},
		},
		{
			name: "xen guest then host (control_d) overrides",
			// /proc/xen is created implicitly as a directory by MkdirAll
			// when /proc/xen/capabilities is written; fileExists("/proc/xen")
			// returns true via Stat on the directory.
			fs: fsWith(s.T(), map[string]string{
				"/proc/xen/capabilities": "control_d\n",
			}),
			exec:     virtExec(s.T(), nil),
			validate: func(i *virtualization.Info) { s.Equal("host", i.Systems["xen"]) },
		},
		{
			name: "vbox host via /proc/modules",
			fs: fsWith(s.T(), map[string]string{
				"/proc/modules": "vboxdrv 524288 0 - Live 0x0\n",
			}),
			exec:     virtExec(s.T(), nil),
			validate: func(i *virtualization.Info) { s.Equal("host", i.Systems["vbox"]) },
		},
		{
			name: "vbox guest via /proc/modules",
			fs: fsWith(s.T(), map[string]string{
				"/proc/modules": "vboxguest 360448 1 - Live 0x0\n",
			}),
			exec:     virtExec(s.T(), nil),
			validate: func(i *virtualization.Info) { s.Equal("guest", i.Systems["vbox"]) },
		},
		{
			name: "kvm guest via /proc/cpuinfo QEMU string",
			fs: fsWith(s.T(), map[string]string{
				"/proc/cpuinfo": "model name : QEMU Virtual CPU version 2.5+\n",
			}),
			exec:     virtExec(s.T(), nil),
			validate: func(i *virtualization.Info) { s.Equal("guest", i.Systems["kvm"]) },
		},
		{
			name: "kvm host via /sys/devices/virtual/misc/kvm without hypervisor flag",
			fs: fsWith(s.T(), map[string]string{
				"/proc/cpuinfo":                 "flags : vmx\n",
				"/sys/devices/virtual/misc/kvm": "",
			}),
			exec:     virtExec(s.T(), nil),
			validate: func(i *virtualization.Info) { s.Equal("host", i.Systems["kvm"]) },
		},
		{
			name: "kvm guest via hypervisor flag",
			fs: fsWith(s.T(), map[string]string{
				"/proc/cpuinfo":                 "flags : vmx hypervisor lm\n",
				"/sys/devices/virtual/misc/kvm": "",
			}),
			exec:     virtExec(s.T(), nil),
			validate: func(i *virtualization.Info) { s.Equal("guest", i.Systems["kvm"]) },
		},
		{
			name: "DMI vmware",
			fs: fsWith(s.T(), map[string]string{
				"/sys/class/dmi/id/product_name": "VMware Virtual Platform\n",
			}),
			exec:     virtExec(s.T(), nil),
			validate: func(i *virtualization.Info) { s.Equal("guest", i.Systems["vmware"]) },
		},
		{
			name: "DMI hyperv",
			fs: fsWith(s.T(), map[string]string{
				"/sys/class/dmi/id/sys_vendor":   "Microsoft Corporation\n",
				"/sys/class/dmi/id/product_name": "Virtual Machine\n",
			}),
			exec:     virtExec(s.T(), nil),
			validate: func(i *virtualization.Info) { s.Equal("guest", i.Systems["hyperv"]) },
		},
		{
			name: "DMI parallels",
			fs: fsWith(s.T(), map[string]string{
				"/sys/class/dmi/id/product_name": "Parallels Virtual Platform\n",
			}),
			exec:     virtExec(s.T(), nil),
			validate: func(i *virtualization.Info) { s.Equal("guest", i.Systems["parallels"]) },
		},
		{
			name: "DMI xen",
			fs: fsWith(s.T(), map[string]string{
				"/sys/class/dmi/id/product_name": "HVM domU\n",
				"/sys/class/dmi/id/sys_vendor":   "Xen\n",
			}),
			exec:     virtExec(s.T(), nil),
			validate: func(i *virtualization.Info) { s.Equal("guest", i.Systems["xen"]) },
		},
		{
			name: "DMI qemu/kvm",
			fs: fsWith(s.T(), map[string]string{
				"/sys/class/dmi/id/product_name": "Standard PC (Q35 + ICH9, 2009)\n",
				"/sys/class/dmi/id/sys_vendor":   "QEMU\n",
			}),
			exec:     virtExec(s.T(), nil),
			validate: func(i *virtualization.Info) { s.Equal("guest", i.Systems["kvm"]) },
		},
		{
			name: "openvz host then guest precedence (host wins)",
			fs: fsWith(s.T(), map[string]string{
				"/proc/bc/0": "",
			}),
			exec:     virtExec(s.T(), nil),
			validate: func(i *virtualization.Info) { s.Equal("host", i.Systems["openvz"]) },
		},
		{
			name: "openvz guest",
			fs: fsWith(s.T(), map[string]string{
				"/proc/vz": "",
			}),
			exec:     virtExec(s.T(), nil),
			validate: func(i *virtualization.Info) { s.Equal("guest", i.Systems["openvz"]) },
		},
		{
			name: "hyperv guest via kvp_pool_3",
			fs: fsWith(s.T(), map[string]string{
				"/var/lib/hyperv/.kvp_pool_3": "",
			}),
			exec:     virtExec(s.T(), nil),
			validate: func(i *virtualization.Info) { s.Equal("guest", i.Systems["hyperv"]) },
		},
		{
			name: "linux-vserver host via s_context: 0",
			fs: fsWith(s.T(), map[string]string{
				"/proc/self/status": "Name: bash\ns_context: 0\n",
			}),
			exec:     virtExec(s.T(), nil),
			validate: func(i *virtualization.Info) { s.Equal("host", i.Systems["linux-vserver"]) },
		},
		{
			name: "linux-vserver guest via VxID",
			fs: fsWith(s.T(), map[string]string{
				"/proc/self/status": "Name: bash\nVxID: 42\n",
			}),
			exec:     virtExec(s.T(), nil),
			validate: func(i *virtualization.Info) { s.Equal("guest", i.Systems["linux-vserver"]) },
		},
		{
			name: "cgroup docker container",
			fs: fsWith(s.T(), map[string]string{
				"/proc/self/cgroup": "12:devices:/docker/abc123\n",
			}),
			exec:     virtExec(s.T(), nil),
			validate: func(i *virtualization.Info) { s.Equal("guest", i.Systems["docker"]) },
		},
		{
			name: "cgroup containerd remaps to docker",
			fs: fsWith(s.T(), map[string]string{
				"/proc/self/cgroup": "12:devices:/containerd/xyz\n",
			}),
			exec:     virtExec(s.T(), nil),
			validate: func(i *virtualization.Info) { s.Equal("guest", i.Systems["docker"]) },
		},
		{
			name: "cgroup lxc",
			fs: fsWith(s.T(), map[string]string{
				"/proc/self/cgroup": "12:devices:/lxc/c1\n",
			}),
			exec:     virtExec(s.T(), nil),
			validate: func(i *virtualization.Info) { s.Equal("guest", i.Systems["lxc"]) },
		},
		{
			name: "environ container=lxc",
			fs: fsWith(s.T(), map[string]string{
				"/proc/1/environ": "PATH=/usr/bin\x00container=lxc\x00",
			}),
			exec:     virtExec(s.T(), nil),
			validate: func(i *virtualization.Info) { s.Equal("guest", i.Systems["lxc"]) },
		},
		{
			name: "environ container=systemd-nspawn",
			fs: fsWith(s.T(), map[string]string{
				"/proc/1/environ": "container=systemd-nspawn\x00",
			}),
			exec:     virtExec(s.T(), nil),
			validate: func(i *virtualization.Info) { s.Equal("guest", i.Systems["nspawn"]) },
		},
		{
			name: "environ container=podman",
			fs: fsWith(s.T(), map[string]string{
				"/proc/1/environ": "container=podman\x00",
			}),
			exec:     virtExec(s.T(), nil),
			validate: func(i *virtualization.Info) { s.Equal("guest", i.Systems["podman"]) },
		},
		{
			name: "/.dockerenv override forces docker guest",
			fs: fsWith(s.T(), map[string]string{
				"/.dockerenv": "",
			}),
			exec:     virtExec(s.T(), nil),
			validate: func(i *virtualization.Info) { s.Equal("guest", i.Systems["docker"]) },
		},
		{
			name: "/.dockerinit alternate",
			fs: fsWith(s.T(), map[string]string{
				"/.dockerinit": "",
			}),
			exec:     virtExec(s.T(), nil),
			validate: func(i *virtualization.Info) { s.Equal("guest", i.Systems["docker"]) },
		},
		{
			name: "lxd guest via /dev/lxd/sock",
			fs: fsWith(s.T(), map[string]string{
				"/dev/lxd/sock": "",
			}),
			exec:     virtExec(s.T(), nil),
			validate: func(i *virtualization.Info) { s.Equal("guest", i.Systems["lxd"]) },
		},
		{
			name: "lxd host via /var/lib/lxd/devlxd",
			fs: fsWith(s.T(), map[string]string{
				"/var/lib/lxd/devlxd": "",
			}),
			exec:     virtExec(s.T(), nil),
			validate: func(i *virtualization.Info) { s.Equal("host", i.Systems["lxd"]) },
		},
		{
			name: "lxd snap host path",
			fs: fsWith(s.T(), map[string]string{
				"/var/snap/lxd/common/lxd/devlxd": "",
			}),
			exec:     virtExec(s.T(), nil),
			validate: func(i *virtualization.Info) { s.Equal("host", i.Systems["lxd"]) },
		},
		{
			name: "nested: kvm guest + docker host",
			fs: fsWith(s.T(), map[string]string{
				"/proc/cpuinfo": "model name : QEMU Virtual CPU\n",
			}),
			exec: virtExec(s.T(), map[string][]byte{
				"command -v docker": []byte("/usr/bin/docker\n"),
			}),
			validate: func(i *virtualization.Info) {
				s.Equal("guest", i.Systems["kvm"])
				s.Equal("host", i.Systems["docker"])
				s.Len(i.Systems, 2)
			},
		},
		{
			name: "nil Exec: no exec-based detections, file-based still work",
			fs: fsWith(s.T(), map[string]string{
				"/.dockerenv": "",
			}),
			exec:     nil,
			validate: func(i *virtualization.Info) { s.Equal("guest", i.Systems["docker"]) },
		},
		{
			name: "xen guest only (no /proc/xen/capabilities): fileContains miss",
			// /proc/xen exists (as the directory holding /proc/xen/other) but
			// /proc/xen/capabilities is absent — fileContains returns false,
			// xen stays guest.
			fs: fsWith(s.T(), map[string]string{
				"/proc/xen/other": "",
			}),
			exec:     virtExec(s.T(), nil),
			validate: func(i *virtualization.Info) { s.Equal("guest", i.Systems["xen"]) },
		},
		{
			name:     "systemd-detect-virt empty output: skipped",
			fs:       fsWith(s.T(), nil),
			exec:     virtExec(s.T(), map[string][]byte{"systemd-detect-virt --vm": []byte("\n")}),
			validate: func(i *virtualization.Info) { s.Empty(i.Systems) },
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			c := &virtualization.Linux{FS: tt.fs, Exec: tt.exec}
			got, err := c.Collect(context.Background())
			s.Require().NoError(err)
			info, ok := got.(*virtualization.Info)
			s.Require().True(ok)
			tt.validate(info)
		})
	}
}
