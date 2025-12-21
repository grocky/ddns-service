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
- **Smart Client** - The `ddns-client` detects IP changes locally and updates the server
- **SSL Certificates** - Built-in certbot integration for Let's Encrypt wildcard certificates
- **Secure by default** - API key authentication protects your data
- **Rate limited** - Prevents abuse with a maximum of 2 IP changes per hour
- **Multi-location support** - Track IPs for multiple locations (home, office, cabin, etc.)
- **Serverless architecture** - Scales automatically, pay only for what you use
- **Privacy-first** - Self-hostable, your data stays yours

## Architecture

```mermaid
flowchart TB
    subgraph clients["Your Devices"]
        daemon["ddns-client\n(daemon mode)"]
        cron["ddns-client --cron\n(crontab)"]
        ha["Home Assistant\n(coming soon)"]
    end

    subgraph aws["AWS Cloud"]
        apigw["API Gateway\n(ddns.grocky.net)"]
        lambda["Lambda Function\n(Go)"]
        dynamodb[("DynamoDB\n(Owners + IP Mappings)")]
        route53["Route53\n(DNS Records)"]
        ses["SES\n(Email)"]
    end

    subgraph endpoints["API Endpoints"]
        direction LR
        e2["POST /owners"]
        e3["POST /update"]
        e4["GET /lookup/{owner}/{location}"]
    end

    daemon -->|"IP changed?\nCall /update"| apigw
    cron -->|"IP changed?\nCall /update"| apigw
    ha -->|"IP changed?\nCall /update"| apigw

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
2. **Run `ddns-client`** - The client detects your IP and calls the API when it changes
3. **Automatic DNS updates** - When your IP changes, Route53 is updated automatically
4. **Connect using your subdomain** - Use your stable hostname (e.g., `a3f8c2d1.grocky.net`)

Each owner/location combination gets a deterministic subdomain based on a hash of `{ownerId}-{location}`. This subdomain never changes, even when your IP does.

### Client-Server Interaction

The `ddns-client` uses a smart update model that minimizes API calls by detecting IP changes locally:

```mermaid
sequenceDiagram
    participant Client as ddns-client
    participant IPServices as IP Detection Services
    participant API as DDNS API
    participant Route53 as Route53 DNS

    loop Every 15 minutes (daemon) or on-demand (cron)
        Client->>IPServices: Query multiple IP services
        IPServices-->>Client: Consensus IP (e.g., 203.0.113.42)

        alt IP unchanged from last check
            Note over Client: Skip API call<br/>(saves bandwidth)
        else IP changed or first run
            Client->>API: POST /update<br/>{ownerId, location, ip}
            API->>API: Validate API key
            API->>API: Check rate limit (2/hour)

            alt Rate limit OK
                API->>Route53: Upsert A record
                Route53-->>API: Success
                API-->>Client: {subdomain, ip, changed: true}
                Note over Client: Save IP hash to state
            else Rate limit exceeded
                API-->>Client: 429 Too Many Requests<br/>Retry-After: N seconds
            end
        end
    end
```

**Key benefits of this model:**
- **Reduced API calls** - Only calls the API when IP actually changes
- **Reliable IP detection** - Uses consensus from multiple services (icanhazip, ipify, etc.)
- **Offline resilience** - State persisted to disk in cron mode survives restarts
- **Rate limit friendly** - Local change detection prevents hitting rate limits

## API Reference

### Authentication

Most endpoints require authentication via API key. Include your API key in the `Authorization` header:

```
Authorization: Bearer ddns_sk_your_api_key_here
```

| Endpoint | Authentication | Rate Limited |
|----------|----------------|--------------|
| `POST /owners` | Not required | No |
| `POST /owners/{id}/recover` | Not required | No |
| `POST /owners/{id}/rotate` | Required | No |
| `POST /update` | Required | Yes (2/hour) |
| `GET /lookup/{owner}/{location}` | Required | No |
| `POST /acme-challenge` | Required | Yes (10/hour) |
| `DELETE /acme-challenge` | Required | No |

### Create an Owner Account

Create an owner account to receive your API key. **Save your API key securely - it is only shown once!**

```bash
curl -X POST https://ddns.grocky.net/owners \
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

### Update DNS

Update your DNS record when your IP changes. The client can optionally send its detected IP, or the server will detect it from the request.

**Rate limited to 2 IP changes per hour.**

```bash
# Client sends detected IP (recommended - used by ddns-client)
curl -X POST https://ddns.grocky.net/update \
  -H "Authorization: Bearer ddns_sk_your_api_key_here" \
  -H "Content-Type: application/json" \
  -d '{
    "ownerId": "my-home-lab",
    "location": "home",
    "ip": "203.0.113.42"
  }'

# Or let server detect IP (backward compatible)
curl -X POST https://ddns.grocky.net/update \
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
curl https://ddns.grocky.net/lookup/my-home-lab/home \
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
curl -X POST https://ddns.grocky.net/owners/my-home-lab/rotate \
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
curl -X POST https://ddns.grocky.net/owners/my-home-lab/recover \
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

The easiest way to get started is to use the hosted service at `ddns.grocky.net`.

**Step 1: Create your owner account**

```bash
curl -X POST https://ddns.grocky.net/owners \
  -H "Content-Type: application/json" \
  -d '{"ownerId":"my-home","email":"you@example.com"}'
```

Save the `apiKey` from the response - you'll need it for all future requests!

**Step 2: Install and run the ddns-client**

```bash
# Build the client
make build-client

# Set environment variables
export DDNS_API_KEY=ddns_sk_your_api_key_here
export DDNS_OWNER=my-home
export DDNS_LOCATION=home

# Run as daemon (default - checks every 15 minutes)
./bin/ddns-client

# Or run once for crontab
./bin/ddns-client --cron
```

**Or use crontab with the --cron flag:**

```bash
# Add to crontab (runs every 15 minutes)
*/15 * * * * DDNS_API_KEY=xxx DDNS_OWNER=my-home DDNS_LOCATION=home /usr/local/bin/ddns-client --cron
```

**Step 3: Connect using your subdomain**

Once set up, you can always connect to your home network using your stable subdomain:

```bash
ssh user@6abf7de6.grocky.net
```

### Option 2: Use the ddns-client CLI

The `ddns-client` is a full-featured DDNS client that:
- Queries multiple IP detection services with consensus for reliability
- Detects IP changes locally (only calls API when IP changes)
- Supports both daemon mode (continuous) and cron mode (one-shot)
- Works with both IPv4 and IPv6

```bash
# Build the client
make build-client

# Daemon mode (runs continuously, checks every 15 minutes)
export DDNS_API_KEY=ddns_sk_your_api_key_here
export DDNS_OWNER=myuser
export DDNS_LOCATION=home
./bin/ddns-client

# Cron mode (runs once, for crontab)
./bin/ddns-client --cron

# IPv6 mode
./bin/ddns-client -6

# Custom check interval
./bin/ddns-client --interval 5m

# Verbose logging
./bin/ddns-client --verbose
```

## Use Cases

### Home Lab Access

Keep track of your home lab's public IP and always connect via a stable hostname:

```bash
# First, create your owner account (one-time setup)
curl -X POST https://ddns.grocky.net/owners \
  -H "Content-Type: application/json" \
  -d '{"ownerId":"homelab","email":"you@example.com"}'
# Save the API key!

# On your home server - run ddns-client as a daemon
export DDNS_API_KEY=ddns_sk_your_api_key
export DDNS_OWNER=homelab
export DDNS_LOCATION=primary
./bin/ddns-client

# Or add to crontab for one-shot updates
*/15 * * * * DDNS_API_KEY=xxx DDNS_OWNER=homelab DDNS_LOCATION=primary /usr/local/bin/ddns-client --cron

# From your laptop, anywhere in the world - use your stable subdomain
ssh user@a1b2c3d4.grocky.net
```

### Multi-Site Monitoring

Track IP addresses across multiple locations:

```bash
# Create owner account first
curl -X POST https://ddns.grocky.net/owners \
  -H "Content-Type: application/json" \
  -d '{"ownerId":"acme-corp","email":"admin@acme.com"}'

# At headquarters - run ddns-client with location=headquarters
DDNS_API_KEY=xxx DDNS_OWNER=acme-corp DDNS_LOCATION=headquarters ./bin/ddns-client

# At warehouse - run ddns-client with location=warehouse
DDNS_API_KEY=xxx DDNS_OWNER=acme-corp DDNS_LOCATION=warehouse ./bin/ddns-client

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

## SSL Certificates with Let's Encrypt

DDNS Service supports automatic SSL certificate generation using Let's Encrypt's DNS-01 challenge. This allows you to get wildcard certificates for your dynamic DNS subdomain.

### How It Works

1. The `ddns-client` integrates with certbot as an authentication hook
2. When certbot needs to validate domain ownership, it creates a TXT record via the DDNS API
3. Let's Encrypt validates the TXT record and issues your certificate
4. The cleanup hook removes the TXT record automatically

### Getting SSL Certificates

**Prerequisites:**
- An existing DDNS location (run `ddns-client` at least once to register your IP)
- certbot installed on your system

**Step 1: Set up environment variables**

```bash
export DDNS_API_KEY=ddns_sk_your_api_key_here
export DDNS_OWNER=my-home
export DDNS_LOCATION=home
```

**Step 2: Request a certificate with certbot**

```bash
# Get a wildcard certificate for your subdomain
certbot certonly --manual --preferred-challenges dns \
  --manual-auth-hook "ddns-client --acme-auth" \
  --manual-cleanup-hook "ddns-client --acme-cleanup" \
  -d "*.a1b2c3d4.grocky.net" \
  -d "a1b2c3d4.grocky.net"
```

The `ddns-client` automatically:
- Creates the `_acme-challenge` TXT record with the validation token
- Waits 60 seconds for DNS propagation (configurable with `--propagation-wait`)
- Cleans up the TXT record after validation

**Step 3: Set up automatic renewal**

Add to your crontab or systemd timer:

```bash
# Renew certificates daily (certbot only renews when needed)
0 3 * * * certbot renew --manual-auth-hook "ddns-client --acme-auth" --manual-cleanup-hook "ddns-client --acme-cleanup"
```

### ACME API Endpoints

| Endpoint | Method | Authentication | Description |
|----------|--------|----------------|-------------|
| `/acme-challenge` | POST | Required | Create TXT record for DNS-01 challenge |
| `/acme-challenge` | DELETE | Required | Delete TXT record after validation |

**Create ACME Challenge:**

```bash
curl -X POST https://ddns.grocky.net/acme-challenge \
  -H "Authorization: Bearer ddns_sk_your_api_key_here" \
  -H "Content-Type: application/json" \
  -d '{
    "ownerId": "my-home",
    "location": "home",
    "txtValue": "abc123validationtoken"
  }'
```

**Response:**
```json
{
  "ownerId": "my-home",
  "location": "home",
  "subdomain": "a1b2c3d4.grocky.net",
  "txtRecord": "_acme-challenge.a1b2c3d4.grocky.net",
  "txtValue": "abc123validationtoken",
  "createdAt": "2025-01-15T10:30:00Z",
  "expiresAt": "2025-01-15T11:30:00Z"
}
```

**Delete ACME Challenge:**

```bash
curl -X DELETE https://ddns.grocky.net/acme-challenge \
  -H "Authorization: Bearer ddns_sk_your_api_key_here" \
  -H "Content-Type: application/json" \
  -d '{
    "ownerId": "my-home",
    "location": "home"
  }'
```

**Response:**
```json
{
  "ownerId": "my-home",
  "location": "home",
  "subdomain": "a1b2c3d4.grocky.net",
  "txtRecord": "_acme-challenge.a1b2c3d4.grocky.net",
  "deleted": true
}
```

### Notes

- ACME challenges expire after 1 hour (auto-deleted by DynamoDB TTL)
- Rate limited to 10 challenges per hour per owner
- Requires an existing IP mapping for the location (run `ddns-client` first to register)
- A daily cleanup job removes any orphaned DNS records

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
