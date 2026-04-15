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

// Package network reports network interfaces, structured per-address
// data (family / prefix / scope / netmask / broadcast), per-interface
// I/O counters, the routing table, and top-level default interface +
// gateway facts (v4 + v6). On Linux we additionally derive the
// canonical encapsulation name from sysfs ARPHRD types and merge
// OpenVZ `venet0:N` aliases under the primary `venet0` interface.
package network

import (
	"context"
	"net"
	"strconv"
	"strings"

	"github.com/jaypipes/ghw"
	gpnet "github.com/shirou/gopsutil/v4/net"
	"github.com/vishvananda/netlink"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/platform"
)

// Info holds network interfaces plus top-level routing + neighbour
// facts.
type Info struct {
	Interfaces            []Interface `json:"interfaces"`
	Routes                []Route     `json:"routes,omitempty"`
	Neighbours            []Neighbour `json:"neighbours,omitempty"`
	DefaultInterface      string      `json:"default_interface,omitempty"`
	DefaultGateway        string      `json:"default_gateway,omitempty"`
	DefaultInet6Interface string      `json:"default_inet6_interface,omitempty"`
	DefaultInet6Gateway   string      `json:"default_inet6_gateway,omitempty"`
}

// Interface describes a single network interface.
type Interface struct {
	Name          string       `json:"name"`
	Number        int          `json:"number,omitempty"` // kernel interface index (Ohai: iface[:number])
	State         string       `json:"state,omitempty"`  // admin state: "up" | "down" (Ohai: iface["state"])
	MTU           int          `json:"mtu"`
	HardwareAddr  string       `json:"hardware_addr,omitempty"` // OCSF: network_interface.mac
	Encapsulation string       `json:"encapsulation,omitempty"` // canonical: Ethernet / Loopback / PPP / SLIP / IPIP / 6to4
	Driver        string       `json:"driver,omitempty"`        // sysfs driver name (e1000e, virtio_net, ixgbe, ...)
	Speed         string       `json:"speed,omitempty"`         // ghw link speed string ("1000Mb/s")
	Duplex        string       `json:"duplex,omitempty"`        // half | full | unknown
	Flags         []string     `json:"flags,omitempty"`
	Addresses     []Address    `json:"addresses,omitempty"`
	Routes        []Route      `json:"routes,omitempty"`
	Counters      *Counters    `json:"counters,omitempty"`
	Ethtool       *EthtoolInfo `json:"ethtool,omitempty"` // Linux only, when ethtool binary is on PATH
}

// EthtoolInfo holds data sourced from `ethtool` subcommands per
// interface. Mirrors Ohai's six per-interface ethtool calls:
// driver_info, ring_params, channel_params, coalesce_params,
// offload_params, pause_params.
//
// Populated only on Linux for interfaces whose Encapsulation is
// "Ethernet" (matching Ohai's `iface[:encapsulation] == "Ethernet"`
// gate). Hosts without the ethtool binary leave Ethtool nil. Per-
// subcommand failures are independent — one ethtool subcommand
// erroring out doesn't suppress the others.
type EthtoolInfo struct {
	// DriverInfo mirrors `ethtool -i <iface>` output as a map. Common
	// keys: driver, version, firmware_version, bus_info,
	// supports_statistics, supports_test, supports_eeprom_access,
	// supports_register_dump, supports_priv_flags. Keys are normalized
	// to snake_case (Ohai only replaces spaces; we additionally
	// translate hyphens — `firmware-version` → `firmware_version` —
	// for Go-idiomatic consistency).
	DriverInfo map[string]string `json:"driver_info,omitempty"`

	// RingParams mirrors `ethtool -g <iface>` output. Keys are
	// prefixed with `max_` (from "Pre-set maximums") or `current_`
	// (from "Current hardware settings") then the ethtool field name
	// snake_cased — e.g. `max_rx`, `current_rx_jumbo`, `max_tx`.
	// Values are integers (buffer descriptor counts).
	RingParams map[string]int `json:"ring_params,omitempty"`

	// ChannelParams mirrors `ethtool -l <iface>` output with the
	// same `max_` / `current_` prefix convention. Keys: `rx`, `tx`,
	// `other`, `combined`. Values are integers (queue counts).
	ChannelParams map[string]int `json:"channel_params,omitempty"`

	// CoalesceParams mirrors `ethtool -c <iface>` output. Most
	// values are integers (microseconds, frame counts) — `rx_usecs`,
	// `tx_max_coalesced_frames`, etc. The exception is the
	// `Adaptive RX: on  TX: off` line which Ohai parses into two
	// string entries `adaptive_rx` and `adaptive_tx` carrying
	// "on"/"off". To preserve that mixed-type Ohai shape we use
	// `any` here; JSON serializes integers and strings cleanly.
	CoalesceParams map[string]any `json:"coalesce_params,omitempty"`

	// OffloadParams mirrors `ethtool -k <iface>` output. Values are
	// strings ("on" / "off"); Ohai strips the trailing
	// `[fixed]` / `[requested ...]` annotations.
	OffloadParams map[string]string `json:"offload_params,omitempty"`

	// PauseParams mirrors `ethtool -a <iface>` output. Values are
	// booleans — Ohai converts the "on"/"off" strings via
	// `.eql? "on"`. Keys: `autonegotiate`, `rx`, `tx`.
	PauseParams map[string]bool `json:"pause_params,omitempty"`
}

// Neighbour is one entry from the ARP / NDP cache.
type Neighbour struct {
	Address   string `json:"address"`             // IPv4 / IPv6
	Family    string `json:"family"`              // inet | inet6
	MAC       string `json:"mac,omitempty"`       // hardware address
	Interface string `json:"interface,omitempty"` // egress interface
	State     string `json:"state,omitempty"`     // REACHABLE / STALE / DELAY / PROBE / PERMANENT / NOARP
}

// Address represents a single IP bound to an interface, structured
// the way Ohai emits it: family, prefix length, netmask (IPv4),
// broadcast (IPv4), scope.
type Address struct {
	Addr      string `json:"addr"`
	Family    string `json:"family"`              // inet | inet6
	Prefixlen int    `json:"prefixlen"`           // 24, 64, ...
	Netmask   string `json:"netmask,omitempty"`   // IPv4 only
	Broadcast string `json:"broadcast,omitempty"` // IPv4 only
	Scope     string `json:"scope,omitempty"`     // Global | Link | Host | Site
}

// Route is one entry from the kernel routing table.
type Route struct {
	Destination string `json:"destination"`
	Family      string `json:"family"`
	Gateway     string `json:"gateway,omitempty"`
	Interface   string `json:"interface,omitempty"`
	Source      string `json:"source,omitempty"`
	Scope       string `json:"scope,omitempty"`
	Proto       string `json:"proto,omitempty"`
	Metric      int    `json:"metric,omitempty"`
}

// Counters holds I/O counters for one interface.
type Counters struct {
	BytesSent   uint64 `json:"bytes_sent"`
	BytesRecv   uint64 `json:"bytes_recv"`
	PacketsSent uint64 `json:"packets_sent"`
	PacketsRecv uint64 `json:"packets_recv"`
	Errin       uint64 `json:"errin,omitempty"`
	Errout      uint64 `json:"errout,omitempty"`
	Dropin      uint64 `json:"dropin,omitempty"`
	Dropout     uint64 `json:"dropout,omitempty"`
}

// Collector is the public interface every network variant satisfies.
type Collector interface {
	collector.Collector
}

type base struct{}

func (base) Name() string           { return "network" }
func (base) Category() string       { return collector.CategoryNetwork }
func (base) DefaultEnabled() bool   { return true }
func (base) Dependencies() []string { return nil }

// New returns the network variant for the host OS.
func New() Collector {
	if platform.Detect() == "darwin" {
		return NewDarwin()
	}
	return NewLinux()
}

// Package-level injection seams for gopsutil (kept private so
// importers don't transitively need gopsutil).
var (
	interfacesFn = gpnet.InterfacesWithContext
	ioCountersFn = gpnet.IOCountersWithContext
)

// readInterfaces is the production bridge to gopsutil. Enumerates
// interfaces, structures each address, and merges per-interface I/O
// counters (matched by name).
func readInterfaces(
	ctx context.Context,
) ([]Interface, error) {
	ifs, err := interfacesFn(ctx)
	if err != nil {
		return nil, err
	}
	counts, _ := ioCountersFn(ctx, true)
	countersByName := map[string]*Counters{}
	for _, c := range counts {
		countersByName[c.Name] = &Counters{
			BytesSent: c.BytesSent, BytesRecv: c.BytesRecv,
			PacketsSent: c.PacketsSent, PacketsRecv: c.PacketsRecv,
			Errin: c.Errin, Errout: c.Errout,
			Dropin: c.Dropin, Dropout: c.Dropout,
		}
	}
	out := make([]Interface, 0, len(ifs))
	for _, i := range ifs {
		item := Interface{
			Name:         i.Name,
			Number:       i.Index,
			State:        stateFromFlags(i.Flags),
			MTU:          i.MTU,
			HardwareAddr: i.HardwareAddr,
			Flags:        i.Flags,
		}
		for _, a := range i.Addrs {
			if addr, ok := parseAddress(a.Addr); ok {
				item.Addresses = append(item.Addresses, addr)
			}
		}
		if c, ok := countersByName[i.Name]; ok {
			item.Counters = c
		}
		out = append(out, item)
	}
	return out, nil
}

// parseAddress turns a CIDR string into a structured Address.
// gopsutil emits everything as `<ip>/<prefix>`; non-parseable input
// (defensive) returns ok=false and is skipped by the caller.
func parseAddress(
	cidr string,
) (Address, bool) {
	ip, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return Address{}, false
	}
	ones, _ := ipnet.Mask.Size()
	addr := Address{Addr: ip.String(), Prefixlen: ones}
	if v4 := ip.To4(); v4 != nil {
		addr.Family = "inet"
		addr.Netmask = net.IP(net.CIDRMask(ones, 32)).String()
		addr.Broadcast = ipv4Broadcast(v4, ones)
	} else {
		addr.Family = "inet6"
	}
	addr.Scope = scopeOf(ip)
	return addr, true
}

// ipv4Broadcast computes the broadcast address for the given IPv4 +
// prefix length.
func ipv4Broadcast(
	ip net.IP,
	prefixlen int,
) string {
	mask := net.CIDRMask(prefixlen, 32)
	bcast := make(net.IP, 4)
	for i := 0; i < 4; i++ {
		bcast[i] = ip[i] | ^mask[i]
	}
	return bcast.String()
}

// scopeOf classifies an IP into Ohai's title-cased scope buckets.
// Multicast classifications are intentionally collapsed into the
// nearest single answer — Ohai uses Global for routable, Link for
// link-local-anything, Host for loopback. We don't emit Site (no
// caller needs IPv6 site-local distinction in 2026).
func scopeOf(
	ip net.IP,
) string {
	switch {
	case ip.IsLoopback():
		return "Host"
	case ip.IsLinkLocalUnicast(), ip.IsLinkLocalMulticast():
		return "Link"
	}
	return "Global"
}

// parseEthtoolDriverInfo turns `ethtool -i <iface>` output into the
// DriverInfo map. Mirrors Ohai's ethernet_driver_info parse: split
// each line on `<key>: <value>`, normalize the key to snake_case
// (Ohai replaces only spaces; we additionally replace hyphens since
// real ethtool keys like `firmware-version` and `bus-info` would
// otherwise pollute Go consumers with non-idiomatic identifiers).
//
// Empty lines and lines without a colon are skipped. Trailing
// whitespace on values is trimmed (matches Ohai's `.chomp`).
func parseEthtoolDriverInfo(
	raw []byte,
) map[string]string {
	out := map[string]string{}
	for _, line := range strings.Split(string(raw), "\n") {
		key, val, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		key = strings.ToLower(key)
		key = strings.ReplaceAll(key, " ", "_")
		key = strings.ReplaceAll(key, "-", "_")
		out[key] = strings.TrimSpace(val)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// parseEthtoolSectionedInts parses ethtool subcommand output that
// has two sections — "Pre-set maximums:" and "Current hardware
// settings:" — into a flat map keyed `max_<field>` / `current_<field>`
// with integer values. Used for both ring_params (`ethtool -g`) and
// channel_params (`ethtool -l`); Ohai's ethernet_ring_parameters and
// ethernet_channel_parameters share this exact shape.
//
// headerPrefix is the per-iface header line ethtool prints first
// ("Ring parameters for" / "Channel parameters for") which we skip.
func parseEthtoolSectionedInts(
	raw []byte,
	headerPrefix string,
) map[string]int {
	out := map[string]int{}
	section := ""
	for _, line := range strings.Split(string(raw), "\n") {
		if strings.HasPrefix(line, headerPrefix) {
			continue
		}
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		switch {
		case strings.HasPrefix(trimmed, "Pre-set maximums"):
			section = "max"
			continue
		case strings.HasPrefix(trimmed, "Current hardware settings"):
			section = "current"
			continue
		}
		if section == "" {
			continue
		}
		key, val, ok := strings.Cut(trimmed, ":")
		if !ok {
			continue
		}
		val = strings.TrimSpace(val)
		if val == "" {
			continue
		}
		n, err := strconv.Atoi(val)
		if err != nil {
			continue
		}
		out[section+"_"+ethtoolKey(key)] = n
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// parseEthtoolCoalesceParams parses `ethtool -c <iface>` output. The
// `Adaptive RX: on  TX: off` line splits into two string entries
// (adaptive_rx / adaptive_tx); every other line is a `key: value`
// pair where value is parsed as an integer. Mirrors Ohai's
// ethernet_coalesce_parameters exactly, including the special
// Adaptive handling.
func parseEthtoolCoalesceParams(
	raw []byte,
) map[string]any {
	out := map[string]any{}
	for _, line := range strings.Split(string(raw), "\n") {
		if strings.HasPrefix(line, "Coalesce parameters for") {
			continue
		}
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "Adaptive") {
			rx, tx, ok := parseAdaptiveLine(trimmed)
			if ok {
				out["adaptive_rx"] = rx
				out["adaptive_tx"] = tx
			}
			continue
		}
		key, val, ok := strings.Cut(trimmed, ":")
		if !ok {
			continue
		}
		val = strings.TrimSpace(val)
		if val == "" {
			continue
		}
		n, err := strconv.Atoi(val)
		if err != nil {
			continue
		}
		out[ethtoolKey(key)] = n
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// parseAdaptiveLine pulls the two on/off tokens out of an ethtool
// "Adaptive RX: on  TX: off" line. Returns ok=false when the shape
// doesn't match (so the caller skips a malformed line cleanly).
func parseAdaptiveLine(
	line string,
) (rx, tx string, ok bool) {
	rest := strings.TrimSpace(strings.TrimPrefix(line, "Adaptive"))
	rxRaw, txPart, ok := strings.Cut(rest, "TX:")
	if !ok {
		return "", "", false
	}
	_, rxVal, ok := strings.Cut(rxRaw, ":")
	if !ok {
		return "", "", false
	}
	return strings.TrimSpace(rxVal), strings.TrimSpace(txPart), true
}

// parseEthtoolOffloadParams parses `ethtool -k <iface>` output. Each
// non-header line is `feature: state[ annotation]`. Ohai lowercases
// the value and strips bracketed annotations like `[fixed]` or
// `[requested on]`. We mirror that exactly — the canonical state
// (`on` / `off`) is what consumers want; the annotation is noise.
func parseEthtoolOffloadParams(
	raw []byte,
) map[string]string {
	out := map[string]string{}
	for _, line := range strings.Split(string(raw), "\n") {
		if strings.HasPrefix(line, "Features for") {
			continue
		}
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		key, val, ok := strings.Cut(trimmed, ":")
		if !ok {
			continue
		}
		val = strings.ToLower(strings.TrimSpace(val))
		if i := strings.Index(val, "["); i >= 0 {
			val = strings.TrimSpace(val[:i])
		}
		if val == "" {
			continue
		}
		out[ethtoolKey(key)] = val
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// parseEthtoolPauseParams parses `ethtool -a <iface>` output. Values
// are booleans — Ohai's `.eql? "on"` truthy-check, lowercased.
func parseEthtoolPauseParams(
	raw []byte,
) map[string]bool {
	out := map[string]bool{}
	for _, line := range strings.Split(string(raw), "\n") {
		if strings.HasPrefix(line, "Pause parameters for") {
			continue
		}
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		key, val, ok := strings.Cut(trimmed, ":")
		if !ok {
			continue
		}
		val = strings.TrimSpace(val)
		if val == "" {
			continue
		}
		out[ethtoolKey(key)] = strings.EqualFold(val, "on")
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// ethtoolKey normalizes an ethtool field label to snake_case by
// lowercasing and replacing both spaces and hyphens with underscores.
// Same convention as parseEthtoolDriverInfo.
func ethtoolKey(
	k string,
) string {
	k = strings.TrimSpace(strings.ToLower(k))
	k = strings.ReplaceAll(k, " ", "_")
	k = strings.ReplaceAll(k, "-", "_")
	return k
}

// stateFromFlags returns the admin state label (`"up"` / `"down"`)
// mirroring Ohai's `iface["state"]` (which reads from `ip link show`).
// gopsutil lowercases the `up` flag, so a simple membership check is
// enough; absence means the kernel has the interface admin-down.
func stateFromFlags(
	flags []string,
) string {
	for _, f := range flags {
		if strings.EqualFold(f, "up") {
			return "up"
		}
	}
	return "down"
}

// arphrdEncapsulation maps the ARPHRD_* integer (read from
// /sys/class/net/<iface>/type) to Ohai's canonical encapsulation
// name.
//
// Source: https://github.com/torvalds/linux/blob/master/include/uapi/linux/if_arp.h
var arphrdEncapsulation = map[int]string{
	1:   "Ethernet", // ARPHRD_ETHER
	24:  "PPP",      // ARPHRD_PPP (kernel uses 512 — keep both)
	512: "PPP",
	256: "SLIP", // ARPHRD_SLIP
	257: "VJSLIP",
	768: "IPIP", // ARPHRD_TUNNEL
	769: "6to4", // ARPHRD_TUNNEL6
	772: "Loopback",
}

// isOpenVZAlias reports whether name looks like `<base>:<n>` — the
// venet0 alias pattern OpenVZ guests use.
func isOpenVZAlias(
	name string,
) (string, bool) {
	if i := strings.Index(name, ":"); i > 0 {
		if _, err := strconv.Atoi(name[i+1:]); err == nil {
			return name[:i], true
		}
	}
	return "", false
}

// NICStat captures the per-interface link-layer fields we surface on
// `Interface.{Driver, Speed, Duplex}`. Speed is a string (matching
// ghw's `"1000Mb/s"` shape) so we can preserve units / "Unknown!".
// Exported so tests can stub the probe via SetNICFn.
type NICStat struct {
	Driver string
	Speed  string
	Duplex string
}

// Package-level seams. Production uses ghw/net for Speed + Duplex
// and a sysfs read for Driver; vishvananda/netlink for the kernel
// ARP+NDP cache. Both libraries compile cross-platform; their darwin
// code paths return errors at runtime, leaving the relevant fields
// blank.
//
// nicFn / neighListFn are the high-level seams collectors use. The
// inner upstream calls (ghwNetworkFn / netlinkNeighListFn /
// netInterfaceByIndex) are also private vars so tests can swap them
// in isolation when exercising readNIC / readNeighbours directly.
var (
	nicFn               = readNIC
	neighListFn         = readNeighbours
	ghwNetworkFn        = ghw.Network
	netlinkNeighListFn  = netlink.NeighList
	netInterfaceByIndex = net.InterfaceByIndex
)

// readNIC asks ghw for the link layer's Speed + Duplex per
// interface name. Driver comes from a separate sysfs read in
// applyNICStats (it's avfs-injectable so tests cover it without
// touching ghw). Returns a name → NICStat map so a single ghw call
// services every interface.
func readNIC() (map[string]NICStat, error) {
	info, err := ghwNetworkFn()
	if err != nil {
		return nil, err
	}
	return nicMapFromGHW(info.NICs), nil
}

// nicMapFromGHW is the pure conversion from ghw's []*NIC to our
// name → NICStat map. Extracted so tests cover the mapping without
// touching the host (readNIC itself stays trivial).
func nicMapFromGHW(
	nics []*ghw.NIC,
) map[string]NICStat {
	out := map[string]NICStat{}
	for _, n := range nics {
		if n == nil {
			continue
		}
		out[n.Name] = NICStat{Speed: n.Speed, Duplex: n.Duplex}
	}
	return out
}

// readNeighbours queries the kernel ARP + NDP cache via netlink and
// returns one Neighbour per entry. The pure conversion lives in
// neighboursFromNetlink so tests can drive it without netlink.
func readNeighbours() ([]Neighbour, error) {
	entries, err := netlinkNeighListFn(0, 0)
	if err != nil {
		return nil, err
	}
	return neighboursFromNetlink(entries, indexToInterfaceName), nil
}

// indexToInterfaceName resolves an interface index via stdlib's
// net.InterfaceByIndex (swappable as netInterfaceByIndex).
func indexToInterfaceName(
	idx int,
) string {
	if iface, err := netInterfaceByIndex(idx); err == nil {
		return iface.Name
	}
	return ""
}

// neighboursFromNetlink is the pure conversion from netlink.Neigh
// to our Neighbour. Takes an indexToName resolver so tests can
// avoid touching the host.
func neighboursFromNetlink(
	entries []netlink.Neigh,
	indexToName func(int) string,
) []Neighbour {
	out := make([]Neighbour, 0, len(entries))
	for _, e := range entries {
		nb := Neighbour{
			Address:   e.IP.String(),
			Family:    neighFamily(e.Family),
			Interface: indexToName(e.LinkIndex),
			State:     neighState(e.State),
		}
		if e.HardwareAddr != nil {
			nb.MAC = e.HardwareAddr.String()
		}
		out = append(out, nb)
	}
	return out
}

// neighFamily turns the netlink AF_* integer into our string label.
func neighFamily(
	fam int,
) string {
	switch fam {
	case 2: // AF_INET
		return "inet"
	case 10: // AF_INET6
		return "inet6"
	}
	return ""
}

// neighState turns netlink's NUD_* bitmask into the canonical
// ip-neigh string. Mirrors `ip neigh show`'s human output. Unknown
// values fall through to a hex-formatted string.
func neighState(
	state int,
) string {
	switch state {
	case 0x01:
		return "INCOMPLETE"
	case 0x02:
		return "REACHABLE"
	case 0x04:
		return "STALE"
	case 0x08:
		return "DELAY"
	case 0x10:
		return "PROBE"
	case 0x20:
		return "FAILED"
	case 0x40:
		return "NOARP"
	case 0x80:
		return "PERMANENT"
	}
	return ""
}
