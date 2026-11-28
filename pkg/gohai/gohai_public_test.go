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
	"fmt"
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
	"github.com/osapi-io/gohai/internal/pattern"
	"github.com/osapi-io/gohai/internal/sourcetest"
	"github.com/osapi-io/gohai/pkg/gohai"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/platform"
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

// newCancellingCollector is a collector named name whose Collect calls
// cancel, as a deadline that passes while a run is in progress.
func newCancellingCollector(
	ctrl *gomock.Controller,
	name string,
	cancel context.CancelFunc,
) *mocks.MockCollector {
	m := mocks.NewMockCollector(ctrl)
	m.EXPECT().Name().Return(name).AnyTimes()
	m.EXPECT().Category().Return(collector.CategoryMisc).AnyTimes()
	m.EXPECT().DefaultEnabled().Return(false).AnyTimes()
	m.EXPECT().Dependencies().Return(nil).AnyTimes()
	m.EXPECT().
		Collect(gomock.Any(), gomock.Any()).
		DoAndReturn(func(context.Context, collector.PriorResults) (any, error) {
			cancel()
			return nil, nil
		}).
		AnyTimes()

	return m
}

// newCollectorReturning is a collector named name whose Collect returns out
// and err.
func newCollectorReturning(
	ctrl *gomock.Controller,
	name string,
	out any,
	err error,
) *mocks.MockCollector {
	m := mocks.NewMockCollector(ctrl)
	m.EXPECT().Name().Return(name).AnyTimes()
	m.EXPECT().Category().Return(collector.CategoryMisc).AnyTimes()
	m.EXPECT().DefaultEnabled().Return(false).AnyTimes()
	m.EXPECT().Dependencies().Return(nil).AnyTimes()
	m.EXPECT().Collect(gomock.Any(), gomock.Any()).Return(out, err).AnyTimes()

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

// newCycleCollector returns a generated Collector mock that declares deps,
// for a dependency cycle. Collect returns nothing.
func newCycleCollector(
	ctrl *gomock.Controller,
	name string,
	deps ...string,
) *mocks.MockCollector {
	m := mocks.NewMockCollector(ctrl)
	m.EXPECT().Name().Return(name).AnyTimes()
	m.EXPECT().Category().Return(collector.CategoryMisc).AnyTimes()
	m.EXPECT().DefaultEnabled().Return(true).AnyTimes()
	m.EXPECT().Dependencies().Return(deps).AnyTimes()
	m.EXPECT().
		Collect(gomock.Any(), gomock.Any()).
		Return(nil, nil).
		AnyTimes()

	return m
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
	// cloudFS is a fake host with the hints of Azure (waagent) and Scaleway
	// (its kernel command line). The other clouds probe when no DMI says
	// otherwise. cloudHTTP records the hosts that each cloud collector asks.
	cloudFS := sourcetest.FS(s.T(), map[string]string{
		"/usr/sbin/waagent": "",
		"/proc/cmdline":     "BOOT_IMAGE=/vmlinuz scaleway",
	})
	cloudHTTP := map[string]*hosts{}
	for _, name := range []string{
		"gce", "ec2", "azure", "digital_ocean", "oci", "alibaba", "openstack", "scaleway",
	} {
		cloudHTTP[name] = &hosts{seen: map[string]bool{}, next: sourcetest.HTTP(nil)}
	}
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
			name: "each pattern of each collector compiles",
			src:  debian,
			validateFunc: func([]collector.Collector) {
				s.NoError(pattern.Errors())
			},
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
		{
			name: "gce reads its metadata service through src.HTTP",
			src:  source.Sources{FS: cloudFS, Family: family.Debian, HTTP: cloudHTTP["gce"]},
			validateFunc: func(cs []collector.Collector) {
				for _, c := range cs {
					if c.Name() != "gce" {
						continue
					}
					got, err := c.Collect(context.Background(), nil)
					s.Require().NoError(err)
					// A metadata service that answers 404 is no cloud.
					s.Exactly(nil, got)
				}
				s.Equal(map[string]bool{"metadata.google.internal.": true}, cloudHTTP["gce"].seen)
			},
		},
		{
			name: "ec2 reads its metadata service through src.HTTP",
			src:  source.Sources{FS: cloudFS, Family: family.Debian, HTTP: cloudHTTP["ec2"]},
			validateFunc: func(cs []collector.Collector) {
				for _, c := range cs {
					if c.Name() != "ec2" {
						continue
					}
					got, err := c.Collect(context.Background(), nil)
					s.Require().NoError(err)
					// A metadata service that answers 404 is no cloud.
					s.Exactly(nil, got)
				}
				s.Equal(map[string]bool{"169.254.169.254": true}, cloudHTTP["ec2"].seen)
			},
		},
		{
			name: "azure reads its metadata service through src.HTTP",
			src:  source.Sources{FS: cloudFS, Family: family.Debian, HTTP: cloudHTTP["azure"]},
			validateFunc: func(cs []collector.Collector) {
				for _, c := range cs {
					if c.Name() != "azure" {
						continue
					}
					got, err := c.Collect(context.Background(), nil)
					s.Require().NoError(err)
					// A metadata service that answers 404 is no cloud.
					s.Exactly(nil, got)
				}
				s.Equal(map[string]bool{"169.254.169.254": true}, cloudHTTP["azure"].seen)
			},
		},
		{
			name: "digital_ocean reads its metadata service through src.HTTP",
			src: source.Sources{
				FS:     cloudFS,
				Family: family.Debian,
				HTTP:   cloudHTTP["digital_ocean"],
			},
			validateFunc: func(cs []collector.Collector) {
				for _, c := range cs {
					if c.Name() != "digital_ocean" {
						continue
					}
					got, err := c.Collect(context.Background(), nil)
					s.Require().NoError(err)
					// A metadata service that answers 404 is no cloud.
					s.Exactly(nil, got)
				}
				s.Equal(map[string]bool{"169.254.169.254": true}, cloudHTTP["digital_ocean"].seen)
			},
		},
		{
			name: "oci reads its metadata service through src.HTTP",
			src:  source.Sources{FS: cloudFS, Family: family.Debian, HTTP: cloudHTTP["oci"]},
			validateFunc: func(cs []collector.Collector) {
				for _, c := range cs {
					if c.Name() != "oci" {
						continue
					}
					got, err := c.Collect(context.Background(), nil)
					s.Require().NoError(err)
					// A metadata service that answers 404 is no cloud.
					s.Exactly(nil, got)
				}
				s.Equal(map[string]bool{"169.254.169.254": true}, cloudHTTP["oci"].seen)
			},
		},
		{
			name: "alibaba reads its metadata service through src.HTTP",
			src:  source.Sources{FS: cloudFS, Family: family.Debian, HTTP: cloudHTTP["alibaba"]},
			validateFunc: func(cs []collector.Collector) {
				for _, c := range cs {
					if c.Name() != "alibaba" {
						continue
					}
					got, err := c.Collect(context.Background(), nil)
					s.Require().NoError(err)
					// A metadata service that answers 404 is no cloud.
					s.Exactly(nil, got)
				}
				s.Equal(map[string]bool{"100.100.100.200": true}, cloudHTTP["alibaba"].seen)
			},
		},
		{
			name: "openstack reads its metadata service through src.HTTP",
			src:  source.Sources{FS: cloudFS, Family: family.Debian, HTTP: cloudHTTP["openstack"]},
			validateFunc: func(cs []collector.Collector) {
				for _, c := range cs {
					if c.Name() != "openstack" {
						continue
					}
					got, err := c.Collect(context.Background(), nil)
					s.Require().NoError(err)
					// A metadata service that answers 404 is no cloud.
					s.Exactly(nil, got)
				}
				s.Equal(map[string]bool{"169.254.169.254": true}, cloudHTTP["openstack"].seen)
			},
		},
		{
			name: "scaleway reads its metadata service through src.HTTP",
			src:  source.Sources{FS: cloudFS, Family: family.Debian, HTTP: cloudHTTP["scaleway"]},
			validateFunc: func(cs []collector.Collector) {
				for _, c := range cs {
					if c.Name() != "scaleway" {
						continue
					}
					got, err := c.Collect(context.Background(), nil)
					s.Require().NoError(err)
					// A metadata service that answers 404 is no cloud.
					s.Exactly(nil, got)
				}
				s.Equal(map[string]bool{"169.254.42.42": true}, cloudHTTP["scaleway"].seen)
			},
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

func (s *GohaiPublicTestSuite) TestCollect() {
	tests := []struct {
		name          string
		opts          []gohai.Option
		injectFailing bool
		injectAs      any
		injectCancel  bool
		injectCycle   bool
		inject        func(*gomock.Controller) []collector.Collector
		builtins      func(source.Sources) []collector.Collector
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
				s.Equal(gohai.StatusOK, entry.Status)
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
				s.Equal(gohai.StatusFailed, entry.Status)
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
			// init runs no command, so the cancel can't stop it: its fact is
			// one that finished. A collector that runs commands, such as
			// platform with its uname, fails on the cancel instead.
			name:         "a cancelled run returns the facts that finished with its error",
			opts:         []gohai.Option{gohai.WithCollectors("init")},
			injectCancel: true,
			validateFunc: func(facts *gohai.Facts, err error) {
				s.Require().ErrorIs(err, context.Canceled)
				s.Require().NotNil(facts)
				s.NotNil(facts.Init)
			},
		},
		{
			name:        "a run error, such as a dependency cycle, is returned",
			injectCycle: true,
			validateFunc: func(_ *gohai.Facts, err error) {
				s.ErrorContains(err, "dependency cycle")
			},
		},
		{
			name: "each collector that ran has its status",
			opts: []gohai.Option{gohai.WithCollectors("platform")},
			inject: func(ctrl *gomock.Controller) []collector.Collector {
				return []collector.Collector{
					newCollectorReturning(ctrl, "nothing", nil, nil),
					newCollectorReturning(
						ctrl,
						"elsewhere",
						nil,
						fmt.Errorf("macOS: %w", collector.ErrUnsupported),
					),
					newCollectorReturning(ctrl, "broken", nil, errors.New("read failed")),
				}
			},
			validateFunc: func(facts *gohai.Facts, err error) {
				s.Require().NoError(err)
				s.Equal(map[string]gohai.Status{
					"platform":  gohai.StatusOK,
					"nothing":   gohai.StatusAbsent,
					"elsewhere": gohai.StatusUnsupported,
					"broken":    gohai.StatusFailed,
				}, facts.Status)
				// Nothing to read on a platform is not an error.
				s.Equal(map[string]string{"broken": "read failed"}, facts.Errors)
			},
		},
		{
			name: "a partial collector keeps its facts and its error",
			opts: []gohai.Option{gohai.WithCollectors("platform"), gohai.WithTimings()},
			builtins: func(source.Sources) []collector.Collector {
				return []collector.Collector{newCollectorReturning(
					gomock.NewController(s.T()),
					"platform",
					&platform.Info{CPUArchitecture: "arm64"},
					errors.New("uname failed"),
				)}
			},
			validateFunc: func(facts *gohai.Facts, err error) {
				s.Require().NoError(err)
				s.Require().NotNil(facts.Platform)
				s.Equal("arm64", facts.Platform.CPUArchitecture)
				s.Equal(map[string]gohai.Status{"platform": gohai.StatusPartial}, facts.Status)
				s.Equal(map[string]string{"platform": "uname failed"}, facts.Errors)
				s.Equal(gohai.StatusPartial, facts.Timings.Collectors["platform"].Status)
				s.Equal("uname failed", facts.Timings.Collectors["platform"].Error)
			},
		},
		{
			name:     "a result that no Facts field holds is failed",
			injectAs: "x",
			validateFunc: func(facts *gohai.Facts, err error) {
				s.Require().NoError(err)
				s.Equal(gohai.StatusFailed, facts.Status["wrongtype"])
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
			if tt.builtins != nil {
				defer gohai.SetBuiltinCollectorsFn(tt.builtins)()
			}
			g, err := gohai.New(tt.opts...)
			s.Require().NoError(err)
			if tt.inject != nil {
				for _, c := range tt.inject(gomock.NewController(s.T())) {
					s.Require().NoError(g.Register(c))
					s.Require().NoError(g.Select(c.Name()))
				}
			}
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
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if tt.injectCancel {
				s.Require().NoError(
					g.Register(newCancellingCollector(gomock.NewController(s.T()), "canceller", cancel)),
				)
				s.Require().NoError(g.Select("canceller"))
			}
			if tt.injectCycle {
				ctrl := gomock.NewController(s.T())
				s.Require().NoError(g.Register(newCycleCollector(ctrl, "cycle-a", "cycle-b")))
				s.Require().NoError(g.Register(newCycleCollector(ctrl, "cycle-b", "cycle-a")))
				s.Require().NoError(g.Select("cycle-a"))
				s.Require().NoError(g.Select("cycle-b"))
			}
			tt.validateFunc(g.Collect(ctx))
		})
	}
}

func (s *GohaiPublicTestSuite) TestNew() {
	tests := []struct {
		name         string
		opts         []gohai.Option
		builtins     func(source.Sources) []collector.Collector
		validateFunc func(any, error)
	}{
		{
			name: "a built-in collector that does not register errors",
			builtins: func(src source.Sources) []collector.Collector {
				return []collector.Collector{platform.NewWith(src), platform.NewWith(src)}
			},
			validateFunc: func(_ any, err error) {
				s.ErrorContains(err, `register built-in collector "platform"`)
			},
		},
		{
			name: "defaults",
			opts: nil,
			validateFunc: func(g any, err error) {
				s.Require().NoError(err)
				s.NotNil(g)
			},
		},
		{
			name: "with disabled",
			opts: []gohai.Option{gohai.WithDisabled("platform")},
			validateFunc: func(g any, err error) {
				s.Require().NoError(err)
				s.NotNil(g)
			},
		},
		{
			name: "with collectors only",
			opts: []gohai.Option{gohai.WithCollectors("platform")},
			validateFunc: func(g any, err error) {
				s.Require().NoError(err)
				s.NotNil(g)
			},
		},
		{
			name: "unknown enabled errors",
			opts: []gohai.Option{gohai.WithEnabled("nope")},
			validateFunc: func(_ any, err error) {
				s.Error(err)
			},
		},
		{
			name: "unknown disabled errors",
			opts: []gohai.Option{gohai.WithDisabled("nope")},
			validateFunc: func(_ any, err error) {
				s.Error(err)
			},
		},
		{
			name: "with collectors and disabled together errors",
			opts: []gohai.Option{
				gohai.WithCollectors("platform"),
				gohai.WithDisabled("hostname"),
			},
			validateFunc: func(_ any, err error) {
				s.ErrorContains(err, "WithCollectors and WithDisabled together")
			},
		},
		{
			name: "unknown only errors",
			opts: []gohai.Option{gohai.WithCollectors("nope")},
			validateFunc: func(_ any, err error) {
				s.Error(err)
			},
		},
		{
			name: "with category cloud",
			opts: []gohai.Option{gohai.WithCategory("cloud")},
			validateFunc: func(g any, err error) {
				s.Require().NoError(err)
				s.NotNil(g)
			},
		},
		{
			name: "with category hardware stacks",
			opts: []gohai.Option{gohai.WithCategory("hardware", "cloud")},
			validateFunc: func(g any, err error) {
				s.Require().NoError(err)
				s.NotNil(g)
			},
		},
		{
			name: "unknown category errors",
			opts: []gohai.Option{gohai.WithCategory("nope")},
			validateFunc: func(_ any, err error) {
				s.Error(err)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			if tt.builtins != nil {
				defer gohai.SetBuiltinCollectorsFn(tt.builtins)()
			}
			tt.validateFunc(gohai.New(tt.opts...))
		})
	}
}
