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

// Package command provides Ohai command/ps parity: runs `ps -ef` and
// stores its output lines. Ohai's command plugin simply stores the
// ps command string; gohai goes one step further and runs it so
// consumers get the actual process listing. Both Linux and Darwin
// support `ps -ef`. DefaultEnabled is false, because the output is large and most
// consumers don't need it.
//
// Each line keeps its columns and the program of its CMD column, and
// drops the program's arguments. An argument can hold a password or a
// token, and the facts are stored, sent and shown.
package command

import (
	"strings"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/pkg/gohai/source"
)

// Info holds the ps output lines without the arguments of each command.
type Info struct {
	// PS contains the lines from `ps -ef` including the header, with
	// the arguments of each command removed.
	PS []string `json:"ps"`
}

// Collector is the public interface every command variant satisfies.
type Collector interface {
	collector.Collector
}

type base struct{}

func (base) Name() string                 { return "command" }
func (base) Category() collector.Category { return collector.CategoryMisc }
func (base) DefaultEnabled() bool         { return false }
func (base) Dependencies() []string       { return nil }

// New returns the command variant for the host OS.
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

// parsePSOutput splits ps output into its lines, trims trailing
// whitespace, skips blank lines and removes the arguments of each
// command. Shared by both Linux and Darwin. The first line is the
// header, and its CMD or COMMAND column tells where each command
// starts, so the layout of `ps -ef` on GNU, BSD and BusyBox all work.
// Output with no such header gives no lines, so an unknown layout can
// never leak an argument.
func parsePSOutput(
	output string,
) *Info {
	info := &Info{PS: []string{}}
	cmd := -1
	// Split, not bufio.Scanner: ps -ef prints a whole command line when
	// stdout is not a tty, and a line can pass the Scanner's 64 KiB limit.
	for _, raw := range strings.Split(output, "\n") {
		line := strings.TrimRight(raw, " \t\r")
		if line == "" {
			continue
		}
		if cmd < 0 {
			cmd = cmdColumn(line)
			if cmd < 0 {
				return info
			}
			info.PS = append(info.PS, line)
			continue
		}
		info.PS = append(info.PS, withoutArgs(line, cmd))
	}
	return info
}

// cmdColumn returns the number of columns before the CMD or COMMAND
// column of a ps header, or -1 when the header has neither.
func cmdColumn(
	header string,
) int {
	for i, name := range strings.Fields(header) {
		if name == "CMD" || name == "COMMAND" {
			return i
		}
	}
	return -1
}

// withoutArgs returns line cut after the first word of its command,
// which starts after the given number of columns, so the arguments of a
// process never reach the facts. A line with no command, or a command
// with no arguments, comes back as it is. A program path that holds a
// space is cut at the space too, which keeps less than the path but
// never an argument.
func withoutArgs(
	line string,
	columns int,
) string {
	rest := line
	for range columns {
		rest = strings.TrimLeft(rest, " \t")
		i := strings.IndexAny(rest, " \t")
		if i < 0 {
			return line
		}
		rest = rest[i:]
	}
	rest = strings.TrimLeft(rest, " \t")
	i := strings.IndexAny(rest, " \t")
	if i < 0 {
		return line
	}
	return line[:len(line)-len(rest)+i]
}
