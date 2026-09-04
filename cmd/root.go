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
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/osapi-io/gohai/internal/cli"
)

// Execute runs the gohai CLI. Called from main.go, which owns the exit
// code so that this stays testable.
func Execute() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		cancel()
	}()

	cmd := newRootCommand()

	return cmd.ExecuteContext(ctx)
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
	cmd.AddCommand(newVersionCommand())

	return cmd
}
