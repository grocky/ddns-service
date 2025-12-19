# DDNS Service Runbook

## Verify A Record for Owner/Location

Use this procedure to verify that a DDNS A record was correctly created and is resolving.

### Prerequisites

- AWS CLI configured with appropriate credentials
- Access to the Route53 hosted zone (Zone ID: `Z030530123PNW3FUSMWUZ`)
- `dig` command available

### Procedure

#### 1. Calculate the subdomain hash

The subdomain is the first 8 characters of the MD5 hash of `{ownerId}-{location}`:

```bash
echo -n "{ownerId}-{location}" | md5 | cut -c1-8
```

**Example:**
```bash
echo -n "grocky-home" | md5 | cut -c1-8
# Output: a7793107
```

#### 2. Verify the Route53 A record

Check that the A record exists in Route53:

```bash
aws route53 list-resource-record-sets \
  --hosted-zone-id Z030530123PNW3FUSMWUZ \
  --query "ResourceRecordSets[?Name=='{subdomain}.grocky.net.']"
```

**Example:**
```bash
aws route53 list-resource-record-sets \
  --hosted-zone-id Z030530123PNW3FUSMWUZ \
  --query "ResourceRecordSets[?Name=='a7793107.grocky.net.']"
```

**Expected output:**
```json
[
    {
        "Name": "a7793107.grocky.net.",
        "Type": "A",
        "TTL": 300,
        "ResourceRecords": [
            {
                "Value": "x.x.x.x"
            }
        ]
    }
]
```

#### 3. Verify DNS resolution

Confirm the record is resolving via public DNS:

```bash
dig +short {subdomain}.grocky.net A
```

**Example:**
```bash
dig +short a7793107.grocky.net A
# Output: 100.36.157.144
```

#### 4. Verify DynamoDB record (optional)

Cross-check with the DynamoDB record:

```bash
aws dynamodb get-item \
  --table-name DdnsServiceIpMapping \
  --key '{"OwnerId": {"S": "{ownerId}"}, "LocationName": {"S": "{location}"}}'
```

**Example:**
```bash
aws dynamodb get-item \
  --table-name DdnsServiceIpMapping \
  --key '{"OwnerId": {"S": "grocky"}, "LocationName": {"S": "home"}}'
```

**Expected fields:**
- `IP`: Should match the Route53 A record value
- `Subdomain`: Should match the calculated hash (e.g., `a7793107`)
- `HourlyChangeCount`: Number of IP changes in the current hour
- `LastIPChangeAt`: Timestamp of last IP change

### Troubleshooting

| Issue | Possible Cause | Resolution |
|-------|---------------|------------|
| No Route53 record | Update endpoint not called or failed | Check Lambda logs in CloudWatch |
| DNS not resolving | DNS propagation delay | Wait 5 minutes and retry |
| IP mismatch between Route53 and DynamoDB | Partial update failure | Check Lambda logs; may need manual reconciliation |
| Rate limit exceeded | More than 2 IP changes per hour | Wait until the next hour |

### Useful Commands

**List all A records in the zone:**
```bash
aws route53 list-resource-record-sets \
  --hosted-zone-id Z030530123PNW3FUSMWUZ \
  --query "ResourceRecordSets[?Type=='A']"
```

**Check Lambda logs:**
```bash
aws logs tail /aws/lambda/ddns-service --follow
```

**Get owner info:**
```bash
aws dynamodb get-item \
  --table-name DdnsServiceOwners \
  --key '{"OwnerId": {"S": "{ownerId}"}}'
```
