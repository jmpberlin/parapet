# Dev Environment: Isolated Gateway Architecture

This document is the complete runbook for standing up the isolated dev environment from scratch: a dev box with **no public internet access at all**, reachable only through a trusted gateway, with all outbound traffic filtered by domain allowlist.

If either box is ever rebuilt, follow this top to bottom.

## Architecture summary

**Threat model:** the dev box runs AI-written code, potentially unreviewed. Assume worst case — full host takeover, credential scraping, attempted exfiltration.

**Two Hetzner Cloud VMs, one private network:**
- **Dev box** (untrusted) — no public IP at all. Only egress is via `HTTP_PROXY`/`HTTPS_PROXY` pointed at the gateway. Runs Docker + the application.
- **Gateway** (trusted) — keeps its public IP (needs it for Tailscale + Squid's own outbound). Runs Tailscale (as a subnet router), Squid (domain allowlist), nftables (one-way containment + NAT).

**Non-goal:** preventing exfiltration entirely. A domain allowlist that permits GitHub or an LLM API is also an exfiltration channel. This design reduces blast radius; it does not eliminate it.

**Traffic paths:**
| Path | Mechanism |
|---|---|
| Runner/laptop → dev box | Tailscale → gateway's advertised subnet route → SSH to private IP |
| Dev box → external APIs | Private network → Squid on gateway → allowlisted domains only |
| Dev box → tailnet | Explicitly dropped by nftables |

---

## Phase 0 — Provision the boxes

1. Create a Hetzner **private network** (e.g. `10.0.0.0/16`).
2. Create two Hetzner Cloud VMs (Ubuntu), both attached to that network. Note their assigned private IPs (this doc uses `10.0.0.2` = gateway, `10.0.0.4` = dev box as examples — yours will differ).
3. Both boxes keep their public IPs **for now** — needed for initial setup. The dev box's public IP is removed at the very end (Phase 10).
4. Verify private connectivity both directions before doing anything else:
   ```bash
   # from gateway
   ping -c 3 <dev-box-private-ip>
   # from dev box
   ping -c 3 <gateway-private-ip>
   ```

## Phase 1 — Gateway → dev-box SSH key (`dev_deploy`)

This is the one-directional trust key: gateway may SSH into the dev box, never the reverse.

On the **gateway**:
```bash
ssh-keygen -t ed25519 -C "gateway-to-dev-deploy" -f /root/.ssh/dev_deploy -N ""
cat /root/.ssh/dev_deploy.pub
```

On the **dev box**:
```bash
mkdir -p /root/.ssh && chmod 700 /root/.ssh
echo "<paste-pubkey>" >> /root/.ssh/authorized_keys
chmod 600 /root/.ssh/authorized_keys
```

Verify from the gateway:
```bash
ssh -i /root/.ssh/dev_deploy root@<dev-box-private-ip> hostname
```

**Back up the private key** (`/root/.ssh/dev_deploy`, no `.pub`) somewhere off-box — password manager or similar. If the gateway is ever rebuilt, this is the only thing standing between you and the dev box once its public IP is gone. The public key is not sensitive; also worth saving as a Hetzner Cloud "SSH Key" so it can be injected automatically if the dev box is ever recreated.

## Phase 2 — Gateway joins Tailscale as a subnet router

On the **gateway**, enable IP forwarding:
```bash
echo 'net.ipv4.ip_forward = 1' | tee -a /etc/sysctl.d/99-tailscale.conf
sysctl -p /etc/sysctl.d/99-tailscale.conf
```

Install and bring up Tailscale, advertising just the dev box's address (not the whole subnet — narrower is better when there's only one thing to route to):
```bash
curl -fsSL https://tailscale.com/install.sh | sh
tailscale up --advertise-routes=<dev-box-private-ip>/32 --ssh --advertise-tags=tag:gateway
```

In the **Tailscale admin console**:
- **Machines** → approve the advertised route (routes start disabled even for the admin).
- Confirm the machine shows tag `tag:gateway`, not untagged.

## Phase 3 — Tailscale ACL policy

```jsonc
{
	"tagOwners": {
		"tag:ci":      ["autogroup:admin"],
		"tag:gateway": ["autogroup:admin"],
	},
	"grants": [
		{ "src": ["autogroup:member"], "dst": ["*"], "ip": ["*"] },
		{ "src": ["tag:ci"], "dst": ["tag:gateway"], "ip": ["tcp:22"] },
		{ "src": ["tag:ci"], "dst": ["<dev-box-private-ip>/32"], "ip": ["tcp:22"] },
	],
	"ssh": [
		{ "action": "check",  "src": ["autogroup:member"], "dst": ["tag:gateway"], "users": ["root"] },
		{ "action": "accept", "src": ["tag:ci"],           "dst": ["tag:gateway"], "users": ["root"] },
	],
}
```

Note the dev box never appears as a tailnet identity anywhere — it has no Tailscale installed, so ACLs can't directly govern it. The `10.0.0.4/32` grant works because it's reachable *through* the gateway's advertised route.

## Phase 4 — One-way containment (nftables)

Find the private interface name first: `ip a` on the gateway (commonly `enp7s0` on Hetzner images).

```bash
apt install -y nftables

cat > /etc/nftables.conf <<'EOF'
#!/usr/sbin/nft -f
flush ruleset

table inet filter {
    chain forward {
        type filter hook forward priority 0; policy drop;
        ct state established,related accept
        iifname "tailscale0" oifname "<PRIVATE_IFACE>" ip daddr <dev-box-private-ip> accept
    }
}
EOF

systemctl enable --now nftables
nft -f /etc/nftables.conf
```

This allows tailnet → dev-box traffic and *only* that — return traffic for established connections is allowed both ways, but the dev box can never initiate a new connection into the tailnet. Verify by attempting to reach a tailnet peer from the dev box; it should fail.

## Phase 5 — NAT masquerade (required, not optional)

**Why:** Hetzner's private network gateway enforces strict unicast reverse path forwarding (uRPF) — it checks a forwarded packet's source IP against addresses actually registered to the sending server. Without masquerading, packets arriving from a tailnet peer (source `100.x.x.x`) get silently dropped by Hetzner's own fabric when the gateway tries to forward them onto the private network — invisible to `tcpdump` on either VM, since the drop happens on Hetzner's infrastructure, not inside either box.

Add to the same nftables config:
```bash
cat >> /etc/nftables.conf <<'EOF'

table ip nat {
    chain postrouting {
        type nat hook postrouting priority 100;
        iifname "tailscale0" oifname "<PRIVATE_IFACE>" masquerade
    }
}
EOF

nft -f /etc/nftables.conf
```

This rewrites the source IP to the gateway's own private IP before the packet crosses onto the private network — now a completely normal, already-registered address, uRPF passes trivially. NAT + conntrack makes this automatically bidirectional; no separate return-path rule is needed.

*(A Hetzner-level "Route" pointing the CGNAT range at the gateway was tried first and rejected by Hetzner's console with "invalid IP range" — likely because `100.64.0.0/10` is RFC 6598 shared address space. Masquerading avoids the problem entirely and is the simpler fix.)*

## Phase 6 — Squid (the domain allowlist)

```bash
apt install -y squid
```

`/etc/squid/squid.conf`:
```
http_port <gateway-private-ip>:3128

acl allowed_src src <dev-box-private-ip>/32

acl allowed_dst dstdomain \
    api.github.com \
    github.com \
    .githubusercontent.com \
    ghcr.io \
    api.anthropic.com \
    .bleepingcomputer.com \
    socket.dev \
    .docker.io \
    .docker.com

acl SSL_ports port 443
acl CONNECT method CONNECT

http_access deny CONNECT !SSL_ports
http_access allow allowed_src allowed_dst
http_access deny all

access_log /var/log/squid/access.log squid
```

```bash
squid -k parse
systemctl restart squid
ss -tlnp | grep 3128   # must show the private IP, not 0.0.0.0
```

**Domain-matching gotcha, hit repeatedly during setup:** `dstdomain` without a leading dot matches *only* that exact hostname. `bleepingcomputer.com` does not match `www.bleepingcomputer.com`; `ghcr.io` does not match its separate blob-storage host. Use a leading dot (`.example.com`) to match a domain and all its subdomains. When adding a new allowlist entry, check what host the *actual request* hits (redirect chains, CDN blob hosts, `www.` prefixes) rather than just the domain you'd type in a browser.

**Ongoing changes to this allowlist go through the `parapet-dev-proxy` repo** (see Phase 9) — never hand-edit `/etc/squid/squid.conf` directly as the permanent fix; live-edit only to unblock urgently, then always follow up with a tracked commit.

## Phase 7 — Dev box: Docker + proxy wiring

Install Docker (standard upstream repo, not distro default):
```bash
apt update && apt install -y ca-certificates curl gnupg
install -m 0755 -d /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg -o /etc/apt/keyrings/docker.asc
chmod a+r /etc/apt/keyrings/docker.asc
echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/ubuntu $(. /etc/os-release && echo "$VERSION_CODENAME") stable" | tee /etc/apt/sources.list.d/docker.list
apt update
apt install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
```

**Daemon-level proxy** (needed for `docker pull`/`docker compose pull` to reach GHCR/Docker Hub through Squid):
```bash
mkdir -p /etc/systemd/system/docker.service.d
cat > /etc/systemd/system/docker.service.d/http-proxy.conf <<EOF
[Service]
Environment="HTTP_PROXY=http://<gateway-private-ip>:3128"
Environment="HTTPS_PROXY=http://<gateway-private-ip>:3128"
Environment="NO_PROXY=localhost,127.0.0.1"
EOF
systemctl daemon-reload
systemctl restart docker
```

**Note:** this only configures the daemon's own outbound (pulls/builds). It does **not** propagate into running containers — application containers need their own `HTTP_PROXY`/`HTTPS_PROXY` set explicitly in `docker-compose.yml` (see Phase 11).

## Phase 8 — `ci_runner` user (GitHub Actions → dev box)

Dedicated, least-privilege user — not root — for the deploy pipeline.

On the **dev box**:
```bash
useradd -m -s /bin/bash ci_runner
usermod -aG docker ci_runner
mkdir -p /home/ci_runner/.ssh && chmod 700 /home/ci_runner/.ssh
```

On your **local machine**, generate a dedicated keypair (never on a server):
```bash
ssh-keygen -t ed25519 -C "github-actions-ci_runner" -f ~/.ssh/parapet_ci_runner -N ""
cat ~/.ssh/parapet_ci_runner.pub
```

Install the pubkey:
```bash
echo "<pubkey>" >> /home/ci_runner/.ssh/authorized_keys
chmod 600 /home/ci_runner/.ssh/authorized_keys
chown -R ci_runner:ci_runner /home/ci_runner/.ssh
```

Store the **private** key as a GitHub Actions secret: `CI_RUNNER_SSH_KEY` (repo: `parapet`).

## Phase 9 — Git access on the dev box (HTTPS + PAT, through the proxy)

SSH doesn't natively tunnel through an HTTP CONNECT proxy, so git uses HTTPS here, not the SSH deploy-key pattern.

1. Create a **fine-grained PAT**: repository access limited to `parapet`, permission `Contents: Read-only`.
2. As **`ci_runner`** specifically (git config is per-user, and file ownership matters for git's safe-directory check):
   ```bash
   sudo -u ci_runner git config --global http.proxy http://<gateway-private-ip>:3128
   sudo -u ci_runner git config --global credential.helper store
   echo "https://<github-user>:<PAT>@github.com" | sudo tee /home/ci_runner/.git-credentials
   sudo chown ci_runner:ci_runner /home/ci_runner/.git-credentials
   sudo chmod 600 /home/ci_runner/.git-credentials
   ```
3. Clone as `ci_runner`, or clone as root and fix ownership + git's safe-directory check afterward:
   ```bash
   git clone https://github.com/<org>/parapet.git /opt/parapet-dev
   chown -R ci_runner:ci_runner /opt/parapet-dev
   sudo -u ci_runner git config --global --add safe.directory /opt/parapet-dev
   ```

**Squid allowlist note:** plain `git fetch` over HTTPS hits the bare `github.com` domain — `api.github.com` alone is not sufficient (hit this exact failure during setup).

## Phase 10 — GHCR pull access on the dev box

Classic PAT scoped `read:packages` only:
```bash
docker login ghcr.io -u <github-user> --password-stdin <<< "<PAT>"
```

## Phase 11 — `.env` and the application's own proxy wiring

Copy `.env.example` → `.env` on the dev box and fill in real values (DB credentials, API keys, etc.) — **or**, preferably, let CI render this automatically every deploy (see "Remaining automation" below); this section describes the manual version for a fresh box's first run.

**In `docker-compose.yml`, the backend service needs its own explicit proxy vars** — the Docker daemon's proxy config (Phase 7) does not propagate into containers automatically:
```yaml
  backend:
    environment:
      # ...existing vars...
      - HTTP_PROXY=http://${GATEWAY_PROXY_PRIVATE_IP}:${GATEWAY_PROXY_PORT}
      - HTTPS_PROXY=http://${GATEWAY_PROXY_PRIVATE_IP}:${GATEWAY_PROXY_PORT}
      - NO_PROXY=postgres,localhost,127.0.0.1
```

**Frontend, Adminer, and Grafana all need dual port bindings** — `127.0.0.1` alone (leftover from the pre-gateway architecture, where `tailscale serve` ran locally on the box) is unreachable from anywhere else, including the private network:
```yaml
  frontend:
    ports:
      - "127.0.0.1:${FRONTEND_HOST_PORT:-8090}:80"
      - "${DEV_SERVER_PRIVATE_IP}:${FRONTEND_HOST_PORT:-8090}:80"

  adminer:
    ports:
      - "127.0.0.1:8081:8080"
      - "${DEV_SERVER_PRIVATE_IP}:8081:8080"

  grafana:
    ports:
      - "127.0.0.1:3000:3000"
      - "${DEV_SERVER_PRIVATE_IP}:3000:3000"
```

`.env` additions needed: `GATEWAY_PROXY_PRIVATE_IP`, `GATEWAY_PROXY_PORT`, `DEV_SERVER_PRIVATE_IP`.

**Gotcha:** editing the compose file or `.env` does not retroactively change an already-running container — environment and port bindings are fixed at container creation. After any such change:
```bash
export IMAGE_TAG=<current-sha>   # match whatever is currently deployed
docker compose up -d --force-recreate <service>
```

## Phase 12 — GitHub repo configuration

**`parapet` repo:**
- Secrets: `TS_OAUTH_CLIENT_ID`, `TS_OAUTH_SECRET` (Tailscale OAuth client, tag `tag:ci`), `CI_RUNNER_SSH_KEY`
- Variables: `DEV_SERVER_PRIVATE_IP`, `GATEWAY_PRIVATE_IP`

**`parapet-dev-proxy` repo** (tracks `squid.conf.template`, deliberately outside the AI pipeline's repo access — see "Security notes"):
- Secrets: same `TS_OAUTH_CLIENT_ID`/`TS_OAUTH_SECRET`
- Variables: `PROXY_TAILNET_HOST` (gateway's MagicDNS name, for deploying config changes), `GATEWAY_PRIVATE_IP`, `DEV_SERVER_PRIVATE_IP`

## Phase 13 — Hetzner Cloud Firewalls (final state)

- **Dev box:** no public IP, no interface at all — no firewall rules needed or possible. This is stronger than "deny all," since there's no interface to misconfigure.
- **Gateway:** inbound — deny all. Outbound — allow all (Tailscale + Squid both need unrestricted outbound; this machine is the trusted one by design).

## Phase 14 — Cutover

Only after everything above is confirmed working **while the dev box still has its public IP** (much easier to debug with a fallback path available):

1. Shut down the dev box.
2. Unassign its public IPv4 (and IPv6 if present) via the Hetzner console.
3. Power back on.
4. Verify access only via: `tailscale ssh root@<gateway-hostname>`, then `ssh -i /root/.ssh/dev_deploy root@<dev-box-private-ip>` from there.
5. Re-run every check — containers up, proxy env vars present in the running container, app reachable at `http://<dev-box-private-ip>:<port>` from a tailnet-connected laptop.

---

## Ongoing operations

**Changing the Squid allowlist:** edit `squid.conf.template` in `parapet-dev-proxy`, PR, merge — deploys automatically. Branch protection (required review) recommended, since this repo enforces a security boundary and is deliberately not reachable by the AI pipeline.

**Reaching the dev box day to day:**
```bash
tailscale ssh root@<gateway-hostname>
ssh -i /root/.ssh/dev_deploy root@<dev-box-private-ip>
```

**Reaching the app from your laptop:** ensure Tailscale is connected and the subnet route is approved, then browse directly to `http://<dev-box-private-ip>:<port>` — no tunnel needed, the approved route makes it a direct peer address.

## Remaining automation (not yet done)

- **`.env` generation from CI**, rendered from GitHub Secrets on every deploy (same pattern as `squid.conf.template`), so a rebuilt box never depends on anyone remembering values by hand. This is the highest-priority remaining gap.
- **Smoke test** in `parapet-dev-proxy`'s own pipeline needs redesigning for this topology — it must originate from the dev box (the only real caller of Squid) via a nested SSH hop through the gateway, not from a stale public-IP/tailnet-hostname assumption.
- Full Hetzner + Tailscale provisioning as IaC (Terraform/similar) — everything in Phases 0–10 is currently manual.

## Known accepted gaps

- Allowlisted domains are themselves an exfiltration channel (e.g. Docker Hub pulls). Containment reduces blast radius; it does not eliminate it.
- Any API key present on the dev box is lost on takeover, usable from anywhere afterward. Future mitigation: terminate credentials at the gateway (dev box calls the proxy with no key, gateway injects the real one before forwarding).
- The GitHub Actions runner itself holds tailnet + deploy access for the duration of each job — ephemeral, single-use keys via OAuth limit this window.