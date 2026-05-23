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

// Package rpm reports RPM macro definitions from `rpm --showrc`. On
// Darwin the collector returns nil gracefully. On Linux the collector
// shells out to `rpm --showrc`, locates the macro section between the
// two `===...===` marker lines, and parses each `-` prefixed macro
// definition (including multi-line continuations). The collector is
// opt-in (DefaultEnabled false) because it requires RPM to be installed
// and forks a subprocess.
package rpm

import (
	"bufio"
	"bytes"
	"regexp"
	"strings"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/platform"
)

// macrosMarkerRE matches the `===...===` separator lines that bracket
// the macros section in `rpm --showrc` output. Mirrors Ohai's
// MACROS_MARKER constant.
var macrosMarkerRE = regexp.MustCompile(`={5,}`)

// Info holds RPM configuration data.
type Info struct {
	// Macros is the map of RPM macro name → definition. Macro names
	// do not include the leading '%' — they are stored exactly as
	// `rpm --showrc` reports them (e.g. "buildroot", "__cc", "%{?_smp_mflags}").
	Macros map[string]string `json:"macros"`
}

// Collector is the public interface every rpm variant satisfies.
type Collector interface {
	collector.Collector
}

// base holds the fields every OS variant has in common.
type base struct{}

// Name returns "rpm".
func (base) Name() string { return "rpm" }

// Category returns "linux".
func (base) Category() string { return collector.CategoryLinux }

// DefaultEnabled returns false — rpm is opt-in.
func (base) DefaultEnabled() bool { return false }

// Dependencies returns no dependencies.
func (base) Dependencies() []string { return nil }

// New returns the rpm collector variant appropriate to the detected
// host OS.
func New() Collector {
	switch platform.Detect() {
	case "darwin":
		return NewDarwin()
	default:
		return NewLinux()
	}
}

// parseShowrc parses the output of `rpm --showrc` and returns the
// macro definitions. The output format has two `===...===` marker
// lines; the macro definitions live between them. Each macro starts
// with a `-` prefix line:
//
//   - %{name} <value>
//
// Continuation lines (no `-` prefix) append to the preceding macro
// with a newline separator — matches Ohai's rpm.rb parsing exactly.
func parseShowrc(
	output []byte,
) map[string]string {
	lines := splitLines(output)

	// Find the two marker lines that bracket the macros section.
	markerIdx := []int{}
	for i, line := range lines {
		if macrosMarkerRE.MatchString(line) {
			markerIdx = append(markerIdx, i)
			if len(markerIdx) == 2 {
				break
			}
		}
	}
	if len(markerIdx) < 2 {
		return map[string]string{}
	}

	macros := map[string]string{}
	var name, value string

	for _, line := range lines[markerIdx[0]+1 : markerIdx[1]] {
		if strings.HasPrefix(line, "-") {
			// Store the previous macro if any.
			if name != "" {
				macros[name] = value
			}
			// Parse new macro: "- <name> <value...>" (split on first two
			// spaces — same as Ohai's `line.split(" ", 3)` picking [1] and [2]).
			parts := strings.SplitN(line, " ", 3)
			if len(parts) < 2 {
				name = ""
				value = ""
				continue
			}
			name = parts[1]
			if len(parts) == 3 {
				value = parts[2]
			} else {
				value = ""
			}
		} else {
			// Continuation line — append to current macro.
			if name != "" {
				value += "\n" + line
			}
		}
	}
	// Store the last parsed macro.
	if name != "" {
		macros[name] = value
	}

	return macros
}

// splitLines splits b on newlines and returns the slice of lines
// without the newline terminators.
func splitLines(
	b []byte,
) []string {
	var lines []string
	sc := bufio.NewScanner(bytes.NewReader(b))
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	return lines
}
