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

package gohai

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/alibaba"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/azure"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/blockdevice"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/cacertificates"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/command"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/cpu"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/digitalocean"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/disk"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/dmi"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/dns"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/docker"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/ec2"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/filesystem"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/fips"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/gce"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/gpu"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/grub2"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/hardware"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/hostname"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/hostnamectl"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/initd"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/interrupts"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/ipc"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/kernel"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/kernelmodules"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/languages"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/libvirt"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/linode"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/livepatch"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/load"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/lsb"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/machineid"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/mdadm"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/memory"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/network"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/oci"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/openstack"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/osrelease"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/packagemgr"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/packages"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/pci"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/platform"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/process"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/rootgroup"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/rpm"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/scaleway"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/scsi"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/selinux"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/services"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/sessions"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/shard"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/shells"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/ssh"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/sysconf"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/sysctl"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/systemdpaths"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/tc"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/timezone"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/uptime"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/users"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/virtualbox"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/virtualization"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/vmware"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/zpools"
	"github.com/osapi-io/gohai/pkg/gohai/source"
)

// Gohai is the SDK entry point for collecting system facts.
type Gohai struct {
	registry    *collector.Registry
	selected    []collector.Collector
	withTimings bool
}

// New constructs a Gohai instance with the given options.
func New(
	opts ...Option,
) (*Gohai, error) {
	var cfg config
	for _, opt := range opts {
		opt(&cfg)
	}

	// Each of the collectors picks its variant by the family, so find it
	// once, not once for each collector.
	cfg.sources.Family = collector.Family(cfg.sources)
	g := &Gohai{registry: collector.NewRegistry()}
	if err := registerBuiltins(g.registry, cfg.sources); err != nil {
		return nil, err
	}

	sel, err := selectCollectors(g.registry, cfg)
	if err != nil {
		return nil, err
	}
	g.selected = sel
	g.withTimings = cfg.withTimings
	return g, nil
}

// Collect runs all selected collectors and returns Facts. Each collector's
// typed result is written to the matching field on Facts; collectors that
// error or aren't selected leave their field nil, and Facts.Errors has the
// error of each that failed. A cancelled ctx is an error. When the instance
// was built with WithTimings, per-collector durations are embedded in
// Facts.Timings.
func (g *Gohai) Collect(
	ctx context.Context,
) (*Facts, error) {
	names := make([]string, 0, len(g.selected))
	for _, c := range g.selected {
		names = append(names, c.Name())
	}
	start := time.Now()

	var (
		samplesMu sync.Mutex
		samples   map[string]CollectorTiming
		errorsMu  sync.Mutex
		errs      map[string]string
	)
	hooks := collector.Hooks{
		OnError: func(name string, err error) {
			errorsMu.Lock()
			defer errorsMu.Unlock()
			if errs == nil {
				errs = make(map[string]string)
			}
			errs[name] = err.Error()
		},
	}
	if g.withTimings {
		samples = make(map[string]CollectorTiming, len(names))
		hooks.OnComplete = func(name string, dur time.Duration, err error) {
			entry := CollectorTiming{
				DurationNs: dur.Nanoseconds(),
				Status:     "ok",
			}
			if err != nil {
				entry.Status = "err"
				entry.Error = err.Error()
			}
			samplesMu.Lock()
			samples[name] = entry
			samplesMu.Unlock()
		}
	}

	results, err := g.registry.Run(ctx, names, hooks)
	if err != nil {
		return nil, fmt.Errorf("run collectors: %w", err)
	}

	facts := &Facts{
		CollectTime:     start,
		CollectDuration: time.Since(start),
		Errors:          errs,
	}
	if g.withTimings {
		facts.Timings = &Timings{Collectors: samples}
	}
	for name, result := range results {
		if result == nil || facts.set(name, result) {
			continue
		}
		// A collector that gives a type its Facts field doesn't hold would
		// lose its fact with no word, so the mismatch is its error.
		if facts.Errors == nil {
			facts.Errors = make(map[string]string)
		}
		facts.Errors[name] = fmt.Sprintf("gave a %T, which its Facts field does not hold", result)
	}
	return facts, nil
}

func selectCollectors(
	reg *collector.Registry,
	cfg config,
) ([]collector.Collector, error) {
	// WithCollectors wins outright: an explicit roster, no defaults. It
	// names each collector, so a disabled list with it is a mistake.
	if len(cfg.only) > 0 {
		if len(cfg.disabled) > 0 {
			return nil, fmt.Errorf(
				"WithCollectors and WithDisabled together: name only the collectors to run",
			)
		}
		out := make([]collector.Collector, 0, len(cfg.only))
		for _, n := range cfg.only {
			c, ok := reg.Get(n)
			if !ok {
				return nil, fmt.Errorf("unknown collector %q", n)
			}
			out = append(out, c)
		}
		return out, nil
	}
	// Expand WithCategory into an additive enable list. Unknown
	// categories (zero collectors match) error so typos surface
	// immediately rather than silently selecting nothing.
	enabled := cfg.enabled
	for _, cat := range cfg.categories {
		names := reg.NamesInCategory(collector.Category(cat))
		if len(names) == 0 {
			return nil, fmt.Errorf("unknown category %q", cat)
		}
		enabled = append(enabled, names...)
	}
	// Without WithDefaults / WithEnabled / WithCategory, return
	// empty, because gohai.New() is opt-in. Still validate the disabled
	// list so unknown names error consistently.
	if !cfg.useDefaults && len(enabled) == 0 {
		if _, err := reg.SelectedWith(false, nil, cfg.disabled); err != nil {
			return nil, fmt.Errorf("select collectors: %w", err)
		}
		return nil, nil
	}
	sel, err := reg.SelectedWith(cfg.useDefaults, enabled, cfg.disabled)
	if err != nil {
		return nil, fmt.Errorf("select collectors: %w", err)
	}
	return sel, nil
}

// registerBuiltins registers every built-in collector. A registration error
// is a duplicate or empty name, or a source field of the wrong type, in
// builtinCollectors: a bug that would drop a collector with no signal, so the
// caller returns it.
func registerBuiltins(
	reg *collector.Registry,
	src source.Sources,
) error {
	for _, c := range builtinCollectorsFn(src) {
		if err := reg.Register(c); err != nil {
			return fmt.Errorf("register built-in collector %q: %w", c.Name(), err)
		}
	}

	return nil
}

// builtinCollectorsFn is builtinCollectors, which a test replaces.
var builtinCollectorsFn = builtinCollectors

// builtinCollectors returns the built-in collectors to register, each
// reading the host through src. The cloud collectors read their provider's
// metadata service, which src does not cover.
func builtinCollectors(
	src source.Sources,
) []collector.Collector {
	return []collector.Collector{
		platform.NewWith(src),
		hostname.NewWith(src),
		kernel.NewWith(src),
		kernelmodules.NewWith(src),
		uptime.NewWith(src),
		virtualization.NewWith(src),
		machineid.NewWith(src),
		cpu.NewWith(src),
		load.NewWith(src),
		memory.NewWith(src),
		filesystem.NewWith(src),
		disk.NewWith(src),
		network.NewWith(src),
		process.NewWith(src),
		users.NewWith(src),
		sessions.NewWith(src),
		timezone.NewWith(src),
		dns.NewWith(src),
		rootgroup.NewWith(src),
		shells.NewWith(src),
		fips.NewWith(src),
		osrelease.NewWith(src),
		lsb.NewWith(src),
		initd.NewWith(src),
		shard.NewWith(src),
		packagemgr.NewWith(src),
		gce.NewWith(src),
		ec2.NewWith(src),
		azure.NewWith(src),
		digitalocean.NewWith(src),
		oci.NewWith(src),
		alibaba.NewWith(src),
		linode.NewWith(src),
		openstack.NewWith(src),
		scaleway.NewWith(src),
		dmi.NewWith(src),
		gpu.NewWith(src),
		pci.NewWith(src),
		scsi.NewWith(src),
		hardware.NewWith(src),
		blockdevice.NewWith(src),
		command.NewWith(src),
		docker.NewWith(src),
		grub2.NewWith(src),
		hostnamectl.NewWith(src),
		interrupts.NewWith(src),
		ipc.NewWith(src),
		languages.NewWith(src),
		libvirt.NewWith(src),
		livepatch.NewWith(src),
		mdadm.NewWith(src),
		packages.NewWith(src),
		rpm.NewWith(src),
		selinux.NewWith(src),
		services.NewWith(src),
		ssh.NewWith(src),
		sysconf.NewWith(src),
		sysctl.NewWith(src),
		systemdpaths.NewWith(src),
		tc.NewWith(src),
		virtualbox.NewWith(src),
		vmware.NewWith(src),
		zpools.NewWith(src),
		cacertificates.NewWith(src),
	}
}
