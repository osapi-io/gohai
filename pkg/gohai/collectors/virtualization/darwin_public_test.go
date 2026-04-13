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
	"testing"

	"github.com/avfs/avfs"
	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/internal/executor"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/virtualization"
)

type VirtualizationDarwinPublicTestSuite struct {
	suite.Suite
}

func TestVirtualizationDarwinPublicTestSuite(t *testing.T) {
	suite.Run(t, new(VirtualizationDarwinPublicTestSuite))
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

func (s *VirtualizationDarwinPublicTestSuite) TestCollect() {
	tests := []struct {
		name     string
		fs       avfs.VFS
		exec     executor.Executor
		validate func(*virtualization.Info)
	}{
		{
			name:     "bare metal Mac: empty",
			fs:       fsWith(s.T(), nil),
			exec:     virtExec(s.T(), nil),
			validate: func(i *virtualization.Info) { s.Empty(i.Systems) },
		},
		{
			name: "docker host on PATH",
			fs:   fsWith(s.T(), nil),
			exec: virtExec(s.T(), map[string][]byte{
				"command -v docker": []byte("/usr/local/bin/docker\n"),
			}),
			validate: func(i *virtualization.Info) { s.Equal("host", i.Systems["docker"]) },
		},
		{
			name: "VBoxManage host on PATH",
			fs:   fsWith(s.T(), nil),
			exec: virtExec(s.T(), map[string][]byte{
				"command -v VBoxManage": []byte("/usr/local/bin/VBoxManage\n"),
			}),
			validate: func(i *virtualization.Info) { s.Equal("host", i.Systems["vbox"]) },
		},
		{
			name: "prlctl host on PATH",
			fs:   fsWith(s.T(), nil),
			exec: virtExec(s.T(), map[string][]byte{
				"command -v prlctl": []byte("/usr/local/bin/prlctl\n"),
			}),
			validate: func(i *virtualization.Info) { s.Equal("host", i.Systems["parallels"]) },
		},
		{
			name: "VMware Fusion app present",
			fs: fsWith(s.T(), map[string]string{
				"/Applications/VMware Fusion.app": "",
			}),
			exec:     virtExec(s.T(), nil),
			validate: func(i *virtualization.Info) { s.Equal("host", i.Systems["vmware"]) },
		},
		{
			name: "QEMU/Virtualization.framework guest via sysctl",
			fs:   fsWith(s.T(), nil),
			exec: virtExec(s.T(), map[string][]byte{
				"sysctl -n kern.hv_vmm_present": []byte("1\n"),
			}),
			validate: func(i *virtualization.Info) { s.Equal("guest", i.Systems["qemu"]) },
		},
		{
			name: "sysctl returns 0: no qemu detection",
			fs:   fsWith(s.T(), nil),
			exec: virtExec(s.T(), map[string][]byte{
				"sysctl -n kern.hv_vmm_present": []byte("0\n"),
			}),
			validate: func(i *virtualization.Info) { s.NotContains(i.Systems, "qemu") },
		},
		{
			name: "Parallels guest via ioreg",
			fs:   fsWith(s.T(), nil),
			exec: virtExec(s.T(), map[string][]byte{
				"ioreg -l": []byte("    | |   \"compatible\" = <\"pci1ab8,4000\">\n"),
			}),
			validate: func(i *virtualization.Info) { s.Equal("guest", i.Systems["parallels"]) },
		},
		{
			name: "VirtualBox guest via system_profiler Boot ROM",
			fs:   fsWith(s.T(), nil),
			exec: virtExec(s.T(), map[string][]byte{
				"system_profiler SPHardwareDataType": []byte(sysProfilerVBox),
			}),
			validate: func(i *virtualization.Info) { s.Equal("guest", i.Systems["vbox"]) },
		},
		{
			name: "VMware guest via system_profiler Boot ROM",
			fs:   fsWith(s.T(), nil),
			exec: virtExec(s.T(), map[string][]byte{
				"system_profiler SPHardwareDataType": []byte(sysProfilerVMware),
			}),
			validate: func(i *virtualization.Info) { s.Equal("guest", i.Systems["vmware"]) },
		},
		{
			name: "Apple VM (macOS-on-macOS) via Model Identifier",
			fs:   fsWith(s.T(), nil),
			exec: virtExec(s.T(), map[string][]byte{
				"system_profiler SPHardwareDataType": []byte(sysProfilerAppleVM),
			}),
			validate: func(i *virtualization.Info) { s.Equal("guest", i.Systems["apple"]) },
		},
		{
			name: "nil Exec: no exec detections; file-based still works",
			fs: fsWith(s.T(), map[string]string{
				"/Applications/VMware Fusion.app": "",
			}),
			exec:     nil,
			validate: func(i *virtualization.Info) { s.Equal("host", i.Systems["vmware"]) },
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			c := &virtualization.Darwin{FS: tt.fs, Exec: tt.exec}
			got, err := c.Collect(context.Background())
			s.Require().NoError(err)
			info, ok := got.(*virtualization.Info)
			s.Require().True(ok)
			tt.validate(info)
		})
	}
}
