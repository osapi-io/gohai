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

package cmd

import (
	"fmt"
	"io"
	"sort"

	"github.com/spf13/cobra"

	"github.com/osapi-io/gohai/pkg/gohai"
)

// flagSet captures collector names toggled on or off via CLI flags.
type flagSet struct {
	set map[string]bool
}

func newFlagSet() *flagSet {
	return &flagSet{set: map[string]bool{}}
}

func (f *flagSet) values() []string {
	out := make([]string, 0, len(f.set))
	for n := range f.set {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

// registerCollectorFlags adds --collector.<name> / --no-collector.<name>
// flags for every registered collector.
func registerCollectorFlags(
	cmd *cobra.Command,
	enabled, disabled *flagSet,
) {
	names := listAllCollectorNames()
	for _, n := range names {
		cmd.Flags().Bool("collector."+n, false, fmt.Sprintf("enable %s collector", n))
		cmd.Flags().Bool("no-collector."+n, false, fmt.Sprintf("disable %s collector", n))
	}
	cmd.PreRunE = func(
		c *cobra.Command,
		_ []string,
	) error {
		for _, n := range names {
			if v, _ := c.Flags().GetBool("collector." + n); v {
				enabled.set[n] = true
			}
			if v, _ := c.Flags().GetBool("no-collector." + n); v {
				disabled.set[n] = true
			}
		}
		return nil
	}
}

func listAllCollectorNames() []string {
	names := gohai.NewRegistry().Names()
	sort.Strings(names)
	return names
}

func printCollectorList(
	out io.Writer,
) error {
	reg := gohai.NewRegistry()
	byCat := map[string][]string{}
	for _, n := range reg.Names() {
		cat := reg.CategoryOf(n)
		byCat[cat] = append(byCat[cat], n)
	}
	cats := make([]string, 0, len(byCat))
	for c := range byCat {
		cats = append(cats, c)
	}
	sort.Strings(cats)
	for _, cat := range cats {
		names := byCat[cat]
		sort.Strings(names)
		if _, err := fmt.Fprintf(out, "[%s]\n", cat); err != nil {
			return fmt.Errorf("write collector list: %w", err)
		}
		for _, n := range names {
			if _, err := fmt.Fprintf(out, "  %s\n", n); err != nil {
				return fmt.Errorf("write collector list: %w", err)
			}
		}
	}
	return nil
}
