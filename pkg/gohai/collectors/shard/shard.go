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

// Package shard derives a deterministic shard seed from the host's
// stable identity. Matches Ohai's shard plugin semantics: a hash
// combining machine_id + hostname so the same host always maps to the
// same shard, but different hosts distribute evenly.
//
// Consumers use this to:
//   - Stagger cron / maintenance jobs across a fleet (shard % N
//     determines which minute/day a host runs something).
//   - Distribute work across parallel pipelines without per-host
//     config.
package shard

import (
	"context"
	"crypto/sha256"
	"encoding/hex"

	"github.com/shirou/gopsutil/v4/host"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/platform"
)

// hostInfoFn is the injection seam for gopsutil's host.InfoWithContext.
// Private — never leaked through a public Fn field. Swapped in tests
// via SetHostInfoFn (export_test.go).
var hostInfoFn = host.InfoWithContext

// readMachineUUID wraps the private gopsutil call and returns the
// macOS IOPlatformUUID as a plain string so importers don't see
// gopsutil types. Returns empty string + wrapped error when gopsutil
// fails (caller may choose to treat empty as "no stable ID available"
// and still compute a hostname-only seed).
func readMachineUUID(
	ctx context.Context,
) (string, error) {
	h, err := hostInfoFn(ctx)
	if err != nil {
		return "", err
	}
	if h == nil {
		return "", nil
	}
	return h.HostID, nil
}

// Info holds the derived shard seed.
type Info struct {
	// Seed is the hex-encoded SHA-256 of "<machine_id>:<hostname>".
	// Stable across reboots when the host has a stable machine-id.
	Seed string `json:"seed"`
}

// Collector is the public interface every shard variant satisfies.
type Collector interface {
	collector.Collector
}

type base struct{}

func (base) Name() string { return "shard" }

// DefaultEnabled returns true — shard is on by default.
func (base) DefaultEnabled() bool { return true }

// Dependencies returns the upstream facts this collector derives from.
// Listed so consumers know shard's output depends on stable inputs.
// (Note: gohai's current collector interface doesn't feed upstream
// values into downstream Collect() — this is documentation-only.)
func (base) Dependencies() []string { return nil }

// New returns the shard variant for the host OS. Both variants use the
// same logic; separate structs preserve the osapi dispatch pattern.
func New() Collector {
	if platform.Detect() == "darwin" {
		return NewDarwin()
	}
	return NewLinux()
}

// readMachineID returns the stable Linux machine-id via the shared
// fallback chain (/etc/machine-id → /var/lib/dbus/machine-id). Empty
// on hosts with neither file.
func readMachineID(
	readFile func(string) ([]byte, error),
) string {
	for _, p := range []string{"/etc/machine-id", "/var/lib/dbus/machine-id"} {
		if b, err := readFile(p); err == nil && len(b) > 0 {
			for _, c := range b {
				if c == '\n' {
					break
				}
			}
			s := string(b)
			for i := 0; i < len(s); i++ {
				if s[i] == '\n' {
					return s[:i]
				}
			}
			return s
		}
	}
	return ""
}

// computeSeed builds the deterministic hash from machine_id + hostname.
// Hostname alone isn't stable (can be renamed); machine_id alone
// doesn't distribute evenly across VMs cloned from the same image —
// combining both gives a practical, stable shard.
func computeSeed(
	machineID, hostname string,
) string {
	h := sha256.Sum256([]byte(machineID + ":" + hostname))
	return hex.EncodeToString(h[:])
}
