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

package gohai

import (
	"github.com/osapi-io/gohai/pkg/gohai/collectors/alibaba"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/azure"
	digitalocean "github.com/osapi-io/gohai/pkg/gohai/collectors/digital_ocean"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/ec2"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/gce"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/linode"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/oci"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/openstack"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/scaleway"
)

// Cloud provider identifiers. Re-exported from each provider
// package so consumers can switch on Facts.Cloud().Name without
// magic strings — a typo like `gohai.CloudAWZ` fails to compile.
const (
	CloudAWS          = ec2.ProviderName
	CloudGCE          = gce.ProviderName
	CloudAzure        = azure.ProviderName
	CloudDigitalOcean = digitalocean.ProviderName
	CloudOCI          = oci.ProviderName
	CloudAlibaba      = alibaba.ProviderName
	CloudLinode       = linode.ProviderName
	CloudOpenStack    = openstack.ProviderName
	CloudScaleway     = scaleway.ProviderName
)

// Cloud is the tiny detection record Facts.Cloud() returns. It
// intentionally exposes only the provider identity — Region, Zone,
// instance IDs, and other fields differ per provider in ways that
// don't normalize cleanly, so consumers wanting those go to the
// per-provider typed field (Facts.Ec2, Facts.Gce, etc.) after
// switching on Name.
type Cloud struct {
	Name string `json:"name"`
}

// Cloud returns a detection record identifying which cloud this host
// is on, or nil when no cloud-provider collector populated data.
//
// Intended for provider-agnostic code that needs to branch on cloud
// identity — a typical flow:
//
//	cloud := facts.Cloud()
//	if cloud == nil {
//	    // not on any cloud
//	    return
//	}
//	switch cloud.Name {
//	case gohai.CloudAWS:
//	    arn := facts.Ec2.IAMInfo.InstanceProfileArn
//	case gohai.CloudGCE:
//	    project := facts.Gce.ProjectID
//	case gohai.CloudAzure:
//	    subscription := facts.Azure.SubscriptionID
//	}
//
// Consumers that already know which cloud they're interested in
// should skip Cloud() and just nil-check the typed field directly
// (for example, `if facts.Ec2 != nil { ... }`). It's faster,
// compile-time checked, and avoids the intermediate indirection.
//
// Cloud() does NOT attempt to normalize Region, Zone, or instance
// identity across providers — those shapes differ too much to
// flatten without losing meaningful information. Go to the typed
// per-provider field for those.
func (f *Facts) Cloud() *Cloud {
	switch {
	case f.Ec2 != nil:
		return &Cloud{Name: CloudAWS}
	case f.Gce != nil:
		return &Cloud{Name: CloudGCE}
	case f.Azure != nil:
		return &Cloud{Name: CloudAzure}
	case f.DigitalOcean != nil:
		return &Cloud{Name: CloudDigitalOcean}
	case f.OCI != nil:
		return &Cloud{Name: CloudOCI}
	case f.Alibaba != nil:
		return &Cloud{Name: CloudAlibaba}
	case f.Linode != nil:
		return &Cloud{Name: CloudLinode}
	case f.OpenStack != nil:
		return &Cloud{Name: CloudOpenStack}
	case f.Scaleway != nil:
		return &Cloud{Name: CloudScaleway}
	}
	return nil
}
