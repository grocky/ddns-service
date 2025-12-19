# DDNS Service

**A serverless Dynamic DNS service that automatically updates DNS records when your IP changes.**

Stop losing access to your home network. Whether you're running a home lab, self-hosting services, or need reliable remote access, DDNS Service automatically tracks your public IP and updates Route53 DNS records so you always have a stable hostname to connect to.

## Why DDNS Service?

Most residential and small business internet connections have dynamic IP addresses that change periodically. This creates a problem: how do you reliably connect to your home network when you're away?

Traditional solutions require:
- Paying for expensive static IP addresses
- Relying on third-party DDNS providers with uncertain privacy practices
- Complex router configurations

**DDNS Service** gives you a simple, privacy-respecting alternative that you control.

## Features

- **Automatic DNS Updates** - Updates Route53 A records when your IP changes
- **Stable Subdomains** - Each location gets a permanent subdomain (e.g., `a3f8c2d1.grocky.net`)
- **Simple REST API** - Poll the `/update` endpoint and let the service handle the rest
- **Secure by default** - API key authentication protects your data
- **Rate limited** - Prevents abuse with a maximum of 2 IP changes per hour
- **Multi-location support** - Track IPs for multiple locations (home, office, cabin, etc.)
- **Serverless architecture** - Scales automatically, pay only for what you use
- **Privacy-first** - Self-hostable, your data stays yours

## Architecture

```mermaid
flowchart TB
    subgraph clients["Your Devices"]
        daemon["pubip CLI\n(scheduled daemon)"]
        cron["cron job\n(curl)"]
        ha["Home Assistant\n(coming soon)"]
    end

    subgraph aws["AWS Cloud"]
        apigw["API Gateway\n(ddns.rockygray.com)"]
        lambda["Lambda Function\n(Go)"]
        dynamodb[("DynamoDB\n(Owners + IP Mappings)")]
        route53["Route53\n(DNS Records)"]
        ses["SES\n(Email)"]
    end

    subgraph endpoints["API Endpoints"]
        direction LR
        e1["GET /public-ip"]
        e2["POST /owners"]
        e3["POST /update"]
        e4["GET /lookup/{owner}/{location}"]
    end

    daemon -->|"poll /update\n(with API key)"| apigw
    cron -->|"poll /update\n(with API key)"| apigw
    ha -->|"poll /update\n(with API key)"| apigw

    apigw --> lambda
    lambda --> dynamodb
    lambda -->|"update A record"| route53
    lambda --> ses
    lambda --> endpoints

    remote["Remote Access\n(You, away from home)"] -->|"DNS lookup"| route53
    remote -->|"Connect via\na3f8c2d1.grocky.net"| clients
```

## How It Works

1. **Create an owner account** - Get your API key
2. **Poll the `/update` endpoint** - The service detects your IP automatically
3. **Automatic DNS updates** - When your IP changes, the service updates Route53
4. **Connect using your subdomain** - Use your stable hostname (e.g., `a3f8c2d1.grocky.net`)

Each owner/location combination gets a deterministic subdomain based on a hash of `{ownerId}-{location}`. This subdomain never changes, even when your IP does.

## API Reference

### Authentication

Most endpoints require authentication via API key. Include your API key in the `Authorization` header:

```
Authorization: Bearer ddns_sk_your_api_key_here
```

| Endpoint | Authentication | Rate Limited |
|----------|----------------|--------------|
| `GET /public-ip` | Not required | No |
| `POST /owners` | Not required | No |
| `POST /owners/{id}/recover` | Not required | No |
| `POST /owners/{id}/rotate` | Required | No |
| `POST /update` | Required | Yes (2/hour) |
| `GET /lookup/{owner}/{location}` | Required | No |

### Get Your Public IP

Returns the public IP address of the caller. No authentication required.

```bash
curl https://ddns.rockygray.com/public-ip
```

```json
{
  "publicIp": "203.0.113.42"
}
```

### Create an Owner Account

Create an owner account to receive your API key. **Save your API key securely - it is only shown once!**

```bash
curl -X POST https://ddns.rockygray.com/owners \
  -H "Content-Type: application/json" \
  -d '{
    "ownerId": "my-home-lab",
    "email": "you@example.com"
  }'
```

```json
{
  "ownerId": "my-home-lab",
  "email": "you@example.com",
  "apiKey": "ddns_sk_7Kx9mP2qR5vW8yB3nF6hJ4tL1cA0eD9gXXXXXXXXXXXX",
  "createdAt": "2025-01-15T10:30:00Z"
}
```

### Update DNS (Poll Endpoint)

Poll this endpoint to update your DNS record. The service automatically:
1. Detects your public IP from the request
2. Checks if it has changed since the last update
3. Updates Route53 if needed
4. Returns your stable subdomain

**Rate limited to 2 IP changes per hour.**

```bash
curl -X POST https://ddns.rockygray.com/update \
  -H "Authorization: Bearer ddns_sk_your_api_key_here" \
  -H "Content-Type: application/json" \
  -d '{
    "ownerId": "my-home-lab",
    "location": "home"
  }'
```

**Response (IP unchanged or updated):**
```json
{
  "ownerId": "my-home-lab",
  "location": "home",
  "ip": "203.0.113.42",
  "subdomain": "6abf7de6.grocky.net",
  "changed": false,
  "updatedAt": "2025-01-15T10:30:00Z"
}
```

**Response (429 Too Many Requests - rate limit exceeded):**
```json
{
  "description": "rate limit exceeded: maximum 2 IP changes per hour"
}
```
The `Retry-After` header indicates how many seconds until you can try again.

### Lookup an IP Address

Retrieve the registered IP and subdomain for a specific owner and location. **Requires authentication.**

```bash
curl https://ddns.rockygray.com/lookup/my-home-lab/home \
  -H "Authorization: Bearer ddns_sk_your_api_key_here"
```

```json
{
  "ownerId": "my-home-lab",
  "location": "home",
  "ip": "203.0.113.42",
  "subdomain": "6abf7de6.grocky.net",
  "updatedAt": "2025-01-15T10:30:00Z"
}
```

### Rotate API Key

Generate a new API key (invalidates the old one). **Requires authentication with your current key.**

```bash
curl -X POST https://ddns.rockygray.com/owners/my-home-lab/rotate \
  -H "Authorization: Bearer ddns_sk_your_current_key"
```

```json
{
  "ownerId": "my-home-lab",
  "apiKey": "ddns_sk_new_key_here",
  "rotatedAt": "2025-01-15T10:30:00Z"
}
```

### Recover API Key

If you lose your API key, request a new one via email. A new key will be sent to the email address on file.

```bash
curl -X POST https://ddns.rockygray.com/owners/my-home-lab/recover \
  -H "Content-Type: application/json" \
  -d '{"email": "you@example.com"}'
```

```json
{
  "message": "If this email matches our records, a new API key has been sent."
}
```

## Getting Started

### Option 1: Use the Hosted Service

The easiest way to get started is to use the hosted service at `ddns.rockygray.com`.

**Step 1: Create your owner account**

```bash
curl -X POST https://ddns.rockygray.com/owners \
  -H "Content-Type: application/json" \
  -d '{"ownerId":"my-home","email":"you@example.com"}'
```

Save the `apiKey` from the response - you'll need it for all future requests!

**Step 2: Set up automatic polling with cron**

```bash
# Add to crontab (runs every 15 minutes)
*/15 * * * * curl -s -X POST https://ddns.rockygray.com/update \
  -H "Authorization: Bearer ddns_sk_your_api_key_here" \
  -H "Content-Type: application/json" \
  -d '{"ownerId":"my-home","location":"home"}' > /dev/null
```

**Or create a simple shell script:**

```bash
#!/bin/bash
# save as: update-ddns.sh

OWNER_ID="my-home"
LOCATION="home"
API_KEY="ddns_sk_your_api_key_here"

response=$(curl -s -X POST https://ddns.rockygray.com/update \
  -H "Authorization: Bearer ${API_KEY}" \
  -H "Content-Type: application/json" \
  -d "{\"ownerId\":\"${OWNER_ID}\",\"location\":\"${LOCATION}\"}")

# Extract subdomain and check if changed
subdomain=$(echo "$response" | jq -r '.subdomain')
changed=$(echo "$response" | jq -r '.changed')

if [ "$changed" = "true" ]; then
    echo "IP updated! Your subdomain: $subdomain"
else
    echo "IP unchanged. Your subdomain: $subdomain"
fi
```

**Step 3: Connect using your subdomain**

Once set up, you can always connect to your home network using your stable subdomain:

```bash
ssh user@6abf7de6.grocky.net
```

### Option 2: Use the pubip CLI

The `pubip` CLI tool queries multiple IP detection services and uses consensus to reliably determine your public IP.

```bash
# Build the CLI
make build-pubip

# Get your IPv4 address (default)
./bin/pubip
203.0.113.42

# Get your IPv6 address
./bin/pubip -6
2601:123:4567:89ab::1
```

## Use Cases

### Home Lab Access

Keep track of your home lab's public IP and always connect via a stable hostname:

```bash
# First, create your owner account (one-time setup)
curl -X POST https://ddns.rockygray.com/owners \
  -H "Content-Type: application/json" \
  -d '{"ownerId":"homelab","email":"you@example.com"}'
# Save the API key!

# On your home server (via cron)
*/15 * * * * curl -s -X POST https://ddns.rockygray.com/update \
  -H "Authorization: Bearer ddns_sk_your_api_key" \
  -H "Content-Type: application/json" \
  -d '{"ownerId":"homelab","location":"primary"}' > /dev/null

# From your laptop, anywhere in the world - use your stable subdomain
ssh user@a1b2c3d4.grocky.net
```

### Multi-Site Monitoring

Track IP addresses across multiple locations:

```bash
# Create owner account first
curl -X POST https://ddns.rockygray.com/owners \
  -H "Content-Type: application/json" \
  -d '{"ownerId":"acme-corp","email":"admin@acme.com"}'

# Set up polling at each location
# At headquarters:
curl -X POST https://ddns.rockygray.com/update \
  -H "Authorization: Bearer ddns_sk_your_api_key" \
  -H "Content-Type: application/json" \
  -d '{"ownerId":"acme-corp","location":"headquarters"}'

# At warehouse:
curl -X POST https://ddns.rockygray.com/update \
  -H "Authorization: Bearer ddns_sk_your_api_key" \
  -H "Content-Type: application/json" \
  -d '{"ownerId":"acme-corp","location":"warehouse"}'

# Each location gets its own stable subdomain
# headquarters: abc12345.grocky.net
# warehouse: def67890.grocky.net
```

### VPN Endpoint Discovery

Use your stable subdomain directly in your VPN client configuration:

```
# /etc/openvpn/client.conf
remote a1b2c3d4.grocky.net 1194
```

No need to update the configuration when your IP changes - the subdomain always points to your current IP!

## Rate Limiting

The `/update` endpoint is rate limited to **2 IP changes per hour** per owner/location. This prevents abuse while allowing for normal IP changes.

- Polling when your IP hasn't changed does NOT count against the limit
- Only actual IP changes count toward the limit
- The limit resets at the top of each hour
- When rate limited, the response includes a `Retry-After` header

## Roadmap

- **Home Assistant Integration** - A native Home Assistant plugin for seamless smart home integration
- **IP Change Webhooks** - Get notified when your IP address changes
- **Historical IP Tracking** - View IP address change history
- **Custom Domain Support** - Use your own domain for the API

## Self-Hosting

DDNS Service is open source and can be self-hosted on your own AWS account. See the `terraform/` directory for infrastructure-as-code deployment.

```bash
# Initialize Terraform
make tf-init

# Deploy to your AWS account
make deploy
```

After deploying the Route53 zone, update your domain registrar's nameservers to the values from the Terraform output.

## License

MIT License - See [LICENSE](LICENSE) for details.
