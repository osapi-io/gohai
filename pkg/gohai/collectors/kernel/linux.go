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

package kernel

import (
	"bufio"
	"context"
	"os"
	"strconv"
	"strings"
)

const procModulesPath = "/proc/modules"

// Linux collects kernel facts on Linux using syscall.Uname plus
// /proc/modules for loaded module data. UnameFn and ReadFileFn are
// injected so tests cover every branch without touching the real host.
type Linux struct {
	base

	UnameFn    func() (name, release, version, machine string, err error)
	ReadFileFn func(string) ([]byte, error)
}

// NewLinux returns a Linux variant wired to real syscalls / os.ReadFile.
func NewLinux() *Linux {
	return &Linux{
		UnameFn:    defaultUname,
		ReadFileFn: os.ReadFile,
	}
}

// Collect returns kernel Info. Module parsing failures yield an empty
// Modules map rather than failing the whole collector.
func (l *Linux) Collect(_ context.Context) (any, error) {
	name, release, version, machine, err := l.UnameFn()
	if err != nil {
		return nil, err
	}
	info := &Info{
		Name:    name,
		Release: release,
		Version: version,
		Machine: machine,
	}
	if b, err := l.ReadFileFn(procModulesPath); err == nil {
		info.Modules = parseProcModules(b)
	}
	return info, nil
}

// parseProcModules turns /proc/modules contents into our Module map.
// Format per line:
//
//	<name> <size> <refcount> [<used_by>] <state> <offset>
func parseProcModules(
	b []byte,
) map[string]Module {
	out := map[string]Module{}
	sc := bufio.NewScanner(strings.NewReader(string(b)))
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) < 3 {
			continue
		}
		m := Module{}
		if sz, err := strconv.ParseUint(fields[1], 10, 64); err == nil {
			m.Size = sz
		}
		if rc, err := strconv.Atoi(fields[2]); err == nil {
			m.RefCount = rc
		}
		out[fields[0]] = m
	}
	return out
}
