# Cloud Detection

> **Status:** Not a collector

This page documents the SDK pattern for detecting which cloud provider a host is
running on, and how to branch on that identity to access the right per-provider
typed fact data.

Unlike the per-provider collector docs in this directory, there is no `cloud`
collector in gohai. The detection flow is a small API on `gohai.Facts` that
inspects which per-provider field was populated.

## Why not an aggregator collector?

Earlier versions of this SDK exposed a `cloud` collector that merged
per-provider facts into a normalized view. That design had two problems:

1. **The normalization is lossy.** EC2's `Region` is `"us-east-1"`, Azure's
   equivalent is `Location: "eastus"`, OCI's is a combined
   `availabilityDomain: "pPrU:PHX-AD-1"`. Papering them over with a single
   `Cloud.Region` field throws away shape that consumers care about.
2. **It duplicates information.** Every provider's typed `Info` struct is
   already exposed on `Facts` (`Facts.Ec2`, `Facts.Gce`, …). An aggregator is
   another struct to maintain, synced manually with the per-provider structs on
   every field change.

So gohai takes a different tack: **expose provider identity at the SDK level,
point consumers to the per-provider field for rich data.**

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

`Facts.Cloud()` walks `Facts.Ec2`, `Facts.Gce`, `Facts.Azure`, etc. in a fixed
order and returns a `*Cloud` whose `Name` is set to the first non-nil provider's
identifier. Returns `nil` when no cloud provider was detected (bare metal,
unknown infrastructure, or no cloud collector was enabled).

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

- **Provider-agnostic code** (logging "this host is on \<cloud\>", fleet
  inventory, feature-flag gating) — use `Cloud()`.
- **Provider-specific code** that already knows it cares about AWS — skip
  `Cloud()` and check `facts.Ec2 != nil` directly. It's shorter, faster, and
  compile-time checked.

## Enabling cloud collectors

Every cloud collector is opt-in (`DefaultEnabled() = false`). Enable them one of
several ways:

```bash
# CLI
gohai --category=cloud
gohai --collector.ec2 --collector.gce

# SDK
gohai.New(gohai.WithCategory("cloud"))
gohai.New(gohai.WithCollectors("ec2", "gce"))
gohai.New(gohai.WithEnabled("ec2"))
```

`WithCategory("cloud")` pulls every cloud collector plus their `dmi` dependency
automatically — `dmi` is read once even though multiple cloud collectors declare
it as a dependency.

## Per-provider docs

Collection methodology, field coverage, and Ohai-parity details live in each
provider's own doc:

- [ec2.md](ec2.md) — AWS EC2
- [gce.md](gce.md) — Google Compute Engine
- [azure.md](azure.md) — Microsoft Azure
- [digital_ocean.md](digital_ocean.md) — DigitalOcean
- [oci.md](oci.md) — Oracle Cloud Infrastructure
- [alibaba.md](alibaba.md) — Alibaba Cloud ECS
- [linode.md](linode.md) — Linode
- [openstack.md](openstack.md) — OpenStack
- [scaleway.md](scaleway.md) — Scaleway
