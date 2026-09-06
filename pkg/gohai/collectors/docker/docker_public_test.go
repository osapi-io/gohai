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

package docker_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/executor"
	execmocks "github.com/osapi-io/gohai/internal/executor/mocks"
	"github.com/osapi-io/gohai/internal/platform"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/docker"
)

var (
	_ collector.Collector = (*docker.Linux)(nil)
	_ collector.Collector = (*docker.Darwin)(nil)
)

// canned output constants.
const (
	versionOut    = "24.0.5\n"
	containersOut = `{"ID":"abc123","Names":"/web","Image":"nginx:latest","State":"running","Status":"Up 2 hours"}
{"ID":"def456","Names":"/db","Image":"postgres:15","State":"exited","Status":"Exited (0)"}
`
	imagesOut = `{"ID":"sha256:abc","Repository":"nginx","Tag":"latest","Size":"187MB"}
{"ID":"sha256:def","Repository":"postgres","Tag":"15","Size":"379MB"}
`
)

type DockerPublicTestSuite struct {
	suite.Suite
}

func TestDockerPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(DockerPublicTestSuite))
}

// buildMock creates a mock Executor that responds to all three docker
// commands. Pass nil/error to simulate failure for individual commands.
func buildMock(
	t *testing.T,
	versionOut []byte,
	versionErr error,
	psOut []byte,
	psErr error,
	imagesOut []byte,
	imagesErr error,
) executor.Executor {
	t.Helper()
	ctrl := gomock.NewController(t)
	m := execmocks.NewMockExecutor(ctrl)
	if versionOut != nil || versionErr != nil {
		m.EXPECT().Execute(gomock.Any(), "docker", "version", "--format", "{{.Server.Version}}").
			Return(versionOut, versionErr).AnyTimes()
	}
	if psOut != nil || psErr != nil {
		m.EXPECT().Execute(gomock.Any(), "docker", "ps", "-a", "--format", "{{json .}}").
			Return(psOut, psErr).AnyTimes()
	}
	if imagesOut != nil || imagesErr != nil {
		m.EXPECT().Execute(gomock.Any(), "docker", "images", "--format", "{{json .}}").
			Return(imagesOut, imagesErr).AnyTimes()
	}
	return m
}

func (s *DockerPublicTestSuite) TestNew() {
	orig := platform.Detect
	defer func() { platform.Detect = orig }()

	tests := []struct {
		name         string
		detect       string
		validateFunc func(docker.Collector)
	}{
		{
			name:   "darwin dispatches to Darwin",
			detect: "darwin",
			validateFunc: func(c docker.Collector) {
				_, ok := c.(*docker.Darwin)
				s.True(ok)
			},
		},
		{
			name:   "debian dispatches to Linux",
			detect: "debian",
			validateFunc: func(c docker.Collector) {
				_, ok := c.(*docker.Linux)
				s.True(ok)
			},
		},
		{
			name:   "rhel dispatches to Linux",
			detect: "rhel",
			validateFunc: func(c docker.Collector) {
				_, ok := c.(*docker.Linux)
				s.True(ok)
			},
		},
		{
			name:   "arch dispatches to Linux",
			detect: "arch",
			validateFunc: func(c docker.Collector) {
				_, ok := c.(*docker.Linux)
				s.True(ok)
			},
		},
		{
			name:   "unknown dispatches to Linux",
			detect: "",
			validateFunc: func(c docker.Collector) {
				_, ok := c.(*docker.Linux)
				s.True(ok)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			platform.Detect = func() string { return tt.detect }
			c := docker.New()
			s.Equal("docker", c.Name())
			s.Equal("software", c.Category())
			s.False(c.DefaultEnabled())
			s.Empty(c.Dependencies())
			tt.validateFunc(c)
		})
	}
}

func (s *DockerPublicTestSuite) TestCollect() {
	tests := []struct {
		name         string
		variant      string
		exec         executor.Executor
		validateFunc func(any, error)
	}{
		{
			name:    "linux: docker present, full result",
			variant: "linux",
			exec: buildMock(
				s.T(),
				[]byte(versionOut), nil,
				[]byte(containersOut), nil,
				[]byte(imagesOut), nil,
			),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*docker.Info)
				s.Require().True(ok)
				s.Equal(&docker.Info{
					Version: "24.0.5",
					Containers: []docker.Container{
						{
							ID:     "abc123",
							Name:   "web",
							Image:  "nginx:latest",
							State:  "running",
							Status: "Up 2 hours",
						},
						{
							ID:     "def456",
							Name:   "db",
							Image:  "postgres:15",
							State:  "exited",
							Status: "Exited (0)",
						},
					},
					Images: []docker.Image{
						{ID: "sha256:abc", Repository: "nginx", Tag: "latest", Size: "187MB"},
						{ID: "sha256:def", Repository: "postgres", Tag: "15", Size: "379MB"},
					},
				}, info)
			},
		},
		{
			name:    "linux: docker version fails, returns nil",
			variant: "linux",
			exec:    buildMock(s.T(), nil, errors.New("no docker"), nil, nil, nil, nil),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				if true {
					s.Nil(got)
					return
				}
				info, ok := got.(*docker.Info)
				s.Require().True(ok)
				s.Equal((*docker.Info)(nil), info)
			},
		},
		{
			name:    "linux: nil Exec returns nil",
			variant: "linux",
			exec:    nil,
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				if true {
					s.Nil(got)
					return
				}
				info, ok := got.(*docker.Info)
				s.Require().True(ok)
				s.Equal((*docker.Info)(nil), info)
			},
		},
		{
			name:    "linux: ps fails, containers empty",
			variant: "linux",
			exec: buildMock(
				s.T(),
				[]byte(versionOut), nil,
				nil, errors.New("ps failed"),
				[]byte(imagesOut), nil,
			),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*docker.Info)
				s.Require().True(ok)
				s.Equal(&docker.Info{
					Version:    "24.0.5",
					Containers: []docker.Container{},
					Images: []docker.Image{
						{ID: "sha256:abc", Repository: "nginx", Tag: "latest", Size: "187MB"},
						{ID: "sha256:def", Repository: "postgres", Tag: "15", Size: "379MB"},
					},
				}, info)
			},
		},
		{
			name:    "linux: images fails, images empty",
			variant: "linux",
			exec: buildMock(
				s.T(),
				[]byte(versionOut), nil,
				[]byte(containersOut), nil,
				nil, errors.New("images failed"),
			),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*docker.Info)
				s.Require().True(ok)
				s.Equal(&docker.Info{
					Version: "24.0.5",
					Containers: []docker.Container{
						{
							ID:     "abc123",
							Name:   "web",
							Image:  "nginx:latest",
							State:  "running",
							Status: "Up 2 hours",
						},
						{
							ID:     "def456",
							Name:   "db",
							Image:  "postgres:15",
							State:  "exited",
							Status: "Exited (0)",
						},
					},
					Images: []docker.Image{},
				}, info)
			},
		},
		{
			name:    "linux: blank lines in containers output skipped",
			variant: "linux",
			exec: buildMock(
				s.T(),
				[]byte(versionOut),
				nil,
				[]byte(
					"\n"+`{"ID":"abc123","Names":"/web","Image":"nginx","State":"running","Status":"Up"}`+"\n\n",
				),
				nil,
				[]byte(""),
				nil,
			),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*docker.Info)
				s.Require().True(ok)
				s.Equal(&docker.Info{
					Version: "24.0.5",
					Containers: []docker.Container{
						{ID: "abc123", Name: "web", Image: "nginx", State: "running", Status: "Up"},
					},
					Images: nil,
				}, info)
			},
		},
		{
			name:    "linux: invalid JSON lines in containers skipped",
			variant: "linux",
			exec: buildMock(
				s.T(),
				[]byte(versionOut),
				nil,
				[]byte(
					"not-json\n"+`{"ID":"abc123","Names":"/web","Image":"nginx","State":"running","Status":"Up"}`+"\n",
				),
				nil,
				[]byte(""),
				nil,
			),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*docker.Info)
				s.Require().True(ok)
				s.Equal(&docker.Info{
					Version: "24.0.5",
					Containers: []docker.Container{
						{ID: "abc123", Name: "web", Image: "nginx", State: "running", Status: "Up"},
					},
					Images: nil,
				}, info)
			},
		},
		{
			name:    "linux: blank lines in images output skipped",
			variant: "linux",
			exec: buildMock(
				s.T(),
				[]byte(versionOut),
				nil,
				[]byte(""),
				nil,
				[]byte(
					"\n"+`{"ID":"sha256:abc","Repository":"nginx","Tag":"latest","Size":"187MB"}`+"\n\n",
				),
				nil,
			),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*docker.Info)
				s.Require().True(ok)
				s.Equal(&docker.Info{
					Version:    "24.0.5",
					Containers: nil,
					Images: []docker.Image{
						{ID: "sha256:abc", Repository: "nginx", Tag: "latest", Size: "187MB"},
					},
				}, info)
			},
		},
		{
			name:    "linux: invalid JSON lines in images skipped",
			variant: "linux",
			exec: buildMock(
				s.T(),
				[]byte(versionOut),
				nil,
				[]byte(""),
				nil,
				[]byte(
					"bad-json\n"+`{"ID":"sha256:abc","Repository":"nginx","Tag":"latest","Size":"187MB"}`+"\n",
				),
				nil,
			),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*docker.Info)
				s.Require().True(ok)
				s.Equal(&docker.Info{
					Version:    "24.0.5",
					Containers: nil,
					Images: []docker.Image{
						{ID: "sha256:abc", Repository: "nginx", Tag: "latest", Size: "187MB"},
					},
				}, info)
			},
		},
		{
			name:    "darwin: docker present, full result",
			variant: "darwin",
			exec: buildMock(
				s.T(),
				[]byte(versionOut), nil,
				[]byte(containersOut), nil,
				[]byte(imagesOut), nil,
			),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*docker.Info)
				s.Require().True(ok)
				s.Equal(&docker.Info{
					Version: "24.0.5",
					Containers: []docker.Container{
						{
							ID:     "abc123",
							Name:   "web",
							Image:  "nginx:latest",
							State:  "running",
							Status: "Up 2 hours",
						},
						{
							ID:     "def456",
							Name:   "db",
							Image:  "postgres:15",
							State:  "exited",
							Status: "Exited (0)",
						},
					},
					Images: []docker.Image{
						{ID: "sha256:abc", Repository: "nginx", Tag: "latest", Size: "187MB"},
						{ID: "sha256:def", Repository: "postgres", Tag: "15", Size: "379MB"},
					},
				}, info)
			},
		},
		{
			name:    "darwin: docker absent, returns nil",
			variant: "darwin",
			exec:    buildMock(s.T(), nil, errors.New("not found"), nil, nil, nil, nil),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				if true {
					s.Nil(got)
					return
				}
				info, ok := got.(*docker.Info)
				s.Require().True(ok)
				s.Equal((*docker.Info)(nil), info)
			},
		},
		{
			name:    "darwin: nil Exec returns nil",
			variant: "darwin",
			exec:    nil,
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				if true {
					s.Nil(got)
					return
				}
				info, ok := got.(*docker.Info)
				s.Require().True(ok)
				s.Equal((*docker.Info)(nil), info)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			var c docker.Collector
			switch tt.variant {
			case "linux":
				c = &docker.Linux{Exec: tt.exec}
			case "darwin":
				c = &docker.Darwin{Exec: tt.exec}
			}
			tt.validateFunc(c.Collect(context.Background(), nil))
		})
	}
}
