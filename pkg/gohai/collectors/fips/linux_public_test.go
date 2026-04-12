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
	"os"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/pkg/gohai/collectors/fips"
)

type FipsLinuxPublicTestSuite struct {
	suite.Suite
}

func TestFipsLinuxPublicTestSuite(t *testing.T) {
	suite.Run(t, new(FipsLinuxPublicTestSuite))
}

// fakeFS builds a ReadFileFn that returns canned content per path and
// canned errors per path. Missing paths return ErrNotExist (mimics
// real filesystem semantics).
func fakeFS(
	contents map[string]string,
	errs map[string]error,
) func(string) ([]byte, error) {
	return func(path string) ([]byte, error) {
		if e, ok := errs[path]; ok {
			return nil, e
		}
		if v, ok := contents[path]; ok {
			return []byte(v), nil
		}
		return nil, os.ErrNotExist
	}
}

func (s *FipsLinuxPublicTestSuite) TestCollect() {
	tests := []struct {
		name              string
		contents          map[string]string
		errs              map[string]error
		wantErr           bool
		wantEnabled       bool
		wantPolicyNil     bool
		wantPolicyName    string
		wantFIPSEffective bool
	}{
		{
			name:          "kernel enabled, no crypto-policies",
			contents:      map[string]string{"/proc/sys/crypto/fips_enabled": "1\n"},
			wantEnabled:   true,
			wantPolicyNil: true,
		},
		{
			name: "kernel enabled + FIPS policy effective",
			contents: map[string]string{
				"/proc/sys/crypto/fips_enabled": "1\n",
				"/etc/crypto-policies/config":   "FIPS\n",
			},
			wantEnabled:       true,
			wantPolicyName:    "FIPS",
			wantFIPSEffective: true,
		},
		{
			name: "kernel enabled + FIPS subpolicy",
			contents: map[string]string{
				"/proc/sys/crypto/fips_enabled": "1\n",
				"/etc/crypto-policies/config":   "FIPS:OSPP\n",
			},
			wantEnabled:       true,
			wantPolicyName:    "FIPS:OSPP",
			wantFIPSEffective: true,
		},
		{
			name: "kernel enabled, policy toggled to DEFAULT (drift)",
			contents: map[string]string{
				"/proc/sys/crypto/fips_enabled": "1\n",
				"/etc/crypto-policies/config":   "DEFAULT\n",
			},
			wantEnabled:       true,
			wantPolicyName:    "DEFAULT",
			wantFIPSEffective: false,
		},
		{
			name: "policy with comments and blanks",
			contents: map[string]string{
				"/proc/sys/crypto/fips_enabled": "1\n",
				"/etc/crypto-policies/config":   "# set by update-crypto-policies\n\nFIPS\n",
			},
			wantEnabled:       true,
			wantPolicyName:    "FIPS",
			wantFIPSEffective: true,
		},
		{
			name: "policy file comments only → no policy",
			contents: map[string]string{
				"/proc/sys/crypto/fips_enabled": "1\n",
				"/etc/crypto-policies/config":   "# comment\n",
			},
			wantEnabled:   true,
			wantPolicyNil: true,
		},
		{
			name:          "kernel disabled",
			contents:      map[string]string{"/proc/sys/crypto/fips_enabled": "0\n"},
			wantEnabled:   false,
			wantPolicyNil: true,
		},
		{
			name:          "kernel file missing → disabled",
			contents:      map[string]string{},
			wantEnabled:   false,
			wantPolicyNil: true,
		},
		{
			name:     "kernel read error propagated",
			contents: map[string]string{},
			errs: map[string]error{
				"/proc/sys/crypto/fips_enabled": errors.New("permission denied"),
			},
			wantErr:       true,
			wantPolicyNil: true,
		},
		{
			name:          "policy read error ignored (Policy omitted)",
			contents:      map[string]string{"/proc/sys/crypto/fips_enabled": "1\n"},
			errs:          map[string]error{"/etc/crypto-policies/config": errors.New("perm")},
			wantEnabled:   true,
			wantPolicyNil: true,
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			c := &fips.Linux{ReadFileFn: fakeFS(tt.contents, tt.errs)}
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
