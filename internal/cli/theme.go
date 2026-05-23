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

// Package cli holds CLI-only output helpers — never imported by the
// SDK under pkg/gohai/, only by cmd/. Mirrors jot's theme system
// (same Theme struct, same role names) so all retr0h CLIs share one
// shape. Uses raw ANSI to avoid charmbracelet dependency conflicts.
package cli

import (
	"fmt"
	"io"
	"os"

	"golang.org/x/term"
)

const (
	ansiReset = "\033[0m"
	ansiFaint = "\033[0;2m"
)

// rgb returns a 24-bit truecolor foreground escape.
func rgb(
	r, g, b int,
) string {
	return fmt.Sprintf("\033[38;2;%d;%d;%dm", r, g, b)
}

// Theme is a six-role palette covering every place gohai's CLI
// surface emits styled text. Roles are stable across themes so a
// theme swap is a pure recolor — no callers change.
type Theme struct {
	Name      string
	Mute      string
	Accent    string
	OK        string
	Err       string
	Info      string
	BannerTop string
	BannerBot string
}

// ThemeMaxheadroom is gohai's default — pulled from the maxheadroom
// palette. The primary accent is a soft lavender (#b4a7d6) — cooler
// and more muted than the magenta used by jot/grind; supporting
// roles are drawn from the same palette. Truecolor (24-bit) so the
// install banner and `gohai --help` paint with the exact same hue.
var ThemeMaxheadroom = Theme{
	Name:      "maxheadroom",
	Mute:      ansiFaint,
	Accent:    rgb(180, 167, 214),
	OK:        rgb(80, 250, 123),
	Err:       rgb(255, 110, 199),
	Info:      rgb(0, 212, 255),
	BannerTop: ansiFaint,
	BannerBot: rgb(180, 167, 214),
}

var active = &ThemeMaxheadroom

var isTerminalFn = term.IsTerminal

func isTTY(
	w io.Writer,
) bool {
	if f, ok := w.(*os.File); ok {
		return isTerminalFn(int(f.Fd()))
	}

	return false
}

func colorize(
	w io.Writer,
	ansi, s string,
) string {
	if !isTTY(w) {
		return s
	}

	return ansi + s + ansiReset
}

// Mute returns s rendered as secondary text per the active theme.
func Mute(
	w io.Writer,
	s string,
) string {
	return colorize(w, active.Mute, s)
}

// Accent returns s rendered as the brand accent color.
func Accent(
	w io.Writer,
	s string,
) string {
	return colorize(w, active.Accent, s)
}

// OK returns s in the success color.
func OK(
	w io.Writer,
	s string,
) string {
	return colorize(w, active.OK, s)
}

// Err returns s in the error color.
func Err(
	w io.Writer,
	s string,
) string {
	return colorize(w, active.Err, s)
}

// Info returns s in the cool-toned info/hint color.
func Info(
	w io.Writer,
	s string,
) string {
	return colorize(w, active.Info, s)
}

// Banner returns the GOHAI block-letter logo, themed via the active
// theme's BannerTop/BannerBot colors. Line-level coloring matches
// the install summary so curl|bash and `gohai --help` look the same.
func Banner(
	w io.Writer,
) string {
	const top = "█▀▀ █▀█ █░█ █▀█ █"
	const bot = "█▄█ █▄█ █▀█ █░█ █"

	return colorize(w, active.BannerTop, top) + "\n" +
		colorize(w, active.BannerBot, bot) + "\n"
}

// Success renders a leading mark in the OK color followed by msg.
func Success(
	w io.Writer,
	msg string,
) string {
	if !isTTY(w) {
		return "[ok] " + msg
	}

	return OK(w, "✓") + " " + msg
}

// Failure mirrors Success for error one-liners.
func Failure(
	w io.Writer,
	msg string,
) string {
	if !isTTY(w) {
		return "[err] " + msg
	}

	return Err(w, "✗") + " " + msg
}

// Print writes a line to w.
func Print(
	w io.Writer,
	s string,
) {
	_, _ = fmt.Fprintln(w, s)
}

// Printf writes a formatted string to w.
func Printf(
	w io.Writer,
	format string,
	a ...any,
) {
	_, _ = fmt.Fprintf(w, format, a...)
}
