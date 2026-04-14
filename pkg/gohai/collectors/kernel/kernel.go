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

// Package kernel reports kernel identity (name, release, version,
// machine) plus the set of currently loaded modules. Module entries
// carry size, reference count, and version so downstream tooling can
// correlate loaded modules against CVE feeds or policy baselines. On
// Linux versions come from /sys/module/<name>/version; on macOS the
// module list is produced from `kextstat -k -l` (legacy kexts only —
// System Extensions are not yet enumerated).
package kernel

import (
	"fmt"

	"golang.org/x/sys/unix"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/platform"
)

// Info holds kernel identification data. Field semantics follow POSIX
// uname(1) plus two synthesized fields (processor, os) matching what
// the `uname(1)` CLI reports versus the raw syscall.
type Info struct {
	Name              string            `json:"name"`                         // uname -s: "Linux", "Darwin"
	Release           string            `json:"release"`                      // uname -r: "5.15.0-47-generic" (OCSF: os.kernel_release — leaf stripped per CLAUDE.md)
	Version           string            `json:"version,omitempty"`            // uname -v: build string
	Machine           string            `json:"machine"`                      // uname -m: hardware arch ("x86_64", "arm64")
	Processor         string            `json:"processor,omitempty"`          // uname -p synthesis: same as Machine on Linux/Darwin
	OS                string            `json:"os,omitempty"`                 // uname -o synthesis: "GNU/Linux" on Linux, "Darwin" on Darwin (OCSF: os.type)
	RosettaTranslated bool              `json:"rosetta_translated,omitempty"` // macOS only: true when process is under Rosetta 2
	Modules           map[string]Module `json:"modules,omitempty"`            // loaded kernel modules (Linux: /proc/modules; macOS: kextstat)
}

// Module describes a loaded kernel module.
type Module struct {
	Size     uint64 `json:"size,omitempty"`     // bytes
	RefCount int    `json:"refcount,omitempty"` // instances currently loaded
	Version  string `json:"version,omitempty"`  // Linux: /sys/module/<m>/version; macOS: parens field from kextstat
}

// Collector is the public interface every kernel variant satisfies.
type Collector interface {
	collector.Collector
}

type base struct{}

func (base) Name() string           { return "kernel" }
func (base) Category() string       { return collector.CategorySystem }
func (base) DefaultEnabled() bool   { return true }
func (base) Dependencies() []string { return nil }

// New returns the kernel variant for the host OS.
func New() Collector {
	if platform.Detect() == "darwin" {
		return NewDarwin()
	}
	return NewLinux()
}

// unameSyscall is the injection seam for unix.Uname. Tests swap this
// via export_test.go to force the error branch of defaultUname.
var unameSyscall = unix.Uname

// defaultUname invokes unix.Uname (works on any unix-like OS) and
// converts the fixed-length byte arrays to Go strings. Shared by both
// Linux and Darwin factories.
func defaultUname() (name, release, version, machine string, err error) {
	var u unix.Utsname
	if err = unameSyscall(&u); err != nil {
		return "", "", "", "", fmt.Errorf("uname: %w", err)
	}
	return bytesToString(u.Sysname[:]),
		bytesToString(u.Release[:]),
		bytesToString(u.Version[:]),
		bytesToString(u.Machine[:]),
		nil
}

// bytesToString trims trailing NUL bytes from a C-style fixed-length
// byte array and returns the Go string.
func bytesToString(
	b []byte,
) string {
	for i, c := range b {
		if c == 0 {
			return string(b[:i])
		}
	}
	return string(b)
}
