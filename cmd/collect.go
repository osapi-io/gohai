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
	"context"
	"fmt"
	"io"
	"sort"

	"github.com/spf13/cobra"

	"github.com/osapi-io/gohai/internal/cli"
	"github.com/osapi-io/gohai/pkg/gohai"
)

func newCollectCommand() *cobra.Command {
	var (
		pretty         bool
		flat           bool
		format         string
		listCollectors bool
		noDefaults     bool
		withTimings    bool
		categories     []string
	)

	enabled := newFlagSet()
	disabled := newFlagSet()

	cmd := &cobra.Command{
		Use:   "collect",
		Short: "Collect system facts",
		Long: `Collect system facts and output as JSON.

Output formats:
  --format ohai    Collector-centric JSON (default)
  --format ocsf    OCSF inventory_info event (class_uid 5001)

Examples:
  gohai collect --pretty
  gohai collect --format ocsf --pretty
  gohai collect --no-defaults --collector.cpu --collector.memory
  gohai collect --pretty | gohai validate`,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(
			c *cobra.Command,
			_ []string,
		) error {
			if listCollectors {
				return cli.WriteCollectorList(c.OutOrStdout())
			}

			return runCollect(
				c.Context(),
				c.OutOrStdout(),
				enabled,
				disabled,
				categories,
				format,
				pretty,
				flat,
				noDefaults,
				withTimings,
			)
		},
	}

	cmd.Flags().StringVar(
		&format,
		"format",
		"ohai",
		"output format: ohai (default) or ocsf",
	)
	cmd.Flags().BoolVar(&pretty, "pretty", false, "pretty-print JSON output")
	cmd.Flags().BoolVar(&flat, "flat", false, "output flat key=value pairs")
	cmd.Flags().
		BoolVar(&listCollectors, "list-collectors", false, "list available collectors and exit")
	cmd.Flags().BoolVar(
		&noDefaults,
		"no-defaults",
		false,
		"skip the recommended default collector set; only --collector.X flags are honoured",
	)
	cmd.Flags().BoolVar(
		&withTimings,
		"with-timings",
		false,
		"embed per-collector timings and errors under _timings in the JSON output",
	)
	cmd.Flags().StringSliceVar(
		&categories,
		"category",
		nil,
		"enable every collector in a category (repeatable): system, hardware, network, cloud, virtualization, security, software, users, linux, misc",
	)
	registerCollectorFlags(cmd, enabled, disabled)

	return cmd
}

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

func runCollect(
	ctx context.Context,
	out io.Writer,
	enabled, disabled *flagSet,
	categories []string,
	format string,
	pretty, flat, noDefaults, withTimings bool,
) error {
	switch format {
	case "ohai", "ocsf":
	default:
		return fmt.Errorf("unknown format %q: must be ohai or ocsf", format)
	}

	var opts []gohai.Option
	if !noDefaults {
		opts = append(opts, gohai.WithDefaults())
	}

	if names := enabled.values(); len(names) > 0 {
		opts = append(opts, gohai.WithEnabled(names...))
	}

	if names := disabled.values(); len(names) > 0 {
		opts = append(opts, gohai.WithDisabled(names...))
	}

	if len(categories) > 0 {
		opts = append(opts, gohai.WithCategory(categories...))
	}

	if withTimings {
		opts = append(opts, gohai.WithTimings())
	}

	g, err := gohai.New(opts...)
	if err != nil {
		return err
	}

	facts, err := g.Collect(ctx)
	if err != nil {
		return err
	}

	if format == "ocsf" {
		return cli.WriteOCSF(out, facts, pretty)
	}

	return cli.WriteOutput(out, facts, pretty, flat)
}
