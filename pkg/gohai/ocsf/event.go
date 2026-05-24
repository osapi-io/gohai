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

// Package ocsf provides OCSF-shaped output for gohai's system facts.
// The InventoryInfo type is an OCSF Device Inventory Info event
// (class_uid 5001, category_uid 5) constructed from gohai's
// collector-centric Facts via FromFacts().
package ocsf

// OCSF event constants for the inventory_info event class.
const (
	CategoryUIDDiscovery  = 5
	ClassUIDInventoryInfo = 5001
	TypeUIDCollect        = 500102
	ActivityIDCollect     = 2
	SeverityIDInfo        = 1
)

// InventoryInfo is an OCSF Device Inventory Info event (class_uid 5001).
type InventoryInfo struct {
	ActivityID  int    `json:"activity_id"`
	CategoryUID int    `json:"category_uid"`
	ClassUID    int    `json:"class_uid"`
	ClassName   string `json:"class_name"`
	SeverityID  int    `json:"severity_id"`
	Time        int64  `json:"time"`
	TypeUID     int    `json:"type_uid"`

	Metadata *Metadata `json:"metadata"`
	Device   *Device   `json:"device"`
	Cloud    *Cloud    `json:"cloud,omitempty"`

	Unmapped map[string]any `json:"unmapped,omitempty"`
}

// Metadata describes the event producer and schema version.
type Metadata struct {
	Version string   `json:"version"`
	Product *Product `json:"product"`
}

// Product identifies the tool that produced the event.
type Product struct {
	Name       string `json:"name"`
	VendorName string `json:"vendor_name"`
	Version    string `json:"version,omitempty"`
}
