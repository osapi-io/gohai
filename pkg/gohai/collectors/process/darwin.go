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
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/pkg/gohai/source"
)

// lstartLayout is ps's lstart: "Thu Nov 20 12:04:00 2026".
const lstartLayout = "Mon Jan 2 15:04:05 2006"

// Darwin collects processes on macOS with ps: the ids, the owner, the
// state and the start time in one call, and the name and the command line
// each in its own, because either can hold spaces.
type Darwin struct {
	base

	Exec source.Runner
}

// NewDarwin returns a Darwin variant that runs the host's ps.
func NewDarwin() *Darwin {
	return &Darwin{Exec: source.Commands()}
}

// Collect returns a process snapshot.
func (d *Darwin) Collect(
	ctx context.Context,
	_ collector.PriorResults,
) (any, error) {
	procs, err := d.list(ctx, "-ax")
	if err != nil {
		return nil, err
	}

	return &Info{Count: len(procs), Processes: procs}, nil
}

// Read is the process pid, from `ps -p <pid>` alone.
func (d *Darwin) Read(
	ctx context.Context,
	pid int32,
) (*Process, error) {
	procs, err := d.list(ctx, "-p", strconv.Itoa(int(pid)))
	if err != nil {
		return nil, err
	}
	if len(procs) == 0 {
		return nil, fmt.Errorf("no process %d", pid)
	}

	return &procs[0], nil
}

// list is the processes that the ps selector picks: "-ax" for each
// process, "-p <pid>" for one.
func (d *Darwin) list(
	ctx context.Context,
	selector ...string,
) ([]Process, error) {
	ps := func(args ...string) []string { return append(append([]string{}, selector...), args...) }
	out, err := d.Exec.Execute(
		ctx,
		"ps",
		ps("-o", "pid=,ppid=,user=,state=,pcpu=,pmem=,rss=,time=,lstart=")...)
	if err != nil {
		return nil, fmt.Errorf("ps: %w", err)
	}
	names := d.column(ctx, ps("-o", "pid=,ucomm=")...)
	args := d.column(ctx, ps("-ww", "-o", "pid=,args=")...)
	procs := []Process{}
	for _, line := range strings.Split(string(out), "\n") {
		f := strings.Fields(line)
		if len(f) < 4 {
			continue
		}
		pid, err := strconv.ParseInt(f[0], 10, 32)
		if err != nil {
			continue
		}
		ppid, _ := strconv.ParseInt(f[1], 10, 32)
		p := Process{
			PID: int32(pid), ParentPID: int32(ppid), Owner: f[2], State: state(f[3]),
			Name: names[f[0]], CmdLine: args[f[0]],
		}
		if len(f) >= 8 {
			p.CPUPercent, _ = strconv.ParseFloat(f[4], 64)
			p.MemPercent, _ = strconv.ParseFloat(f[5], 64)
			kb, _ := strconv.ParseInt(f[6], 10, 64)
			p.RSS = kb * 1024
			p.CPUSeconds = cpuTime(f[7])
		}
		if len(f) >= 13 {
			if t, err := time.ParseInLocation(
				lstartLayout,
				strings.Join(f[8:13], " "),
				time.Local,
			); err == nil {
				p.CreationTime = t.Unix()
			}
		}
		procs = append(procs, p)
	}

	return procs, nil
}

// cpuTime is the seconds of ps's time column: "[[dd-]hh:]mm:ss.ss".
// A value that is not a time is 0.
func cpuTime(
	v string,
) float64 {
	days := 0.0
	if d, rest, ok := strings.Cut(v, "-"); ok {
		n, err := strconv.ParseFloat(d, 64)
		if err != nil {
			return 0
		}
		days, v = n, rest
	}
	total := 0.0
	for _, part := range strings.Split(v, ":") {
		n, err := strconv.ParseFloat(part, 64)
		if err != nil {
			return 0
		}
		total = total*60 + n
	}

	return days*86400 + total
}

// column is the rest of each line of ps with args, after the pid, by pid.
// It is empty when ps fails.
func (d *Darwin) column(
	ctx context.Context,
	args ...string,
) map[string]string {
	col := map[string]string{}
	out, err := d.Exec.Execute(ctx, "ps", args...)
	if err != nil {
		return col
	}
	for _, line := range strings.Split(string(out), "\n") {
		pid, rest, ok := strings.Cut(strings.TrimSpace(line), " ")
		if ok {
			col[pid] = strings.TrimSpace(rest)
		}
	}

	return col
}
