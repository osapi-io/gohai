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

package dns

import (
	"context"
	"regexp"
	"strings"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/osfs"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/executor"
)

// Darwin reads /etc/resolv.conf through FS, and each interface's
// resolver from `scutil --dns`.
type Darwin struct {
	base

	FS   avfs.VFS
	Exec executor.Executor
}

// NewDarwin returns a Darwin variant that reads the host.
func NewDarwin() *Darwin {
	return &Darwin{FS: osfs.NewWithNoIdm(), Exec: executor.New()}
}

var (
	scutilNameserverRE = regexp.MustCompile(`^nameserver\[\d+\]\s*:\s*(\S+)`)
	scutilSearchRE     = regexp.MustCompile(`^search domain\[\d+\]\s*:\s*(\S+)`)
	scutilIfIndexRE    = regexp.MustCompile(`^if_index\s*:\s*\d+\s*\(([^)]+)\)`)
)

// Collect returns the dns Info.
func (d *Darwin) Collect(
	ctx context.Context,
	_ collector.PriorResults,
) (any, error) {
	info := &Info{}
	if b, err := d.FS.ReadFile(resolvConf); err == nil {
		parseResolvConf(info, b)
	}
	if d.Exec == nil {
		return info, nil
	}
	out, err := d.Exec.Execute(ctx, "scutil", "--dns")
	if err != nil {
		return info, nil
	}
	// Each "resolver #N" block has its servers, its search domains and,
	// when it belongs to one, its interface. The first block of an
	// interface wins: scutil prints the scoped resolvers again after.
	var servers, domains []string
	flush := func(iface string) {
		if _, seen := info.Interfaces[iface]; iface != "" && !seen {
			setResolver(info, iface, servers, domains)
		}
		servers, domains = nil, nil
	}
	iface := ""
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "resolver #"):
			flush(iface)
			iface = ""
		case scutilNameserverRE.MatchString(line):
			servers = append(servers, scutilNameserverRE.FindStringSubmatch(line)[1])
		case scutilSearchRE.MatchString(line):
			domains = append(domains, scutilSearchRE.FindStringSubmatch(line)[1])
		case scutilIfIndexRE.MatchString(line):
			iface = scutilIfIndexRE.FindStringSubmatch(line)[1]
		}
	}
	flush(iface)

	return info, nil
}
