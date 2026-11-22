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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/osapi-io/gohai/pkg/gohai"
)

// errProblems is the error of validate when a collector breaks a rule.
var errProblems = errors.New("a collector breaks a rule")

// newCollectorCommand is gohai collector: list, get, spec get and validate,
// the commands that describe and check the collectors, as osapi's DSLs
// describe and check themselves.
func newCollectorCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "collector",
		Short: "List, describe and validate the collectors this binary has",
		Long: `A collector reads one kind of fact from a host, such as its services.
These commands read the collectors in the binary, so what they print
always agrees with the build that you run.`,
	}
	spec := &cobra.Command{Use: "spec", Short: "Print the parts of a collector"}
	spec.AddCommand(newCollectorSpecGetCommand())
	cmd.AddCommand(
		newCollectorListCommand(),
		newCollectorGetCommand(),
		spec,
		newCollectorValidateCommand(),
	)

	return cmd
}

func newCollectorListCommand() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List the collectors: category, whether it runs by default, dependencies",
		Example: `  gohai collector list
  gohai collector list --json`,
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(c *cobra.Command, _ []string) error {
			return writeCollectorList(c.OutOrStdout(), gohai.Collectors(), asJSON)
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "print JSON")

	return cmd
}

func newCollectorGetCommand() *cobra.Command {
	var (
		name   string
		asJSON bool
	)
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Show a collector: its constructor, its fields, and what it reads on each platform",
		Long: `The reads are what the collector reads on an empty host of each platform:
the files that it opens or looks at, and the commands that it runs. With
content, it also reads under these paths. A fake host of a test gives each
one, and a fact that no collector reads is a change to gohai first.`,
		Example: `  gohai collector get --name services
  gohai collector get --name services --json`,
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(c *cobra.Command, _ []string) error {
			spec, ok := gohai.CollectorByName(name)
			if !ok {
				return fmt.Errorf("no collector named %q: gohai collector list shows them", name)
			}

			return writeCollector(c.OutOrStdout(), spec, asJSON)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "the collector, from gohai collector list")
	cmd.Flags().BoolVar(&asJSON, "json", false, "print JSON")
	_ = cmd.MarkFlagRequired("name")

	return cmd
}

func newCollectorSpecGetCommand() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Print each part of a collector, what it does and when to use it",
		Example: `  gohai collector spec get
  gohai collector spec get --json`,
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(c *cobra.Command, _ []string) error {
			return writeTerms(c.OutOrStdout(), gohai.CollectorTerms(), asJSON)
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "print JSON")

	return cmd
}

func newCollectorValidateCommand() *cobra.Command {
	var asJSON bool
	var rules strings.Builder
	for _, c := range gohai.CollectorChecks() {
		fmt.Fprintf(&rules, "  %s: %s\n", c.Rule, c.Requires)
	}
	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Check every collector against the rules, and print the fix of each problem",
		Long:  "It exits 1 when a collector breaks a rule.\n\nThe rules:\n\n" + rules.String(),
		Example: `  gohai collector validate
  gohai collector validate --json`,
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(c *cobra.Command, _ []string) error {
			return writeProblems(
				c.OutOrStdout(),
				gohai.CollectorProblems(gohai.Collectors()),
				asJSON,
			)
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "print JSON")

	return cmd
}

// writeJSON writes v as indented JSON.
func writeJSON(
	out io.Writer,
	v any,
) error {
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")

	return enc.Encode(v)
}

func writeCollectorList(
	out io.Writer,
	specs []gohai.CollectorSpec,
	asJSON bool,
) error {
	if asJSON {
		return writeJSON(out, specs)
	}
	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "NAME\tCATEGORY\tDEFAULT\tDEPENDS ON")
	for _, c := range specs {
		def := "no"
		if c.DefaultEnabled {
			def = "yes"
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", c.Name, c.Category, def, orNone(c.Dependencies))
	}

	return w.Flush()
}

func writeCollector(
	out io.Writer,
	c gohai.CollectorSpec,
	asJSON bool,
) error {
	if asJSON {
		return writeJSON(out, c)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Collector:    %s\n", c.Name)
	fmt.Fprintf(&b, "Category:     %s\n", c.Category)
	fmt.Fprintf(&b, "Default:      %t\n", c.DefaultEnabled)
	fmt.Fprintf(&b, "Depends on:   %s\n", orNone(c.Dependencies))
	fmt.Fprintf(&b, "Package:      %s\n", c.Package)
	fmt.Fprintf(&b, "Constructor:  %s\n", c.Constructor)
	if c.Network != "" {
		fmt.Fprintf(&b, "Network:      %s\n", c.Network)
	}
	fmt.Fprintf(&b, "\nFields (%d):\n", len(c.Fields))
	for _, f := range c.Fields {
		fmt.Fprintf(&b, "  %s  %s\n", f.Name, f.Type)
	}
	for _, r := range c.Reads {
		fmt.Fprintf(&b, "\nReads on %s:\n", r.Platform)
		if len(r.Files)+len(r.Commands) == 0 {
			b.WriteString("  nothing\n")
		}
		for _, f := range r.Files {
			fmt.Fprintf(&b, "  file     %s\n", f)
		}
		for _, cmd := range r.Commands {
			fmt.Fprintf(&b, "  command  %s\n", cmd)
		}
	}
	_, err := io.WriteString(out, b.String())

	return err
}

func writeTerms(
	out io.Writer,
	terms []gohai.Term,
	asJSON bool,
) error {
	if asJSON {
		return writeJSON(out, terms)
	}
	var b strings.Builder
	for _, t := range terms {
		fmt.Fprintf(&b, "%s\n  %s\n  Use: %s\n\n", t.Name, t.Does, t.When)
	}
	_, err := io.WriteString(out, b.String())

	return err
}

func writeProblems(
	out io.Writer,
	problems []gohai.Problem,
	asJSON bool,
) error {
	if asJSON {
		if problems == nil {
			problems = []gohai.Problem{}
		}
		if err := writeJSON(out, problems); err != nil {
			return err
		}
	} else {
		if len(problems) == 0 {
			_, _ = fmt.Fprintln(out, "Every collector passes each rule.")

			return nil
		}
		for _, p := range problems {
			_, _ = fmt.Fprintf(out, "%s [%s]: %s Fix: %s\n", p.Collector, p.Rule, p.Message, p.Fix)
		}
	}
	if len(problems) > 0 {
		return errProblems
	}

	return nil
}

// orNone is list joined, or none.
func orNone(
	list []string,
) string {
	if len(list) == 0 {
		return "none"
	}

	return strings.Join(list, ", ")
}
