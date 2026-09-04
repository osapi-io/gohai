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
	"slices"

	"github.com/spf13/cobra"

	"github.com/osapi-io/gohai/internal/cli"
	"github.com/osapi-io/gohai/pkg/gohai"
)

// outputFlags holds what `gohai collect` was asked to produce.
type outputFlags struct {
	pretty         bool
	flat           bool
	format         string
	listCollectors bool
	noDefaults     bool
	withTimings    bool
	categories     []string
}

// registerOutputFlags wires the output flags onto the command.
func registerOutputFlags(
	cmd *cobra.Command,
	out *outputFlags,
) {
	cmd.Flags().StringVar(
		&out.format,
		"format",
		"ohai",
		"output format: ohai (default) or ocsf",
	)
	cmd.Flags().BoolVar(&out.pretty, "pretty", false, "pretty-print JSON output")
	cmd.Flags().BoolVar(&out.flat, "flat", false, "output flat key=value pairs")
	cmd.Flags().BoolVar(
		&out.listCollectors,
		"list-collectors",
		false,
		"list available collectors and exit",
	)
	cmd.Flags().BoolVar(
		&out.noDefaults,
		"no-defaults",
		false,
		"skip the recommended default collector set; only --collector.X flags are honoured",
	)
	cmd.Flags().BoolVar(
		&out.withTimings,
		"with-timings",
		false,
		"embed per-collector timings and errors under _timings in the JSON output",
	)
	cmd.Flags().StringSliceVar(
		&out.categories,
		"category",
		nil,
		"enable every collector in a category (repeatable): system, hardware, network, cloud, virtualization, security, software, users, linux, misc",
	)
}

func newCollectCommand() *cobra.Command {
	var out outputFlags

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
			if out.listCollectors {
				return cli.WriteCollectorList(c.OutOrStdout())
			}

			return runCollect(c.Context(), c.OutOrStdout(), collectRequest{
				enabled:     enabled,
				disabled:    disabled,
				categories:  out.categories,
				format:      out.format,
				pretty:      out.pretty,
				flat:        out.flat,
				noDefaults:  out.noDefaults,
				withTimings: out.withTimings,
			})
		},
	}

	registerOutputFlags(cmd, &out)
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
	slices.Sort(out)

	return out
}

// readCollectorFlags moves the per-collector flags into the two sets the
// command was given.
func readCollectorFlags(
	c *cobra.Command,
	names []string,
	enabled *flagSet,
	disabled *flagSet,
) {
	for _, n := range names {
		if v, _ := c.Flags().GetBool("collector." + n); v {
			enabled.set[n] = true
		}

		if v, _ := c.Flags().GetBool("no-collector." + n); v {
			disabled.set[n] = true
		}
	}
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
		readCollectorFlags(c, names, enabled, disabled)

		return nil
	}
}

func listAllCollectorNames() []string {
	names := gohai.NewRegistry().Names()
	slices.Sort(names)

	return names
}

// collectRequest is what one invocation of `gohai collect` was asked
// for. It travels as a struct because the flags outgrew a parameter list.
type collectRequest struct {
	enabled     *flagSet
	disabled    *flagSet
	categories  []string
	format      string
	pretty      bool
	flat        bool
	noDefaults  bool
	withTimings bool
}

// collectorOptions turns the command's flags into library options.
func collectorOptions(
	req collectRequest,
) []gohai.Option {
	var opts []gohai.Option

	if !req.noDefaults {
		opts = append(opts, gohai.WithDefaults())
	}

	if names := req.enabled.values(); len(names) > 0 {
		opts = append(opts, gohai.WithEnabled(names...))
	}

	if names := req.disabled.values(); len(names) > 0 {
		opts = append(opts, gohai.WithDisabled(names...))
	}

	if len(req.categories) > 0 {
		opts = append(opts, gohai.WithCategory(req.categories...))
	}

	if req.withTimings {
		opts = append(opts, gohai.WithTimings())
	}

	return opts
}

func runCollect(
	ctx context.Context,
	out io.Writer,
	req collectRequest,
) error {
	switch req.format {
	case "ohai", "ocsf":
	default:
		return fmt.Errorf("unknown format %q: must be ohai or ocsf", req.format)
	}

	opts := collectorOptions(req)

	g, err := gohai.New(opts...)
	if err != nil {
		return err
	}

	facts, err := g.Collect(ctx)
	if err != nil {
		return err
	}

	if req.format == "ocsf" {
		return cli.WriteOCSF(out, facts, req.pretty)
	}

	format := cli.FormatJSON
	if req.flat {
		format = cli.FormatFlat
	}

	return cli.WriteOutput(out, facts, format, req.pretty)
}
