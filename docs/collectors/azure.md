# Azure

> **Status:** Implemented ✅

## Description

Collects Azure VM metadata by hitting the link-local Instance Metadata Service
at `http://169.254.169.254/metadata/instance`. Requires the `Metadata: true`
header (Azure rejects requests without it) and **negotiates an api-version**
with the service at collect time — matches Ohai's `best_api_version` handshake.
`facts.Azure != nil` is the detection signal.

Detection uses two Linux signals (matches Ohai's non-Windows chain):

- `/usr/sbin/waagent` exists (Azure Linux Agent installed)
- `/var/lib/dhcp/dhclient.eth0.leases` contains the DHCP option-245 signature
  (`unknown-245`)

If neither fires, the collector short-circuits with no HTTP call.

## Collected Fields

| Field                            | Type                      | Description                                                                                             | Schema mapping                 |
| -------------------------------- | ------------------------- | ------------------------------------------------------------------------------------------------------- | ------------------------------ |
| `vm_id`                          | `string`                  | VM UUID.                                                                                                | OTel `cloud.resource_id`       |
| `name`                           | `string`                  | VM name.                                                                                                | OTel `host.name`               |
| `vm_size`                        | `string`                  | VM SKU (e.g. `Standard_D2s_v3`).                                                                        | OTel `host.type`               |
| `resource_id`                    | `string`                  | Full ARM resource ID.                                                                                   | No direct schema mapping.      |
| `resource_group_name`            | `string`                  | Resource group name.                                                                                    | No direct schema mapping.      |
| `vm_scale_set_name`              | `string`                  | VMSS name (if part of one).                                                                             | No direct schema mapping.      |
| `priority`                       | `string`                  | `Regular` / `Low` / `Spot`.                                                                             | No direct schema mapping.      |
| `eviction_policy`                | `string`                  | Spot eviction policy.                                                                                   | No direct schema mapping.      |
| `location`                       | `string`                  | Azure region (e.g. `eastus`).                                                                           | OTel `cloud.region`            |
| `zone`                           | `string`                  | Availability zone (e.g. `1`).                                                                           | OTel `cloud.availability_zone` |
| `placement_group_id`             | `string`                  | Placement group.                                                                                        | No direct schema mapping.      |
| `platform_fault_domain`          | `string`                  | Fault domain index.                                                                                     | No direct schema mapping.      |
| `platform_update_domain`         | `string`                  | Update domain index.                                                                                    | No direct schema mapping.      |
| `subscription_id`                | `string`                  | Azure subscription UUID.                                                                                | OTel `cloud.account.id`        |
| `az_environment`                 | `string`                  | `AzurePublicCloud` / `AzureUSGovernment` / etc.                                                         | No direct schema mapping.      |
| `offer`                          | `string`                  | Marketplace offer.                                                                                      | No direct schema mapping.      |
| `publisher`                      | `string`                  | Marketplace publisher.                                                                                  | No direct schema mapping.      |
| `sku`                            | `string`                  | Marketplace SKU.                                                                                        | No direct schema mapping.      |
| `version`                        | `string`                  | Marketplace image version.                                                                              | OTel `host.image.version`      |
| `license_type`                   | `string`                  | BYO license type.                                                                                       | No direct schema mapping.      |
| `os_type`                        | `string`                  | `Linux` / `Windows`.                                                                                    | OTel `os.type`                 |
| `provider`                       | `string`                  | Azure RP (usually `Microsoft.Compute`).                                                                 | No direct schema mapping.      |
| `plan`                           | `*Plan`                   | Marketplace plan.                                                                                       | No direct schema mapping.      |
| `storage_profile`                | `*StorageProfile`         | OS disk + data disks — see below.                                                                       | No direct schema mapping.      |
| `tags`                           | `string`                  | Semicolon-delimited `key:value` tag list.                                                               | No direct schema mapping.      |
| `tags_list`                      | `[]Tag`                   | Parsed tags.                                                                                            | No direct schema mapping.      |
| `user_data`                      | `string`                  | Base64-encoded custom data.                                                                             | No direct schema mapping.      |
| `custom_data`                    | `string`                  | Legacy custom data field.                                                                               | No direct schema mapping.      |
| `is_host_compatibility_layer_vm` | `bool`                    | Compatibility-layer indicator.                                                                          | No direct schema mapping.      |
| `security_profile`               | `*SecurityProfile`        | Secure boot / vTPM / encryption flags.                                                                  | No direct schema mapping.      |
| `public_keys`                    | `[]PublicKey`             | SSH keys attached via the profile.                                                                      | No direct schema mapping.      |
| `host`                           | `*Host`                   | Azure Dedicated Host the VM is pinned to (`{id}`). Empty on pooled-host VMs.                            | No direct schema mapping.      |
| `host_group`                     | `*HostGroup`              | Azure Dedicated Host Group containing the host.                                                         | No direct schema mapping.      |
| `os_profile`                     | `*OSProfile`              | OS provisioning settings (`admin_username`, `computer_name`, `disable_password_authentication`).        | No direct schema mapping.      |
| `additional_capabilities`        | `*AdditionalCapabilities` | Feature toggles (`hibernation_enabled`). Values are Azure-style string booleans (`"true"` / `"false"`). | No direct schema mapping.      |
| `extended_location`              | `*ExtendedLocation`       | Non-standard placement (`{name, type}`) for edge zones / Azure Arc / Azure Local.                       | No direct schema mapping.      |
| `interfaces`                     | `map[string]Interface`    | VNICs keyed by MAC address (matches Ohai's `metadata.network.interfaces[<mac>]` shape).                 | OCSF `network_interface`       |
| `public_ipv4` / `local_ipv4`     | `[]string`                | Flat IP lists aggregated across all interfaces.                                                         | No direct schema mapping.      |
| `public_ipv6` / `local_ipv6`     | `[]string`                | Flat IPv6 lists aggregated across all interfaces.                                                       | No direct schema mapping.      |

Sub-struct details (`Plan`, `StorageProfile`, `Disk`, `ManagedDisk`, `Tag`,
`SecurityProfile`, `PublicKey`, `Host`, `HostGroup`, `OSProfile`,
`AdditionalCapabilities`, `ExtendedLocation`, `Interface`, `IPAddrs`,
`IPAddress`, `Subnet`) live on pkg.go.dev.

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

None. Detection uses Azure-specific file signals (waagent binary + DHCP option
245), not DMI.

## Data Sources

1. **Detection gate:** returns `(nil, nil)` unless one of these is true:
   - `/usr/sbin/waagent` exists (Azure Linux Agent installed). Matches Ohai's
     `has_waagent?`.
   - `/var/lib/dhcp/dhclient.eth0.leases` contains `"unknown-245"`. Matches
     Ohai's `has_dhcp_option_245?`.
2. **API version negotiation:** `GET /metadata/instance` without `api-version` —
   Azure returns HTTP 400 with `{"newest-versions": [...]}`. We intersect with
   our supported-versions list and pick the latest. Falls back to the latest
   hardcoded version on any negotiation failure. Matches Ohai's
   `best_api_version` in `mixin/azure_metadata.rb`.
3. **Metadata fetch:** `GET /metadata/instance?api-version=<negotiated>`.
4. **Required header:** `Metadata: true` — Azure rejects requests without it.
   Protects against lateral SSRF.
5. **User-Agent:** `gohai` (the cloudmetadata default).
6. **Timeout:** 6 seconds — matches Ohai's `read_timeout`.
7. **Failure handling:** transport / 404 / etc. → `(nil, nil)`.
8. **Transformation:** per-interface IP addresses are aggregated into flat
   top-level `public_ipv4` / `local_ipv4` / `public_ipv6` / `local_ipv6` lists
   in addition to the per-interface map keyed by MAC address.

Mirrors Ohai's `Ohai::Mixin::AzureMetadata` methodology — same detection chain
(non-Windows signals), same version-negotiation handshake, same 6s timeout, same
MAC-keyed network-interfaces structure. Windows-specific detection signals
(`C:\WindowsAzure`, `DhcpDomain` registry key) are not implemented.

## Backing library

- [`internal/cloudmetadata`](../../internal/cloudmetadata/) — the shared HTTP
  client used by every cloud-provider collector.
