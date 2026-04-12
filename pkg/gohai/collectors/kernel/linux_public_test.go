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

	"github.com/osapi-io/gohai/pkg/gohai/collectors/kernel"
)

type KernelLinuxPublicTestSuite struct {
	suite.Suite
}

func TestKernelLinuxPublicTestSuite(t *testing.T) {
	suite.Run(t, new(KernelLinuxPublicTestSuite))
}

func (s *KernelLinuxPublicTestSuite) TestCollect() {
	okUname := func() (string, string, string, string, error) {
		return "Linux", "5.15.0-47-generic",
			"#51-Ubuntu SMP Thu Aug 11 07:51:15 UTC 2022",
			"x86_64", nil
	}

	tests := []struct {
		name     string
		unameFn  func() (string, string, string, string, error)
		readFile func(string) ([]byte, error)
		wantErr  bool
		want     kernel.Info
	}{
		{
			name:    "canonical Linux host with modules",
			unameFn: okUname,
			readFile: func(string) ([]byte, error) {
				return []byte(
					"nf_tables 217088 25 rfkill,nf_conntrack - Live 0x0000000000000000\n" +
						"ipv6 557056 24 - Live 0x0000000000000000\n",
				), nil
			},
			want: kernel.Info{
				Name: "Linux", Release: "5.15.0-47-generic",
				Version: "#51-Ubuntu SMP Thu Aug 11 07:51:15 UTC 2022",
				Machine: "x86_64",
				Modules: map[string]kernel.Module{
					"nf_tables": {Size: 217088, RefCount: 25},
					"ipv6":      {Size: 557056, RefCount: 24},
				},
			},
		},
		{
			name:     "missing /proc/modules omits Modules field",
			unameFn:  okUname,
			readFile: func(string) ([]byte, error) { return nil, errors.New("not found") },
			want: kernel.Info{
				Name: "Linux", Release: "5.15.0-47-generic",
				Version: "#51-Ubuntu SMP Thu Aug 11 07:51:15 UTC 2022",
				Machine: "x86_64",
			},
		},
		{
			name:     "malformed module line skipped",
			unameFn:  okUname,
			readFile: func(string) ([]byte, error) { return []byte("short\nvalid_mod 1024 3 - Live 0x0\n"), nil },
			want: kernel.Info{
				Name: "Linux", Release: "5.15.0-47-generic",
				Version: "#51-Ubuntu SMP Thu Aug 11 07:51:15 UTC 2022",
				Machine: "x86_64",
				Modules: map[string]kernel.Module{
					"valid_mod": {Size: 1024, RefCount: 3},
				},
			},
		},
		{
			name:     "unparseable size/refcount leaves field zero",
			unameFn:  okUname,
			readFile: func(string) ([]byte, error) { return []byte("broken abc xyz - Live 0x0\n"), nil },
			want: kernel.Info{
				Name: "Linux", Release: "5.15.0-47-generic",
				Version: "#51-Ubuntu SMP Thu Aug 11 07:51:15 UTC 2022",
				Machine: "x86_64",
				Modules: map[string]kernel.Module{"broken": {}},
			},
		},
		{
			name: "uname error propagated",
			unameFn: func() (string, string, string, string, error) {
				return "", "", "", "", errors.New("uname failed")
			},
			readFile: func(string) ([]byte, error) { return nil, nil },
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			c := &kernel.Linux{UnameFn: tt.unameFn, ReadFileFn: tt.readFile}
			got, err := c.Collect(context.Background())
			if tt.wantErr {
				s.Error(err)
				return
			}
			s.Require().NoError(err)
			info, ok := got.(*kernel.Info)
			s.Require().True(ok)
			s.Equal(tt.want, *info)
		})
	}
}
