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

package gohai

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/pkg/gohai/collectors/cpu"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/disk"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/filesystem"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/fips"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/hostname"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/kernel"
	machineid "github.com/osapi-io/gohai/pkg/gohai/collectors/machine_id"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/memory"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/network"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/platform"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/process"
	rootgroup "github.com/osapi-io/gohai/pkg/gohai/collectors/root_group"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/shells"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/timezone"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/uptime"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/users"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/virtualization"
)

func hostnameInfoPtr() *hostname.Info     { return &hostname.Info{} }
func kernelInfoPtr() *kernel.Info         { return &kernel.Info{} }
func uptimeInfoPtr() *uptime.Info         { return &uptime.Info{} }
func virtInfoPtr() *virtualization.Info   { return &virtualization.Info{} }
func machineIDInfoPtr() *machineid.Info   { return &machineid.Info{} }
func cpuInfoPtr() *cpu.Info               { return &cpu.Info{} }
func memoryInfoPtr() *memory.Info         { return &memory.Info{} }
func filesystemInfoPtr() *filesystem.Info { return &filesystem.Info{} }
func diskInfoPtr() *disk.Info             { return &disk.Info{} }
func networkInfoPtr() *network.Info       { return &network.Info{} }
func processInfoPtr() *process.Info       { return &process.Info{} }
func usersInfoPtr() *users.Info           { return &users.Info{} }
func timezoneInfoPtr() *timezone.Info     { return &timezone.Info{} }
func rootGroupInfoPtr() *rootgroup.Info   { return &rootgroup.Info{} }
func shellsInfoPtr() *shells.Info         { return &shells.Info{} }
func fipsInfoPtr() *fips.Info             { return &fips.Info{} }

type FactsTestSuite struct {
	suite.Suite
}

func TestFactsTestSuite(t *testing.T) {
	suite.Run(t, new(FactsTestSuite))
}

func (s *FactsTestSuite) TestFlattenMap() {
	tests := []struct {
		name string
		in   map[string]any
		want map[string]any
	}{
		{
			name: "scalars",
			in:   map[string]any{"a": 1, "b": "x"},
			want: map[string]any{"a": 1, "b": "x"},
		},
		{
			name: "nested map",
			in:   map[string]any{"cpu": map[string]any{"total": 8, "model": "intel"}},
			want: map[string]any{"cpu.total": 8, "cpu.model": "intel"},
		},
		{
			name: "deep nest",
			in:   map[string]any{"a": map[string]any{"b": map[string]any{"c": 1}}},
			want: map[string]any{"a.b.c": 1},
		},
		{
			name: "empty",
			in:   map[string]any{},
			want: map[string]any{},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			got := flattenMap("", tt.in)
			s.Equal(tt.want, got)
		})
	}
}

func (s *FactsTestSuite) TestSet() {
	tests := []struct {
		name     string
		collName string
		result   any
		check    func(*Facts) bool
	}{
		{
			name:     "platform populated",
			collName: "platform",
			result:   &platform.Info{Name: "ubuntu"},
			check:    func(f *Facts) bool { return f.Platform != nil && f.Platform.Name == "ubuntu" },
		},
		{
			name:     "wrong type ignored",
			collName: "platform",
			result:   "not an Info",
			check:    func(f *Facts) bool { return f.Platform == nil },
		},
		{
			name:     "unknown name ignored",
			collName: "nope",
			result:   &platform.Info{Name: "x"},
			check:    func(f *Facts) bool { return f.Platform == nil },
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			f := &Facts{}
			f.set(tt.collName, tt.result)
			s.True(tt.check(f))
		})
	}
}

func (s *FactsTestSuite) TestCountPopulated() {
	tests := []struct {
		name  string
		facts *Facts
		want  int
	}{
		{"empty", &Facts{}, 0},
		{"platform only", &Facts{Platform: &platform.Info{}}, 1},
		{
			name: "all fields populated",
			facts: &Facts{
				Platform:       &platform.Info{},
				Hostname:       hostnameInfoPtr(),
				Kernel:         kernelInfoPtr(),
				Uptime:         uptimeInfoPtr(),
				Virtualization: virtInfoPtr(),
				MachineID:      machineIDInfoPtr(),
				CPU:            cpuInfoPtr(),
				Memory:         memoryInfoPtr(),
				Filesystem:     filesystemInfoPtr(),
				Disk:           diskInfoPtr(),
				Network:        networkInfoPtr(),
				Process:        processInfoPtr(),
				Users:          usersInfoPtr(),
				Timezone:       timezoneInfoPtr(),
				RootGroup:      rootGroupInfoPtr(),
				Shells:         shellsInfoPtr(),
				Fips:           fipsInfoPtr(),
			},
			want: 17,
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Equal(tt.want, tt.facts.countPopulated())
		})
	}
}
