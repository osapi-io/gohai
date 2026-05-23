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

// Package zpools reports ZFS pool status by running
// `zpool list -H -o name,size,alloc,free,health,altroot`. Both Linux and
// macOS can host ZFS (OpenZFS), so both variants run zpool when available.
package zpools

import (
	"bufio"
	"bytes"
	"context"
	"strings"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/executor"
	"github.com/osapi-io/gohai/internal/platform"
)

// Pool holds the attributes of a single ZFS pool.
type Pool struct {
	// Name is the pool name.
	Name string `json:"name"`
	// Size is the total pool size as reported by zpool (e.g. "1.82T").
	// Empty when zpool reports "-".
	Size string `json:"size,omitempty"`
	// Alloc is the amount of storage space allocated (e.g. "672G").
	// Empty when zpool reports "-".
	Alloc string `json:"alloc,omitempty"`
	// Free is the amount of unallocated storage space (e.g. "1.17T").
	// Empty when zpool reports "-".
	Free string `json:"free,omitempty"`
	// Health is the pool health status: "ONLINE", "DEGRADED", "FAULTED",
	// "OFFLINE", "REMOVED", or "UNAVAIL".
	Health string `json:"health,omitempty"`
	// Altroot is the alternate root directory for the pool.
	// Empty when none is set (zpool reports "-").
	Altroot string `json:"altroot,omitempty"`
}

// Info holds the list of ZFS pools found on the host.
type Info struct {
	Pools []Pool `json:"pools"`
}

// Collector is the public interface every zpools variant satisfies.
type Collector interface {
	collector.Collector
}

// base holds the fields every OS variant has in common.
type base struct{}

// Name returns "zpools".
func (base) Name() string { return "zpools" }

// Category returns "linux".
func (base) Category() string { return collector.CategoryLinux }

// DefaultEnabled returns false — zpools is opt-in.
func (base) DefaultEnabled() bool { return false }

// Dependencies returns no dependencies.
func (base) Dependencies() []string { return nil }

// New returns the zpools collector variant appropriate to the detected
// host OS.
func New() Collector {
	switch platform.Detect() {
	case "darwin":
		return NewDarwin()
	default:
		return NewLinux()
	}
}

// zpoolArgs are the arguments passed to zpool to enumerate pools.
// -H suppresses the header; -o selects exactly the fields we want.
var zpoolArgs = []string{"list", "-H", "-o", "name,size,alloc,free,health,altroot"}

// sanitize converts "-" (zpool's "not available" sentinel) to empty
// string so consumers don't have to special-case the sentinel.
func sanitize(
	v string,
) string {
	if v == "-" {
		return ""
	}
	return v
}

// parseZpoolList parses the tab-separated output of
// `zpool list -H -o name,size,alloc,free,health,altroot`.
// Lines that don't have exactly 6 tab-separated fields are skipped.
func parseZpoolList(
	out []byte,
) []Pool {
	pools := []Pool{}
	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Split(line, "\t")
		if len(fields) != 6 {
			continue
		}
		pools = append(pools, Pool{
			Name:    fields[0],
			Size:    sanitize(fields[1]),
			Alloc:   sanitize(fields[2]),
			Free:    sanitize(fields[3]),
			Health:  sanitize(fields[4]),
			Altroot: sanitize(fields[5]),
		})
	}
	return pools
}

// collectPools runs `zpool list` via execFn and parses the output. If
// zpool is not found (command error) it returns an empty list without
// error — ZFS is not installed on this host.
func collectPools(
	ctx context.Context,
	exec executor.Executor,
) (*Info, error) {
	if exec == nil {
		return &Info{Pools: []Pool{}}, nil
	}
	out, err := exec.Execute(ctx, "zpool", zpoolArgs...)
	if err != nil {
		// zpool not installed or no pools — return empty, not an error.
		return &Info{Pools: []Pool{}}, nil
	}
	return &Info{Pools: parseZpoolList(out)}, nil
}
