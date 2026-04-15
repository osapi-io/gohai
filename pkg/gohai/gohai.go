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
	"github.com/osapi-io/gohai/pkg/gohai/collectors/cpu"
	digitalocean "github.com/osapi-io/gohai/pkg/gohai/collectors/digital_ocean"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/disk"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/dmi"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/ec2"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/filesystem"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/fips"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/gce"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/gpu"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/hostname"
	initd "github.com/osapi-io/gohai/pkg/gohai/collectors/init"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/kernel"
	kernelmodules "github.com/osapi-io/gohai/pkg/gohai/collectors/kernel_modules"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/linode"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/load"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/lsb"
	machineid "github.com/osapi-io/gohai/pkg/gohai/collectors/machine_id"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/memory"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/network"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/oci"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/openstack"
	osrelease "github.com/osapi-io/gohai/pkg/gohai/collectors/os_release"
	packagemgr "github.com/osapi-io/gohai/pkg/gohai/collectors/package_mgr"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/platform"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/process"
	rootgroup "github.com/osapi-io/gohai/pkg/gohai/collectors/root_group"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/scaleway"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/shard"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/shells"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/timezone"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/uptime"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/users"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/virtualization"
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

	var (
		samplesMu sync.Mutex
		samples   map[string]CollectorTiming
	)
	hooks := collector.Hooks{}
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
	}
	if g.withTimings {
		facts.Timings = &Timings{Collectors: samples}
	}
	for name, result := range results {
		facts.set(name, result)
	}
	return facts, nil
}

func selectCollectors(
	reg *collector.Registry,
	cfg config,
) ([]collector.Collector, error) {
	// WithCollectors wins outright — explicit roster, no defaults.
	if len(cfg.only) > 0 {
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
		names := reg.NamesInCategory(cat)
		if len(names) == 0 {
			return nil, fmt.Errorf("unknown category %q", cat)
		}
		enabled = append(enabled, names...)
	}
	// Without WithDefaults / WithEnabled / WithCategory, return
	// empty — gohai.New() is opt-in. Still validate the disabled
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
	}
}
