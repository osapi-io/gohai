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
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/osapi-io/gohai/internal/cli"
	"github.com/osapi-io/gohai/internal/validate"
)

func newValidateCommand() *cobra.Command {
	var file string

	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate JSON against the gohai schema",
		Long: `Validate a gohai JSON document against the embedded JSON Schema.

Reads from --file or stdin. Exits 0 if valid, 1 if invalid.

Examples:
  gohai collect --pretty | gohai validate
  gohai validate --file facts.json`,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(
			c *cobra.Command,
			_ []string,
		) error {
			return runValidate(c.Context(), c.OutOrStdout(), c.ErrOrStderr(), file)
		},
	}

	cmd.Flags().StringVarP(&file, "file", "f", "", "path to JSON file (default: stdin)")

	return cmd
}

func runValidate(
	_ context.Context,
	out, errOut io.Writer,
	file string,
) error {
	input, err := readInput(file)
	if err != nil {
		return err
	}

	if err := validate.JSON(input); err != nil {
		cli.Print(errOut, cli.Failure(errOut, "schema validation failed"))
		cli.Print(errOut, "")
		cli.Print(errOut, cli.Mute(errOut, err.Error()))
		cli.Print(errOut, "")

		return errors.New("schema validation failed")
	}

	cli.Print(out, cli.Success(out, "schema valid"))

	return nil
}

func readInput(
	file string,
) ([]byte, error) {
	if file != "" {
		data, err := os.ReadFile(file)
		if err != nil {
			return nil, fmt.Errorf("read file %s: %w", file, err)
		}

		return data, nil
	}

	info, err := os.Stdin.Stat()
	if err == nil && info.Mode()&os.ModeCharDevice != 0 {
		return nil, fmt.Errorf("no input: pass --file or pipe JSON to stdin")
	}

	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return nil, fmt.Errorf("read stdin: %w", err)
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("no input: pass --file or pipe JSON to stdin")
	}

	return data, nil
}
