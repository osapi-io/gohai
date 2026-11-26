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

// Darwin collects the launchd daemons of the system domain on macOS: `launchctl
// list` for each label, its PID and the status of its last exit, and
// `launchctl print-disabled system` for the labels that are disabled. As root,
// launchctl list gives the system domain; as a user, it gives that user's
// domain.
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
	out, err := d.Exec.Execute(ctx, "launchctl", "list")
	if err != nil {
		return &Info{Services: []Service{}}, nil
	}
	// When print-disabled fails, no daemon is reported disabled: launchd
	// runs a daemon unless it is disabled.
	disabled, _ := d.Exec.Execute(ctx, "launchctl", "print-disabled", "system")

	return parseLaunchctlList(string(out), parseDisabled(string(disabled))), nil
}

// parseLaunchctlList parses `launchctl list`:
//
//	PID	Status	Label
//	412	0	com.apple.example
//	-	78	io.osapi.app
//
// A daemon with a PID runs. One with none and a status of 0 is dead, as
// systemd says of a stopped unit, and one with another status failed.
func parseLaunchctlList(
	out string,
	disabled map[string]bool,
) *Info {
	info := &Info{Services: []Service{}}
	loaded := map[string]bool{}
	for _, line := range strings.Split(out, "\n") {
		f := strings.Fields(line)
		if len(f) != 3 || !isPID(f[0]) || !isStatus(f[1]) {
			continue
		}
		state := "running"
		if f[0] == "-" {
			state = "dead"
			if f[1] != "0" {
				state = "failed"
			}
		}
		loaded[f[2]] = true
		info.Services = append(info.Services, Service{
			Name: f[2], State: state, Enabled: !disabled[f[2]], Type: "launchd",
		})
	}
	// A daemon that launchd has not loaded is not in launchctl list. One
	// that print-disabled names is there too, dead, as systemd's list adds
	// the unit files that it has not loaded.
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

// parseDisabled parses `launchctl print-disabled system`:
//
//	disabled services = {
//		"com.apple.ftp-proxy" => disabled
//		"io.osapi.app" => enabled
//	}
func parseDisabled(
	out string,
) map[string]bool {
	disabled := map[string]bool{}
	for _, line := range strings.Split(out, "\n") {
		label, state, ok := strings.Cut(strings.TrimSpace(line), " => ")
		if ok && strings.HasPrefix(label, `"`) {
			disabled[strings.Trim(label, `"`)] = strings.TrimSpace(state) == "disabled"
		}
	}

	return disabled
}

// isPID is whether s is a process ID or -, for a daemon that does not run.
func isPID(
	s string,
) bool {
	_, err := strconv.Atoi(s)

	return s == "-" || err == nil
}

// isStatus is whether s is the status of a daemon's last exit, which is
// negative for a signal.
func isStatus(
	s string,
) bool {
	_, err := strconv.Atoi(s)

	return err == nil
}
