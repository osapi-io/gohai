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

package shard

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/shirou/gopsutil/v4/host"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/executor"
)

// hostInfoFn is the injection seam for gopsutil's host.InfoWithContext.
var hostInfoFn = host.InfoWithContext

// Darwin computes a shard seed on macOS from machinename +
// serial (system_profiler) + IOPlatformUUID (gopsutil).
// DMI is empty on macOS so we source serial/uuid directly.
type Darwin struct {
	base
	Exec executor.Executor
}

// NewDarwin returns a Darwin variant wired to real executors.
func NewDarwin() *Darwin {
	return &Darwin{Exec: executor.New()}
}

// Collect derives the shard seed. On macOS, Ohai uses
// hardware["serial_number"] and hardware["platform_UUID"] — we read
// the same sources via system_profiler and gopsutil respectively.
func (d *Darwin) Collect(
	ctx context.Context,
	prior collector.PriorResults,
) (any, error) {
	machinename := getMachineName(prior)
	serial := readDarwinSerial(ctx, d.Exec)
	uuid := readDarwinUUID(ctx)
	return &Info{Seed: computeSeed(machinename, serial, uuid)}, nil
}

// readDarwinSerial gets the hardware serial from system_profiler,
// matching Ohai's hardware["serial_number"] source.
func readDarwinSerial(
	ctx context.Context,
	exec executor.Executor,
) string {
	out, err := exec.Execute(ctx, "system_profiler", "SPHardwareDataType", "-json")
	if err != nil {
		return ""
	}
	var sp struct {
		Items []struct {
			SerialNumber string `json:"serial_number"`
		} `json:"SPHardwareDataType"`
	}
	if err := json.Unmarshal(out, &sp); err != nil || len(sp.Items) == 0 {
		return ""
	}
	return strings.TrimSpace(sp.Items[0].SerialNumber)
}

// readDarwinUUID reads IOPlatformUUID via gopsutil, matching Ohai's
// hardware["platform_UUID"] source.
func readDarwinUUID(
	ctx context.Context,
) string {
	h, err := hostInfoFn(ctx)
	if err != nil || h == nil {
		return ""
	}
	return h.HostID
}
