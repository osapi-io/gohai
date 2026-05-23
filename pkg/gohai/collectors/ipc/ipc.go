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

// Package ipc reports Linux IPC subsystem kernel parameters: semaphore
// limits, message queue limits, and shared memory limits, all read from
// the /proc/sys/kernel/ sysctl tree.
package ipc

import (
	"strings"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/platform"
)

// SemLimits holds the kernel semaphore configuration from
// /proc/sys/kernel/sem. The file contains four whitespace-separated
// values on one line: SEMMSL SEMMNS SEMOPM SEMMNI.
type SemLimits struct {
	// SEMMSL is the maximum number of semaphores per semaphore set.
	SEMMSL string `json:"semmsl"`
	// SEMMNS is the maximum number of semaphores system-wide.
	SEMMNS string `json:"semmns"`
	// SEMOPM is the maximum number of operations per semop call.
	SEMOPM string `json:"semopm"`
	// SEMMNI is the maximum number of semaphore sets system-wide.
	SEMMNI string `json:"semmni"`
}

// MsgLimits holds the message queue kernel parameters.
type MsgLimits struct {
	// MSGMNB is the default maximum size in bytes for a message queue,
	// from /proc/sys/kernel/msgmnb.
	MSGMNB string `json:"msgmnb"`
	// MSGMNI is the maximum number of message queue identifiers,
	// from /proc/sys/kernel/msgmni.
	MSGMNI string `json:"msgmni"`
	// MSGMAX is the maximum message size in bytes,
	// from /proc/sys/kernel/msgmax.
	MSGMAX string `json:"msgmax"`
}

// ShmLimits holds the shared memory kernel parameters.
type ShmLimits struct {
	// SHMALL is the total amount of shared memory (in pages or bytes
	// depending on kernel version), from /proc/sys/kernel/shmall.
	SHMALL string `json:"shmall"`
	// SHMMAX is the maximum size of a shared memory segment in bytes,
	// from /proc/sys/kernel/shmmax.
	SHMMAX string `json:"shmmax"`
	// SHMMNI is the maximum number of shared memory segments,
	// from /proc/sys/kernel/shmmni.
	SHMMNI string `json:"shmmni"`
}

// Info holds all Linux IPC subsystem kernel parameters.
type Info struct {
	Sem SemLimits `json:"sem"`
	Msg MsgLimits `json:"msg"`
	Shm ShmLimits `json:"shm"`
}

// Collector is the public interface every ipc variant satisfies.
type Collector interface {
	collector.Collector
}

// base holds the fields every OS variant has in common.
type base struct{}

// Name returns "ipc".
func (base) Name() string { return "ipc" }

// Category returns "linux".
func (base) Category() string { return collector.CategoryLinux }

// DefaultEnabled returns false — ipc is opt-in.
func (base) DefaultEnabled() bool { return false }

// Dependencies returns no dependencies.
func (base) Dependencies() []string { return nil }

// New returns the ipc collector variant appropriate to the detected
// host OS.
func New() Collector {
	switch platform.Detect() {
	case "darwin":
		return NewDarwin()
	default:
		return NewLinux()
	}
}

// readTrimmed reads a sysfs/procfs file using readFn and returns its
// trimmed content, or empty string on error.
func readTrimmed(
	readFn func(string) ([]byte, error),
	path string,
) string {
	b, err := readFn(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

// parseSem parses the four-value /proc/sys/kernel/sem line into a
// SemLimits struct. Returns a zero-value struct if the content is
// missing or malformed.
func parseSem(
	content string,
) SemLimits {
	fields := strings.Fields(content)
	lim := SemLimits{}
	if len(fields) >= 1 {
		lim.SEMMSL = fields[0]
	}
	if len(fields) >= 2 {
		lim.SEMMNS = fields[1]
	}
	if len(fields) >= 3 {
		lim.SEMOPM = fields[2]
	}
	if len(fields) >= 4 {
		lim.SEMMNI = fields[3]
	}
	return lim
}
