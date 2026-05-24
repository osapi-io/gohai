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
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/internal/validate"
	"github.com/osapi-io/gohai/pkg/gohai"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/cpu"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/hostname"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/kernel"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/load"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/memory"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/network"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/platform"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/timezone"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/uptime"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/virtualization"
)

type ValidatePublicTestSuite struct {
	suite.Suite
}

func TestValidatePublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ValidatePublicTestSuite))
}

func (s *ValidatePublicTestSuite) TestCompileSchema() {
	tests := []struct {
		name        string
		schemaFn    func() []byte
		resourceURL string
		wantErr     bool
	}{
		{
			name:    "valid embedded schema compiles",
			wantErr: false,
		},
		{
			name:     "invalid JSON schema fails unmarshal",
			schemaFn: func() []byte { return []byte(`{not json}`) },
			wantErr:  true,
		},
		{
			name:        "invalid resource URL fails AddResource",
			schemaFn:    func() []byte { return []byte(`{"type": "object"}`) },
			resourceURL: "://bad\x00url",
			wantErr:     true,
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			if tc.schemaFn != nil {
				restore := validate.SetSchemaJSONFn(tc.schemaFn)
				defer restore()
			}
			if tc.resourceURL != "" {
				restore := validate.SetSchemaResourceURL(tc.resourceURL)
				defer restore()
			}

			sch, err := validate.CompileSchema()

			if tc.wantErr {
				s.Error(err)
				s.Nil(sch)
			} else {
				s.NoError(err)
				s.NotNil(sch)
			}
		})
	}
}

func (s *ValidatePublicTestSuite) TestJSON() {
	tests := []struct {
		name     string
		data     []byte
		schemaFn func() []byte
		wantErr  bool
	}{
		{
			name:    "valid minimal facts",
			data:    mustMarshal(s.T(), &gohai.Facts{CollectTime: time.Now()}),
			wantErr: false,
		},
		{
			name:    "valid fully populated facts",
			data:    mustMarshal(s.T(), fullyPopulatedFacts()),
			wantErr: false,
		},
		{
			name:    "invalid JSON",
			data:    []byte(`{not json}`),
			wantErr: true,
		},
		{
			name:    "schema violation: wrong type",
			data:    []byte(`{"platform": {"os": 123}}`),
			wantErr: true,
		},
		{
			name:     "compile schema error",
			data:     []byte(`{}`),
			schemaFn: func() []byte { return []byte(`{bad}`) },
			wantErr:  true,
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			if tc.schemaFn != nil {
				restore := validate.SetSchemaJSONFn(tc.schemaFn)
				defer restore()
			}

			err := validate.JSON(tc.data)

			if tc.wantErr {
				s.Error(err)
			} else {
				s.NoError(err)
			}
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
