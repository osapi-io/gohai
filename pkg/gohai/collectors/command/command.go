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
// header. Its CMD or COMMAND column tells where each command starts,
// and its PID column tells a process row from the rest of an argument
// that held a newline, so the GNU, BSD and BusyBox layouts of `ps -ef`
// all work. Output with no such header gives no lines, and a line that
// does not fit the header is dropped, so the fix fails closed and does
// not rely on ps to clean its output.
func parsePSOutput(
	output string,
) *Info {
	info := &Info{PS: []string{}}
	var cols psColumns
	header := true
	// Split, not bufio.Scanner: ps -ef prints a whole command line when
	// stdout is not a tty, and a line can pass the Scanner's 64 KiB limit.
	for _, raw := range strings.Split(output, "\n") {
		line := strings.TrimRight(raw, " \t\r")
		if line == "" {
			continue
		}
		if header {
			header = false
			var ok bool
			if cols, ok = headerColumns(line); !ok {
				return info
			}
			info.PS = append(info.PS, line)
			continue
		}
		if kept, ok := withoutArgs(line, cols); ok {
			info.PS = append(info.PS, kept)
		}
	}
	return info
}

// psColumns holds where the PID and the command start in a ps line,
// counted in columns from the left.
type psColumns struct {
	pid int
	cmd int
}

// headerColumns finds the PID and the CMD or COMMAND columns of a ps
// header. It gives false when the header has either one missing, or
// when the command is not the last column.
func headerColumns(
	header string,
) (psColumns, bool) {
	cols := psColumns{pid: -1, cmd: -1}
	names := strings.Fields(header)
	for i, name := range names {
		switch name {
		case "PID":
			cols.pid = i
		case "CMD", "COMMAND":
			cols.cmd = i
		}
	}
	if cols.pid < 0 || cols.cmd != len(names)-1 {
		return cols, false
	}
	return cols, true
}

// withoutArgs returns line cut after the first word of its command, so
// the arguments of a process never reach the facts. It gives false,
// and the line is dropped, when the line does not fit the header: it
// has no field at the command column, or its PID column is not a
// number, as for the rest of an argument that held a newline. A program
// path that holds a space is cut at the space too, which keeps less
// than the path but never an argument.
func withoutArgs(
	line string,
	cols psColumns,
) (string, bool) {
	rest := line
	for col := range cols.cmd {
		rest = strings.TrimLeft(rest, " \t")
		i := strings.IndexAny(rest, " \t")
		if i < 0 {
			return "", false
		}
		if col == cols.pid && !digits(rest[:i]) {
			return "", false
		}
		rest = rest[i:]
	}
	// line has no trailing whitespace, so rest holds the command here.
	rest = strings.TrimLeft(rest, " \t")
	i := strings.IndexAny(rest, " \t")
	if i < 0 {
		return line, true
	}
	return line[:len(line)-len(rest)+i], true
}

// digits reports whether s, which is never empty here, is all ASCII
// digits.
func digits(
	s string,
) bool {
	for _, c := range []byte(s) {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
