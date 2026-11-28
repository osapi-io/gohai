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
	"github.com/osapi-io/gohai/internal/platform"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/docker"
	"github.com/osapi-io/gohai/pkg/gohai/source"
	sourcemocks "github.com/osapi-io/gohai/pkg/gohai/source/mocks"
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
) source.Runner {
	t.Helper()
	ctrl := gomock.NewController(t)
	m := sourcemocks.NewMockRunner(ctrl)
	if versionOut != nil || versionErr != nil {
		m.EXPECT().Execute(gomock.Any(), "docker", "version", "--format", "{{.Server.Version}}").
			Return(versionOut, versionErr).AnyTimes()
	}
	if psOut != nil || psErr != nil {
		m.EXPECT().
			Execute(gomock.Any(), "docker", "ps", "-a", "--no-trunc", "--format", "{{json .}}").
			Return(psOut, psErr).
			AnyTimes()
	}
	if imagesOut != nil || imagesErr != nil {
		m.EXPECT().Execute(gomock.Any(), "docker", "images", "--format", "{{json .}}").
			Return(imagesOut, imagesErr).AnyTimes()
	}
	inspect(m, nil, errors.New("inspect unavailable"))
	return m
}

// inspect makes m answer each docker inspect with out and err.
func inspect(
	m *sourcemocks.MockRunner,
	out []byte,
	err error,
) {
	m.EXPECT().Execute(gomock.Any(), "docker", gomock.Cond(func(x any) bool {
		args, ok := x.([]string)

		return ok && len(args) > 0 && args[0] == "inspect"
	})).Return(out, err).AnyTimes()
}

// inspectOut is docker inspect of the two containers of containersOut: web
// publishes a port, mounts a volume, has a health check and holds a secret in
// its environment; db has no health check and publishes nothing.
const inspectOut = `[
  {
    "Id": "abc123",
    "Created": "2026-11-22T10:00:00.123456789Z",
    "State": {"Health": {"Status": "healthy"}},
    "Config": {"Env": ["PATH=/usr/bin", "DB_PASSWORD=hunter2", "EMPTY"]},
    "Mounts": [
      {"Type": "volume", "Source": "/var/lib/docker/volumes/data/_data", "Destination": "/data", "RW": true},
      {"Type": "bind", "Source": "/etc/nginx", "Destination": "/etc/nginx", "RW": false}
    ],
    "NetworkSettings": {
      "Ports": {
        "80/tcp": [{"HostIp": "0.0.0.0", "HostPort": "8080"}, {"HostIp": "::", "HostPort": "8080"}],
        "443/tcp": null
      },
      "Networks": {
        "frontend": {"IPAddress": "172.19.0.2", "Gateway": "172.19.0.1"},
        "bridge": {"IPAddress": "172.17.0.2", "Gateway": "172.17.0.1"}
      }
    }
  },
  {
    "Id": "def456",
    "Created": "2026-11-21T09:00:00Z",
    "State": {},
    "Config": {"Env": []},
    "Mounts": [],
    "NetworkSettings": {"Ports": {}, "Networks": {}}
  },
  {
    "Id": "not-listed",
    "Created": "2026-11-20T09:00:00Z"
  }
]`

func (s *DockerPublicTestSuite) TestInspect() {
	tests := []struct {
		name         string
		inspectOut   []byte
		inspectErr   error
		validateFunc func([]docker.Container)
	}{
		{
			name:       "when inspect answers, each container has its details in sorted order, and no env value",
			inspectOut: []byte(inspectOut),
			validateFunc: func(cs []docker.Container) {
				s.Require().Len(cs, 2)
				web := cs[0]
				s.Equal("abc123", web.ID)
				s.Equal("2026-11-22T10:00:00.123456789Z", web.Created)
				s.Equal("healthy", web.Health)
				s.Equal([]string{"PATH", "DB_PASSWORD", "EMPTY"}, web.EnvNames)
				for _, n := range web.EnvNames {
					s.NotContains(n, "hunter2")
				}
				s.Equal([]docker.Mount{
					{
						Type:        "volume",
						Source:      "/var/lib/docker/volumes/data/_data",
						Destination: "/data",
					},
					{Type: "bind", Source: "/etc/nginx", Destination: "/etc/nginx", ReadOnly: true},
				}, web.Mounts)
				s.Equal([]docker.Port{
					{ContainerPort: "443/tcp"},
					{ContainerPort: "80/tcp", HostIP: "0.0.0.0", HostPort: "8080"},
					{ContainerPort: "80/tcp", HostIP: "::", HostPort: "8080"},
				}, web.Ports)
				s.Equal([]docker.Network{
					{Name: "bridge", IPAddress: "172.17.0.2", Gateway: "172.17.0.1"},
					{Name: "frontend", IPAddress: "172.19.0.2", Gateway: "172.19.0.1"},
				}, web.Networks)

				db := cs[1]
				s.Equal("2026-11-21T09:00:00Z", db.Created)
				s.Empty(db.Health)
				s.Nil(db.Ports)
				s.Nil(db.Mounts)
				s.Nil(db.Networks)
				s.Nil(db.EnvNames)
			},
		},
		{
			name:       "when inspect has only one of the containers, the other is as ps gave it",
			inspectOut: []byte(`[{"Id": "def456", "Created": "2026-11-21T09:00:00Z"}]`),
			validateFunc: func(cs []docker.Container) {
				s.Require().Len(cs, 2)
				s.Empty(cs[0].Created)
				s.Equal("2026-11-21T09:00:00Z", cs[1].Created)
			},
		},
		{
			name:       "when inspect answers with JSON that is not a list, the containers are as ps gave them",
			inspectOut: []byte(`{"not": "a list"}`),
			validateFunc: func(cs []docker.Container) {
				s.Require().Len(cs, 2)
				s.Empty(cs[0].Created)
			},
		},
		{
			name:       "when inspect fails for a container that went away, the others have their details",
			inspectOut: []byte(`[{"Id": "def456", "Created": "2026-11-21T09:00:00Z"}]`),
			inspectErr: errors.New("exit status 1: no such object: abc123"),
			validateFunc: func(cs []docker.Container) {
				s.Require().Len(cs, 2)
				s.Empty(cs[0].Created)
				s.Equal("2026-11-21T09:00:00Z", cs[1].Created)
			},
		},
		{
			name:       "when inspect fails, the containers are as ps gave them",
			inspectErr: errors.New("daemon gone"),
			validateFunc: func(cs []docker.Container) {
				s.Require().Len(cs, 2)
				s.Empty(cs[0].Created)
				s.Equal("web", cs[0].Name)
			},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			ctrl := gomock.NewController(s.T())
			m := sourcemocks.NewMockRunner(ctrl)
			m.EXPECT().
				Execute(gomock.Any(), "docker", "version", "--format", "{{.Server.Version}}").
				Return([]byte(versionOut), nil)
			m.EXPECT().
				Execute(gomock.Any(), "docker", "ps", "-a", "--no-trunc", "--format", "{{json .}}").
				Return([]byte(containersOut), nil)
			m.EXPECT().Execute(gomock.Any(), "docker", "images", "--format", "{{json .}}").
				Return([]byte(imagesOut), nil)
			inspect(m, tc.inspectOut, tc.inspectErr)

			got, err := (&docker.Linux{Exec: m}).Collect(context.Background(), nil)
			s.Require().NoError(err)
			tc.validateFunc(got.(*docker.Info).Containers)
		})
	}
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
			s.Equal("software", string(c.Category()))
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
		exec         source.Runner
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
			name:    "linux: no containers and no images give empty lists that the schema takes",
			variant: "linux",
			exec: buildMock(
				s.T(),
				[]byte(versionOut), nil,
				[]byte(""), nil,
				[]byte(""), nil,
			),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*docker.Info)
				s.Require().True(ok)
				s.Equal([]docker.Container{}, info.Containers)
				s.Equal([]docker.Image{}, info.Images)
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
					Images: []docker.Image{},
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
					Images: []docker.Image{},
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
					Containers: []docker.Container{},
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
					Containers: []docker.Container{},
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
