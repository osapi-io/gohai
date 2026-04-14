# EC2

> **Status:** Implemented ✅

## Description

Collects AWS EC2 instance metadata from the link-local server at
`http://169.254.169.254/latest/`. Uses IMDSv2 (token-authenticated) by default
and falls back to IMDSv1 when the token endpoint isn't reachable.
`facts.Ec2 != nil` is the detection signal.

Detection is gated on DMI `bios_vendor` containing `"Amazon"` — matches Ohai's
`has_ec2_amazon_dmi?`. Three classes of metadata are fetched: the `/meta-data/`
tree, the IAM instance info (`iam/info`), and the dynamic instance identity
document (`/dynamic/instance-identity/document`) which provides the canonical
`accountId` + `region` + `availabilityZone` fields.

## Collected Fields

| Field                 | Type               | Description                              | Schema mapping                 |
| --------------------- | ------------------ | ---------------------------------------- | ------------------------------ |
| `instance_id`         | `string`           | EC2 instance ID.                         | OTel `cloud.resource_id`       |
| `instance_type`       | `string`           | Instance type (e.g. `t3.micro`).         | OTel `host.type`               |
| `instance_life_cycle` | `string`           | `on-demand` / `spot` / `scheduled`.      | No direct schema mapping.      |
| `ami_id`              | `string`           | Boot AMI ID.                             | OTel `host.image.id`           |
| `ami_launch_index`    | `string`           | Launch index within the reservation.     | No direct schema mapping.      |
| `ami_manifest_path`   | `string`           | Legacy manifest path.                    | No direct schema mapping.      |
| `hostname`            | `string`           | EC2-supplied hostname.                   | No direct schema mapping.      |
| `local_hostname`      | `string`           | Internal DNS name.                       | OCSF `device.hostname`         |
| `public_hostname`     | `string`           | Public DNS name (if assigned).           | No direct schema mapping.      |
| `local_ipv4`          | `string`           | Primary private IPv4.                    | No direct schema mapping.      |
| `public_ipv4`         | `string`           | Primary public IPv4.                     | No direct schema mapping.      |
| `mac`                 | `string`           | Primary interface MAC.                   | OCSF `network_interface.mac`   |
| `security_groups`     | `[]string`         | Security group names.                    | No direct schema mapping.      |
| `region`              | `string`           | AWS region (e.g. `us-east-1`).           | OTel `cloud.region`            |
| `availability_zone`   | `string`           | Availability zone (e.g. `us-east-1a`).   | OTel `cloud.availability_zone` |
| `account_id`          | `string`           | AWS account ID (from identity document). | OTel `cloud.account.id`        |
| `reservation_id`      | `string`           | Reservation ID.                          | No direct schema mapping.      |
| `profile`             | `string`           | Virtualization profile (`default-hvm`).  | No direct schema mapping.      |
| `iam_info`            | `*IAMInstanceInfo` | Attached IAM profile — see below.        | No direct schema mapping.      |

### IAMInstanceInfo

| Field                  | Type     | Description                           |
| ---------------------- | -------- | ------------------------------------- |
| `code`                 | `string` | `Success` or error code.              |
| `last_updated`         | `string` | Last-rotated timestamp.               |
| `instance_profile_arn` | `string` | Full ARN of the attached IAM profile. |
| `instance_profile_id`  | `string` | IAM profile ID.                       |

`security-credentials` (access keys) is **deliberately dropped** — matches
Ohai's explicit scrub of the secrets-bearing sub-tree.

## Platform Support

| Platform | Supported                               |
| -------- | --------------------------------------- |
| Linux    | ✅                                      |
| macOS    | ✅ (only meaningful on an EC2 instance) |
| Other    | ✅ (only meaningful on an EC2 instance) |

## Example Output

```json
{
  "ec2": {
    "instance_id": "i-abc",
    "instance_type": "t3.micro",
    "region": "us-east-1",
    "availability_zone": "us-east-1a",
    "local_ipv4": "10.0.0.5",
    "public_ipv4": "1.2.3.4",
    "account_id": "123456789012",
    "iam_info": {
      "code": "Success",
      "instance_profile_arn": "arn:aws:iam::123456789012:instance-profile/web"
    }
  }
}
```

## SDK Usage

```go
g, _ := gohai.New(gohai.WithCollectors("ec2"))
facts, _ := g.Collect(context.Background())

if facts.Ec2 != nil {
    fmt.Println(facts.Ec2.Region, facts.Ec2.AccountID)
}
```

## Enable/Disable

```bash
gohai --collector.ec2      # enable (opt-in)
gohai --no-collector.ec2   # disable (default)
gohai --category=cloud     # pulls this + all cloud collectors
```

## Dependencies

`dmi`. EC2 writes `"Amazon EC2"` / `"Amazon"` as `bios_vendor`. Fails open when
dmi wasn't run.

## Data Sources

1. **DMI gate:** `dmi.BIOS.Vendor` contains `"Amazon"`. Matches Ohai's
   `has_ec2_amazon_dmi?`.
2. **IMDSv2 token:** `PUT /latest/api/token` with
   `X-aws-ec2-metadata-token-ttl-seconds: 60`. Response body is the token
   string. A 404 here triggers IMDSv1 fallback (no token header on subsequent
   requests), matching Ohai's behavior.
3. **Meta-data tree:** GETs against a curated list of well-known
   `/latest/meta-data/*` paths. All requests carry the token header when a token
   was obtained.
4. **IAM info:** `GET /latest/meta-data/iam/info` — parsed as JSON. Absent on
   instances without an attached role (tolerated).
5. **Identity document:** `GET /latest/dynamic/instance-identity/document` —
   JSON with `accountId`, `region`, `availabilityZone`, `instanceId`. These fill
   the top-level fields when the meta-data tree doesn't have them directly.
6. **Timeout:** 2 seconds per request.
7. **Failure handling:** first-probe (`/ami-id`) failure returns `(nil, nil)`.
   Per-path failures on subsequent paths are tolerated.

Mirrors Ohai's `Ohai::Mixin::Ec2Metadata` collection approach, with the same
IMDSv2→IMDSv1 fallback behavior and the same security scrub of IAM credentials.

## Backing library

- [`internal/cloudmetadata`](../../internal/cloudmetadata/) — the shared HTTP
  client used by every cloud-provider collector.
