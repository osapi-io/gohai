# Azure

> **Status:** Implemented ✅

## Description

Collects Azure VM metadata by hitting the link-local Instance Metadata Service
at `http://169.254.169.254/metadata/instance`. Requires the `Metadata: true`
header (Azure rejects requests without it) and pins the `api-version` query
parameter to `2023-07-01` — the latest version Ohai supports.
`facts.Azure != nil` is the detection signal.

Detection uses the Azure Linux Agent binary at `/usr/sbin/waagent` (Ohai's
`has_waagent?`). Hosts without that binary short-circuit with no HTTP call.

## Collected Fields

| Field                            | Type               | Description                                       | Schema mapping                 |
| -------------------------------- | ------------------ | ------------------------------------------------- | ------------------------------ |
| `vm_id`                          | `string`           | VM UUID.                                          | OTel `cloud.resource_id`       |
| `name`                           | `string`           | VM name.                                          | OTel `host.name`               |
| `vm_size`                        | `string`           | VM SKU (e.g. `Standard_D2s_v3`).                  | OTel `host.type`               |
| `resource_id`                    | `string`           | Full ARM resource ID.                             | No direct schema mapping.      |
| `resource_group_name`            | `string`           | Resource group name.                              | No direct schema mapping.      |
| `vm_scale_set_name`              | `string`           | VMSS name (if part of one).                       | No direct schema mapping.      |
| `priority`                       | `string`           | `Regular` / `Low` / `Spot`.                       | No direct schema mapping.      |
| `eviction_policy`                | `string`           | Spot eviction policy.                             | No direct schema mapping.      |
| `location`                       | `string`           | Azure region (e.g. `eastus`).                     | OTel `cloud.region`            |
| `zone`                           | `string`           | Availability zone (e.g. `1`).                     | OTel `cloud.availability_zone` |
| `placement_group_id`             | `string`           | Placement group.                                  | No direct schema mapping.      |
| `platform_fault_domain`          | `string`           | Fault domain index.                               | No direct schema mapping.      |
| `platform_update_domain`         | `string`           | Update domain index.                              | No direct schema mapping.      |
| `subscription_id`                | `string`           | Azure subscription UUID.                          | OTel `cloud.account.id`        |
| `az_environment`                 | `string`           | `AzurePublicCloud` / `AzureUSGovernment` / etc.   | No direct schema mapping.      |
| `offer`                          | `string`           | Marketplace offer.                                | No direct schema mapping.      |
| `publisher`                      | `string`           | Marketplace publisher.                            | No direct schema mapping.      |
| `sku`                            | `string`           | Marketplace SKU.                                  | No direct schema mapping.      |
| `version`                        | `string`           | Marketplace image version.                        | OTel `host.image.version`      |
| `license_type`                   | `string`           | BYO license type.                                 | No direct schema mapping.      |
| `os_type`                        | `string`           | `Linux` / `Windows`.                              | OTel `os.type`                 |
| `provider`                       | `string`           | Azure RP (usually `Microsoft.Compute`).           | No direct schema mapping.      |
| `plan`                           | `*Plan`            | Marketplace plan.                                 | No direct schema mapping.      |
| `storage_profile`                | `*StorageProfile`  | OS disk + data disks — see below.                 | No direct schema mapping.      |
| `tags`                           | `string`           | Semicolon-delimited `key:value` tag list.         | No direct schema mapping.      |
| `tags_list`                      | `[]Tag`            | Parsed tags.                                      | No direct schema mapping.      |
| `user_data`                      | `string`           | Base64-encoded custom data.                       | No direct schema mapping.      |
| `custom_data`                    | `string`           | Legacy custom data field.                         | No direct schema mapping.      |
| `is_host_compatibility_layer_vm` | `bool`             | Compatibility-layer indicator.                    | No direct schema mapping.      |
| `security_profile`               | `*SecurityProfile` | Secure boot / vTPM / encryption flags.            | No direct schema mapping.      |
| `public_keys`                    | `[]PublicKey`      | SSH keys attached via the profile.                | No direct schema mapping.      |
| `interfaces`                     | `[]Interface`      | VNICs (with full per-family subnets + addresses). | OCSF `network_interface`       |
| `public_ipv4` / `local_ipv4`     | `[]string`         | Flat IP lists across all interfaces.              | No direct schema mapping.      |
| `public_ipv6` / `local_ipv6`     | `[]string`         | Flat IPv6 lists across all interfaces.            | No direct schema mapping.      |

Sub-struct details (`Plan`, `StorageProfile`, `Disk`, `ManagedDisk`, `Tag`,
`SecurityProfile`, `PublicKey`, `Interface`, `IPAddrs`, `IPAddress`, `Subnet`)
live on pkg.go.dev.

## Platform Support

| Platform | Supported                           |
| -------- | ----------------------------------- |
| Linux    | ✅                                  |
| macOS    | ✅ (only meaningful on an Azure VM) |
| Other    | ✅ (only meaningful on an Azure VM) |

## Example Output

```json
{
  "azure": {
    "vm_id": "abcd-1234",
    "name": "web-1",
    "vm_size": "Standard_D2s_v3",
    "location": "eastus",
    "zone": "1",
    "subscription_id": "sub-uuid",
    "interfaces": [
      {
        "mac_address": "000D3A...",
        "ipv4": { "ip_addresses": [{ "private_ip": "10.0.0.4" }] }
      }
    ]
  }
}
```

## SDK Usage

```go
g, _ := gohai.New(gohai.WithCollectors("azure"))
facts, _ := g.Collect(context.Background())

if facts.Azure != nil {
    fmt.Println(facts.Azure.SubscriptionID, facts.Azure.Location)
}
```

## Enable/Disable

```bash
gohai --collector.azure      # enable (opt-in)
gohai --no-collector.azure   # disable (default)
gohai --category=cloud       # pulls this + all cloud collectors
```

## Dependencies

None. Detection uses the presence of `/usr/sbin/waagent` (Ohai's
`has_waagent?`), not DMI.

## Data Sources

1. **waagent gate:** `os.Stat("/usr/sbin/waagent")` — if absent, short circuit.
   Matches Ohai's has_waagent? on Linux. Hosts without waagent installed return
   `(nil, nil)` immediately.
2. **Endpoint:**
   `http://169.254.169.254/metadata/instance?api-version=2023-07-01`.
3. **Required header:** `Metadata: true` — Azure rejects requests without it.
   Protects against lateral SSRF.
4. **API version:** hardcoded to `2023-07-01` (latest Ohai supports). Azure's
   version-negotiation handshake (which responds 400 with `newest-versions` list
   when no `api-version` is sent) is skipped for simplicity — if Azure retires
   `2023-07-01` we'll bump.
5. **Timeout:** 2 seconds.
6. **Failure handling:** transport / 404 / etc. → `(nil, nil)`.
7. **Transformation:** per-interface IP addresses are aggregated into flat
   top-level `public_ipv4` / `local_ipv4` / `public_ipv6` / `local_ipv6` lists
   in addition to the nested per-interface arrays.

Mirrors Ohai's `Ohai::Mixin::AzureMetadata` collection approach, with the
version-negotiation handshake simplified to a single hardcoded latest version.

## Backing library

- [`internal/cloudmetadata`](../../internal/cloudmetadata/) — the shared HTTP
  client used by every cloud-provider collector.
