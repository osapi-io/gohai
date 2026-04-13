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

// Package executor abstracts OS command execution. Collectors that need
// to shell out (sysctl, sw_vers, lsb_release, loginctl, lscpu, etc.)
// accept an Executor so tests can assert the command + args and return
// canned stdout via a gomock-generated mock.
package executor

import (
	"context"
	"fmt"
	"os/exec"
)

// Executor runs OS commands and returns their combined stdout+stderr.
// Context cancellation propagates to the underlying process via
// exec.CommandContext — collectors can honor deadlines and SIGINT.
//
// Shape is intentionally minimal (single method, combined output).
// Matches osapi's CommandExecutor with one addition: we thread
// context for cancellation, osapi passes timeout as an integer.
type Executor interface {
	Execute(
		ctx context.Context,
		name string,
		args ...string,
	) ([]byte, error)
}

// defaultExecutor is the production implementation wrapping
// exec.CommandContext. Returns combined output so both stdout and
// stderr are visible to parsers — most of our collectors want stdout
// only and ignore stderr noise, which combined output naturally
// accommodates since stderr is rare on the success path.
type defaultExecutor struct{}

// New returns the production Executor implementation.
func New() Executor {
	return &defaultExecutor{}
}

// Execute runs the command and returns its combined stdout+stderr.
// On non-zero exit, returns the captured output plus a wrapped error
// carrying the exit status via errors.As(*exec.ExitError).
func (d *defaultExecutor) Execute(
	ctx context.Context,
	name string,
	args ...string,
) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return out, fmt.Errorf("exec %s: %w", name, err)
	}
	return out, nil
}
