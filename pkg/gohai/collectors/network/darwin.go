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

package network

import (
	"context"
	"math/bits"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/executor"
)

// Darwin collects network facts on macOS: each interface of `ifconfig
// -a -v`, with its index, flags, MTU, MAC and addresses, and its
// counters from the link rows of `netstat -ibdnW`. Encapsulation, routes and
// OpenVZ alias merging are Linux only.
type Darwin struct {
	base

	Exec executor.Executor
}

// NewDarwin returns a Darwin variant that runs the host's ifconfig and
// netstat.
func NewDarwin() *Darwin {
	return &Darwin{Exec: executor.New()}
}

// Collect returns network Info.
func (d *Darwin) Collect(
	ctx context.Context,
	_ collector.PriorResults,
) (any, error) {
	out, err := d.Exec.Execute(ctx, "ifconfig", "-a", "-v")
	if err != nil {
		return nil, err
	}
	ifs := parseIfconfig(out)
	if out, err := d.Exec.Execute(ctx, "netstat", "-ibdnW"); err == nil {
		links := parseNetstatLinks(out)
		for i := range ifs {
			if l, ok := links[ifs[i].Name]; ok {
				ifs[i].Counters = l
			}
		}
	}
	sort.SliceStable(ifs, func(i, j int) bool { return ifs[i].Number < ifs[j].Number })

	return &Info{Interfaces: ifs}, nil
}

// ifconfigHeadRE is the first line of an interface in `ifconfig -a`:
// "en0: flags=8863<UP,BROADCAST,RUNNING> mtu 1500 index 14".
var ifconfigHeadRE = regexp.MustCompile(
	`^(\S+): flags=[0-9a-f]+<([^>]*)> mtu (\d+)(?: index (\d+))?`,
)

// parseIfconfig is the interfaces of `ifconfig -a -v`, in its order.
func parseIfconfig(
	out []byte,
) []Interface {
	ifs := []Interface{}
	for _, line := range strings.Split(string(out), "\n") {
		if m := ifconfigHeadRE.FindStringSubmatch(line); m != nil {
			set := map[string]bool{}
			for _, f := range strings.Split(strings.ToLower(m[2]), ",") {
				if f == "pointopoint" {
					f = "pointtopoint"
				}
				set[f] = true
			}
			item := Interface{Name: m[1], Flags: orderFlags(set)}
			item.MTU, _ = strconv.Atoi(m[3])
			item.Number, _ = strconv.Atoi(m[4])
			item.State = stateFromFlags(item.Flags)
			ifs = append(ifs, item)

			continue
		}
		f := strings.Fields(line)
		if len(ifs) == 0 || len(f) < 2 || line[0] != '\t' {
			continue
		}
		cur := &ifs[len(ifs)-1]
		switch f[0] {
		case "ether":
			cur.MAC = f[1]
		case "inet":
			if a, ok := parseAddress(
				f[1] + "/" + strconv.Itoa(hexMaskBits(field(f, "netmask"))),
			); ok {
				cur.Addresses = append(cur.Addresses, a)
			}
		case "inet6":
			ip, _, _ := strings.Cut(f[1], "%")
			cidr := withPrefix(ip)
			if p := field(f, "prefixlen"); p != "" {
				cidr = ip + "/" + p
			}
			if a, ok := parseAddress(cidr); ok {
				cur.Addresses = append(cur.Addresses, a)
			}
		}
	}

	return ifs
}

// field is the word after key in f, or "".
func field(
	f []string,
	key string,
) string {
	for i := 0; i+1 < len(f); i++ {
		if f[i] == key {
			return f[i+1]
		}
	}

	return ""
}

// hexMaskBits is the prefix length of a netmask as ifconfig prints it,
// "0xffffff00", or 32 when it has none or is not hex.
func hexMaskBits(
	mask string,
) int {
	n, err := strconv.ParseUint(strings.TrimPrefix(mask, "0x"), 16, 32)
	if err != nil {
		return 32
	}

	return bits.OnesCount32(uint32(n))
}

// netstatLinkRE is the network column of a link row: "<Link#14>".
var netstatLinkRE = regexp.MustCompile(`^<Link#\d+>$`)

// parseNetstatLinks is the counters of the link rows of `netstat
// -ibdnW`, by interface. A row is the name (with a "*" when the interface
// is down), the MTU, "<Link#N>", the MAC when the interface has
// one, then Ipkts Ierrs Ibytes Opkts Oerrs Obytes Coll Drop. Drop is
// output drops, as gopsutil read it.
func parseNetstatLinks(
	out []byte,
) map[string]*Counters {
	links := map[string]*Counters{}
	for _, line := range strings.Split(string(out), "\n") {
		f := strings.Fields(line)
		if len(f) < 11 {
			continue
		}
		if !netstatLinkRE.MatchString(f[2]) {
			continue
		}
		n := func(i int) uint64 {
			v, _ := strconv.ParseUint(f[len(f)-8+i], 10, 64)
			return v
		}
		links[strings.TrimSuffix(f[0], "*")] = &Counters{
			PacketsRecv: n(0), Errin: n(1), BytesRecv: n(2),
			PacketsSent: n(3), Errout: n(4), BytesSent: n(5), Dropout: n(7),
		}
	}

	return links
}
