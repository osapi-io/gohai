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
	"errors"
	"net/http"
	"sync"
	"testing"

	"github.com/avfs/avfs/vfs/memfs"
	"github.com/osapi-io/gohai/pkg/gohai/family"
	"github.com/osapi-io/gohai/pkg/gohai/source"

	"github.com/stretchr/testify/suite"

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
	"go.uber.org/mock/gomock"
)

// failingCollector is a test double that always returns a canned error
// from Collect. Used to exercise the error-path branch of
// Facts.Timings without depending on a real collector's upstream seam.
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
func (s *GohaiPublicTestSuite) TestWithSources() {
	fs := memfs.New()
	s.Require().NoError(fs.MkdirAll("/etc", 0o755))
	s.Require().NoError(fs.WriteFile("/etc/os-release",
		[]byte("ID=gohaitest\nNAME=\"Gohai Test\"\nVERSION_ID=\"1.0\"\n"), 0o644))

	g, err := gohai.New(
		gohai.WithSources(source.Sources{FS: fs, Family: family.Debian}),
		gohai.WithCollectors("os_release"),
	)
	s.Require().NoError(err)
	facts, err := g.Collect(context.Background())
	s.Require().NoError(err)
	s.Require().NotNil(facts.OSRelease)
	s.Equal("gohaitest", facts.OSRelease.ID)
	s.Equal("1.0", facts.OSRelease.VersionID)
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
			s.Nil(got, "a metadata service that answers 404 is no cloud")
			s.Equal(map[string]bool{tt.host: true}, rt.seen)
		})
	}
}

func (s *GohaiPublicTestSuite) TestCollect() {
	tests := []struct {
		name          string
		opts          []gohai.Option
		injectFailing bool
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
			tt.validateFunc(g.Collect(context.Background()))
		})
	}
}
