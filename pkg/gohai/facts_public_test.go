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

package gohai_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/pkg/gohai"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/cpu"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/platform"
)

type FactsPublicTestSuite struct {
	suite.Suite
	facts *gohai.Facts
}

func TestFactsPublicTestSuite(t *testing.T) {
	suite.Run(t, new(FactsPublicTestSuite))
}

func (s *FactsPublicTestSuite) SetupTest() {
	s.facts = &gohai.Facts{
		Platform: &platform.Info{
			OS:           "linux",
			Name:         "ubuntu",
			Version:      "24.04",
			Family:       "debian",
			Architecture: "amd64",
		},
		CPU: &cpu.Info{
			Total: 8,
			Cores: 4,
		},
		CollectTime:     time.Date(2026, 4, 11, 12, 0, 0, 0, time.UTC),
		CollectDuration: 100 * time.Millisecond,
	}
}

func (s *FactsPublicTestSuite) TestJSON() {
	tests := []struct {
		name string
		fn   func() ([]byte, error)
	}{
		{"compact", s.facts.JSON},
		{"pretty", s.facts.PrettyJSON},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			b, err := tt.fn()
			s.Require().NoError(err)
			var got map[string]any
			s.Require().NoError(json.Unmarshal(b, &got))
			s.Contains(got, "platform")
			s.Contains(got, "cpu")
			s.Contains(got, "collect_time")
		})
	}
}

// TestUnmarshalStoredBlob pins the on-the-wire JSON schema. If any Info
// struct's JSON tag gets renamed or a Facts field changes without bumping
// the schema, this test fails — catching backward-incompatible changes
// for consumers that store serialized Facts (e.g., OSAPI's agent→server
// fact handoff).
func (s *FactsPublicTestSuite) TestUnmarshalStoredBlob() {
	const storedBlob = `{
  "platform": {
    "os": "linux",
    "name": "ubuntu",
    "version": "24.04",
    "family": "debian",
    "architecture": "amd64"
  },
  "hostname": {
    "hostname": "web01",
    "fqdn": "web01.example.com",
    "domain": "example.com"
  },
  "kernel": {
    "os": "linux",
    "version": "6.8.0-31-generic",
    "arch": "x86_64"
  },
  "cpu": {
    "total": 8,
    "cores": 4,
    "model_name": "Intel Core i7"
  },
  "virtualization": {
    "system": "docker",
    "role": "guest"
  },
  "collect_time": "2026-04-12T00:00:00Z",
  "collect_duration_ns": 12345
}`

	tests := []struct {
		name  string
		check func(*gohai.Facts)
	}{
		{
			name: "platform typed fields",
			check: func(f *gohai.Facts) {
				s.Require().NotNil(f.Platform)
				s.Equal("linux", f.Platform.OS)
				s.Equal("ubuntu", f.Platform.Name)
				s.Equal("24.04", f.Platform.Version)
				s.Equal("debian", f.Platform.Family)
				s.Equal("amd64", f.Platform.Architecture)
			},
		},
		{
			name: "hostname typed fields",
			check: func(f *gohai.Facts) {
				s.Require().NotNil(f.Hostname)
				s.Equal("web01", f.Hostname.Hostname)
				s.Equal("web01.example.com", f.Hostname.FQDN)
				s.Equal("example.com", f.Hostname.Domain)
			},
		},
		{
			name: "kernel typed fields",
			check: func(f *gohai.Facts) {
				s.Require().NotNil(f.Kernel)
				s.Equal("linux", f.Kernel.OS)
				s.Equal("6.8.0-31-generic", f.Kernel.Version)
				s.Equal("x86_64", f.Kernel.Arch)
			},
		},
		{
			name: "cpu typed fields",
			check: func(f *gohai.Facts) {
				s.Require().NotNil(f.CPU)
				s.Equal(8, f.CPU.Total)
				s.Equal(4, f.CPU.Cores)
				s.Equal("Intel Core i7", f.CPU.ModelName)
			},
		},
		{
			name: "virtualization typed fields",
			check: func(f *gohai.Facts) {
				s.Require().NotNil(f.Virtualization)
				s.Equal("docker", f.Virtualization.System)
				s.Equal("guest", f.Virtualization.Role)
			},
		},
		{
			name: "omitted collectors leave fields nil",
			check: func(f *gohai.Facts) {
				s.Nil(f.Memory)
				s.Nil(f.Disk)
				s.Nil(f.Network)
				s.Nil(f.Process)
				s.Nil(f.Users)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			var f gohai.Facts
			s.Require().NoError(json.Unmarshal([]byte(storedBlob), &f))
			tt.check(&f)
		})
	}
}

func (s *FactsPublicTestSuite) TestFlat() {
	flat := s.facts.Flat()
	s.Equal("ubuntu", flat["platform.name"])
	s.Equal("24.04", flat["platform.version"])
	s.EqualValues(8, flat["cpu.total"])
}

func (s *FactsPublicTestSuite) TestGet() {
	tests := []struct {
		path string
		want any
	}{
		{"platform.name", "ubuntu"},
		{"cpu.total", float64(8)},
		{"missing", nil},
	}
	for _, tt := range tests {
		s.Run(tt.path, func() {
			s.Equal(tt.want, s.facts.Get(tt.path))
		})
	}
}

func (s *FactsPublicTestSuite) TestString() {
	s.Contains(s.facts.String(), "Facts{")
}
