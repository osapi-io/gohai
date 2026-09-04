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
// A macro line is "- <name> <value...>", split on the first two spaces.
const macroFields = 3

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
	body, ok := macrosSection(splitLines(output))
	if !ok {
		return map[string]string{}
	}

	acc := &macroAccumulator{macros: map[string]string{}}
	for _, line := range body {
		acc.add(line)
	}

	acc.flush()

	return acc.macros
}

// macroAccumulator assembles macros as the lines arrive. A value can run
// over several lines, so a macro is only complete when the next one
// begins or the section ends.
type macroAccumulator struct {
	macros map[string]string
	name   string
	value  string
}

// add takes one line: either the start of a macro or a continuation of
// the one before it.
func (a *macroAccumulator) add(
	line string,
) {
	if !strings.HasPrefix(line, "-") {
		if a.name != "" {
			a.value += "\n" + line
		}

		return
	}

	a.flush()
	a.name, a.value = parseMacroLine(line)
}

// flush stores whatever macro is in hand.
func (a *macroAccumulator) flush() {
	if a.name != "" {
		a.macros[a.name] = a.value
	}
}

// macrosSection returns the lines between the two markers that bracket
// rpm --showrc's macro list.
func macrosSection(
	lines []string,
) ([]string, bool) {
	var markers []int

	for i, line := range lines {
		if macrosMarkerRE.MatchString(line) {
			markers = append(markers, i)
			if len(markers) == 2 {
				return lines[markers[0]+1 : markers[1]], true
			}
		}
	}

	return nil, false
}

// parseMacroLine reads "- <name> <value...>", splitting on the first two
// spaces as Ohai's line.split(" ", 3) does.
func parseMacroLine(
	line string,
) (name string, value string) {
	parts := strings.SplitN(line, " ", macroFields)
	if len(parts) < 2 {
		return "", ""
	}

	if len(parts) == macroFields {
		return parts[1], parts[2]
	}

	return parts[1], ""
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
