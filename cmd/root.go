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

// Package cmd implements the gohai CLI.
package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/osapi-io/gohai/pkg/gohai"
)

// Execute runs the gohai CLI. Called from main.go. Matches OSAPI's
// cmd/root.go pattern: context.WithCancel bound to SIGINT/SIGTERM so
// Ctrl-C cancels in-flight collection cleanly.
func Execute() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		cancel()
	}()

	cmd := newRootCommand()
	if err := cmd.ExecuteContext(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

// newRootCommand builds the Cobra root command with all flags and collector
// toggles wired up.
func newRootCommand() *cobra.Command {
	var (
		pretty         bool
		flat           bool
		listCollectors bool
		noDefaults     bool
	)
	enabled := newFlagSet()
	disabled := newFlagSet()

	cmd := &cobra.Command{
		Use:           "gohai",
		Short:         "Collect system facts",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(
			c *cobra.Command,
			_ []string,
		) error {
			if listCollectors {
				return printCollectorList(c.OutOrStdout())
			}
			return runCollect(
				c.Context(),
				c.OutOrStdout(),
				enabled,
				disabled,
				pretty,
				flat,
				noDefaults,
			)
		},
	}

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
	registerCollectorFlags(cmd, enabled, disabled)

	return cmd
}

func runCollect(
	ctx context.Context,
	out io.Writer,
	enabled, disabled *flagSet,
	pretty, flat, noDefaults bool,
) error {
	opts := []gohai.Option{}
	if !noDefaults {
		opts = append(opts, gohai.WithDefaults())
	}
	if names := enabled.values(); len(names) > 0 {
		opts = append(opts, gohai.WithEnabled(names...))
	}
	if names := disabled.values(); len(names) > 0 {
		opts = append(opts, gohai.WithDisabled(names...))
	}

	g, err := gohai.New(opts...)
	if err != nil {
		return err
	}
	facts, err := g.Collect(ctx)
	if err != nil {
		return err
	}
	return writeOutput(out, facts, pretty, flat)
}
