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

package collector

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"
)

// Hooks are optional per-run observability callbacks. Both fields may
// be nil.
type Hooks struct {
	// OnError is invoked when a collector's Collect returns an error.
	// Called from the collector's goroutine, concurrent with other
	// collectors in the same level.
	OnError func(name string, err error)
	// OnComplete is invoked after every collector's Collect returns,
	// regardless of success or failure, with the wall-clock duration
	// of that call. Useful for --debug timing output.
	OnComplete func(name string, dur time.Duration, err error)
}

// Registry holds the set of registered collectors.
type Registry struct {
	mu         sync.RWMutex
	collectors map[string]Collector
}

// NewRegistry returns an empty Registry.
func NewRegistry() *Registry {
	return &Registry{
		collectors: make(map[string]Collector),
	}
}

// Register adds a collector to the registry. Returns an error if the name is
// empty or already registered.
func (r *Registry) Register(
	c Collector,
) error {
	if c.Name() == "" {
		return errors.New("collector name must not be empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.collectors[c.Name()]; exists {
		return fmt.Errorf("collector %q already registered", c.Name())
	}
	r.collectors[c.Name()] = c
	return nil
}

// Get returns the collector with the given name, if registered.
func (r *Registry) Get(
	name string,
) (Collector, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.collectors[name]
	return c, ok
}

// NamesInCategory returns the names of every collector whose Category
// matches cat. Returns an empty slice (never nil) when no collector
// has that category — callers should treat that as a user error
// ("unknown category") rather than silently running nothing.
func (r *Registry) NamesInCategory(
	cat string,
) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := []string{}
	for name, c := range r.collectors {
		if c.Category() == cat {
			out = append(out, name)
		}
	}
	return out
}

// Names returns the names of all registered collectors.
func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.collectors))
	for n := range r.collectors {
		names = append(names, n)
	}
	return names
}

// Selected returns the collectors that should run treating every
// DefaultEnabled() collector as on, plus anything in `enable`, minus
// anything in `disable`. Kept for backward compatibility; new callers
// should use SelectedWith and decide explicitly whether to honour
// DefaultEnabled().
func (r *Registry) Selected(
	enable []string,
	disable []string,
) ([]Collector, error) {
	return r.SelectedWith(Selection{UseDefaults: true, Enable: enable, Disable: disable})
}

// SelectedWith returns the collectors that should run. When
// useDefaults is true, every collector with DefaultEnabled()==true is
// included. The enable list adds names regardless of DefaultEnabled.
// The disable list subtracts names. Unknown names in enable/disable
// return an error.
// Selection describes which collectors a caller wants: the defaults or
// not, plus names to add and names to take away.
type Selection struct {
	UseDefaults bool
	Enable      []string
	Disable     []string
}

func (r *Registry) SelectedWith(
	sel Selection,
) ([]Collector, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if err := r.checkKnown(sel.Enable, "enable"); err != nil {
		return nil, err
	}

	if err := r.checkKnown(sel.Disable, "disable"); err != nil {
		return nil, err
	}

	return r.pick(sel), nil
}

// pick chooses what sel asks for: the defaults when it wants them,
// plus anything named in Enable, less anything named in Disable.
func (r *Registry) pick(
	sel Selection,
) []Collector {
	enabled := asSet(sel.Enable)
	disabled := asSet(sel.Disable)

	out := make([]Collector, 0, len(r.collectors))

	for name, c := range r.collectors {
		if disabled[name] {
			continue
		}

		if (sel.UseDefaults && c.DefaultEnabled()) || enabled[name] {
			out = append(out, c)
		}
	}

	return out
}

// checkKnown rejects a name no collector answers to, naming which list
// it came from.
func (r *Registry) checkKnown(
	names []string,
	list string,
) error {
	for _, n := range names {
		if _, ok := r.collectors[n]; !ok {
			return fmt.Errorf("unknown collector %q in %s list", n, list)
		}
	}

	return nil
}

// asSet turns a name list into a lookup.
func asSet(
	names []string,
) map[string]bool {
	set := make(map[string]bool, len(names))
	for _, n := range names {
		set[n] = true
	}

	return set
}

// Run executes the named collectors in dependency order. Dependencies are
// auto-included even if not in the names list. Collectors at the same level
// (no inter-dependencies) run concurrently. Returns a map of collector name
// to result. Failed collectors are omitted from the result map. hooks may
// be zero-valued; any non-nil field is invoked per collector.
func (r *Registry) Run(
	ctx context.Context,
	names []string,
	hooks Hooks,
) (map[string]any, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	wanted, err := r.expandWithDeps(names)
	if err != nil {
		return nil, err
	}

	levels, err := r.topoLevels(wanted)
	if err != nil {
		return nil, err
	}

	results := make(map[string]any, len(wanted))

	var resultsMu sync.Mutex

	for _, level := range levels {
		r.runLevel(ctx, level, results, &resultsMu, hooks)
	}

	return results, nil
}

// runLevel runs one level's collectors at the same time. They have no
// dependencies on each other by definition, so they all see the same
// snapshot of what earlier levels produced.
func (r *Registry) runLevel(
	ctx context.Context,
	level []string,
	results map[string]any,
	mu *sync.Mutex,
	hooks Hooks,
) {
	mu.Lock()
	prior := make(PriorResults, len(results))

	for k, v := range results {
		prior[k] = v
	}
	mu.Unlock()

	var wg sync.WaitGroup

	for _, name := range level {
		c := r.collectors[name]
		wg.Go(func() {
			out, ok := runOne(ctx, c, prior, hooks)
			if !ok {
				return
			}

			mu.Lock()
			results[c.Name()] = out
			mu.Unlock()
		})
	}

	wg.Wait()
}

// runOne runs a single collector, timing it and reporting through the
// hooks. It returns false when the collector failed, which keeps it out
// of the results.
func runOne(
	ctx context.Context,
	c Collector,
	prior PriorResults,
	hooks Hooks,
) (any, bool) {
	start := time.Now()
	out, err := c.Collect(ctx, prior)

	if hooks.OnComplete != nil {
		hooks.OnComplete(c.Name(), time.Since(start), err)
	}

	if err != nil {
		if hooks.OnError != nil {
			hooks.OnError(c.Name(), err)
		}

		return nil, false
	}

	return out, true
}

func (r *Registry) expandWithDeps(
	names []string,
) (map[string]bool, error) {
	wanted := make(map[string]bool)

	for _, n := range names {
		if err := r.visitDeps(n, wanted); err != nil {
			return nil, err
		}
	}

	return wanted, nil
}

// visitDeps marks a collector and everything it depends on, depth-first.
func (r *Registry) visitDeps(
	name string,
	wanted map[string]bool,
) error {
	if wanted[name] {
		return nil
	}

	c, ok := r.collectors[name]
	if !ok {
		return fmt.Errorf("unknown collector %q", name)
	}

	wanted[name] = true

	for _, dep := range c.Dependencies() {
		if err := r.visitDeps(dep, wanted); err != nil {
			return err
		}
	}

	return nil
}

func (r *Registry) topoLevels(
	wanted map[string]bool,
) ([][]string, error) {
	indeg := make(map[string]int, len(wanted))
	for n := range wanted {
		indeg[n] = len(r.collectors[n].Dependencies())
	}

	var levels [][]string

	for len(indeg) > 0 {
		level := readyNames(indeg)
		if len(level) == 0 {
			return nil, errors.New("dependency cycle detected among collectors")
		}

		slices.Sort(level)
		levels = append(levels, level)
		r.removeLevel(indeg, level)
	}

	return levels, nil
}

// readyNames lists the collectors whose dependencies have all run.
func readyNames(
	indeg map[string]int,
) []string {
	var level []string

	for n, d := range indeg {
		if d == 0 {
			level = append(level, n)
		}
	}

	return level
}

// removeLevel drops a level from the graph and decrements whatever was
// waiting on it.
func (r *Registry) removeLevel(
	indeg map[string]int,
	level []string,
) {
	for _, n := range level {
		delete(indeg, n)
		r.decrementDependents(indeg, n)
	}
}

// decrementDependents drops one from every collector still waiting on n.
func (r *Registry) decrementDependents(
	indeg map[string]int,
	n string,
) {
	for m := range indeg {
		for _, dep := range r.collectors[m].Dependencies() {
			if dep == n {
				indeg[m]--
			}
		}
	}
}
