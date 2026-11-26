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

package services

import (
	"context"
	"sort"
	"strconv"
	"strings"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/executor"
)

// Darwin collects the launchd daemons of the system domain on macOS from
// `launchctl print system`: its services block has the PID, the last exit
// and the label of each daemon that launchd has loaded, and its disabled
// services block has launchd's override of each label. A user can read the
// system domain, so the result does not depend on who runs the collector:
// `launchctl list` as a user gives that user's domain.
type Darwin struct {
	base
	Exec executor.Executor
}

// NewDarwin returns a Darwin variant wired to the production Executor.
func NewDarwin() *Darwin {
	return &Darwin{Exec: executor.New()}
}

// Collect runs launchctl and parses its output. It returns an empty list,
// not an error, when launchctl cannot run.
func (d *Darwin) Collect(
	ctx context.Context,
	_ collector.PriorResults,
) (any, error) {
	if d.Exec == nil {
		return &Info{Services: []Service{}}, nil
	}
	out, err := d.Exec.Execute(ctx, "launchctl", "print", "system")
	if err != nil {
		return &Info{Services: []Service{}}, nil
	}

	return parsePrintSystem(string(out)), nil
}

// parsePrintSystem parses the two blocks of `launchctl print system`:
//
//	services = {
//		     412      - 	com.apple.example
//		       0     78 	io.osapi.app
//	}
//	disabled services = {
//		"io.osapi.app" => disabled
//	}
//
// A daemon with a PID that is not 0 runs. One with none is failed when its
// last exit is a number that is not 0, and dead otherwise, as systemd says of
// a stopped unit. A daemon is enabled unless the disabled block says
// disabled. A label of the disabled block that launchd has not loaded is in
// the list too, dead, as systemd's list adds the unit files that it has not
// loaded.
func parsePrintSystem(
	out string,
) *Info {
	disabled := parseDisabled(block(out, "disabled services"))
	info := &Info{Services: []Service{}}
	loaded := map[string]bool{}
	for _, line := range block(out, "services") {
		f := strings.Fields(line)
		if len(f) != 3 {
			continue
		}
		pid, err := strconv.Atoi(f[0])
		if err != nil {
			continue
		}
		state := "running"
		if pid == 0 {
			state = "dead"
			if code, err := strconv.Atoi(f[1]); err == nil && code != 0 {
				state = "failed"
			}
		}
		loaded[f[2]] = true
		info.Services = append(info.Services, Service{
			Name: f[2], State: state, Enabled: !disabled[f[2]], Type: "launchd",
		})
	}
	labels := make([]string, 0, len(disabled))
	for label := range disabled {
		if !loaded[label] {
			labels = append(labels, label)
		}
	}
	sort.Strings(labels)
	for _, label := range labels {
		info.Services = append(info.Services, Service{
			Name: label, State: "dead", Enabled: !disabled[label], Type: "launchd",
		})
	}

	return info
}

// block is the lines of the top-level block name of `launchctl print`, from
// its "name = {" line to its closing brace.
func block(
	out string,
	name string,
) []string {
	var lines []string
	in := false
	for _, line := range strings.Split(out, "\n") {
		switch {
		case line == "\t"+name+" = {":
			in = true
		case in && line == "\t}":
			return lines
		case in:
			lines = append(lines, line)
		}
	}

	return lines
}

// parseDisabled parses the lines of the disabled services block:
//
//	"com.apple.ftp-proxy" => disabled
//	"io.osapi.app" => enabled
func parseDisabled(
	lines []string,
) map[string]bool {
	disabled := map[string]bool{}
	for _, line := range lines {
		label, state, ok := strings.Cut(strings.TrimSpace(line), " => ")
		if ok && strings.HasPrefix(label, `"`) {
			disabled[strings.Trim(label, `"`)] = strings.TrimSpace(state) == "disabled"
		}
	}

	return disabled
}
