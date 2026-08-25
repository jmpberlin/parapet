# Dev Environment: Isolated Gateway Architecture

This document is the complete runbook for standing up the isolated dev environment from scratch: a dev box with **no public internet access at all**, reachable only through a trusted gateway, with all outbound traffic filtered by domain allowlist.

If either box is ever rebuilt, follow this top to bottom. Hetzner snapshots of both boxes also exist as a faster recovery path — see "Disaster recovery" at the end.

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

**Compare to prod:** prod deliberately does **not** use any of this isolation — no gateway, no Squid, no egress restriction. It shares the same `docker-compose.yml` and image-build pipeline shape, but runs on the open internet behind Caddy with real domains. See `DEPLOY.md`.

---

## Phase 0 — Provision the boxes

1. Create a Hetzner **private network** (e.g. `10.0.0.0/16`).
2. Create two Hetzner Cloud VMs (Ubuntu), both attached to that network. Note their assigned private IPs (this doc uses `10.0.0.2` = gateway, `10.0.0.4` = dev box as examples — yours will differ).
3. Both boxes keep their public IPs **for now** — needed for initial setup. The dev box's public IP is removed at the very end (Phase 15).
4. Verify private connectivity both directions before doing anything else:
   ```bash
   # from gateway
   ping -c 3 <dev-box-private-ip>
   # from dev box
   ping -c 3 <gateway-private-ip>
   ```

## Phase 1 — Gateway → dev-box SSH key (`dev_deploy`)

This is the one-directional trust key: gateway may SSH into the dev box, never the reverse. It is **not** a GitHub secret — it lives only on the gateway's disk and in your password manager.

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

**Back up the private key** (`/root/.ssh/dev_deploy`, no `.pub`) somewhere off-box. If the gateway is ever rebuilt, this is the only thing standing between you and the dev box once its public IP is gone. Also worth saving the public key as a Hetzner Cloud "SSH Key" for automatic injection if the dev box is ever recreated.

## Phase 2 — Gateway joins Tailscale as a subnet router

On the **gateway**, enable IP forwarding:
```bash
echo 'net.ipv4.ip_forward = 1' | tee -a /etc/sysctl.d/99-tailscale.conf
sysctl -p /etc/sysctl.d/99-tailscale.conf
```

Install and bring up Tailscale, advertising just the dev box's address:
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

The dev box never appears as a tailnet identity anywhere — it has no Tailscale installed. The `<dev-box-private-ip>/32` grant works because it's reachable *through* the gateway's advertised route.

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

Verify by attempting to reach a tailnet peer from the dev box (e.g. `ping` your own laptop's tailnet IP after forcing a route to it) — it should fail.

## Phase 5 — NAT masquerade (required, not optional)

**Why:** Hetzner's private network gateway enforces strict unicast reverse path forwarding (uRPF) — it checks a forwarded packet's source IP against addresses actually registered to the sending server. Without masquerading, packets arriving from a tailnet peer (source `100.x.x.x`) get silently dropped by Hetzner's own fabric when the gateway tries to forward them onto the private network — invisible to `tcpdump` on either VM, since the drop happens on Hetzner's infrastructure, not inside either box.

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

NAT + conntrack makes this automatically bidirectional; no separate return-path rule is needed.

*(A Hetzner-level "Route" pointing the CGNAT range at the gateway was tried first and rejected by Hetzner's console with "invalid IP range" — `100.64.0.0/10` is RFC 6598 shared address space. Masquerading avoids the problem entirely.)*

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

**Domain-matching gotcha, hit repeatedly during setup:** `dstdomain` without a leading dot matches *only* that exact hostname. `bleepingcomputer.com` does not match `www.bleepingcomputer.com`; `ghcr.io` does not match its separate blob-storage host; plain `git fetch` over HTTPS hits bare `github.com`, not just `api.github.com`. Use a leading dot (`.example.com`) to match a domain and all its subdomains. When adding a new entry, check what host the *actual request* hits rather than assuming.

**Ongoing changes to this allowlist go through the `parapet-dev-proxy` repo** (Phase 13) — never hand-edit `/etc/squid/squid.conf` as the permanent fix; live-edit only to unblock urgently, then always follow up with a tracked commit. This repo is deliberately kept outside the AI pipeline's own repo access, since the thing enforcing containment shouldn't be editable by the thing it's containing.

## Phase 7 — Dev box: Docker + daemon-level proxy

```bash
apt update && apt install -y ca-certificates curl gnupg
install -m 0755 -d /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg -o /etc/apt/keyrings/docker.asc
chmod a+r /etc/apt/keyrings/docker.asc
echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/ubuntu $(. /etc/os-release && echo "$VERSION_CODENAME") stable" | tee /etc/apt/sources.list.d/docker.list
apt update
apt install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
```

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

**Note:** this configures only the daemon's own outbound (pulls/builds). It does **not** propagate into running containers — the application container gets its own proxy vars via `docker-compose.yml` (Phase 11).

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
ssh-keygen -t ed25519 -C "github-actions-ci_runner-dev" -f ~/.ssh/parapet_ci_runner -N ""
cat ~/.ssh/parapet_ci_runner.pub
```

```bash
echo "<pubkey>" >> /home/ci_runner/.ssh/authorized_keys
chmod 600 /home/ci_runner/.ssh/authorized_keys
chown -R ci_runner:ci_runner /home/ci_runner/.ssh
```

Store the **private** key as the GitHub Actions secret `CI_RUNNER_SSH_KEY`, scoped to the **`dev` Environment** (not repo-level — prod has its own, different value under the same name; see Phase 13).

## Phase 9 — Git access on the dev box (HTTPS + PAT, through the proxy)

SSH doesn't natively tunnel through an HTTP CONNECT proxy, so git uses HTTPS here, not an SSH deploy-key pattern.

1. Create a **fine-grained PAT**: repository access limited to `parapet`, permission `Contents: Read-only`.
2. As **`ci_runner`** specifically (git config is per-user, and file ownership matters for git's safe-directory check):
   ```bash
   sudo -u ci_runner git config --global http.proxy http://<gateway-private-ip>:3128
   sudo -u ci_runner git config --global credential.helper store
   echo "https://<github-user>:<PAT>@github.com" | sudo tee /home/ci_runner/.git-credentials
   sudo chown ci_runner:ci_runner /home/ci_runner/.git-credentials
   sudo chmod 600 /home/ci_runner/.git-credentials
   ```
3. Clone as `ci_runner`, or clone as root and fix ownership + safe-directory afterward:
   ```bash
   git clone https://github.com/<org>/parapet.git /opt/parapet-dev
   chown -R ci_runner:ci_runner /opt/parapet-dev
   sudo -u ci_runner git config --global --add safe.directory /opt/parapet-dev
   ```

**Ownership gotcha, hit during setup:** if `/opt/parapet-dev` or `.env` inside it is ever created/edited as `root` (e.g. during manual debugging), subsequent `ci_runner`-driven `git` or file-write operations fail with a permissions error. After any manual root-level touch, re-run `chown -R ci_runner:ci_runner /opt/parapet-dev`.

## Phase 10 — GHCR authentication: fully automated, no manual step

Earlier versions of this doc had a manual `docker login ghcr.io` with a static `read:packages` PAT here. **This is now obsolete.** The deploy pipeline (Phase 12) logs into GHCR fresh on every run using GitHub's own ephemeral, auto-issued `secrets.GITHUB_TOKEN` — forwarded once over the SSH session, never stored on the box. Nothing to create, rotate, or revoke for this specifically. If a fresh box has never had this run yet, the very first pipeline deploy performs the login as part of that run — no separate bootstrap needed.

## Phase 11 — `docker-compose.yml`: proxy wiring and port bindings

This compose file is **shared between dev and prod** — differences are expressed entirely through which `.env` values are present, never a second compose file.

**Backend's proxy vars** — full URLs, not just an IP (the daemon-level proxy from Phase 7 does not propagate into containers automatically):
```yaml
  backend:
    environment:
      # ...existing vars...
      - HTTP_PROXY=${GATEWAY_HTTP_PROXY:-}
      - HTTPS_PROXY=${GATEWAY_HTTPS_PROXY:-}
      - NO_PROXY=postgres,localhost,127.0.0.1
```
On dev, `.env` sets `GATEWAY_HTTP_PROXY=http://<gateway-private-ip>:3128` (full URL, scheme included) — **do not** prepend an extra `http://` in the compose file itself, or the resulting value doubles up (`http://http://...`) and breaks. On prod, these are simply never set in `.env`, and the `:-` fallback correctly resolves to an empty string, meaning "no proxy" — not a broken one.

**Port bindings** — `127.0.0.1`-only (leftover from an earlier design where `tailscale serve` ran locally on the box) is unreachable from anywhere else, including the private network. All app-facing services bind to **both** loopback and the dev box's private IP:
```yaml
  backend:
    ports:
      - '${BACKEND_HOST:-127.0.0.1}:${BACKEND_HOST_PORT}:${BACKEND_PORT}'

  frontend:
    ports:
      - "127.0.0.1:${FRONTEND_HOST_PORT:-8090}:80"
      - "${SERVER_PRIVATE_IP:-127.0.0.1}:${FRONTEND_HOST_PORT:-8090}:80"

  adminer:
    ports:
      - "127.0.0.1:8081:8080"
      - "${SERVER_PRIVATE_IP:-127.0.0.1}:8081:8080"

  grafana:
    ports:
      - "127.0.0.1:3000:3000"
      - "${SERVER_PRIVATE_IP:-127.0.0.1}:3000:3000"
```
The `:-127.0.0.1` fallback on every line matters: if `SERVER_PRIVATE_IP` (or `BACKEND_HOST`) is ever unset — a deleted repo var, a typo — an empty host-IP in a Docker port binding can silently bind to *all* interfaces instead of failing loudly. The fallback makes the safe behavior (loopback-only) the default, not the accident.

Confirmed working: `http://<dev-box-private-ip>:8090` (app), `:8081` (Adminer, no auth layer — dev-only, reachable solely by tailnet members with the approved route), `:3000` (Grafana), all directly from a tailnet-connected laptop, no tunnel needed.

**Gotcha:** editing the compose file or `.env` does not retroactively change an already-running container — environment and port bindings are fixed at container creation.
```bash
export IMAGE_TAG=<current-sha>
docker compose up -d --force-recreate <service>
```

## Phase 12 — `.env`: fully automated, rendered by CI every deploy

Earlier versions of this doc described manually creating `.env` on the box. **This is now obsolete** — the deploy workflow writes a fresh `.env` on every run, from GitHub Secrets/Variables, so a rebuilt box never depends on anyone remembering values by hand. Relevant workflow step, in `parapet`'s `.github/workflows/deploy-dev.yml`:

```yaml
      - name: Write .env
        run: |
          ssh -o StrictHostKeyChecking=accept-new -i ~/.ssh/ci_runner_key \
            ci_runner@${{ vars.SERVER_PRIVATE_IP }} "cat > /opt/parapet-dev/.env" <<EOF
          POSTGRES_USER=${{ vars.POSTGRES_USER }}
          POSTGRES_PASSWORD=${{ secrets.POSTGRES_PASSWORD }}
          POSTGRES_DB=${{ vars.POSTGRES_DB }}
          POSTGRES_PORT=${{ vars.POSTGRES_PORT }}
          BACKEND_PORT=${{ vars.BACKEND_PORT }}
          BACKEND_HOST_PORT=${{ vars.BACKEND_HOST_PORT }}
          BACKEND_HOST=${{ vars.BACKEND_HOST }}
          FRONTEND_HOST_PORT=${{ vars.FRONTEND_HOST_PORT }}
          CLAUDE_API_KEY=${{ secrets.CLAUDE_API_KEY }}
          GITHUB_TOKEN=${{ secrets.PAT_GITHUB }}
          VITE_API_URL=${{ vars.VITE_API_URL }}
          GRAFANA_PASSWORD=${{ secrets.GRAFANA_PASSWORD }}
          GATEWAY_HTTP_PROXY=${{ vars.GATEWAY_HTTP_PROXY }}
          GATEWAY_HTTPS_PROXY=${{ vars.GATEWAY_HTTPS_PROXY }}
          SERVER_PRIVATE_IP=${{ vars.SERVER_PRIVATE_IP }}
          EOF
```

**`PAT_GITHUB` vs `GITHUB_TOKEN` naming, a real gotcha hit during setup:** GitHub Actions reserves the `GITHUB_` prefix for *secret names* only (you cannot create a repo/environment secret literally called `GITHUB_TOKEN` — that name is the auto-injected one). This restriction does **not** extend to `.env` variable names or application code — `PAT_GITHUB` is just the name chosen for *storing* this credential as a secret; it's still written into `.env` as plain `GITHUB_TOKEN`, matching what the application code actually reads. Renaming the app-facing variable to work around the reserved-prefix rule was an unnecessary detour taken once during setup — revert if you find `PAT_GITHUB` (or any renamed variant) referenced anywhere in application code.

**Heredoc gotcha:** the closing `EOF` must be indented to *exactly* match the block's established indentation level in the YAML source (not flush-left) — `run: |` strips a uniform amount of leading whitespace from every line, so the closing marker needs to sit at the same source-indentation as the content lines to land at true column-zero in the executed script. Trailing whitespace after `EOF` on that line also breaks the match.

## Phase 13 — GitHub configuration

GitHub Environments (`dev` and, separately for prod, `prod`) hold anything whose *name* needs to stay the same but *value* legitimately differs per environment. Everything else — identical in both, or only relevant to one — is a plain repo-level Secret/Variable.

**`parapet` repo — repo-level:**
- Secrets: `TS_OAUTH_CLIENT_ID`, `TS_OAUTH_SECRET` (Tailscale OAuth client, tag `tag:ci`; identical value needed by both environments, so left at repo level rather than duplicated)
- Variables: `BACKEND_HOST`, `BACKEND_HOST_PORT`, `BACKEND_PORT`, `FRONTEND_HOST_PORT`, `POSTGRES_DB`, `POSTGRES_USER`, `POSTGRES_PORT`, `VITE_API_URL`

**`parapet` repo — `dev` Environment:**
- Secrets: `CI_RUNNER_SSH_KEY`, `CLAUDE_API_KEY`, `GRAFANA_PASSWORD`, `PAT_GITHUB`, `POSTGRES_PASSWORD`
- Variables: `GATEWAY_HTTP_PROXY`, `GATEWAY_HTTPS_PROXY`, `SERVER_PRIVATE_IP`

(See `DEPLOY.md` for the parallel `prod` Environment — same secret/variable *names* where they overlap, different values, plus a few prod-only entries like `ADMINER_BEARER`.)

**`parapet-dev-proxy` repo** (tracks `squid.conf.template`, deliberately outside the AI pipeline's own repo access):
- Secrets: same `TS_OAUTH_CLIENT_ID`/`TS_OAUTH_SECRET` values, stored separately (different repo = separate copy)
- Variables: `PROXY_TAILNET_HOST` (gateway's MagicDNS name — the deploy target for pushing Squid config changes), `GATEWAY_PRIVATE_IP` (used to render `squid.conf`'s `http_port` binding, and as the smoke test's proxy target), `SERVER_PRIVATE_IP` (used to render `allowed_src`, and as the smoke test's SSH target)

## Phase 14 — Hetzner Cloud Firewalls (final state)

- **Dev box:** no public IP, no interface at all — no firewall rules needed or possible. Stronger than "deny all," since there's no interface to misconfigure.
- **Gateway:** inbound — deny all. Outbound — allow all (Tailscale + Squid both need unrestricted outbound; this machine is the trusted one by design).

## Phase 15 — Cutover

Only after everything above is confirmed working **while the dev box still has its public IP**:

1. Shut down the dev box.
2. Unassign its public IPv4 (and IPv6 if present) via the Hetzner console.
3. Power back on.
4. Verify access only via: `tailscale ssh root@<gateway-hostname>`, then `ssh -i /root/.ssh/dev_deploy root@<dev-box-private-ip>` from there.
5. Re-run every check — containers up, proxy env vars present in the running container (`docker exec parapet-backend env | grep -i proxy`), app reachable at `http://<dev-box-private-ip>:8090` from a tailnet-connected laptop.

---

## Ongoing operations

**Changing the Squid allowlist:** edit `squid.conf.template` in `parapet-dev-proxy`, PR, merge — deploys automatically, its own smoke test verifies the change against the real dev-box traffic path (nested SSH: CI → gateway → dev box → curl through Squid). Branch protection (required review) recommended.

**Reaching the dev box day to day:**
```bash
tailscale ssh root@<gateway-hostname>
ssh -i /root/.ssh/dev_deploy root@<dev-box-private-ip>
```

**Reaching the app from your laptop:** ensure Tailscale is connected and the subnet route is approved, then browse directly to `http://<dev-box-private-ip>:8090` (app), `:3000` (Grafana), `:8081` (Adminer) — no tunnel needed, the approved route makes it a direct peer address.

**The GitHub Actions run itself links directly to the app** via the job's `environment: { name: dev, url: http://${{ vars.SERVER_PRIVATE_IP }}:8090 }` block — click through from the PR or the Actions run summary rather than typing the address by hand.

## Disaster recovery

Hetzner snapshots of both boxes exist as of the images taken during setup. To restore:
1. Restore from the snapshot.
2. Assign the **same private IP** the box had before (`10.0.0.2` gateway / `10.0.0.4` dev box, or your actual values) — doing so means Squid config, nftables rules, and every GitHub repo variable keyed off these IPs needs **no changes at all**.
3. On the restored **gateway** specifically: run `tailscale logout` then re-authenticate (`tailscale up ...` as in Phase 2). A snapshot restore duplicates the machine's Tailscale identity; without this step, two devices would claim the same tailnet identity.

Full Terraform/IaC provisioning of Phases 0–13 remains a known gap — snapshots are the current stopgap, not a replacement for it.

## Known accepted gaps

- Allowlisted domains are themselves an exfiltration channel (e.g. Docker Hub pulls). Containment reduces blast radius; it does not eliminate it.
- Any API key present on the dev box is lost on takeover, usable from anywhere afterward. Future mitigation: terminate credentials at the gateway (dev box calls the proxy with no key, gateway injects the real one before forwarding).
- The GitHub Actions runner itself holds tailnet + deploy access for the duration of each job — ephemeral, single-use keys via OAuth limit this window.
- Adminer on dev has no authentication layer of its own; access control is entirely "must be a tailnet member with the approved subnet route." Acceptable for a dev box; would not be for prod (see `DEPLOY.md`, which layers real basic-auth in front via Caddy).