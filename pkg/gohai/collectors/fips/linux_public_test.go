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

package fips_test

import (
	"context"
	"errors"
	"io/fs"
	"path/filepath"
	"testing"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/memfs"
	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/pkg/gohai/collectors/fips"
)

type FipsLinuxPublicTestSuite struct {
	suite.Suite
}

func TestFipsLinuxPublicTestSuite(t *testing.T) {
	suite.Run(t, new(FipsLinuxPublicTestSuite))
}

// pathErrorFS wraps a memfs and forces ReadFile to error on a
// specific path — exercises non-ErrNotExist read failures.
type pathErrorFS struct {
	avfs.VFS
	failPath string
}

func (p pathErrorFS) ReadFile(path string) ([]byte, error) {
	if path == p.failPath {
		return nil, errors.New("permission denied")
	}
	return p.VFS.ReadFile(path)
}

// newFipsFS builds a memfs with the given path→contents mapping.
func newFipsFS(
	contents map[string]string,
) avfs.VFS {
	f := memfs.New()
	for path, body := range contents {
		_ = f.MkdirAll(filepath.Dir(path), 0o755)
		_ = f.WriteFile(path, []byte(body), fs.FileMode(0o644))
	}
	return f
}

func (s *FipsLinuxPublicTestSuite) TestCollect() {
	tests := []struct {
		name              string
		setupFS           func() avfs.VFS
		wantErr           bool
		wantEnabled       bool
		wantPolicyNil     bool
		wantPolicyName    string
		wantFIPSEffective bool
	}{
		{
			name: "kernel enabled, no crypto-policies",
			setupFS: func() avfs.VFS {
				return newFipsFS(map[string]string{"/proc/sys/crypto/fips_enabled": "1\n"})
			},
			wantEnabled:   true,
			wantPolicyNil: true,
		},
		{
			name: "kernel enabled + FIPS policy effective",
			setupFS: func() avfs.VFS {
				return newFipsFS(map[string]string{
					"/proc/sys/crypto/fips_enabled": "1\n",
					"/etc/crypto-policies/config":   "FIPS\n",
				})
			},
			wantEnabled:       true,
			wantPolicyName:    "FIPS",
			wantFIPSEffective: true,
		},
		{
			name: "kernel enabled + FIPS subpolicy",
			setupFS: func() avfs.VFS {
				return newFipsFS(map[string]string{
					"/proc/sys/crypto/fips_enabled": "1\n",
					"/etc/crypto-policies/config":   "FIPS:OSPP\n",
				})
			},
			wantEnabled:       true,
			wantPolicyName:    "FIPS:OSPP",
			wantFIPSEffective: true,
		},
		{
			name: "kernel enabled, policy toggled to DEFAULT (drift)",
			setupFS: func() avfs.VFS {
				return newFipsFS(map[string]string{
					"/proc/sys/crypto/fips_enabled": "1\n",
					"/etc/crypto-policies/config":   "DEFAULT\n",
				})
			},
			wantEnabled:       true,
			wantPolicyName:    "DEFAULT",
			wantFIPSEffective: false,
		},
		{
			name: "policy with comments and blanks",
			setupFS: func() avfs.VFS {
				return newFipsFS(map[string]string{
					"/proc/sys/crypto/fips_enabled": "1\n",
					"/etc/crypto-policies/config":   "# set by update-crypto-policies\n\nFIPS\n",
				})
			},
			wantEnabled:       true,
			wantPolicyName:    "FIPS",
			wantFIPSEffective: true,
		},
		{
			name: "policy file comments only → no policy",
			setupFS: func() avfs.VFS {
				return newFipsFS(map[string]string{
					"/proc/sys/crypto/fips_enabled": "1\n",
					"/etc/crypto-policies/config":   "# comment\n",
				})
			},
			wantEnabled:   true,
			wantPolicyNil: true,
		},
		{
			name: "kernel disabled",
			setupFS: func() avfs.VFS {
				return newFipsFS(map[string]string{"/proc/sys/crypto/fips_enabled": "0\n"})
			},
			wantEnabled:   false,
			wantPolicyNil: true,
		},
		{
			name:          "kernel file missing → disabled",
			setupFS:       func() avfs.VFS { return memfs.New() },
			wantEnabled:   false,
			wantPolicyNil: true,
		},
		{
			name: "kernel read error propagated",
			setupFS: func() avfs.VFS {
				return pathErrorFS{VFS: memfs.New(), failPath: "/proc/sys/crypto/fips_enabled"}
			},
			wantErr:       true,
			wantPolicyNil: true,
		},
		{
			name: "policy read error ignored (Policy omitted)",
			setupFS: func() avfs.VFS {
				base := newFipsFS(map[string]string{"/proc/sys/crypto/fips_enabled": "1\n"})
				return pathErrorFS{VFS: base, failPath: "/etc/crypto-policies/config"}
			},
			wantEnabled:   true,
			wantPolicyNil: true,
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			c := &fips.Linux{FS: tt.setupFS()}
			got, err := c.Collect(context.Background())
			if tt.wantErr {
				s.Error(err)
				return
			}
			s.Require().NoError(err)
			info, ok := got.(*fips.Info)
			s.Require().True(ok)
			s.Equal(tt.wantEnabled, info.Kernel.Enabled)
			if tt.wantPolicyNil {
				s.Nil(info.Policy)
				return
			}
			s.Require().NotNil(info.Policy)
			s.Equal(tt.wantPolicyName, info.Policy.Name)
			s.Equal(tt.wantFIPSEffective, info.Policy.FIPSEffective)
		})
	}
}
