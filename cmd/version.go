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

	goversion "github.com/caarlos0/go-version"
	"github.com/spf13/cobra"
)

var (
	version   = ""
	commit    = ""
	treestate = ""
	date      = ""
	builtBy   = ""
)

func buildVersion() goversion.Info {
	return goversion.GetVersionInfo(
		goversion.WithAppDetails(
			"gohai",
			"SDK-first Go library for collecting system facts.\n",
			"https://github.com/osapi-io/gohai",
		),
		func(i *goversion.Info) {
			if commit != "" {
				i.GitCommit = commit
			}
			if treestate != "" {
				i.GitTreeState = treestate
			}
			if date != "" {
				i.BuildDate = date
			}
			if version != "" {
				i.GitVersion = version
			}
			if builtBy != "" {
				i.BuiltBy = builtBy
			}
		},
	)
}

func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:           "version",
		Short:         "Display the version of gohai",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(
			c *cobra.Command,
			_ []string,
		) error {
			v := buildVersion()
			jsonOut, err := v.JSONString()
			if err != nil {
				return fmt.Errorf("render version JSON: %w", err)
			}

			if _, err := fmt.Fprintln(c.OutOrStdout(), jsonOut); err != nil {
				return fmt.Errorf("write version: %w", err)
			}

			return nil
		},
	}
}
