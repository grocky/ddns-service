# DDNS Service

**A serverless Dynamic DNS service for keeping track of your home or office public IP address.**

Stop losing access to your home network. Whether you're running a home lab, self-hosting services, or need reliable remote access, DDNS Service automatically tracks your public IP so you always know how to connect back home.

## Why DDNS Service?

Most residential and small business internet connections have dynamic IP addresses that change periodically. This creates a problem: how do you reliably connect to your home network when you're away?

Traditional solutions require:
- Paying for expensive static IP addresses
- Relying on third-party DDNS providers with uncertain privacy practices
- Complex router configurations

**DDNS Service** gives you a simple, privacy-respecting alternative that you control.

## Features

- **Simple REST API** - Register and lookup IPs with straightforward HTTP calls
- **Multi-location support** - Track IPs for multiple locations (home, office, cabin, etc.)
- **Multi-tenant** - Support multiple users/owners with isolated data
- **Instant IP detection** - Get your current public IP with a single request
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
        dynamodb[("DynamoDB\n(IP Mappings)")]
    end

    subgraph endpoints["API Endpoints"]
        direction LR
        e1["GET /public-ip"]
        e2["POST /register"]
        e3["GET /lookup/{owner}/{location}"]
    end

    daemon -->|"register IP"| apigw
    cron -->|"register IP"| apigw
    ha -->|"register IP"| apigw

    apigw --> lambda
    lambda --> dynamodb
    lambda --> endpoints

    remote["Remote Access\n(You, away from home)"] -->|"lookup IP"| apigw
```

## API Reference

### Get Your Public IP

Returns the public IP address of the caller.

```bash
curl https://ddns.rockygray.com/public-ip
```

```json
{
  "publicIp": "203.0.113.42"
}
```

### Register an IP Address

Register or update the IP address for a location. Use `"ip": "auto"` to automatically use your current public IP.

```bash
curl -X POST https://ddns.rockygray.com/register \
  -H "Content-Type: application/json" \
  -d '{
    "ownerId": "your-unique-id",
    "location": "home",
    "ip": "auto"
  }'
```

```json
{
  "ownerId": "your-unique-id",
  "location": "home",
  "ip": "203.0.113.42",
  "updatedAt": "2025-01-15T10:30:00Z"
}
```

### Lookup an IP Address

Retrieve the registered IP for a specific owner and location.

```bash
curl https://ddns.rockygray.com/lookup/your-unique-id/home
```

```json
{
  "ownerId": "your-unique-id",
  "location": "home",
  "ip": "203.0.113.42",
  "updatedAt": "2025-01-15T10:30:00Z"
}
```

## Getting Started

### Option 1: Use the Hosted Service

The easiest way to get started is to use the hosted service at `ddns.rockygray.com`.

**Set up automatic IP registration with cron:**

```bash
# Add to crontab (runs every 15 minutes)
*/15 * * * * curl -s -X POST https://ddns.rockygray.com/register \
  -H "Content-Type: application/json" \
  -d '{"ownerId":"my-home","location":"home","ip":"auto"}' > /dev/null
```

**Or create a simple shell script:**

```bash
#!/bin/bash
# save as: update-ddns.sh

OWNER_ID="my-home"
LOCATION="home"

curl -s -X POST https://ddns.rockygray.com/register \
  -H "Content-Type: application/json" \
  -d "{\"ownerId\":\"${OWNER_ID}\",\"location\":\"${LOCATION}\",\"ip\":\"auto\"}"
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

**Use pubip in a daemon script:**

```bash
#!/bin/bash
# save as: ddns-daemon.sh

OWNER_ID="my-home"
LOCATION="home"
INTERVAL=900  # 15 minutes

while true; do
    IP=$(./bin/pubip)
    if [ -n "$IP" ]; then
        curl -s -X POST https://ddns.rockygray.com/register \
          -H "Content-Type: application/json" \
          -d "{\"ownerId\":\"${OWNER_ID}\",\"location\":\"${LOCATION}\",\"ip\":\"${IP}\"}"
        echo "Registered IP: $IP"
    fi
    sleep $INTERVAL
done
```

## Use Cases

### Home Lab Access

Keep track of your home lab's public IP so you can SSH in from anywhere:

```bash
# On your home server (via cron)
*/15 * * * * curl -s -X POST https://ddns.rockygray.com/register \
  -d '{"ownerId":"homelab","location":"primary","ip":"auto"}' > /dev/null

# From your laptop, anywhere in the world
HOME_IP=$(curl -s https://ddns.rockygray.com/lookup/homelab/primary | jq -r '.ip')
ssh user@$HOME_IP
```

### Multi-Site Monitoring

Track IP addresses across multiple locations:

```bash
# Register each location
curl -X POST https://ddns.rockygray.com/register \
  -d '{"ownerId":"acme-corp","location":"headquarters","ip":"auto"}'

curl -X POST https://ddns.rockygray.com/register \
  -d '{"ownerId":"acme-corp","location":"warehouse","ip":"auto"}'

# Look up any location
curl https://ddns.rockygray.com/lookup/acme-corp/headquarters
curl https://ddns.rockygray.com/lookup/acme-corp/warehouse
```

### VPN Endpoint Discovery

Automatically update your VPN client configuration when your home IP changes:

```bash
#!/bin/bash
NEW_IP=$(curl -s https://ddns.rockygray.com/lookup/my-vpn/home | jq -r '.ip')
sed -i "s/remote .* 1194/remote $NEW_IP 1194/" /etc/openvpn/client.conf
```

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

## License

MIT License - See [LICENSE](LICENSE) for details.
