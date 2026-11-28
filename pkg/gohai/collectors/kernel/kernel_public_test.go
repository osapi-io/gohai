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

package kernel_test

import (
	"context"
	"errors"
	"testing"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/memfs"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/platform"
	"github.com/osapi-io/gohai/internal/sourcetest"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/kernel"
	"github.com/osapi-io/gohai/pkg/gohai/family"
	"github.com/osapi-io/gohai/pkg/gohai/source"
	"github.com/osapi-io/gohai/pkg/gohai/source/mocks"
)

var (
	_ collector.Collector = (*kernel.Linux)(nil)
	_ collector.Collector = (*kernel.Darwin)(nil)
)

type KernelPublicTestSuite struct {
	suite.Suite
}

func TestKernelPublicTestSuite(t *testing.T) {
	suite.Run(t, new(KernelPublicTestSuite))
}

// hostFS is a filesystem with each file of files.
func (s *KernelPublicTestSuite) hostFS(
	files map[string]string,
) avfs.VFS {
	fs := memfs.New()
	s.Require().NoError(fs.MkdirAll("/proc/sys/kernel", 0o755))
	for path, content := range files {
		s.Require().NoError(fs.WriteFile(path, []byte(content), 0o644))
	}

	return fs
}

// ubuntu are the /proc/sys/kernel files of an Ubuntu host.
func ubuntu() map[string]string {
	return map[string]string{
		"/proc/sys/kernel/ostype":    "Linux\n",
		"/proc/sys/kernel/osrelease": "5.15.0-47-generic\n",
		"/proc/sys/kernel/version":   "#51-Ubuntu SMP Thu Aug 11 07:51:15 UTC 2022\n",
	}
}

// answer is one command a mock executor expects, and its output.
type answer struct {
	args []string
	out  string
	err  error
}

func (s *KernelPublicTestSuite) exec(
	answers ...answer,
) source.Runner {
	m := mocks.NewMockRunner(gomock.NewController(s.T()))
	for _, a := range answers {
		rest := make([]any, 0, len(a.args)-1)
		for _, x := range a.args[1:] {
			rest = append(rest, x)
		}
		m.EXPECT().Execute(gomock.Any(), a.args[0], rest...).Return([]byte(a.out), a.err)
	}

	return m
}

// sysctlKernel is the sysctl that Darwin runs for the uname values.
func sysctlKernel(
	out string,
	err error,
) answer {
	return answer{
		args: []string{
			"sysctl",
			"-n",
			"kern.ostype",
			"kern.osrelease",
			"kern.version",
			"hw.machine",
		},
		out: out,
		err: err,
	}
}

func rosetta(
	out string,
	err error,
) answer {
	return answer{args: []string{"sysctl", "-n", "sysctl.proc_translated"}, out: out, err: err}
}

func (s *KernelPublicTestSuite) TestNew() {
	orig := platform.Detect
	defer func() { platform.Detect = orig }()

	tests := []struct {
		name      string
		detect    string
		wantLinux bool
	}{
		{name: "darwin dispatches to Darwin", detect: "darwin"},
		{name: "debian dispatches to Linux", detect: "debian", wantLinux: true},
		{name: "rhel dispatches to Linux", detect: "rhel", wantLinux: true},
		{name: "unknown dispatches to Linux", detect: "", wantLinux: true},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			platform.Detect = func() string { return tt.detect }
			c := kernel.New()
			s.Equal("kernel", c.Name())
			s.Equal("system", string(c.Category()))
			s.True(c.DefaultEnabled())
			s.Empty(c.Dependencies())
			_, linux := c.(*kernel.Linux)
			s.Equal(tt.wantLinux, linux)
		})
	}
}

// TestNewWithReadsTheGivenHost: a fake host gives the collector its
// kernel, not the kernel of the machine that runs the test.
func (s *KernelPublicTestSuite) TestNewWith() {
	tests := []struct {
		name         string
		files        func() map[string]string
		validateFunc func(any, error)
	}{
		{
			name: "reads the host that its sources give",
			files: func() map[string]string {
				files := ubuntu()
				files["/proc/sys/kernel/arch"] = "aarch64\n"

				return files
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				s.Equal(&kernel.Info{
					Name:      "Linux",
					Release:   "5.15.0-47-generic",
					Version:   "#51-Ubuntu SMP Thu Aug 11 07:51:15 UTC 2022",
					Machine:   "aarch64",
					Processor: "aarch64",
					OS:        "GNU/Linux",
				}, got)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			c := kernel.NewWith(source.Sources{FS: s.hostFS(tt.files()), Family: family.Debian})
			tt.validateFunc(c.Collect(context.Background(), nil))
		})
	}
}

func (s *KernelPublicTestSuite) TestCollect() {
	arm := "Darwin\n23.4.0\nDarwin Kernel Version 23.4.0: Wed Feb 21 21:44:31 PST 2024\narm64\n"
	intel := "Darwin\n22.6.0\nDarwin Kernel Version 22.6.0\nx86_64\n"

	tests := []struct {
		name         string
		variant      string
		files        func() map[string]string
		failRead     []string
		answers      []answer
		nilExec      bool
		validateFunc func(any, error)
	}{
		{
			name:    "when arch has the machine",
			variant: "linux",
			files: func() map[string]string {
				f := ubuntu()
				f["/proc/sys/kernel/arch"] = "x86_64\n"

				return f
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i := got.(*kernel.Info)
				s.Equal("Linux", i.Name)
				s.Equal("5.15.0-47-generic", i.Release)
				s.Equal("x86_64", i.Machine)
				s.Equal("x86_64", i.Processor)
				s.Equal("GNU/Linux", i.OS)
			},
		},
		{
			name:    "when the kernel has no arch, uname -m has the machine",
			variant: "linux",
			files:   ubuntu,
			answers: []answer{{args: []string{"uname", "-m"}, out: "x86_64\n"}},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				s.Equal("x86_64", got.(*kernel.Info).Machine)
			},
		},
		{
			name:    "when there is no version, it is empty",
			variant: "linux",
			files: func() map[string]string {
				f := ubuntu()
				delete(f, "/proc/sys/kernel/version")
				f["/proc/sys/kernel/arch"] = "x86_64"

				return f
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				s.Empty(got.(*kernel.Info).Version)
			},
		},
		{
			name:    "when the version can't be read, it is an error",
			variant: "linux",
			files: func() map[string]string {
				f := ubuntu()
				delete(f, "/proc/sys/kernel/version")
				f["/proc/sys/kernel/arch"] = "x86_64"

				return f
			},
			failRead: []string{"/proc/sys/kernel/version"},
			validateFunc: func(got any, err error) {
				s.Exactly(nil, got)
				s.ErrorContains(err, "read /proc/sys/kernel/version:")
			},
		},
		{
			name:    "when there is no ostype, it is an error",
			variant: "linux",
			files: func() map[string]string {
				f := ubuntu()
				delete(f, "/proc/sys/kernel/ostype")

				return f
			},
			validateFunc: func(_ any, err error) {
				s.ErrorContains(err, "/proc/sys/kernel/ostype")
			},
		},
		{
			name:    "when there is no osrelease, it is an error",
			variant: "linux",
			files: func() map[string]string {
				f := ubuntu()
				delete(f, "/proc/sys/kernel/osrelease")

				return f
			},
			validateFunc: func(_ any, err error) {
				s.ErrorContains(err, "/proc/sys/kernel/osrelease")
			},
		},
		{
			name:    "when the kernel has no arch and there is no Exec, the machine is empty",
			variant: "linux",
			files:   ubuntu,
			nilExec: true,
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i := got.(*kernel.Info)
				s.Equal("Linux", i.Name)
				s.Empty(i.Machine)
			},
		},
		{
			name:    "when uname -m fails, it is an error",
			variant: "linux",
			files:   ubuntu,
			answers: []answer{{args: []string{"uname", "-m"}, err: errors.New("no uname")}},
			validateFunc: func(_ any, err error) {
				s.ErrorContains(err, "uname -m")
			},
		},
		{
			name:    "when there is no Exec, there is no fact",
			variant: "darwin",
			nilExec: true,
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				s.Exactly(nil, got)
			},
		},
		{
			name:    "when it runs on Apple Silicon, with no Rosetta",
			variant: "darwin",
			answers: []answer{sysctlKernel(arm, nil)},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				s.Equal(&kernel.Info{
					Name:      "Darwin",
					Release:   "23.4.0",
					Version:   "Darwin Kernel Version 23.4.0: Wed Feb 21 21:44:31 PST 2024",
					Machine:   "arm64",
					Processor: "arm64",
					OS:        "Darwin",
				}, got)
			},
		},
		{
			name:    "when it runs on an Intel Mac, which has no sysctl.proc_translated, there is no Rosetta",
			variant: "darwin",
			answers: []answer{
				sysctlKernel(intel, nil),
				rosetta("", errors.New("unknown oid 'sysctl.proc_translated'")),
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i := got.(*kernel.Info)
				s.Equal("x86_64", i.Machine)
				s.False(i.RosettaTranslated)
			},
		},
		{
			name:    "when the Rosetta sysctl fails, there is no Rosetta",
			variant: "darwin",
			answers: []answer{sysctlKernel(intel, nil), rosetta("", errors.New("no sysctl"))},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				s.False(got.(*kernel.Info).RosettaTranslated)
			},
		},
		{
			name:    "when it runs under Rosetta, the machine is arm64",
			variant: "darwin",
			answers: []answer{sysctlKernel(intel, nil), rosetta("1\n", nil)},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i := got.(*kernel.Info)
				s.Equal("arm64", i.Machine)
				s.Equal("arm64", i.Processor)
				s.True(i.RosettaTranslated)
			},
		},
		{
			name:    "when sysctl fails, it is an error",
			variant: "darwin",
			answers: []answer{sysctlKernel("", errors.New("no sysctl"))},
			validateFunc: func(_ any, err error) {
				s.ErrorContains(err, "sysctl")
			},
		},
		{
			name:    "when sysctl gives too few values, it is an error",
			variant: "darwin",
			answers: []answer{sysctlKernel("Darwin\n23.4.0\n", nil)},
			validateFunc: func(_ any, err error) {
				s.ErrorContains(err, "not 4")
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			var c kernel.Collector
			switch tt.variant {
			case "linux":
				l := &kernel.Linux{
					FS:   sourcetest.FailRead(s.T(), s.hostFS(tt.files()), tt.failRead...),
					Exec: s.exec(tt.answers...),
				}
				if tt.nilExec {
					l.Exec = nil
				}
				c = l
			case "darwin":
				d := &kernel.Darwin{Exec: s.exec(tt.answers...)}
				if tt.nilExec {
					d.Exec = nil
				}
				c = d
			}
			tt.validateFunc(c.Collect(context.Background(), nil))
		})
	}
}
