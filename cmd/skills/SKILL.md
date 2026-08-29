---
name: nubulus
description: >
  Manages Nubulus Cloud DNS zones, DNS records (RRsets), WireGuard tunnels, and tunnel routes, and exposes local development ports to internet (ngrok style) using the nubulus CLI. Trigger when the user wants to expose a local service/port to internet, create or manage DNS records, verify DNS domains, set up WireGuard tunnels, configure public routes, or manage Nubulus Cloud resources.
user-invocable: true
disable-model-invocation: false
allowed-tools: Bash, Read, Write, Edit, Glob, Grep, Agent, AskUserQuestion
---

# Nubulus Cloud CLI (`nubulus`)

The `nubulus` CLI manages DNS zones, RRsets, WireGuard tunnels, and public routes on Nubulus Cloud, and provides an automatic port-forwarding tool (`nubulus expose`) to expose local servers like ngrok.

## Quick Reference / Intents

| Goal | Command |
| :--- | :--- |
| **Expose local port (ngrok mode)** | `nubulus expose <port>` (e.g. `nubulus expose 3000`) |
| **Expose with custom subdomain** | `nubulus expose 3000 --subdomain <subdomain>` |
| **Expose in background (permanent)** | `nubulus expose 3000 --subdomain <name> --detach` |
| **List DNS zones** | `nubulus zones list` |
| **Get zone details** | `nubulus zones get <zone>` |
| **Create DNS zone** | `nubulus zones create <zone>` |
| **Verify DNS challenge** | `nubulus zones verify <zone>` / `nubulus zones verification <zone>` |
| **List DNS records** | `nubulus records list <zone>` |
| **Create/Update DNS record** | `nubulus records set <zone> <name> <type> <values...> --ttl <ttl>` |
| **Delete DNS record** | `nubulus records delete <zone> <name> <type> --yes` |
| **List WireGuard tunnels** | `nubulus tunnels list` |
| **Get tunnel & routes** | `nubulus tunnels get <tunnel-id>` |
| **Create tunnel & export wg0.conf** | `nubulus tunnels create --name <name> --save-wg wg0.conf` |
| **List routes on tunnel** | `nubulus routes list <tunnel-id>` |
| **Create route on tunnel** | `nubulus routes create <tunnel-id> --hostname <fqdn> --upstream-host <ip> --upstream-port <port>` |
| **Show active config** | `nubulus config show` |
| **Initialize config** | `nubulus config init` |

---

## 1. Exposing Local Ports (`nubulus expose`)

Use `nubulus expose` to make any local web server publicly accessible via Nubulus WireGuard tunnel and DNS.

### Standard Foreground Mode (Auto-cleanup on exit)
```bash
# Random generated subdomain (e.g. swift-fox-42.tun.aleix.cloud):
nubulus expose 3000

# Custom subdomain (e.g. myapp.tun.aleix.cloud):
nubulus expose 3000 --subdomain myapp

# Custom zone & tunnel:
nubulus expose 3000 -s test --zone tun.aleix.cloud --tunnel cb781a20-708e-429a-9e2b-cf54e1e81d9d
```

**How it works:**
1. Creates `CNAME` record `<subdomain>.<zone>` -> tunnel CNAME target in Nubulus DNS.
2. Creates host route on the WireGuard tunnel -> `http://127.0.0.1:<port>`.
3. Displays a live status dashboard with public HTTPS URL.
4. On process interrupt (Ctrl+C / SIGINT), cleanly deletes the tunnel route and the DNS CNAME record.

### Permanent / Detached Mode
```bash
nubulus expose 3000 --subdomain myapp --detach
```
Creates the DNS CNAME and tunnel route permanently without waiting for Ctrl+C.

---

## 2. Managing DNS Zones (`nubulus zones`)

```bash
# List zones
nubulus zones list

# Get zone details (serial SOA, nameservers, verification state)
nubulus zones get example.com

# Create / claim zone
nubulus zones create example.com

# Check TXT verification instructions
nubulus zones verification example.com

# Trigger domain verification check
nubulus zones verify example.com

# Delete zone
nubulus zones delete example.com --yes
```

---

## 3. Managing DNS Records / RRsets (`nubulus records`)

The unit of work is an **RRset** (all values sharing the same name and record type).

```bash
# List all records in a zone
nubulus records list example.com

# Get a specific record set
nubulus records get example.com www A

# Set / replace a record (A, CNAME, TXT, MX, etc.)
nubulus records set example.com www A 198.51.100.10 --ttl 300
nubulus records set example.com api A 198.51.100.10 198.51.100.11 --ttl 300
nubulus records set example.com app CNAME tun-xyz.nubulustun.com. --ttl 60
nubulus records set example.com @ MX "10 mail.example.com."

# Delete a record set
nubulus records delete example.com www A --yes
```

---

## 4. Managing WireGuard Tunnels (`nubulus tunnels`)

```bash
# List tunnels
nubulus tunnels list

# Inspect tunnel and its routes
nubulus tunnels get <tunnel-id>

# Create tunnel and export wg0.conf
nubulus tunnels create --name "my-tunnel" --save-wg wg0.conf

# Rotate tunnel credentials
nubulus tunnels rotate-token <tunnel-id>

# Delete tunnel
nubulus tunnels delete <tunnel-id> --yes
```

---

## 5. Managing Tunnel Routes (`nubulus routes`)

```bash
# List routes on a tunnel
nubulus routes list <tunnel-id>

# Get route details
nubulus routes get <tunnel-id> <route-id>

# Create host route
nubulus routes create <tunnel-id> \
  --hostname app.example.com \
  --upstream-host 127.0.0.1 \
  --upstream-port 8080 \
  --upstream-scheme http

# Create path-based route
nubulus routes create <tunnel-id> \
  --type path \
  --hostname api.example.com \
  --path-prefix /v1 \
  --upstream-host 10.0.0.5 \
  --upstream-port 3000 \
  --strip-prefix

# Update route
nubulus routes update <tunnel-id> <route-id> --upstream-port 9090 --enabled false

# Delete route
nubulus routes delete <tunnel-id> <route-id> --yes
```

---

## 6. Output Formats

All commands accept `-o` / `--output`:
- `table` (default)
- `json` (for scripting and JSON processing)
- `yaml` (for YAML manifests)

Example:
```bash
nubulus zones list -o json
```
