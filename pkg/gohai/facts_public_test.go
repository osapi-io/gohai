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
	"go/ast"
	"go/parser"
	"go/token"
	"math"
	"testing"
	"time"

	"github.com/avfs/avfs/vfs/osfs"
	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/pkg/gohai"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/cpu"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/load"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/memory"
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
			OS:              "linux",
			Name:            "ubuntu",
			Version:         "24.04",
			Family:          "debian",
			CPUArchitecture: "amd64",
		},
		CPU: &cpu.Info{
			Count: 8,
			Cores: 4,
		},
		CollectTime:     time.Date(2026, 4, 11, 12, 0, 0, 0, time.UTC),
		CollectDuration: 100 * time.Millisecond,
	}
}

func (s *FactsPublicTestSuite) TestJSON() {
	tests := []struct {
		name         string
		fn           func() ([]byte, error)
		validateFunc func([]byte, error)
	}{
		{
			name: "compact",
			fn:   s.facts.JSON,
			validateFunc: func(b []byte, err error) {
				s.Require().NoError(err)

				var got map[string]any
				s.Require().NoError(json.Unmarshal(b, &got))
				s.Contains(got, "platform")
				s.Contains(got, "cpu")
				s.Contains(got, "collect_time")
			},
		},
		{
			name: "pretty",
			fn:   s.facts.PrettyJSON,
			validateFunc: func(b []byte, err error) {
				s.Require().NoError(err)

				var got map[string]any
				s.Require().NoError(json.Unmarshal(b, &got))
				s.Contains(got, "platform")
				s.Contains(got, "cpu")
				s.Contains(got, "collect_time")
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.validateFunc(tt.fn())
		})
	}
}

// TestUnmarshalStoredBlob pins the on-the-wire JSON schema. If any Info
// struct's JSON tag gets renamed or a Facts field changes without bumping
// the schema, this test fails. It catches backward-incompatible changes
// for consumers that store serialized Facts (e.g., OSAPI's agent→server
// fact handoff).
func (s *FactsPublicTestSuite) TestUnmarshalStoredBlob() {
	const storedBlob = `{
  "platform": {
    "os": "linux",
    "name": "ubuntu",
    "version": "24.04",
    "family": "debian",
    "cpu_architecture": "amd64"
  },
  "hostname": {
    "name": "web01",
    "fqdn": "web01.example.com",
    "domain": "example.com"
  },
  "kernel": {
    "name": "Linux",
    "release": "6.8.0-31-generic",
    "machine": "x86_64"
  },
  "cpu": {
    "count": 8,
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
		name         string
		validateFunc func(*gohai.Facts)
	}{
		{
			name: "platform typed fields",
			validateFunc: func(f *gohai.Facts) {
				s.Require().NotNil(f.Platform)
				s.Equal("linux", f.Platform.OS)
				s.Equal("ubuntu", f.Platform.Name)
				s.Equal("24.04", f.Platform.Version)
				s.Equal("debian", f.Platform.Family)
				s.Equal("amd64", f.Platform.CPUArchitecture)
			},
		},
		{
			name: "hostname typed fields",
			validateFunc: func(f *gohai.Facts) {
				s.Require().NotNil(f.Hostname)
				s.Equal("web01", f.Hostname.Name)
				s.Equal("web01.example.com", f.Hostname.FQDN)
				s.Equal("example.com", f.Hostname.Domain)
			},
		},
		{
			name: "kernel typed fields",
			validateFunc: func(f *gohai.Facts) {
				s.Require().NotNil(f.Kernel)
				s.Equal("Linux", f.Kernel.Name)
				s.Equal("6.8.0-31-generic", f.Kernel.Release)
				s.Equal("x86_64", f.Kernel.Machine)
			},
		},
		{
			name: "cpu typed fields",
			validateFunc: func(f *gohai.Facts) {
				s.Require().NotNil(f.CPU)
				s.Equal(8, f.CPU.Count)
				s.Equal(4, f.CPU.Cores)
				s.Equal("Intel Core i7", f.CPU.ModelName)
			},
		},
		{
			name: "virtualization typed fields",
			validateFunc: func(f *gohai.Facts) {
				s.Require().NotNil(f.Virtualization)
				s.Equal("docker", f.Virtualization.System)
				s.Equal("guest", f.Virtualization.Role)
			},
		},
		{
			name: "omitted collectors leave fields nil",
			validateFunc: func(f *gohai.Facts) {
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
			tt.validateFunc(&f)
		})
	}
}

func (s *FactsPublicTestSuite) TestFlat() {
	tests := []struct {
		name         string
		facts        *gohai.Facts
		validateFunc func(map[string]any, error)
	}{
		{
			name:  "nested fields get dot-separated keys",
			facts: s.facts,
			validateFunc: func(flat map[string]any, err error) {
				s.Require().NoError(err)
				s.Equal("ubuntu", flat["platform.name"])
				s.Equal("24.04", flat["platform.version"])
				s.Equal(json.Number("8"), flat["cpu.count"])
			},
		},
		{
			name:  "each element of a list gets its index",
			facts: &gohai.Facts{CPU: &cpu.Info{Flags: []string{"fpu", "sse2"}}},
			validateFunc: func(flat map[string]any, err error) {
				s.Require().NoError(err)
				s.Equal("fpu", flat["cpu.flags.0"])
				s.Equal("sse2", flat["cpu.flags.1"])
				s.NotContains(flat, "cpu.flags")
			},
		},
		{
			name:  "an integer above 2^53 keeps every digit",
			facts: &gohai.Facts{Memory: &memory.Info{Total: 9007199254740993}},
			validateFunc: func(flat map[string]any, err error) {
				s.Require().NoError(err)
				s.Equal(json.Number("9007199254740993"), flat["memory.total"])
			},
		},
		{
			name:  "facts that do not encode return the error",
			facts: &gohai.Facts{Load: &load.Info{One: math.NaN()}},
			validateFunc: func(flat map[string]any, err error) {
				s.Nil(flat)
				s.EqualError(err, "flatten facts: json: unsupported value: NaN")
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.validateFunc(tt.facts.Flat())
		})
	}
}

func (s *FactsPublicTestSuite) TestGet() {
	tests := []struct {
		name         string
		facts        *gohai.Facts
		path         string
		validateFunc func(any, error)
	}{
		{
			name:  "a string",
			facts: s.facts,
			path:  "platform.name",
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				s.Equal("ubuntu", got)
			},
		},
		{
			name:  "a number is a json.Number",
			facts: s.facts,
			path:  "cpu.count",
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				s.Equal(json.Number("8"), got)
			},
		},
		{
			name:  "a missing path is nil",
			facts: s.facts,
			path:  "missing",
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				s.Exactly(nil, got)
			},
		},
		{
			name:  "facts that do not encode return the error",
			facts: &gohai.Facts{Load: &load.Info{One: math.NaN()}},
			path:  "load.one",
			validateFunc: func(got any, err error) {
				s.Exactly(nil, got)
				s.EqualError(err, "flatten facts: json: unsupported value: NaN")
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.validateFunc(tt.facts.Get(tt.path))
		})
	}
}

func (s *FactsPublicTestSuite) TestString() {
	tests := []struct {
		name         string
		validateFunc func(string)
	}{
		{
			name: "names the Facts type",
			validateFunc: func(got string) {
				s.Contains(got, "Facts{")
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.validateFunc(s.facts.String())
		})
	}
}

// TestStatus checks that the statuses of gohai.Status are the statuses of
// internal/collector, which gohai.Collect converts: a status that one adds
// and the other does not would reach the facts with no constant for it.
func (s *FactsPublicTestSuite) TestStatus() {
	// constants is the name and value of each constant of type Status in the
	// Go file at path, read through avfs.
	constants := func(path string) map[string]string {
		repo := osfs.New()
		src, err := repo.ReadFile(path)
		s.Require().NoError(err)
		file, err := parser.ParseFile(token.NewFileSet(), path, src, 0)
		s.Require().NoError(err)
		out := map[string]string{}
		for _, decl := range file.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.CONST {
				continue
			}
			for _, spec := range gen.Specs {
				v, ok := spec.(*ast.ValueSpec)
				if !ok || len(v.Values) != 1 {
					continue
				}
				if t, ok := v.Type.(*ast.Ident); !ok || t.Name != "Status" {
					continue
				}
				lit, ok := v.Values[0].(*ast.BasicLit)
				s.Require().True(ok)
				out[v.Names[0].Name] = lit.Value
			}
		}

		return out
	}
	tests := []struct {
		name         string
		validateFunc func()
	}{
		{
			name: "the statuses are those of internal/collector",
			validateFunc: func() {
				public := constants("facts.go")
				internal := constants("../../internal/collector/collector.go")
				s.Len(public, 5)
				s.Equal(internal, public)
			},
		},
		{
			name: "each status has the value that its constant gives",
			validateFunc: func() {
				s.Equal(gohai.Status(collector.StatusOK), gohai.StatusOK)
				s.Equal(gohai.Status(collector.StatusAbsent), gohai.StatusAbsent)
				s.Equal(gohai.Status(collector.StatusUnsupported), gohai.StatusUnsupported)
				s.Equal(gohai.Status(collector.StatusFailed), gohai.StatusFailed)
				s.Equal(gohai.Status(collector.StatusPartial), gohai.StatusPartial)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, tt.validateFunc)
	}
}
