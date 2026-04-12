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

// Package users collects currently logged-in users.
package users

import (
	"context"

	"github.com/osapi-io/gohai/internal/collector"
)

// Info holds the set of logged-in sessions.
type Info struct {
	LoggedIn []Session `json:"logged_in"`
}

// Session represents one logged-in user session.
type Session struct {
	User     string `json:"user"`
	Terminal string `json:"terminal,omitempty"`
	Host     string `json:"host,omitempty"`
	Started  uint64 `json:"started,omitempty"` // unix timestamp
}

// Collector implements the collector.Collector interface.
type Collector struct{}

// New returns a new users Collector.
func New() *Collector {
	return &Collector{}
}

// Name returns "users".
func (c *Collector) Name() string {
	return "users"
}

// Tier returns TierCore.
func (c *Collector) Tier() collector.Tier {
	return collector.TierCore
}

// Dependencies returns no dependencies.
func (c *Collector) Dependencies() []string {
	return nil
}

// Collect gathers currently-logged-in-user facts.
func (c *Collector) Collect(
	ctx context.Context,
) (any, error) {
	return collect(ctx)
}
