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

	"github.com/stretchr/testify/suite"
	"golang.org/x/sys/unix"

	"github.com/osapi-io/gohai/pkg/gohai/collectors/kernel"
)

type KernelDarwinPublicTestSuite struct {
	suite.Suite
}

func TestKernelDarwinPublicTestSuite(t *testing.T) {
	suite.Run(t, new(KernelDarwinPublicTestSuite))
}

func (s *KernelDarwinPublicTestSuite) TestCollect() {
	okUname := fakeUtsname(
		"Darwin", "23.4.0",
		"Darwin Kernel Version 23.4.0: Wed Feb 21 21:44:31 PST 2024",
		"arm64",
	)
	intelUname := fakeUtsname(
		"Darwin", "22.6.0",
		"Darwin Kernel Version 22.6.0",
		"x86_64",
	)

	kextstatOut := []byte(
		`Index Refs Address            Size       Wired      Name (Version) <Linked Against>
    1    0 0xffffff7f80000000 0x8a8      0xa8       com.apple.iokit.IOPCIFamily (2.9)
    2   12 0xffffff7f80001000 0x1000     0x100      com.apple.driver.AppleACPIPlatform (6.1)
`)

	tests := []struct {
		name              string
		uname             func(*unix.Utsname) error
		sysctlOut         []byte
		sysctlErr         error
		kextOut           []byte
		kextErr           error
		exec              bool // true wires a gomock Executor; false leaves Exec nil
		wantErr           bool
		wantMachine       string
		wantProcessor     string
		wantRosetta       bool
		wantModulesCount  int
		wantModuleName    string
		wantModuleVersion string
	}{
		{
			name:              "native arm64 Apple Silicon — no rosetta, kexts parsed",
			uname:             okUname,
			sysctlOut:         []byte("0\n"),
			kextOut:           kextstatOut,
			exec:              true,
			wantMachine:       "arm64",
			wantProcessor:     "arm64",
			wantModulesCount:  2,
			wantModuleName:    "com.apple.iokit.IOPCIFamily",
			wantModuleVersion: "2.9",
		},
		{
			name:             "native Intel Mac — sysctl returns 0, machine stays x86_64",
			uname:            intelUname,
			sysctlOut:        []byte("0\n"),
			kextOut:          kextstatOut,
			exec:             true,
			wantMachine:      "x86_64",
			wantProcessor:    "x86_64",
			wantModulesCount: 2,
		},
		{
			name:             "native Intel Mac — sysctl errors, no rosetta",
			uname:            intelUname,
			sysctlErr:        errors.New("no sysctl"),
			kextOut:          kextstatOut,
			exec:             true,
			wantMachine:      "x86_64",
			wantProcessor:    "x86_64",
			wantModulesCount: 2,
		},
		{
			name:             "Rosetta on Apple Silicon — machine corrected to arm64",
			uname:            intelUname,
			sysctlOut:        []byte("1\n"),
			kextOut:          kextstatOut,
			exec:             true,
			wantMachine:      "arm64",
			wantProcessor:    "arm64",
			wantRosetta:      true,
			wantModulesCount: 2,
		},
		{
			name:          "kextstat error: modules left empty",
			uname:         okUname,
			sysctlOut:     []byte("0\n"),
			kextErr:       errors.New("not found"),
			exec:          true,
			wantMachine:   "arm64",
			wantProcessor: "arm64",
		},
		{
			name:          "kextstat unparseable line skipped",
			uname:         okUname,
			sysctlOut:     []byte("0\n"),
			kextOut:       []byte("garbage line that cannot match\n"),
			exec:          true,
			wantMachine:   "arm64",
			wantProcessor: "arm64",
		},
		{
			name:          "nil Exec: no rosetta, no modules",
			uname:         okUname,
			exec:          false,
			wantMachine:   "arm64",
			wantProcessor: "arm64",
		},
		{
			name:    "uname error propagated",
			uname:   func(*unix.Utsname) error { return errors.New("uname failed") },
			exec:    false,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			defer kernel.SetUnameSyscall(tt.uname)()

			c := &kernel.Darwin{}
			if tt.exec {
				c.Exec = darwinKernelExec(s.T(), tt.sysctlOut, tt.sysctlErr, tt.kextOut, tt.kextErr)
			}

			got, err := c.Collect(context.Background())
			if tt.wantErr {
				s.Error(err)
				return
			}
			s.Require().NoError(err)
			info, ok := got.(*kernel.Info)
			s.Require().True(ok)

			s.Equal("Darwin", info.Name)
			s.Equal(tt.wantMachine, info.Machine)
			s.Equal(tt.wantProcessor, info.Processor)
			s.Equal("Darwin", info.OS)
			s.Equal(tt.wantRosetta, info.RosettaTranslated)
			s.Len(info.Modules, tt.wantModulesCount)
			if tt.wantModuleName != "" {
				mod, ok := info.Modules[tt.wantModuleName]
				s.Require().True(ok, "expected module %s", tt.wantModuleName)
				s.Equal(tt.wantModuleVersion, mod.Version)
			}
		})
	}
}
