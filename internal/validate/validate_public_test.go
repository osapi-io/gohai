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

package validate_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/internal/readlog"
	"github.com/osapi-io/gohai/internal/validate"
	"github.com/osapi-io/gohai/pkg/gohai"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/cpu"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/docker"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/hostname"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/kernel"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/load"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/memory"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/network"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/platform"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/timezone"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/uptime"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/virtualization"
	"github.com/osapi-io/gohai/pkg/gohai/family"
	"github.com/osapi-io/gohai/schemas"
)

type ValidatePublicTestSuite struct {
	suite.Suite
}

func TestValidatePublicTestSuite(t *testing.T) {
	suite.Run(t, new(ValidatePublicTestSuite))
}

func (s *ValidatePublicTestSuite) TestCompileSchema() {
	tests := []struct {
		name         string
		schema       []byte
		validateFunc func(*jsonschema.Schema, error)
	}{
		{
			name:   "the embedded schema compiles",
			schema: schemas.SchemaJSON,
			validateFunc: func(sch *jsonschema.Schema, err error) {
				s.NoError(err)
				s.NotNil(sch)
			},
		},
		{
			name:   "a schema that is not JSON fails unmarshal",
			schema: []byte(`{not json}`),
			validateFunc: func(_ *jsonschema.Schema, err error) {
				s.ErrorContains(err, "unmarshal schema")
			},
		},
		{
			name:   "JSON that is not a valid schema fails compile",
			schema: []byte(`{"type": 5}`),
			validateFunc: func(_ *jsonschema.Schema, err error) {
				s.ErrorContains(err, "compile schema")
			},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			tc.validateFunc(validate.CompileSchema(tc.schema))
		})
	}
}

func (s *ValidatePublicTestSuite) TestJSON() {
	tests := []struct {
		name         string
		data         []byte
		validateFunc func(error)
	}{
		{
			name: "facts that match the embedded schema",
			data: mustMarshal(s.T(), &gohai.Facts{CollectTime: time.Now()}),
			validateFunc: func(err error) {
				s.NoError(err)
			},
		},
		{
			name: "facts that break the embedded schema",
			data: []byte(`{"platform": {"os": 123}}`),
			validateFunc: func(err error) {
				s.Error(err)
			},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			tc.validateFunc(validate.JSON(tc.data))
		})
	}
}

func (s *ValidatePublicTestSuite) TestJSONWith() {
	tests := []struct {
		name         string
		data         []byte
		schema       []byte
		validateFunc func(error)
	}{
		{
			name:   "valid minimal facts",
			schema: schemas.SchemaJSON,
			data:   mustMarshal(s.T(), &gohai.Facts{CollectTime: time.Now()}),
			validateFunc: func(err error) {
				s.NoError(err)
			},
		},
		{
			name:   "valid fully populated facts",
			schema: schemas.SchemaJSON,
			data:   mustMarshal(s.T(), fullyPopulatedFacts()),
			validateFunc: func(err error) {
				s.NoError(err)
			},
		},
		{
			name:   "every host collector on an empty debian host",
			schema: schemas.SchemaJSON,
			data:   emptyHostFacts(s.T(), family.Debian),
			validateFunc: func(err error) {
				s.NoError(err)
			},
		},
		{
			name:   "every host collector on an empty rhel host",
			schema: schemas.SchemaJSON,
			data:   emptyHostFacts(s.T(), family.RHEL),
			validateFunc: func(err error) {
				s.NoError(err)
			},
		},
		{
			name:   "every host collector on an empty darwin host",
			schema: schemas.SchemaJSON,
			data:   emptyHostFacts(s.T(), family.Darwin),
			validateFunc: func(err error) {
				s.NoError(err)
			},
		},
		{
			name:   "docker with no container and no image has empty lists",
			schema: schemas.SchemaJSON,
			data: mustMarshal(s.T(), &gohai.Facts{Docker: &docker.Info{
				Version:    "24.0.5",
				Containers: []docker.Container{},
				Images:     []docker.Image{},
			}}),
			validateFunc: func(err error) {
				s.NoError(err)
			},
		},
		{
			name:   "docker lists that are null break the schema",
			schema: schemas.SchemaJSON,
			data:   mustMarshal(s.T(), &gohai.Facts{Docker: &docker.Info{Version: "24.0.5"}}),
			validateFunc: func(err error) {
				s.Error(err)
			},
		},
		{
			name:   "invalid JSON",
			schema: schemas.SchemaJSON,
			data:   []byte(`{not json}`),
			validateFunc: func(err error) {
				s.Error(err)
			},
		},
		{
			name:   "schema violation: wrong type",
			schema: schemas.SchemaJSON,
			data:   []byte(`{"platform": {"os": 123}}`),
			validateFunc: func(err error) {
				s.Error(err)
			},
		},
		{
			name:   "a schema that does not compile gives the error of CompileSchema once",
			data:   []byte(`{}`),
			schema: []byte(`{"type": 5}`),
			validateFunc: func(err error) {
				s.Require().Error(err)
				s.Equal(1, strings.Count(err.Error(), "compile schema:"))
			},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			tc.validateFunc(validate.JSONWith(tc.schema, tc.data))
		})
	}
}

func fullyPopulatedFacts() *gohai.Facts {
	return &gohai.Facts{
		Platform: &platform.Info{
			OS:              "linux",
			Name:            "Ubuntu",
			Version:         "22.04",
			Family:          "debian",
			CPUArchitecture: "amd64",
			Build:           "SMP",
		},
		Hostname: &hostname.Info{
			Name:        "web-01",
			FQDN:        "web-01.example.com",
			Domain:      "example.com",
			MachineName: "web-01",
		},
		Kernel: &kernel.Info{
			Name:    "Linux",
			Release: "5.15.0-76-generic",
			Version: "SMP PREEMPT_DYNAMIC",
			Machine: "x86_64",
		},
		CPU: &cpu.Info{
			Count:           8,
			Cores:           4,
			Sockets:         1,
			ModelName:       "Intel Xeon E5-2686 v4",
			VendorID:        "GenuineIntel",
			Family:          "6",
			ModelID:         "79",
			Stepping:        1,
			Speed:           2300,
			CacheSize:       46080,
			Flags:           []string{"aes", "avx2", "sse4_2"},
			Vulnerabilities: map[string]string{"spectre_v1": "Mitigation: usercopy"},
		},
		Memory: &memory.Info{
			Total:     16384000000,
			Available: 8192000000,
			Used:      8192000000,
			Free:      4096000000,
		},
		Network: &network.Info{
			DefaultInterface: "eth0",
			DefaultGateway:   "10.0.0.1",
			Interfaces: []network.Interface{
				{
					Name:  "eth0",
					MAC:   "00:11:22:33:44:55",
					MTU:   1500,
					State: "up",
					Flags: []string{"up", "broadcast", "multicast"},
					Addresses: []network.Address{
						{Addr: "10.0.0.5", Family: "inet", Prefixlen: 24},
					},
				},
			},
		},
		Load: &load.Info{
			One: 0.5, Five: 1.0, Fifteen: 0.8,
		},
		Uptime: &uptime.Info{
			BootTime: 1700000000,
			Seconds:  86400,
		},
		Timezone: &timezone.Info{
			Name:   "America/New_York",
			Abbrev: "EST",
			Offset: -18000,
		},
		Virtualization: &virtualization.Info{
			System:  "kvm",
			Role:    "guest",
			Systems: map[string]string{"kvm": "guest"},
		},
		CollectTime:     time.Date(2026, 5, 23, 12, 0, 0, 0, time.UTC),
		CollectDuration: 100 * time.Millisecond,
	}
}

// emptyHostFacts runs every collector that reads the host, not a cloud
// metadata service, on an empty host of platform, and gives the facts as
// JSON.
func emptyHostFacts(
	t *testing.T,
	platform string,
) []byte {
	t.Helper()

	names := []string{}
	for _, c := range gohai.Collectors() {
		if c.Network == "" {
			names = append(names, c.Name)
		}
	}
	src, _ := readlog.Sources(platform)
	g, err := gohai.New(gohai.WithSources(src), gohai.WithCollectors(names...))
	if err != nil {
		t.Fatal(err)
	}
	facts, err := g.Collect(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	return mustMarshal(t, facts)
}

func mustMarshal(
	t *testing.T,
	v any,
) []byte {
	t.Helper()

	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}

	return b
}
