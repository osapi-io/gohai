//go:build linux

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
	"errors"
	"io"
	"os"
	"strings"
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

type stringReadCloser struct{ *strings.Reader }

func (stringReadCloser) Close() error { return nil }

type erroringReader struct{}

func (erroringReader) Read([]byte) (int, error) { return 0, errors.New("read fail") }

// fakeFS builds an open function that returns per-path content. A path
// set to nil is treated as not-existing; a path mapped to a sentinel
// error returns that error instead.
func fakeFS(
	contents map[string]string,
	errs map[string]error,
) func(string) (io.ReadCloser, error) {
	return func(path string) (io.ReadCloser, error) {
		if e, ok := errs[path]; ok {
			return nil, e
		}
		c, ok := contents[path]
		if !ok {
			return nil, os.ErrNotExist
		}
		return stringReadCloser{strings.NewReader(c)}, nil
	}
}

func (s *FipsLinuxPublicTestSuite) TestParseFips() {
	tests := []struct {
		name    string
		content string
		want    bool
		wantErr bool
	}{
		{"enabled", "1\n", true, false},
		{"disabled", "0\n", false, false},
		{"no trailing newline enabled", "1", true, false},
		{"empty file", "", false, false},
		{"unexpected value", "yes", false, false},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			kernel, err := fips.ParseFips(strings.NewReader(tt.content))
			if tt.wantErr {
				s.Error(err)
				return
			}
			s.Require().NoError(err)
			s.Equal(tt.want, kernel.Enabled)
		})
	}
}

func (s *FipsLinuxPublicTestSuite) TestParseFipsReadError() {
	_, err := fips.ParseFips(erroringReader{})
	s.Error(err)
}

func (s *FipsLinuxPublicTestSuite) TestParsePolicy() {
	tests := []struct {
		name              string
		content           string
		wantNil           bool
		wantName          string
		wantFIPSEffective bool
	}{
		{"FIPS policy", "FIPS\n", false, "FIPS", true},
		{"FIPS with subpolicy", "FIPS:OSPP\n", false, "FIPS:OSPP", true},
		{"DEFAULT policy", "DEFAULT\n", false, "DEFAULT", false},
		{"DEFAULT with subpolicy", "DEFAULT:NO-SHA1\n", false, "DEFAULT:NO-SHA1", false},
		{"skips comments", "# set by update-crypto-policies\nFIPS\n", false, "FIPS", true},
		{"empty file", "", true, "", false},
		{"only comments", "# comment only\n", true, "", false},
		{"blank lines before value", "\n\n  FIPS\n", false, "FIPS", true},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			p := fips.ParsePolicy(tt.content)
			if tt.wantNil {
				s.Nil(p)
				return
			}
			s.Require().NotNil(p)
			s.Equal(tt.wantName, p.Name)
			s.Equal(tt.wantFIPSEffective, p.FIPSEffective)
		})
	}
}

func (s *FipsLinuxPublicTestSuite) TestCollectFromFunc() {
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
			name:          "kernel enabled, no crypto-policies file",
			contents:      map[string]string{"/proc/sys/crypto/fips_enabled": "1\n"},
			wantEnabled:   true,
			wantPolicyNil: true,
		},
		{
			name: "kernel enabled + FIPS policy",
			contents: map[string]string{
				"/proc/sys/crypto/fips_enabled": "1\n",
				"/etc/crypto-policies/config":   "FIPS\n",
			},
			wantEnabled:       true,
			wantPolicyName:    "FIPS",
			wantFIPSEffective: true,
		},
		{
			name: "kernel enabled but policy switched to DEFAULT",
			contents: map[string]string{
				"/proc/sys/crypto/fips_enabled": "1\n",
				"/etc/crypto-policies/config":   "DEFAULT\n",
			},
			wantEnabled:       true,
			wantPolicyName:    "DEFAULT",
			wantFIPSEffective: false,
		},
		{
			name:          "kernel disabled, no policy file",
			contents:      map[string]string{"/proc/sys/crypto/fips_enabled": "0\n"},
			wantEnabled:   false,
			wantPolicyNil: true,
		},
		{
			name:          "kernel file missing treated as disabled",
			contents:      map[string]string{},
			wantEnabled:   false,
			wantPolicyNil: true,
		},
		{
			name:          "policy read error ignored",
			contents:      map[string]string{"/proc/sys/crypto/fips_enabled": "1\n"},
			errs:          map[string]error{"/etc/crypto-policies/config": errors.New("perm")},
			wantEnabled:   true,
			wantPolicyNil: true,
		},
		{
			name:     "kernel file permission denied propagated",
			contents: map[string]string{},
			errs: map[string]error{
				"/proc/sys/crypto/fips_enabled": errors.New("permission denied"),
			},
			wantErr:       true,
			wantPolicyNil: true,
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			info, err := fips.CollectFromFunc(fakeFS(tt.contents, tt.errs))
			if tt.wantErr {
				s.Error(err)
				return
			}
			s.Require().NoError(err)
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
