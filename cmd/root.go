// Copyright (c) 2024 John Dewey

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

	"github.com/spf13/cobra"

	"github.com/osapi-io/gohai/pkg/gohai"
)

// Execute runs the gohai CLI. Called from main.go.
func Execute() {
	cmd := newRootCommand()
	if err := cmd.Execute(); err != nil {
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
			return runCollect(c.OutOrStdout(), enabled, disabled, pretty, flat)
		},
	}

	cmd.Flags().BoolVar(&pretty, "pretty", false, "pretty-print JSON output")
	cmd.Flags().BoolVar(&flat, "flat", false, "output flat key=value pairs")
	cmd.Flags().
		BoolVar(&listCollectors, "list-collectors", false, "list available collectors and exit")
	registerCollectorFlags(cmd, enabled, disabled)

	return cmd
}

func runCollect(
	out io.Writer,
	enabled, disabled *flagSet,
	pretty, flat bool,
) error {
	opts := []gohai.Option{}
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
	facts, err := g.Collect(context.Background())
	if err != nil {
		return err
	}
	return writeOutput(out, facts, pretty, flat)
}
