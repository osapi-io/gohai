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
// two `===...===` marker lines, and parses each macro definition, a
// "-13: name<TAB>body" line with its continuation lines. The collector is
// opt-in (DefaultEnabled false) because it requires RPM to be installed
// and forks a subprocess.
package rpm

import (
	"bufio"
	"bytes"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/pattern"
	"github.com/osapi-io/gohai/pkg/gohai/source"
)

// macrosMarkerRE matches the `===...===` separator lines that bracket
// the macros section in `rpm --showrc` output. Mirrors Ohai's
// MACROS_MARKER constant.
var macrosMarkerRE = pattern.New(`={5,}`)

// Info holds RPM configuration data.
type Info struct {
	// Macros is the map of RPM macro name → definition. Macro names
	// do not include the leading '%'. They are stored exactly as
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
func (base) Category() collector.Category { return collector.CategoryLinux }

// DefaultEnabled returns false, because rpm is opt-in.
func (base) DefaultEnabled() bool { return false }

// Dependencies returns no dependencies.
func (base) Dependencies() []string { return nil }

// New returns the rpm collector variant appropriate to the detected
// host OS.
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

// macroLineRE matches the line that starts a macro in `rpm --showrc`: its
// nesting level, a colon and a space, the name, then a tab and the body,
// as in "-13: _builddir\t%{_topdir}/BUILD". The level is negative for
// built-in and file macros, and 0 or more for macros defined at run time.
var macroLineRE = pattern.New(`^\s*-?\d+: ([^\t]+)(?:\t(.*))?$`)

// parseShowrc parses the output of `rpm --showrc` and returns the macro
// definitions. The macros live between the two `===...===` marker lines.
// Each macro starts with a level line (see macroLineRE). The lines after it
// that do not start a macro are its body's continuation, joined with
// newlines.
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
		if m := macroLineRE.FindStringSubmatch(line); m != nil {
			if name != "" {
				macros[name] = value
			}
			name, value = m[1], m[2]
			continue
		}
		if name != "" {
			value += "\n" + line
		}
	}
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
