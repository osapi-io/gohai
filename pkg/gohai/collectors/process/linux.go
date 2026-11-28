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

package process

import (
	"cmp"
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/osfs"

	"github.com/osapi-io/gohai/internal/collector"
)

// clockTicks is USER_HZ, the unit of /proc/<pid>/stat's start time. It is
// 100 on each architecture Linux runs osapi on.
const clockTicks = 100

// Linux collects processes from /proc: each /proc/<pid>/stat, status and
// cmdline, with the boot time from /proc/stat and the owner's name from
// /etc/passwd.
type Linux struct {
	base

	FS  avfs.VFS
	Now func() time.Time
}

// NewLinux returns a Linux variant that reads the host's filesystem.
func NewLinux() *Linux {
	return &Linux{FS: osfs.NewWithNoIdm(), Now: time.Now}
}

// Collect returns a process snapshot. A process that ends while it is
// read is left out. A cancelled ctx stops the read, with its error.
func (l *Linux) Collect(
	ctx context.Context,
	_ collector.PriorResults,
) (any, error) {
	entries, err := l.FS.ReadDir("/proc")
	if err != nil {
		return nil, fmt.Errorf("read /proc: %w", err)
	}
	boot := bootTime(l.FS)
	users := passwd(l.FS)
	total := memTotal(l.FS)
	now := l.Now().Unix()
	procs := []Process{}
	for _, e := range entries {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		pid, err := strconv.ParseInt(e.Name(), 10, 32)
		if err != nil {
			continue
		}
		if p, ok := l.read(int32(pid), boot, users, total, now); ok {
			procs = append(procs, p)
		}
	}

	return &Info{Count: len(procs), Processes: procs}, nil
}

// Read is the process pid, from its /proc directory alone.
func (l *Linux) Read(
	_ context.Context,
	pid int32,
) (*Process, error) {
	p, ok := l.read(pid, bootTime(l.FS), passwd(l.FS), memTotal(l.FS), l.Now().Unix())
	if !ok {
		return nil, fmt.Errorf("no process %d", pid)
	}

	return &p, nil
}

// read is the process pid, or false when its stat cannot be read.
func (l *Linux) read(
	pid int32,
	boot int64,
	users map[string]string,
	total int64,
	now int64,
) (Process, bool) {
	dir := "/proc/" + strconv.Itoa(int(pid))
	stat, err := l.FS.ReadFile(dir + "/stat")
	if err != nil {
		return Process{}, false
	}
	text := string(stat)
	open, closing := strings.IndexByte(text, '('), strings.LastIndexByte(text, ')')
	if open < 0 || closing < open {
		return Process{}, false
	}
	p := Process{PID: pid, Name: text[open+1 : closing]}
	// After the name: state, ppid, and the start time 19 fields on.
	f := strings.Fields(text[closing+1:])
	if len(f) > 1 {
		p.State = state(f[0])
		ppid, _ := strconv.ParseInt(f[1], 10, 32)
		p.ParentPID = int32(ppid)
	}
	if len(f) > 4 {
		nr, _ := strconv.ParseInt(f[4], 10, 64)
		p.TTY = ttyName(nr)
	}
	if len(f) > 19 && boot > 0 {
		if ticks, err := strconv.ParseInt(f[19], 10, 64); err == nil {
			p.CreationTime = boot + ticks/clockTicks
		}
		utime, _ := strconv.ParseInt(f[11], 10, 64)
		stime, _ := strconv.ParseInt(f[12], 10, 64)
		p.CPUSeconds = float64(utime+stime) / clockTicks
		if elapsed := now - p.CreationTime; p.CreationTime > 0 && elapsed > 0 {
			p.CPUPercent = float64(utime+stime) / clockTicks / float64(elapsed) * 100
		}
	}
	if status, err := l.FS.ReadFile(dir + "/status"); err == nil {
		for _, line := range strings.Split(string(status), "\n") {
			if v, ok := strings.CutPrefix(line, "Uid:"); ok {
				if uid := strings.Fields(v); len(uid) > 0 {
					// A user that /etc/passwd does not have, such as
					// one from LDAP, is its uid, as ps shows it.
					p.Owner = cmp.Or(users[uid[0]], uid[0])
				}
			}
			if v, ok := strings.CutPrefix(line, "VmRSS:"); ok {
				kb, _ := strconv.ParseInt(strings.TrimSuffix(strings.TrimSpace(v), " kB"), 10, 64)
				p.RSS = kb * 1024
			}
			if v, ok := strings.CutPrefix(line, "Threads:"); ok {
				p.Threads, _ = strconv.Atoi(strings.TrimSpace(v))
			}
		}
	}
	if total > 0 {
		p.MemPercent = float64(p.RSS) / float64(total) * 100
	}
	if cmd, err := l.FS.ReadFile(dir + "/cmdline"); err == nil {
		p.CmdLine = strings.TrimSpace(strings.ReplaceAll(string(cmd), "\x00", " "))
	}
	// Another user's fd directory is root-only, so a run that is not root
	// leaves OpenFiles empty for it.
	if fds, err := l.FS.ReadDir(dir + "/fd"); err == nil {
		p.OpenFiles = len(fds)
	}

	return p, true
}

// ttyName is the name of the terminal device nr, the tty_nr field of
// /proc/<pid>/stat: "pts/N" for a pseudo-terminal, "ttyN" for a virtual
// console and "ttySN" for a serial port. It is empty for no terminal, and
// for a device that is none of those.
func ttyName(
	nr int64,
) string {
	major := (nr >> 8) & 0xfff
	minor := (nr & 0xff) | ((nr >> 12) & 0xfff00)
	switch {
	case nr == 0:
		return ""
	case major >= 136 && major <= 143:
		return "pts/" + strconv.FormatInt((major-136)*256+minor, 10)
	case major == 4 && minor < 64:
		return "tty" + strconv.FormatInt(minor, 10)
	case major == 4:
		return "ttyS" + strconv.FormatInt(minor-64, 10)
	}
	return ""
}

// bootTime is the btime line of /proc/stat, or 0.
func bootTime(
	fs avfs.VFS,
) int64 {
	b, err := fs.ReadFile("/proc/stat")
	if err != nil {
		return 0
	}
	for _, line := range strings.Split(string(b), "\n") {
		if v, ok := strings.CutPrefix(line, "btime "); ok {
			n, _ := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
			return n
		}
	}

	return 0
}

// passwd is the user name of each uid in /etc/passwd.
func passwd(
	fs avfs.VFS,
) map[string]string {
	users := map[string]string{}
	b, err := fs.ReadFile("/etc/passwd")
	if err != nil {
		return users
	}
	for _, line := range strings.Split(string(b), "\n") {
		f := strings.Split(line, ":")
		if len(f) > 2 {
			users[f[2]] = f[0]
		}
	}

	return users
}

// memTotal is MemTotal of /proc/meminfo, in bytes, or 0.
func memTotal(
	fs avfs.VFS,
) int64 {
	b, err := fs.ReadFile("/proc/meminfo")
	if err != nil {
		return 0
	}
	for _, line := range strings.Split(string(b), "\n") {
		if v, ok := strings.CutPrefix(line, "MemTotal:"); ok {
			kb, _ := strconv.ParseInt(strings.TrimSuffix(strings.TrimSpace(v), " kB"), 10, 64)
			return kb * 1024
		}
	}

	return 0
}
