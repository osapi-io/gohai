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
	"time"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/memfs"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/osapi-io/gohai/pkg/gohai"
	"github.com/osapi-io/gohai/pkg/gohai/family"
	"github.com/osapi-io/gohai/pkg/gohai/source"
	sourcemocks "github.com/osapi-io/gohai/pkg/gohai/source/mocks"
)

type RegistryPublicTestSuite struct {
	suite.Suite
}

func TestRegistryPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(RegistryPublicTestSuite))
}

// TestBuiltinCollectorsTakeSources checks that each field of each built-in
// collector that holds a source has the source that was given. A field that
// collector.Wire does not set, such as one with another name, reads the real
// host under a caller's fake host, and nothing else would say so.
func (s *RegistryPublicTestSuite) TestBuiltinCollectorsTakeSources() {
	ctrl := gomock.NewController(s.T())
	fs := memfs.New()
	exec := sourcemocks.NewMockRunner(ctrl)
	resolver := sourcemocks.NewMockResolver(ctrl)
	now := func() time.Time { return time.Unix(42, 0) }

	for _, platform := range []string{family.Debian, family.Darwin} {
		src := source.Sources{
			FS: fs, Exec: exec, Now: now, Resolver: resolver, Family: platform,
		}
		sources := map[reflect.Type]any{
			reflect.TypeFor[avfs.VFS]():         avfs.VFS(fs),
			reflect.TypeFor[source.Runner]():    source.Runner(exec),
			reflect.TypeFor[source.Resolver]():  source.Resolver(resolver),
			reflect.TypeFor[func() time.Time](): now,
		}
		for _, c := range gohai.BuiltinCollectors(src) {
			v := reflect.ValueOf(c)
			if v.Kind() != reflect.Pointer || v.Elem().Kind() != reflect.Struct {
				continue
			}
			st := v.Elem()
			for i := range st.NumField() {
				field := st.Type().Field(i)
				name := platform + " " + c.Name() + "." + field.Name
				want, isSource := sources[field.Type]
				if !isSource {
					continue
				}
				if !s.True(field.IsExported(), name+" is not exported, so Wire cannot set it") {
					continue
				}
				got := st.Field(i).Interface()
				if field.Type == reflect.TypeFor[func() time.Time]() {
					s.Equal(reflect.ValueOf(want).Pointer(), reflect.ValueOf(got).Pointer(), name)

					continue
				}
				s.True(got == want, name)
			}
		}
	}
}

func (s *RegistryPublicTestSuite) TestNewRegistry() {
	reg := gohai.NewRegistry()
	s.Contains(reg.Names(), "platform")
}

func (s *RegistryPublicTestSuite) TestNamesInCategory() {
	reg := gohai.NewRegistry()
	tests := []struct {
		name         string
		category     string
		validateFunc func([]string)
	}{
		{
			name:     "cloud contains gce",
			category: "cloud",
			validateFunc: func(got []string) {
				s.Contains(got, "gce")
			},
		},
		{
			name:     "hardware contains dmi",
			category: "hardware",
			validateFunc: func(got []string) {
				s.Contains(got, "dmi")
			},
		},
		{
			name:     "unknown category returns empty",
			category: "nope",
			validateFunc: func(got []string) {
				s.Empty(got)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.validateFunc(reg.NamesInCategory(tt.category))
		})
	}
}

func (s *RegistryPublicTestSuite) TestCategoryOf() {
	reg := gohai.NewRegistry()
	tests := []struct {
		name         string
		collector    string
		validateFunc func(string)
	}{
		{
			name:      "gce is cloud",
			collector: "gce",
			validateFunc: func(got string) {
				s.Equal("cloud", got)
			},
		},
		{
			name:      "dmi is hardware",
			collector: "dmi",
			validateFunc: func(got string) {
				s.Equal("hardware", got)
			},
		},
		{
			name:      "missing collector returns empty",
			collector: "nope",
			validateFunc: func(got string) {
				s.Equal("", got)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.validateFunc(reg.CategoryOf(tt.collector))
		})
	}
}
