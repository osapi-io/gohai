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
	"reflect"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/pkg/gohai"
)

type SpecPublicTestSuite struct {
	suite.Suite
}

func TestSpecPublicTestSuite(t *testing.T) {
	suite.Run(t, new(SpecPublicTestSuite))
}

func (s *SpecPublicTestSuite) TestCollectors() {
	specs := gohai.Collectors()
	byName := map[string]gohai.CollectorSpec{}
	for _, c := range specs {
		byName[c.Name] = c
	}

	tests := []struct {
		name         string
		validateFunc func()
	}{
		{
			name: "one spec for each registered collector, with the reads of each platform",
			validateFunc: func() {
				reg, err := gohai.NewRegistry()
				s.Require().NoError(err)
				s.Len(specs, len(reg.Names()))
				for _, c := range specs {
					s.Run(c.Name, func() {
						s.Len(c.Reads, len(gohai.SpecPlatforms()))
					})
				}
			},
		},
		{
			name: "each call returns its own deep copy of the list",
			validateFunc: func() {
				first := gohai.Collectors()
				var dep, field, read int
				for i, c := range first {
					if len(c.Dependencies) > 0 {
						dep = i
					}
					if len(c.Fields) > 0 {
						field = i
					}
					if len(c.Reads) > 0 && len(c.Reads[0].Files) > 0 &&
						len(c.Reads[0].Commands) > 0 {
						read = i
					}
				}
				name := first[0].Name
				depName := first[dep].Dependencies[0]
				fieldName := first[field].Fields[0].Name
				platform := first[read].Reads[0].Platform
				file := first[read].Reads[0].Files[0]
				command := first[read].Reads[0].Commands[0]
				first[0] = gohai.CollectorSpec{Name: "changed"}
				first[dep].Dependencies[0] = "changed"
				first[field].Fields[0].Name = "changed"
				first[read].Reads[0].Platform = "changed"
				first[read].Reads[0].Files[0] = "changed"
				first[read].Reads[0].Commands[0] = "changed"

				next := gohai.Collectors()
				s.Equal(name, next[0].Name)
				s.Equal(depName, next[dep].Dependencies[0])
				s.Equal(fieldName, next[field].Fields[0].Name)
				s.Equal(platform, next[read].Reads[0].Platform)
				s.Equal(file, next[read].Reads[0].Files[0])
				s.Equal(command, next[read].Reads[0].Commands[0])
			},
		},
		{
			name: "the list is built once, whatever the number of calls",
			validateFunc: func() {
				gohai.Collectors()
				before := gohai.SpecBuilds()
				gohai.Collectors()
				_, _ = gohai.CollectorByName("dns")
				gohai.Collectors()
				s.Equal(before, gohai.SpecBuilds())
			},
		},
		{
			name: "dns: its constructor, package, fields and reads",
			validateFunc: func() {
				dns := byName["dns"]
				s.Equal("dns.NewWith(src)", dns.Constructor)
				s.Equal("github.com/osapi-io/gohai/pkg/gohai/collectors/dns", dns.Package)
				s.Empty(dns.Network)
				s.Contains(dns.Fields, gohai.Field{Name: "nameservers", Type: "[]string"})
				s.Equal("debian", dns.Reads[0].Platform)
				s.Contains(dns.Reads[0].Files, "/etc/resolv.conf")
				s.Contains(dns.Reads[0].Commands, "resolvectl dns")
				s.Equal("darwin", dns.Reads[2].Platform)
				s.Contains(dns.Reads[2].Commands, "scutil --dns")
			},
		},
		{
			name: "ec2 reads the network, linode only the host, and alibaba depends on dmi",
			validateFunc: func() {
				ec2 := byName["ec2"]
				s.Equal("ec2.NewWith(src)", ec2.Constructor)
				s.Equal("the metadata service of ec2, over HTTP", ec2.Network)
				linode := byName["linode"]
				s.Equal("linode.NewWith(src)", linode.Constructor)
				s.Empty(linode.Network)
				s.NotEmpty(linode.Reads[0].Files)
				s.Equal([]string{"dmi"}, byName["alibaba"].Dependencies)
			},
		},
		{
			name: "a field is named by its JSON tag, else by its Go name",
			validateFunc: func() {
				type t struct {
					Tagged   string `json:"tagged,omitempty"`
					Untagged string
				}
				rt := reflect.TypeFor[t]()
				s.Equal("tagged", gohai.JSONName(rt.Field(0)))
				s.Equal("Untagged", gohai.JSONName(rt.Field(1)))
			},
		},
		{
			name: "each term of a spec has a name, what it does and when",
			validateFunc: func() {
				terms := gohai.CollectorTerms()
				s.NotEmpty(terms)
				for _, t := range terms {
					s.Run(t.Name, func() {
						s.NotEmpty(t.Name)
						s.NotEmpty(t.Does)
						s.NotEmpty(t.When)
					})
				}
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, tt.validateFunc)
	}
}

func (s *SpecPublicTestSuite) TestCollectorByName() {
	tests := []struct {
		name   string
		lookup string
		wantOK bool
	}{
		{name: "when the collector exists", lookup: "services", wantOK: true},
		{name: "when no collector has the name", lookup: "nope", wantOK: false},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			got, ok := gohai.CollectorByName(tc.lookup)
			s.Equal(tc.wantOK, ok)
			if ok {
				s.Equal(tc.lookup, got.Name)
			}
		})
	}
}
