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
	blockdevice "github.com/osapi-io/gohai/pkg/gohai/collectors/block_device"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/command"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/cpu"
	digitalocean "github.com/osapi-io/gohai/pkg/gohai/collectors/digital_ocean"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/disk"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/dmi"
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
	initd "github.com/osapi-io/gohai/pkg/gohai/collectors/init"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/interrupts"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/ipc"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/kernel"
	kernelmodules "github.com/osapi-io/gohai/pkg/gohai/collectors/kernel_modules"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/languages"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/libvirt"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/linode"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/livepatch"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/load"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/lsb"
	machineid "github.com/osapi-io/gohai/pkg/gohai/collectors/machine_id"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/mdadm"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/memory"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/network"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/oci"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/openstack"
	osrelease "github.com/osapi-io/gohai/pkg/gohai/collectors/os_release"
	packagemgr "github.com/osapi-io/gohai/pkg/gohai/collectors/package_mgr"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/packages"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/pci"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/platform"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/process"
	rootgroup "github.com/osapi-io/gohai/pkg/gohai/collectors/root_group"
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
	systemdpaths "github.com/osapi-io/gohai/pkg/gohai/collectors/systemd_paths"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/tc"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/timezone"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/uptime"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/users"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/virtualbox"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/virtualization"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/vmware"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/zpools"
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
	g := &Gohai{registry: collector.NewRegistry()}
	registerBuiltins(g.registry)

	var cfg config
	for _, opt := range opts {
		opt(&cfg)
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
// error or aren't selected leave their field nil. When the instance was
// built with WithTimings, per-collector durations and error messages are
// embedded in Facts.Timings.
func (g *Gohai) Collect(
	ctx context.Context,
) (*Facts, error) {
	names := make([]string, 0, len(g.selected))
	for _, c := range g.selected {
		names = append(names, c.Name())
	}

	start := time.Now()

	timings, hooks := g.timingHooks(len(names))

	results, err := g.registry.Run(ctx, names, hooks)
	if err != nil {
		return nil, fmt.Errorf("run collectors: %w", err)
	}

	facts := &Facts{
		CollectTime:     start,
		CollectDuration: time.Since(start),
	}

	if timings != nil {
		facts.Timings = &Timings{Collectors: timings.samples}
	}

	for name, result := range results {
		facts.set(name, result)
	}

	return facts, nil
}

// timingRecorder gathers per-collector timings from the run hooks.
type timingRecorder struct {
	mu      sync.Mutex
	samples map[string]CollectorTiming
}

// record stores one collector's outcome.
func (t *timingRecorder) record(
	name string,
	dur time.Duration,
	err error,
) {
	entry := CollectorTiming{
		DurationNs: dur.Nanoseconds(),
		Status:     "ok",
	}

	if err != nil {
		entry.Status = "err"
		entry.Error = err.Error()
	}

	t.mu.Lock()
	t.samples[name] = entry
	t.mu.Unlock()
}

// timingHooks returns a recorder and the hooks that feed it, or nil
// hooks when timings were not asked for.
func (g *Gohai) timingHooks(
	n int,
) (*timingRecorder, collector.Hooks) {
	if !g.withTimings {
		return nil, collector.Hooks{}
	}

	rec := &timingRecorder{samples: make(map[string]CollectorTiming, n)}

	return rec, collector.Hooks{OnComplete: rec.record}
}

func selectCollectors(
	reg *collector.Registry,
	cfg config,
) ([]collector.Collector, error) {
	// WithCollectors wins outright: an explicit roster, no defaults.
	if len(cfg.only) > 0 {
		return namedCollectors(reg, cfg.only)
	}

	enabled, err := expandCategories(reg, cfg)
	if err != nil {
		return nil, err
	}

	// gohai.New() is opt-in: with none of WithDefaults, WithEnabled or
	// WithCategory it collects nothing. The disabled list is still
	// validated, so an unknown name errors either way.
	if !cfg.useDefaults && len(enabled) == 0 {
		_, err := reg.SelectedWith(collector.Selection{Disable: cfg.disabled})
		if err != nil {
			return nil, fmt.Errorf("select collectors: %w", err)
		}

		return nil, nil
	}

	sel, err := reg.SelectedWith(collector.Selection{
		UseDefaults: cfg.useDefaults,
		Enable:      enabled,
		Disable:     cfg.disabled,
	})
	if err != nil {
		return nil, fmt.Errorf("select collectors: %w", err)
	}

	return sel, nil
}

// namedCollectors resolves an explicit roster, erroring on a name no
// collector answers to.
func namedCollectors(
	reg *collector.Registry,
	names []string,
) ([]collector.Collector, error) {
	out := make([]collector.Collector, 0, len(names))

	for _, n := range names {
		c, ok := reg.Get(n)
		if !ok {
			return nil, fmt.Errorf("unknown collector %q", n)
		}

		out = append(out, c)
	}

	return out, nil
}

// expandCategories turns WithCategory into names to add. A category no
// collector belongs to errors, so a typo surfaces rather than quietly
// selecting nothing.
func expandCategories(
	reg *collector.Registry,
	cfg config,
) ([]string, error) {
	enabled := cfg.enabled

	for _, cat := range cfg.categories {
		names := reg.NamesInCategory(cat)
		if len(names) == 0 {
			return nil, fmt.Errorf("unknown category %q", cat)
		}

		enabled = append(enabled, names...)
	}

	return enabled, nil
}

// registerBuiltins registers every built-in collector. Registration errors
// would only occur from programmer bugs (duplicate names, empty names),
// which are caught by tests — callers can rely on registration succeeding.
func registerBuiltins(
	reg *collector.Registry,
) {
	for _, c := range builtinCollectors() {
		_ = reg.Register(c)
	}
}

// builtinCollectors returns the list of built-in collectors to register.
func builtinCollectors() []collector.Collector {
	return []collector.Collector{
		platform.New(),
		hostname.New(),
		kernel.New(),
		kernelmodules.New(),
		uptime.New(),
		virtualization.New(),
		machineid.New(),
		cpu.New(),
		load.New(),
		memory.New(),
		filesystem.New(),
		disk.New(),
		network.New(),
		process.New(),
		users.New(),
		sessions.New(),
		timezone.New(),
		rootgroup.New(),
		shells.New(),
		fips.New(),
		osrelease.New(),
		lsb.New(),
		initd.New(),
		shard.New(),
		packagemgr.New(),
		gce.New(),
		ec2.New(),
		azure.New(),
		digitalocean.New(),
		oci.New(),
		alibaba.New(),
		linode.New(),
		openstack.New(),
		scaleway.New(),
		dmi.New(),
		gpu.New(),
		pci.New(),
		scsi.New(),
		hardware.New(),
		blockdevice.New(),
		command.New(),
		docker.New(),
		grub2.New(),
		hostnamectl.New(),
		interrupts.New(),
		ipc.New(),
		languages.New(),
		libvirt.New(),
		livepatch.New(),
		mdadm.New(),
		packages.New(),
		rpm.New(),
		selinux.New(),
		services.New(),
		ssh.New(),
		sysconf.New(),
		sysctl.New(),
		systemdpaths.New(),
		tc.New(),
		virtualbox.New(),
		vmware.New(),
		zpools.New(),
	}
}
