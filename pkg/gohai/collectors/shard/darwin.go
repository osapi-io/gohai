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
	"errors"
	"fmt"
	"strings"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/pattern"
	"github.com/osapi-io/gohai/pkg/gohai/source"
)

// platformUUIDRE is the IOPlatformUUID line of ioreg's output.
var platformUUIDRE = pattern.New(`"IOPlatformUUID" = "([^"]+)"`)

// Darwin computes a shard seed on macOS from machinename +
// serial (system_profiler) + IOPlatformUUID (ioreg).
// DMI is empty on macOS so we source serial/uuid directly.
type Darwin struct {
	base
	Exec source.Runner
}

// NewDarwin returns a Darwin variant wired to real executors.
func NewDarwin() *Darwin {
	return &Darwin{Exec: source.Commands()}
}

// Collect derives the shard seed. On macOS, Ohai uses
// hardware["serial_number"] and hardware["platform_UUID"]. We read
// the same sources via system_profiler and ioreg. With no Exec, or with
// a command that is not on the host, the seed comes from the inputs that
// remain. A command that is there and fails, or whose output does not
// parse, is an error and gives no seed: a seed from fewer inputs moves the
// host to a different shard, which is worse than no seed.
func (d *Darwin) Collect(
	ctx context.Context,
	prior collector.PriorResults,
) (any, error) {
	machinename, err := getMachineName(prior)
	if err != nil {
		return nil, err
	}
	if d.Exec == nil {
		return newInfo(machinename, "", ""), nil
	}
	serial, serialErr := readDarwinSerial(ctx, d.Exec)
	uuid, uuidErr := readDarwinUUID(ctx, d.Exec)
	if err := errors.Join(serialErr, uuidErr); err != nil {
		return nil, err
	}

	return newInfo(machinename, serial, uuid), nil
}

// readDarwinSerial gets the hardware serial from system_profiler,
// matching Ohai's hardware["serial_number"] source. It is "" with no
// error when system_profiler is not on the host or lists no hardware.
func readDarwinSerial(
	ctx context.Context,
	exec source.Runner,
) (string, error) {
	out, err := exec.Execute(ctx, "system_profiler", "SPHardwareDataType", "-json")
	if collector.Absent(err) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("system_profiler: %w", err)
	}
	var sp struct {
		Items []struct {
			SerialNumber string `json:"serial_number"`
		} `json:"SPHardwareDataType"`
	}
	if err := json.Unmarshal(out, &sp); err != nil {
		return "", fmt.Errorf("system_profiler: %w", err)
	}
	if len(sp.Items) == 0 {
		return "", nil
	}

	return strings.TrimSpace(sp.Items[0].SerialNumber), nil
}

// readDarwinUUID reads IOPlatformUUID from ioreg, matching Ohai's
// hardware["platform_UUID"] source. It is "" with no error when ioreg is
// not on the host or prints no IOPlatformUUID.
func readDarwinUUID(
	ctx context.Context,
	exec source.Runner,
) (string, error) {
	out, err := exec.Execute(ctx, "ioreg", "-rd1", "-c", "IOPlatformExpertDevice")
	if collector.Absent(err) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("ioreg: %w", err)
	}
	if m := platformUUIDRE.FindSubmatch(out); m != nil {
		return string(m[1]), nil
	}

	return "", nil
}
