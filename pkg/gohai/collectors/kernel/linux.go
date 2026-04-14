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
	"strconv"
	"strings"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/osfs"
	"github.com/osapi-io/gohai/internal/collector"
)

const (
	procModulesPath     = "/proc/modules"
	sysModuleVersionFmt = "/sys/module/%s/version"
)

// Linux collects kernel facts on Linux. Uses unix.Uname for the top-level
// identity fields (via the package-level unameSyscall seam), reads
// /proc/modules for the module list, and reads /sys/module/<name>/version
// to populate each module's Version — matches Ohai's kernel plugin on
// Linux. `processor` and `os` are synthesized per Option A of issue #29
// (Machine and the static string "GNU/Linux") rather than shelling out
// to `uname -p` / `uname -o`.
type Linux struct {
	base

	FS avfs.VFS
}

// NewLinux returns a Linux variant wired to the real OS filesystem.
func NewLinux() *Linux {
	return &Linux{FS: osfs.NewWithNoIdm()}
}

// Collect returns kernel Info.
func (l *Linux) Collect(
	_ context.Context,
	_ collector.PriorResults,
) (any, error) {
	name, release, version, machine, err := defaultUname()
	if err != nil {
		return nil, err
	}
	info := &Info{
		Name:      name,
		Release:   release,
		Version:   version,
		Machine:   machine,
		Processor: machine,
		OS:        "GNU/Linux",
	}
	if b, err := l.FS.ReadFile(procModulesPath); err == nil {
		info.Modules = parseProcModules(b)
		enrichModuleVersions(l.FS, info.Modules)
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

// enrichModuleVersions reads /sys/module/<name>/version per module and
// assigns the trimmed contents to Module.Version. Many built-in or
// stripped-down modules do not expose a version file; those leave the
// field empty (matches Ohai's silent-on-miss behaviour).
func enrichModuleVersions(
	fs avfs.VFS,
	modules map[string]Module,
) {
	for name, m := range modules {
		path := "/sys/module/" + name + "/version"
		b, err := fs.ReadFile(path)
		if err != nil {
			continue
		}
		v := strings.TrimSpace(string(b))
		if v == "" {
			continue
		}
		m.Version = v
		modules[name] = m
	}
}
