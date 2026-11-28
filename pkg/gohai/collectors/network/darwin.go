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
	"fmt"
	"math/bits"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/pkg/gohai/source"
)

// Darwin collects network facts on macOS: each interface of `ifconfig
// -a -v`, with its index, flags, MTU, MAC and addresses; its counters
// from the link rows of `netstat -ibdnW`; and the routes, with the
// default interface and gateway, from `netstat -rn`. Encapsulation and
// OpenVZ alias merging are Linux only.
type Darwin struct {
	base

	Exec source.Runner
}

// NewDarwin returns a Darwin variant that runs the host's ifconfig and
// netstat.
func NewDarwin() *Darwin {
	return &Darwin{Exec: source.Commands()}
}

// Collect returns network Info. With no Exec it gives no fact, as every
// value comes from ifconfig and netstat. A host with no netstat has no
// counters and no routes; a netstat that runs and fails is an error.
func (d *Darwin) Collect(
	ctx context.Context,
	_ collector.PriorResults,
) (any, error) {
	if d.Exec == nil {
		return nil, nil
	}
	out, err := d.Exec.Execute(ctx, "ifconfig", "-a", "-v")
	if err != nil {
		return nil, fmt.Errorf("ifconfig -a -v: %w", err)
	}
	ifs := parseIfconfig(out)
	out, err = d.Exec.Execute(ctx, "netstat", "-ibdnW")
	switch {
	case err == nil:
		links := parseNetstatLinks(out)
		for i := range ifs {
			if l, ok := links[ifs[i].Name]; ok {
				ifs[i].Counters = l
			}
		}
	case !collector.Absent(err):
		return nil, fmt.Errorf("netstat -ibdnW: %w", err)
	}
	sort.SliceStable(ifs, func(i, j int) bool { return ifs[i].Number < ifs[j].Number })
	info := &Info{Interfaces: ifs}
	out, err = d.Exec.Execute(ctx, "netstat", "-rn")
	switch {
	case err == nil:
		applyNetstatRoutes(info, out)
	case !collector.Absent(err):
		return nil, fmt.Errorf("netstat -rn: %w", err)
	}

	return info, nil
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

// applyNetstatRoutes adds the routes of `netstat -rn`: the rows of its
// "Internet:" and "Internet6:" tables, "Destination Gateway Flags Netif".
// The first default route with a gateway (flag G) of each family is the
// default interface and gateway, as Ohai reads it from the same table.
// A gateway's "%en0" zone is dropped, as `ip route` prints none.
func applyNetstatRoutes(
	info *Info,
	out []byte,
) {
	family := ""
	for _, line := range strings.Split(string(out), "\n") {
		switch strings.TrimSpace(line) {
		case "Internet:":
			family = "inet"
			continue
		case "Internet6:":
			family = "inet6"
			continue
		}
		f := strings.Fields(line)
		if family == "" || len(f) < 4 || f[0] == "Destination" {
			continue
		}
		gateway, _, _ := strings.Cut(f[1], "%")
		r := Route{Destination: f[0], Family: family, Gateway: gateway, Interface: f[3]}
		info.Routes = append(info.Routes, r)
		attachToInterface(info, r)
		if r.Destination != "default" || !strings.Contains(f[2], "G") {
			continue
		}
		if family == "inet" && info.DefaultInterface == "" {
			info.DefaultInterface, info.DefaultGateway = r.Interface, r.Gateway
		}
		if family == "inet6" && info.DefaultInet6Interface == "" {
			info.DefaultInet6Interface, info.DefaultInet6Gateway = r.Interface, r.Gateway
		}
	}
}
