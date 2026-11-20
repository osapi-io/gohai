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
	"github.com/osapi-io/gohai/internal/executor"
)

// lstartLayout is ps's lstart: "Thu Nov 20 12:04:00 2026".
const lstartLayout = "Mon Jan 2 15:04:05 2006"

// Darwin collects processes on macOS with ps: the ids, the owner, the
// state and the start time in one call, and the name and the command line
// each in its own, because either can hold spaces.
type Darwin struct {
	base

	Exec executor.Executor
}

// NewDarwin returns a Darwin variant that runs the host's ps.
func NewDarwin() *Darwin {
	return &Darwin{Exec: executor.New()}
}

// Collect returns a process snapshot.
func (d *Darwin) Collect(
	ctx context.Context,
	_ collector.PriorResults,
) (any, error) {
	out, err := d.Exec.Execute(
		ctx,
		"ps",
		"-axo",
		"pid=,ppid=,user=,state=,pcpu=,pmem=,rss=,lstart=",
	)
	if err != nil {
		return nil, fmt.Errorf("ps: %w", err)
	}
	names := d.column(ctx, "-axo", "pid=,ucomm=")
	args := d.column(ctx, "-axww", "-o", "pid=,args=")
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
		if len(f) >= 7 {
			p.CPUPercent, _ = strconv.ParseFloat(f[4], 64)
			p.MemPercent, _ = strconv.ParseFloat(f[5], 64)
			kb, _ := strconv.ParseInt(f[6], 10, 64)
			p.RSS = kb * 1024
		}
		if len(f) >= 12 {
			if t, err := time.ParseInLocation(
				lstartLayout,
				strings.Join(f[7:12], " "),
				time.Local,
			); err == nil {
				p.CreationTime = t.Unix()
			}
		}
		procs = append(procs, p)
	}

	return &Info{Count: len(procs), Processes: procs}, nil
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
