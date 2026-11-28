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

// Package source is what a collector reads a host through: a filesystem, a
// command runner, a clock and the host's platform family. Host is the real
// host. A caller that gives other sources, such as osapi's fake host in its
// tests, gets the same collectors reading that host instead.
package source

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/osfs"

	"github.com/osapi-io/gohai/pkg/gohai/family"
)

// Runner runs a command and returns its stdout. Context cancellation stops
// the process. On a failure, the error has the command's stderr.
type Runner interface {
	Execute(
		ctx context.Context,
		name string,
		args ...string,
	) ([]byte, error)
}

// Sources is what a collector reads. A zero field is the real host's.
type Sources struct {
	// FS is the filesystem: /proc, /sys, /etc and the rest.
	FS avfs.VFS
	// Exec runs the host's programs, such as sysctl, df and vm_stat.
	Exec Runner
	// Now is the clock, for a value computed from the time, such as uptime
	// from the boot time.
	Now func() time.Time
	// Family picks the collector's variant, as family.Detect names it, such
	// as debian or darwin. When it is zero and FS is given, it is the family
	// that family.FromFS reads from FS: darwin for a filesystem with macOS's
	// SystemVersion.plist, else the family that its /etc/os-release names.
	Family string
	// Resolver resolves names, as the system resolver does: /etc/hosts and
	// then DNS. The hostname collector canonicalizes the FQDN with it.
	Resolver Resolver
	// HTTP is the transport of the cloud collectors' metadata reads, such as
	// http://169.254.169.254. A fake host gives one that answers them, so a
	// test reads no network. When it is zero, the collectors share one
	// transport that uses no proxy.
	HTTP http.RoundTripper
}

// Resolver is the name lookups that a collector makes. *net.Resolver has
// them.
type Resolver interface {
	LookupHost(ctx context.Context, host string) ([]string, error)
	LookupAddr(ctx context.Context, addr string) ([]string, error)
}

// Host is the real host: its filesystem, its programs, its clock and its
// family.
func Host() Sources {
	return Sources{}.Defaults()
}

// Defaults fills each zero field of s with the real host's. A zero Family
// with a given FS is the family of that FS's /etc/os-release, not the real
// host's. A zero HTTP stays zero: it is the cloud collectors' shared
// transport.
func (s Sources) Defaults() Sources {
	if s.Family == "" {
		if s.FS != nil {
			s.Family = family.FromFS(s.FS)
		} else {
			s.Family = family.Detect()
		}
	}
	if s.FS == nil {
		s.FS = osfs.NewWithNoIdm()
	}
	if s.Exec == nil {
		s.Exec = Commands()
	}
	if s.Now == nil {
		s.Now = time.Now
	}
	if s.Resolver == nil {
		s.Resolver = net.DefaultResolver
	}

	return s
}

// CommandTimeout is the longest that one command of the real host runs.
// A hung program, such as docker with a stopped daemon, then fails its
// collector and not the whole collection.
const CommandTimeout = 30 * time.Second

// MaxOutput is the most stdout that one command keeps. A command that
// writes more fails.
const MaxOutput = 64 << 20

// commandWaitDelay is how long Execute waits for the pipes to close after
// the command exits or is stopped. A child that keeps the pipes open, such
// as a daemon that the command starts, then does not block Execute.
const commandWaitDelay = 2 * time.Second

// maxStderr is the most stderr that one command keeps, for its error.
const maxStderr = 4 << 10

// commands runs the host's programs.
type commands struct {
	timeout   time.Duration
	waitDelay time.Duration
	limit     int
}

// Commands is the Runner of the real host. Each command runs with
// LC_ALL=C, so that its output is the same in each locale, and stops after
// CommandTimeout.
func Commands() Runner {
	return commands{timeout: CommandTimeout, waitDelay: commandWaitDelay, limit: MaxOutput}
}

// Execute runs name with args, and returns its stdout. Stderr is not data:
// on a failure, the error has it. On a non-zero exit the error wraps the
// *exec.ExitError, and the stdout is returned with it.
func (c commands) Execute(
	ctx context.Context,
	name string,
	args ...string,
) ([]byte, error) {
	runCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	// A command whose stdout passes the limit is stopped at once: its output
	// is discarded, so it would only run on to the timeout.
	stdout := &capped{limit: c.limit, stop: cancel}
	stderr := &capped{limit: maxStderr}
	cmd := exec.CommandContext(runCtx, name, args...)
	cmd.Env = append(os.Environ(), "LC_ALL=C")
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.WaitDelay = c.waitDelay

	err := cmd.Run()
	// A child that kept the pipes open after the command exited is not a
	// failure of the command: its output is complete.
	if errors.Is(err, exec.ErrWaitDelay) {
		err = nil
	}
	out := stdout.Bytes()
	switch {
	case stdout.over:
		return out, fmt.Errorf("exec %s: stdout is more than %d bytes", name, c.limit)
	case err != nil && ctx.Err() == nil && errors.Is(runCtx.Err(), context.DeadlineExceeded):
		return out, fmt.Errorf("exec %s: stopped after %s: %w", name, c.timeout, err)
	case err != nil:
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return out, fmt.Errorf("exec %s: %w: %s", name, err, msg)
		}

		return out, fmt.Errorf("exec %s: %w", name, err)
	}

	return out, nil
}

// capped keeps the first limit bytes written to it, and records that more
// came. It reads all that comes, so that the command does not block on a
// full pipe. When more comes, it calls stop, if stop is set. The buffer is
// a field, not embedded: io.Copy would use an embedded buffer's ReadFrom and
// pass by Write.
type capped struct {
	buf   bytes.Buffer
	limit int
	over  bool
	stop  func()
}

// Write keeps what fits in the limit.
func (c *capped) Write(
	p []byte,
) (int, error) {
	if room := c.limit - c.buf.Len(); room < len(p) {
		if !c.over && c.stop != nil {
			c.stop()
		}
		c.over = true
		if room > 0 {
			c.buf.Write(p[:room])
		}

		return len(p), nil
	}

	return c.buf.Write(p)
}

// Bytes is what was kept.
func (c *capped) Bytes() []byte { return c.buf.Bytes() }

// String is what was kept, as a string.
func (c *capped) String() string { return c.buf.String() }
