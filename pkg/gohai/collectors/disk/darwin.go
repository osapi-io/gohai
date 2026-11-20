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

package disk

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/executor"
)

var (
	// bsdNameRE is the BSD name of a driver's media: "disk0".
	bsdNameRE = regexp.MustCompile(`"BSD Name" = "([^"]+)"`)
	// statRE is one counter of a driver's Statistics:
	// "Operations (Read)"=378856037.
	statRE = regexp.MustCompile(`"([^"]+)"=(\d+)`)
)

// Darwin collects I/O counters from IOKit's block storage drivers, with
// ioreg: each driver's Statistics, and the BSD name of the media under
// it. A driver with no media is left out, as gopsutil leaves it out.
type Darwin struct {
	base

	Exec executor.Executor
}

// NewDarwin returns a Darwin variant that runs the host's ioreg.
func NewDarwin() *Darwin {
	return &Darwin{Exec: executor.New()}
}

// Collect returns per-device I/O counters. The times are in
// milliseconds, and IoTime is the read and the write time.
func (d *Darwin) Collect(
	ctx context.Context,
	_ collector.PriorResults,
) (any, error) {
	out, err := d.Exec.Execute(
		ctx,
		"ioreg",
		"-r",
		"-c",
		"IOBlockStorageDriver",
		"-d",
		"2",
		"-l",
		"-w0",
	)
	if err != nil {
		return nil, fmt.Errorf("ioreg IOBlockStorageDriver: %w", err)
	}
	devs := []Device{}
	var cur *Device
	for _, line := range strings.Split(string(out), "\n") {
		switch {
		case strings.HasPrefix(line, "+-o "):
			cur = nil
		case strings.Contains(line, `"Statistics" = {`):
			stats := map[string]uint64{}
			for _, m := range statRE.FindAllStringSubmatch(line, -1) {
				stats[m[1]], _ = strconv.ParseUint(m[2], 10, 64)
			}
			read := stats["Total Time (Read)"] / 1_000_000
			write := stats["Total Time (Write)"] / 1_000_000
			cur = &Device{
				ReadCount:  stats["Operations (Read)"],
				WriteCount: stats["Operations (Write)"],
				ReadBytes:  stats["Bytes (Read)"],
				WriteBytes: stats["Bytes (Write)"],
				ReadTime:   read,
				WriteTime:  write,
				IoTime:     read + write,
			}
		default:
			if m := bsdNameRE.FindStringSubmatch(line); m != nil && cur != nil {
				cur.Name = m[1]
				devs = append(devs, *cur)
				cur = nil
			}
		}
	}

	return &Info{Devices: devs}, nil
}
