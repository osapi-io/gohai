# Cloud detection

> **Status:** Not a collector

This page shows how to use the SDK to find the cloud provider of a host. It also
shows how to use that identity to get the correct typed facts of each provider.

The other pages in this directory each describe the collector of one provider.
gohai has no `cloud` collector. The detection is a small API on `gohai.Facts`.
It examines which provider field has data.

## Why not an aggregator collector?

Earlier versions of this SDK had a `cloud` collector. It merged the facts of
each provider into one normalized view. That design had two problems:

1. **The normalization loses data.** EC2's `Region` is `"us-east-1"`. The Azure
   equivalent is `Location: "eastus"`. The OCI equivalent is a combined
   `availabilityDomain: "pPrU:PHX-AD-1"`. One `Cloud.Region` field for all of
   them removes the shape that consumers need.
2. **It duplicates data.** `Facts` already has the typed `Info` struct of each
   provider (`Facts.Ec2`, `Facts.Gce`, …). An aggregator is one more struct to
   maintain. A person must change it manually each time a field of a provider
   struct changes.

Thus gohai uses a different design: **the SDK gives the identity of the
provider, and the consumer reads the provider field for the full data.**

## The API

```go
// pkg/gohai exports:
type Cloud struct {
    Name string `json:"name"` // "aws", "gce", "azure", ...
}

func (f *Facts) Cloud() *Cloud

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
```

`Facts.Cloud()` examines `Facts.Ec2`, `Facts.Gce`, `Facts.Azure` and the other
provider fields in a fixed order. It returns a `*Cloud`. Its `Name` is the
identifier of the first provider that is not nil. `Facts.Cloud()` returns `nil`
if gohai found no cloud provider. This occurs on bare metal, on unknown
infrastructure, or when no cloud collector is enabled.

## Typical flow

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/osapi-io/gohai/pkg/gohai"
)

func main() {
    // Opt in to cloud collectors — none run by default.
    g, err := gohai.New(gohai.WithCategory("cloud"))
    if err != nil {
        log.Fatal(err)
    }
    facts, err := g.Collect(context.Background())
    if err != nil {
        log.Fatal(err)
    }

    cloud := facts.Cloud()
    if cloud == nil {
        fmt.Println("not running on a supported cloud")
        return
    }

    switch cloud.Name {
    case gohai.CloudAWS:
        // facts.Ec2 is guaranteed non-nil here
        fmt.Println("AWS account", facts.Ec2.AccountID,
            "region", facts.Ec2.Region,
            "profile", facts.Ec2.IAMInfo.InstanceProfileArn)
    case gohai.CloudGCE:
        fmt.Println("GCE project", facts.Gce.ProjectID,
            "zone", facts.Gce.Zone)
    case gohai.CloudAzure:
        fmt.Println("Azure subscription", facts.Azure.SubscriptionID,
            "location", facts.Azure.Location)
    }
}
```

## When to use `Cloud()` vs. the typed field directly

- **Code for all providers:** use `Cloud()`. Examples are a log line "this host
  is on \<cloud>", a fleet inventory, or a feature-flag gate.
- **Code for one provider:** if the code is only for AWS, do not use `Cloud()`.
  Check `facts.Ec2 != nil` directly. This is shorter and faster, and the
  compiler checks it.

## Enabling cloud collectors

All cloud collectors are opt-in (`DefaultEnabled() = false`). Use one of these
methods to enable them:

```bash
# CLI
gohai collect --category=cloud
gohai collect --collector.ec2 --collector.gce

# SDK
gohai.New(gohai.WithCategory("cloud"))
gohai.New(gohai.WithCollectors("ec2", "gce"))
gohai.New(gohai.WithEnabled("ec2"))
```

`WithCategory("cloud")` automatically enables all cloud collectors and their
`dmi` dependency. Many cloud collectors declare `dmi` as a dependency, but gohai
reads `dmi` only one time.

## Per-provider docs

The page of each provider gives the collection method, the fields, and the
differences from Ohai:

- [ec2.md](ec2.md). AWS EC2
- [gce.md](gce.md). Google Compute Engine
- [azure.md](azure.md). Microsoft Azure
- [digital_ocean.md](digital_ocean.md). DigitalOcean
- [oci.md](oci.md). Oracle Cloud Infrastructure
- [alibaba.md](alibaba.md). Alibaba Cloud ECS
- [linode.md](linode.md). Linode
- [openstack.md](openstack.md). OpenStack
- [scaleway.md](scaleway.md). Scaleway
