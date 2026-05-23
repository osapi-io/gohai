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

// Package services reports systemd service states on Linux. macOS uses
// launchd, which has a substantially different model, so Darwin returns
// nil gracefully.
package services

import (
	"bufio"
	"strings"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/platform"
)

// Service holds metadata for a single systemd service unit.
type Service struct {
	Name    string `json:"name"`
	State   string `json:"state"`
	Enabled bool   `json:"enabled"`
	Type    string `json:"type,omitempty"`
}

// Info holds the list of systemd services.
type Info struct {
	Services []Service `json:"services"`
}

// Collector is the public interface every services variant satisfies.
type Collector interface {
	collector.Collector
}

// base holds fields shared by every OS variant.
type base struct{}

// Name returns "services".
func (base) Name() string { return "services" }

// Category returns "software".
func (base) Category() string { return collector.CategorySoftware }

// DefaultEnabled returns false — full service list can be large.
func (base) DefaultEnabled() bool { return false }

// Dependencies returns no dependencies.
func (base) Dependencies() []string { return nil }

// New returns the services collector variant for the detected host OS.
func New() Collector {
	switch platform.Detect() {
	case "darwin":
		return NewDarwin()
	default:
		return NewLinux()
	}
}

// parseSystemctlOutput parses `systemctl list-units --type=service
// --all --no-pager --plain` output.
//
// Plain output columns (no header, no legend):
//
//	UNIT  LOAD  ACTIVE  SUB  DESCRIPTION
//
// A service is considered enabled when its ACTIVE state is "active".
func parseSystemctlOutput(
	output string,
) *Info {
	info := &Info{Services: []Service{}}
	sc := bufio.NewScanner(strings.NewReader(output))
	for sc.Scan() {
		line := sc.Text()
		// Skip header, blank lines, and legend lines that start with spaces
		// or contain "UNIT".
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "UNIT") {
			continue
		}
		// Skip legend/summary lines (e.g. "LOAD   = Reflects whether the unit...").
		if !strings.Contains(line, ".service") {
			continue
		}
		fields := strings.Fields(trimmed)
		if len(fields) < 4 {
			continue
		}
		name := strings.TrimSuffix(fields[0], ".service")
		// fields[1] = LOAD, fields[2] = ACTIVE, fields[3] = SUB
		active := fields[2]
		sub := fields[3]
		info.Services = append(info.Services, Service{
			Name:    name,
			State:   sub,
			Enabled: active == "active",
		})
	}
	return info
}
