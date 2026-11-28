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
	"github.com/osapi-io/gohai/pkg/gohai/source"
)

// Service holds metadata for a single systemd service unit.
type Service struct {
	Name    string `json:"name"`
	State   string `json:"state"`
	Enabled bool   `json:"enabled"`
	// Loaded is whether the init system has the service loaded: systemd's
	// LOAD column is "loaded", or launchd's services block has the label.
	// A unit that only has a unit file, or a label that only has a launchd
	// override, is not loaded.
	Loaded bool   `json:"loaded"`
	Type   string `json:"type,omitempty"`
	// UnitFileState is the state of the unit file, as systemctl
	// list-unit-files prints it: enabled, enabled-runtime, disabled,
	// static, masked and the others. Empty when list-unit-files failed.
	UnitFileState string `json:"unit_file_state,omitempty"`
	// Description is the unit's Description=, as systemctl list-units
	// prints it.
	Description string `json:"description,omitempty"`
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
func (base) Category() collector.Category { return collector.CategorySoftware }

// DefaultEnabled returns false, because the full service list can be large.
func (base) DefaultEnabled() bool { return false }

// Dependencies returns no dependencies.
func (base) Dependencies() []string { return nil }

// New returns the services collector variant for the detected host OS.
func New() Collector {
	return NewWith(source.Sources{})
}

// NewWith returns the variant for src's family, reading the host through
// src: its filesystem, its commands and its clock.
func NewWith(
	src source.Sources,
) Collector {
	if collector.IsDarwin(src) {
		return collector.Wire(NewDarwin(), src)
	}

	return collector.Wire(NewLinux(), src)
}

// parseSystemctlOutput parses `systemctl list-units --type=service
// --all --no-pager --plain` output, then adds each unit of files that
// list-units did not give.
//
// Plain output columns (no header, no legend):
//
//	UNIT  LOAD  ACTIVE  SUB  DESCRIPTION
//
// list-units --all lists only the units that systemd has loaded. A unit
// that is stopped and disabled is usually not loaded, but it is installed,
// so it is in the list from its unit file, with the state dead and no
// description. A template, such as getty@, is not a unit that runs, and an
// alias is a second name of a unit that is in the list already, so the list
// has neither.
func parseSystemctlOutput(
	output string,
	files []unitFile,
) *Info {
	state := map[string]string{}
	for _, f := range files {
		state[f.name] = f.state
	}
	info := &Info{Services: []Service{}}
	seen := map[string]bool{}
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
		seen[name] = true
		// fields[1] = LOAD, fields[2] = ACTIVE, fields[3] = SUB, then the
		// description, which can have spaces.
		info.Services = append(info.Services, Service{
			Name:          name,
			State:         fields[3],
			Loaded:        fields[1] == "loaded",
			Enabled:       enabled(state[name]),
			UnitFileState: state[name],
			Description:   strings.Join(fields[4:], " "),
		})
	}
	for _, f := range files {
		if seen[f.name] || strings.HasSuffix(f.name, "@") || f.state == "alias" {
			continue
		}
		seen[f.name] = true
		info.Services = append(info.Services, Service{
			Name:          f.name,
			State:         "dead",
			Enabled:       enabled(f.state),
			UnitFileState: f.state,
		})
	}

	return info
}

// unitFile is one line of systemctl list-unit-files: the unit, without
// .service, and the state of its file.
type unitFile struct {
	name  string
	state string
}

// parseUnitFiles parses `systemctl list-unit-files --type=service
// --no-legend --no-pager`: each line is a unit, its state and its preset,
// in the order that systemctl prints them.
func parseUnitFiles(
	output string,
) []unitFile {
	var files []unitFile
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 || !strings.HasSuffix(fields[0], ".service") {
			continue
		}
		files = append(
			files,
			unitFile{name: strings.TrimSuffix(fields[0], ".service"), state: fields[1]},
		)
	}

	return files
}

// enabled reports whether a unit file state starts the unit at boot,
// whether or not it runs now: enabled or enabled-runtime.
func enabled(
	state string,
) bool {
	return state == "enabled" || state == "enabled-runtime"
}
