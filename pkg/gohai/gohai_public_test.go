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
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/memfs"
	"github.com/avfs/avfs/vfs/osfs"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/collector/mocks"
	"github.com/osapi-io/gohai/internal/sourcetest"
	"github.com/osapi-io/gohai/pkg/gohai"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/alibaba"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/azure"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/digitalocean"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/ec2"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/gce"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/oci"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/openstack"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/scaleway"
	"github.com/osapi-io/gohai/pkg/gohai/family"
	"github.com/osapi-io/gohai/pkg/gohai/source"
	sourcemocks "github.com/osapi-io/gohai/pkg/gohai/source/mocks"
)

// newFailingCollector returns a generated Collector mock whose Collect fails.
func newFailingCollector(
	ctrl *gomock.Controller,
	name string,
	err error,
) *mocks.MockCollector {
	m := mocks.NewMockCollector(ctrl)
	m.EXPECT().Name().Return(name).AnyTimes()
	m.EXPECT().Category().Return(collector.CategoryMisc).AnyTimes()
	m.EXPECT().DefaultEnabled().Return(false).AnyTimes()
	m.EXPECT().Dependencies().Return(nil).AnyTimes()
	m.EXPECT().
		Collect(gomock.Any(), gomock.Any()).
		Return(nil, err).
		AnyTimes()

	return m
}

// newCollectorGiving is a collector named name whose Collect gives result.
func newCollectorGiving(
	ctrl *gomock.Controller,
	name string,
	result any,
) *mocks.MockCollector {
	m := mocks.NewMockCollector(ctrl)
	m.EXPECT().Name().Return(name).AnyTimes()
	m.EXPECT().Category().Return(collector.CategoryMisc).AnyTimes()
	m.EXPECT().DefaultEnabled().Return(false).AnyTimes()
	m.EXPECT().Dependencies().Return(nil).AnyTimes()
	m.EXPECT().Collect(gomock.Any(), gomock.Any()).Return(result, nil).AnyTimes()

	return m
}

type GohaiPublicTestSuite struct {
	suite.Suite
}

func TestGohaiPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(GohaiPublicTestSuite))
}

// TestWithSources: a collector reads the host that WithSources gives, not
// the machine the test runs on.
// TestBuiltinCollectors checks that the lists which describe the built-in
// collectors agree. Each field that holds a source has the source that was
// given: a field that collector.Wire does not set reads the real host under a
// caller's fake host, and nothing else would say so. Facts has a field of each
// collector's name, which Facts.set fills, and each such field of Facts has a
// collector: a collector missing from Facts.set loses its data with no signal.
// Collectors describes each one with its fields, the schema has a property for
// each one, and each one has a page under docs/collectors.
func (s *GohaiPublicTestSuite) TestBuiltinCollectors() {
	ctrl := gomock.NewController(s.T())
	fs := memfs.New()
	exec := sourcemocks.NewMockRunner(ctrl)
	resolver := sourcemocks.NewMockResolver(ctrl)
	now := func() time.Time { return time.Unix(42, 0) }
	sources := map[reflect.Type]any{
		reflect.TypeFor[avfs.VFS]():         avfs.VFS(fs),
		reflect.TypeFor[source.Runner]():    source.Runner(exec),
		reflect.TypeFor[source.Resolver]():  source.Resolver(resolver),
		reflect.TypeFor[func() time.Time](): now,
	}
	ft := reflect.TypeFor[gohai.Facts]()
	byName := map[string]reflect.StructField{}
	for i := range ft.NumField() {
		f := ft.Field(i)
		byName[gohai.JSONName(f)] = f
	}

	// takesSources checks that each source field of c is exported and has
	// the source that was given.
	takesSources := func(c collector.Collector) {
		v := reflect.ValueOf(c)
		if v.Kind() != reflect.Pointer || v.Elem().Kind() != reflect.Struct {
			return
		}
		st := v.Elem()
		for i := range st.NumField() {
			field := st.Type().Field(i)
			want, isSource := sources[field.Type]
			if !isSource {
				continue
			}
			s.Run(field.Name, func() {
				s.Require().True(field.IsExported())
				got := st.Field(i).Interface()
				if field.Type == reflect.TypeFor[func() time.Time]() {
					s.Equal(reflect.ValueOf(want).Pointer(), reflect.ValueOf(got).Pointer())

					return
				}
				s.True(got == want)
			})
		}
	}
	// setsFacts checks that Facts has a field of the name of c, and that
	// Facts.set puts the result of c there.
	setsFacts := func(c collector.Collector) {
		field, ok := byName[c.Name()]
		s.Require().True(ok)
		s.Require().Equal(reflect.Pointer, field.Type.Kind())
		result := reflect.New(field.Type.Elem())
		facts := &gohai.Facts{}
		facts.Set(c.Name(), result.Interface())
		s.Equal(result.Pointer(), reflect.ValueOf(facts).Elem().FieldByIndex(field.Index).Pointer())
	}
	// each runs fn for each collector, as a subtest named for it.
	each := func(fn func(collector.Collector)) func([]collector.Collector) {
		return func(cs []collector.Collector) {
			for _, c := range cs {
				s.Run(c.Name(), func() {
					fn(c)
				})
			}
		}
	}
	// names is the set of names of cs.
	names := func(cs []collector.Collector) map[string]bool {
		out := map[string]bool{}
		for _, c := range cs {
			out[c.Name()] = true
		}

		return out
	}
	// repo reads the repository root, which is two levels above this package.
	repo := osfs.New()
	root := repo.Join("..", "..")

	debian := source.Sources{Family: family.Debian, FS: memfs.New()}
	tests := []struct {
		name         string
		src          source.Sources
		validateFunc func([]collector.Collector)
	}{
		{
			name: "debian: each source field has the source that was given",
			src: source.Sources{
				FS: fs, Exec: exec, Now: now, Resolver: resolver, Family: family.Debian,
			},
			validateFunc: each(takesSources),
		},
		{
			name: "darwin: each source field has the source that was given",
			src: source.Sources{
				FS: fs, Exec: exec, Now: now, Resolver: resolver, Family: family.Darwin,
			},
			validateFunc: each(takesSources),
		},
		{
			name:         "each collector has a Facts field that Facts.set fills",
			src:          debian,
			validateFunc: each(setsFacts),
		},
		{
			name: "each collector field of Facts has a collector",
			src:  debian,
			validateFunc: func(cs []collector.Collector) {
				have := names(cs)
				for name, field := range byName {
					if field.Type.Kind() != reflect.Pointer ||
						field.Type.Elem().Kind() != reflect.Struct ||
						field.Type.Elem() == reflect.TypeFor[gohai.Timings]() {
						continue
					}
					s.Run(name, func() {
						s.True(have[name])
					})
				}
			},
		},
		{
			name: "Collectors describes each collector with its fields",
			src:  debian,
			validateFunc: func(cs []collector.Collector) {
				specs := map[string]gohai.CollectorSpec{}
				for _, spec := range gohai.Collectors() {
					specs[spec.Name] = spec
				}
				s.Len(specs, len(cs))
				for _, c := range cs {
					s.Run(c.Name(), func() {
						spec, ok := specs[c.Name()]
						s.Require().True(ok)
						s.NotEmpty(spec.Fields)
					})
				}
			},
		},
		{
			name: "the schema has a property for each collector",
			src:  debian,
			validateFunc: func(cs []collector.Collector) {
				data, err := repo.ReadFile(repo.Join(root, "schemas", "gohai.schema.json"))
				s.Require().NoError(err)
				var schema struct {
					Defs struct {
						Facts struct {
							Properties map[string]json.RawMessage `json:"properties"`
						} `json:"Facts"`
					} `json:"$defs"`
				}
				s.Require().NoError(json.Unmarshal(data, &schema))
				for _, c := range cs {
					s.Run(c.Name(), func() {
						s.Contains(schema.Defs.Facts.Properties, c.Name())
					})
				}
			},
		},
		{
			name: "each collector has a page under docs/collectors",
			src:  debian,
			validateFunc: each(func(c collector.Collector) {
				_, err := repo.Stat(repo.Join(root, "docs", "collectors", c.Name()+".md"))
				s.NoError(err)
			}),
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.validateFunc(gohai.BuiltinCollectors(tt.src))
		})
	}
}

// TestRegisterBuiltins checks that a registration error of a built-in
// collector is returned, not dropped.
func (s *GohaiPublicTestSuite) TestRegisterBuiltins() {
	src := source.Sources{Family: family.Debian, FS: memfs.New()}
	tests := []struct {
		name         string
		registered   bool
		validateFunc func(error)
	}{
		{
			name: "registers each built-in collector",
			validateFunc: func(err error) {
				s.NoError(err)
			},
		},
		{
			name:       "a duplicate is an error",
			registered: true,
			validateFunc: func(err error) {
				s.EqualError(
					err,
					`register built-in collector "platform": collector "platform" already registered`,
				)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			reg := collector.NewRegistry()
			if tt.registered {
				s.Require().NoError(gohai.RegisterBuiltins(reg, src))
			}
			tt.validateFunc(gohai.RegisterBuiltins(reg, src))
		})
	}
}

func (s *GohaiPublicTestSuite) TestWithSources() {
	tests := []struct {
		name         string
		src          func() source.Sources
		validateFunc func(*gohai.Facts, error)
	}{
		{
			name: "a collector reads the filesystem it is given",
			src: func() source.Sources {
				fs := memfs.New()
				s.Require().NoError(fs.MkdirAll("/etc", 0o755))
				s.Require().NoError(fs.WriteFile("/etc/os-release",
					[]byte("ID=gohaitest\nNAME=\"Gohai Test\"\nVERSION_ID=\"1.0\"\n"), 0o644))

				return source.Sources{FS: fs, Family: family.Debian}
			},
			validateFunc: func(facts *gohai.Facts, err error) {
				s.Require().NoError(err)
				s.Require().NotNil(facts.OSRelease)
				s.Equal("gohaitest", facts.OSRelease.ID)
				s.Equal("1.0", facts.OSRelease.VersionID)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			g, err := gohai.New(
				gohai.WithSources(tt.src()),
				gohai.WithCollectors("os_release"),
			)
			s.Require().NoError(err)
			tt.validateFunc(g.Collect(context.Background()))
		})
	}
}

// hosts is a transport that records the host of each request and answers
// it with the metadata of a host that is on no cloud. http.RoundTripper is
// the standard library's interface, so it has no generated mock.
type hosts struct {
	mu   sync.Mutex
	seen map[string]bool
	next http.RoundTripper
}

func (h *hosts) RoundTrip(
	req *http.Request,
) (*http.Response, error) {
	h.mu.Lock()
	h.seen[req.URL.Hostname()] = true
	h.mu.Unlock()

	return h.next.RoundTrip(req)
}

func (s *GohaiPublicTestSuite) TestWithSourcesHTTP() {
	// A fake host with the hints of Azure (waagent) and Scaleway (its
	// kernel command line). The others probe when no DMI says otherwise.
	fs := sourcetest.FS(s.T(), map[string]string{
		"/usr/sbin/waagent": "",
		"/proc/cmdline":     "BOOT_IMAGE=/vmlinuz scaleway",
	})
	tests := []struct {
		name string
		new  func(source.Sources) collector.Collector
		host string
	}{
		{
			name: "gce",
			new:  func(src source.Sources) collector.Collector { return gce.NewWith(src) },
			host: "metadata.google.internal.",
		},
		{
			name: "ec2",
			new:  func(src source.Sources) collector.Collector { return ec2.NewWith(src) },
			host: "169.254.169.254",
		},
		{
			name: "azure",
			new:  func(src source.Sources) collector.Collector { return azure.NewWith(src) },
			host: "169.254.169.254",
		},
		{
			name: "digital_ocean",
			new:  func(src source.Sources) collector.Collector { return digitalocean.NewWith(src) },
			host: "169.254.169.254",
		},
		{
			name: "oci",
			new:  func(src source.Sources) collector.Collector { return oci.NewWith(src) },
			host: "169.254.169.254",
		},
		{
			name: "alibaba",
			new:  func(src source.Sources) collector.Collector { return alibaba.NewWith(src) },
			host: "100.100.100.200",
		},
		{
			name: "openstack",
			new:  func(src source.Sources) collector.Collector { return openstack.NewWith(src) },
			host: "169.254.169.254",
		},
		{
			name: "scaleway",
			new:  func(src source.Sources) collector.Collector { return scaleway.NewWith(src) },
			host: "169.254.42.42",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name+" reads its metadata service through src.HTTP", func() {
			rt := &hosts{seen: map[string]bool{}, next: sourcetest.HTTP(nil)}
			c := tt.new(source.Sources{FS: fs, Family: family.Debian, HTTP: rt})
			got, err := c.Collect(context.Background(), nil)
			s.Require().NoError(err)
			// A metadata service that answers 404 is no cloud.
			s.Exactly(nil, got)
			s.Equal(map[string]bool{tt.host: true}, rt.seen)
		})
	}
}

func (s *GohaiPublicTestSuite) TestCollect() {
	tests := []struct {
		name          string
		opts          []gohai.Option
		injectFailing bool
		injectAs      any
		validateFunc  func(*gohai.Facts, error)
	}{
		{
			name: "no opts collects nothing",
			opts: nil,
			validateFunc: func(facts *gohai.Facts, err error) {
				s.Require().NoError(err)
				s.Require().NotNil(facts)
				s.Nil(facts.Platform)
				s.Nil(facts.Hostname)
				s.Nil(facts.Timings)
			},
		},
		{
			name: "WithDefaults collects platform and hostname",
			opts: []gohai.Option{gohai.WithDefaults()},
			validateFunc: func(facts *gohai.Facts, err error) {
				s.Require().NoError(err)
				s.Require().NotNil(facts)
				s.NotNil(facts.Platform)
				s.NotNil(facts.Hostname)
				s.Nil(facts.Timings)
			},
		},
		{
			name: "WithDefaults + WithDisabled subtracts platform",
			opts: []gohai.Option{gohai.WithDefaults(), gohai.WithDisabled("platform")},
			validateFunc: func(facts *gohai.Facts, err error) {
				s.Require().NoError(err)
				s.Require().NotNil(facts)
				s.Nil(facts.Platform)
				s.NotNil(facts.Hostname)
				s.Nil(facts.Timings)
			},
		},
		{
			name: "WithEnabled platform without defaults still collects platform",
			opts: []gohai.Option{gohai.WithEnabled("platform")},
			validateFunc: func(facts *gohai.Facts, err error) {
				s.Require().NoError(err)
				s.Require().NotNil(facts)
				s.NotNil(facts.Platform)
				s.Nil(facts.Hostname)
				s.Nil(facts.Timings)
			},
		},
		{
			name: "only platform",
			opts: []gohai.Option{gohai.WithCollectors("platform")},
			validateFunc: func(facts *gohai.Facts, err error) {
				s.Require().NoError(err)
				s.Require().NotNil(facts)
				s.NotNil(facts.Platform)
				s.Nil(facts.Hostname)
				s.Nil(facts.Timings)
			},
		},
		{
			name: "WithTimings populates Facts.Timings per collector",
			opts: []gohai.Option{
				gohai.WithCollectors("platform"),
				gohai.WithTimings(),
			},
			validateFunc: func(facts *gohai.Facts, err error) {
				s.Require().NoError(err)
				s.Require().NotNil(facts)
				s.NotNil(facts.Platform)
				s.Nil(facts.Hostname)
				s.Require().NotNil(facts.Timings)
				s.Require().NotEmpty(facts.Timings.Collectors)
				entry, ok := facts.Timings.Collectors["platform"]
				s.Require().True(ok)
				s.Equal("ok", entry.Status)
				s.Empty(entry.Error)
				s.Greater(entry.DurationNs, int64(0))
			},
		},
		{
			name: "without WithTimings Facts.Timings is nil",
			opts: []gohai.Option{gohai.WithCollectors("platform")},
			validateFunc: func(facts *gohai.Facts, err error) {
				s.Require().NoError(err)
				s.Require().NotNil(facts)
				s.NotNil(facts.Platform)
				s.Nil(facts.Hostname)
				s.Nil(facts.Timings)
			},
		},
		{
			name: "failed collector drops from typed output, surfaces in Timings",
			opts: []gohai.Option{
				gohai.WithTimings(),
			},
			injectFailing: true,
			validateFunc: func(facts *gohai.Facts, err error) {
				s.Require().NoError(err)
				s.Require().NotNil(facts)
				s.Nil(facts.Platform)
				s.Nil(facts.Hostname)
				s.Require().NotNil(facts.Timings)
				s.Require().NotEmpty(facts.Timings.Collectors)
				entry, ok := facts.Timings.Collectors["failtest"]
				s.Require().True(ok)
				s.Equal("err", entry.Status)
				s.Contains(entry.Error, "simulated collector failure")
				s.Greater(entry.DurationNs, int64(0))
				s.Contains(facts.Errors["failtest"], "simulated collector failure")
			},
		},
		{
			name:     "a result that no Facts field holds surfaces in Errors",
			injectAs: "x",
			validateFunc: func(facts *gohai.Facts, err error) {
				s.Require().NoError(err)
				s.Require().NotNil(facts)
				s.Equal(
					map[string]string{
						"wrongtype": "gave a string, which its Facts field does not hold",
					},
					facts.Errors,
				)
			},
		},
		{
			name:          "failed collector surfaces in Errors without WithTimings",
			injectFailing: true,
			validateFunc: func(facts *gohai.Facts, err error) {
				s.Require().NoError(err)
				s.Require().NotNil(facts)
				s.Nil(facts.Timings)
				s.Equal(
					map[string]string{"failtest": "simulated collector failure"},
					facts.Errors,
				)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			g, err := gohai.New(tt.opts...)
			s.Require().NoError(err)
			if tt.injectFailing {
				s.Require().NoError(
					g.Register(newFailingCollector(
						gomock.NewController(s.T()),
						"failtest",
						errors.New("simulated collector failure"),
					)),
				)
				s.Require().NoError(g.Select("failtest"))
			}
			if tt.injectAs != nil {
				s.Require().NoError(
					g.Register(newCollectorGiving(gomock.NewController(s.T()), "wrongtype", tt.injectAs)),
				)
				s.Require().NoError(g.Select("wrongtype"))
			}
			tt.validateFunc(g.Collect(context.Background()))
		})
	}
}
