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

	"github.com/spf13/cobra"

	"github.com/osapi-io/gohai/internal/cli"
)

// Execute runs the gohai CLI with args, the words after the binary's name,
// and returns the exit code: 0, or 1 after it prints the error to errOut.
// main.go gives it the process's arguments and streams, and a ctx that a
// signal cancels. A stream takes color when it is a terminal and NO_COLOR is
// unset.
func Execute(
	ctx context.Context,
	args []string,
	in io.Reader,
	out io.Writer,
	errOut io.Writer,
) int {
	out = cli.Colored(out, colorFor(out, os.Getenv, terminal))
	errOut = cli.Colored(errOut, colorFor(errOut, os.Getenv, terminal))
	cmd := newRootCommand()
	cmd.SetArgs(args)
	cmd.SetIn(in)
	cmd.SetOut(out)
	cmd.SetErr(errOut)
	if err := cmd.ExecuteContext(ctx); err != nil {
		_, _ = fmt.Fprintf(errOut, "error: %v\n", err)

		return 1
	}

	return 0
}

func newRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "gohai",
		Short:         "Collect system facts",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(
			c *cobra.Command,
			_ []string,
		) error {
			return c.Help()
		},
	}

	defaultHelp := cmd.HelpFunc()
	cmd.SetHelpFunc(func(
		c *cobra.Command,
		args []string,
	) {
		if c == cmd {
			out := c.OutOrStdout()
			_, _ = fmt.Fprintln(out)
			_, _ = fmt.Fprint(out, cli.Banner(out))
			_, _ = fmt.Fprintln(out)
		}
		defaultHelp(c, args)
	})

	cmd.AddCommand(newCollectCommand())
	cmd.AddCommand(newValidateCommand())
	cmd.AddCommand(NewCollectorCommand("gohai collector"))
	cmd.AddCommand(newVersionCommand())

	return cmd
}
