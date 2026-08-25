# Parapet — Production Deployment (Hetzner)

Prod runs the same application and the same `docker-compose.yml` as dev, with none of dev's network isolation — this box is on the open internet behind Caddy, with real domains and real TLS certs. See `DEPLOY_DEV.md` for the isolated dev environment and why it differs.

If this box is ever rebuilt, follow this top to bottom. A Hetzner snapshot also exists as a faster recovery path — see "Disaster recovery."

## What changed since the original manual setup

The original version of this doc described building the frontend locally on the box (Node/npm), `scp`-ing the built `dist/` folder over, and running `docker compose up -d --build` to build the backend image on the box itself. **None of that happens anymore.** The frontend is now a Docker image (multi-stage build, nginx), built in CI alongside the backend, both pushed to GHCR, and the box only ever pulls pre-built images — never builds anything itself. This mirrors dev's pipeline shape, with the isolation layer (gateway, Squid, egress restriction) intentionally omitted here — see `DEPLOY_DEV.md`'s architecture summary for why that trade-off is deliberate for prod.

---

## 1. Provision the server

- Hetzner Cloud VM, Ubuntu, with a public IP.
- SSH key for your own admin access — this doc uses `parapet-hetzner-prod` as the example name.

## 2. Connect and update

```bash
ssh -i ~/.ssh/parapet-hetzner-prod root@<server-public-ip>
apt update && apt upgrade -y
```

## 3. Install Docker

```bash
apt update && apt install -y ca-certificates curl gnupg
install -m 0755 -d /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg -o /etc/apt/keyrings/docker.asc
chmod a+r /etc/apt/keyrings/docker.asc
echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/ubuntu $(. /etc/os-release && echo "$VERSION_CODENAME") stable" | tee /etc/apt/sources.list.d/docker.list
apt update
apt install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
```

No Node/npm needed on this box at all — the frontend build happens entirely inside CI's Docker build step.

## 4. Install Caddy

```bash
apt install -y debian-keyring debian-archive-keyring apt-transport-https curl
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/gpg.key' | gpg --dearmor -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt' | tee /etc/apt/sources.list.d/caddy-stable.list
apt update && apt install caddy
systemctl status caddy   # verify: active (running)
```

## 5. DNS

```
Type: A  Name: @        Value: <server-public-ip>  TTL: 300
Type: A  Name: www      Value: <server-public-ip>  TTL: 300
Type: A  Name: grafana  Value: <server-public-ip>  TTL: 300
Type: A  Name: db       Value: <server-public-ip>  TTL: 300
```

```bash
dig parapet.digital +short
```

## 6. Clone the repository

```bash
git clone https://github.com/jmpberlin/parapet.git /opt/parapet
```

Unlike dev, this stays a normal SSH-based (or plain HTTPS, unauthenticated for a public repo) clone — prod has no egress restriction requiring the proxy-aware HTTPS+PAT dance dev needs.

## 7. `ci_runner` deploy key (GitHub Actions → this box)

Same least-privilege pattern as dev: a dedicated keypair for CI, never your own admin key.

On your **local machine**:
```bash
ssh-keygen -t ed25519 -f ~/.ssh/parapet_ci_runner_prod -C "github-actions-prod-deploy" -N ""
cat ~/.ssh/parapet_ci_runner_prod.pub
```

On the **server**, using your own admin key to connect:
```bash
nano ~/.ssh/authorized_keys
```
Add the new pubkey as its own line. **Do not remove your own admin key's line.**

Verify both independently:
```bash
ssh -i ~/.ssh/parapet-hetzner-prod root@<server-public-ip> hostname   # your own access
ssh -i ~/.ssh/parapet_ci_runner_prod root@<server-public-ip> hostname # CI's access
```

Store `parapet_ci_runner_prod`'s **private** key contents as the GitHub Actions secret `CI_RUNNER_SSH_KEY`, scoped to the **`prod` Environment**. Note this authenticates as `root` on prod — unlike dev's `ci_runner`, which is a restricted, non-root user. Tightening prod to a similarly scoped deploy user is a reasonable future improvement, not yet done.

## 8. GHCR authentication: fully automated, no manual step

No static PAT is stored on this box. The deploy pipeline logs into GHCR fresh on every run using GitHub's own ephemeral, auto-issued `secrets.GITHUB_TOKEN`, forwarded once over the SSH session — same mechanism as dev. Nothing to create or rotate for this specifically.

## 9. `docker-compose.yml`

This is the **same file as dev** — see `DEPLOY_DEV.md` Phase 11 for the full annotated version (proxy env vars, dual port bindings). On prod:
- `GATEWAY_HTTP_PROXY` / `GATEWAY_HTTPS_PROXY` are simply never set in `.env` — the compose file's `${VAR:-}` fallback resolves this to an empty string, correctly meaning "no proxy" for the application.
- `SERVER_PRIVATE_IP` is likewise never set — the `:-127.0.0.1` fallback on each port binding means frontend/Adminer/Grafana bind to loopback only, which is exactly right here: **Caddy** is what exposes them to the internet, not a direct port binding to a routable address (unlike dev, which has no reverse proxy in front and needs the private-IP binding instead).

## 10. `Caddyfile.template` — tracked in the repo, rendered by CI

The Caddyfile lives in the repo as `Caddyfile.template`, **not** `Caddyfile` — the tracked version still contains an unresolved `${ADMINER_BEARER}` placeholder, so naming it `.template` is accurate; it isn't usable as-is until CI substitutes the real value.

```caddyfile
parapet.digital {
    reverse_proxy 127.0.0.1:8090
}

www.parapet.digital {
    redir https://parapet.digital{uri} permanent
}

grafana.parapet.digital {
    reverse_proxy 127.0.0.1:3000
}

db.parapet.digital {
    basicauth {
        admin ${ADMINER_BEARER}
    }
    reverse_proxy 127.0.0.1:8081
}
```

**This is simpler than the original Caddyfile** — the old version had a `handle /api/*` block doing `uri strip_prefix /api` and serving static files directly from `/opt/parapet/frontend/dist`. That logic now lives entirely inside the frontend container's own `nginx.conf` (same one dev uses), so Caddy's only job on `parapet.digital` is a single, path-agnostic reverse proxy to the frontend container. This also removes a real risk the old design carried: nginx and Caddy could have both tried to strip the `/api` prefix if the split were ever done differently, silently double-stripping and breaking every API route. With Caddy reduced to a dumb proxy, only nginx owns that logic — no ambiguity.

**`ADMINER_BEARER` must be a bcrypt hash, not the plaintext password** — Caddy's `basicauth` directive requires it. Generate once:
```bash
caddy hash-password --plaintext <your-chosen-password>
```
Store the **hash** output (starts with `$2a$...`) as the `ADMINER_BEARER` secret in the `prod` Environment. Store the plaintext password separately, in your password manager — it cannot be recovered from the hash, and you'll need it to actually log in.

**Confirm the live config path** before the first automated deploy:
```bash
systemctl status caddy
cat /etc/systemd/system/caddy.service | grep ExecStart
```
This doc assumes `/etc/caddy/Caddyfile`, the standard package default — verify rather than assume, since a wrong path means CI writes a file Caddy never actually reads.

## 11. `.env`: fully automated, rendered by CI every deploy

No manual `.env` creation — the workflow writes it fresh on every deploy, same mechanism as dev, from GitHub Secrets/Variables. See the full workflow below.

## 12. GitHub configuration

**`parapet` repo — repo-level** (shared with dev, see `DEPLOY_DEV.md` Phase 13): `BACKEND_HOST`, `BACKEND_HOST_PORT`, `BACKEND_PORT`, `FRONTEND_HOST_PORT`, `POSTGRES_DB`, `POSTGRES_USER`, `POSTGRES_PORT`, `VITE_API_URL`.

**`parapet` repo — `prod` Environment:**
- Secrets: `CI_RUNNER_SSH_KEY`, `CLAUDE_API_KEY`, `GRAFANA_PASSWORD`, `PAT_GITHUB`, `POSTGRES_PASSWORD`, `ADMINER_BEARER`
- Variables: `SERVER_PUBLIC_IP`, `GRAFANA_ROOT_URL`

Same secret/variable *names* as dev's environment where they overlap (`CI_RUNNER_SSH_KEY`, `CLAUDE_API_KEY`, etc.) — deliberately, so the workflow YAML reads identically in shape between the two files, differing only in which Environment block is active. Values are entirely separate per environment; nothing is shared except `TS_OAUTH_CLIENT_ID`/`TS_OAUTH_SECRET` (repo-level, dev-only concept — prod has no Tailscale involvement at all).

## 13. `.github/workflows/deploy.yml`

```yaml
name: Deploy to Hetzner

on:
  push:
    branches: [main]

jobs:
  deploy:
    runs-on: ubuntu-latest
    permissions:
      contents: read
      packages: write
    environment:
      name: prod
      url: https://parapet.digital
    env:
      IMAGE_TAG: ${{ github.sha }}
    steps:
      - uses: actions/checkout@v4

      - uses: docker/login-action@v3
        with:
          registry: ghcr.io
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}

      - name: Build & push images
        run: |
          for svc in backend frontend; do
            img=ghcr.io/jmpberlin/parapet-$svc
            docker build -t $img:${IMAGE_TAG} ./$svc
            docker push $img:${IMAGE_TAG}
          done

      - name: Render Caddyfile
        run: envsubst '$ADMINER_BEARER' < Caddyfile.template > Caddyfile
        env:
          ADMINER_BEARER: ${{ secrets.ADMINER_BEARER }}

      - name: Deploy Caddyfile
        run: |
          cat Caddyfile | ssh -o StrictHostKeyChecking=accept-new -i <(echo "${{ secrets.CI_RUNNER_SSH_KEY }}") \
            root@${{ vars.SERVER_PUBLIC_IP }} \
            'cat > /etc/caddy/Caddyfile && systemctl reload caddy'

      - name: Write .env
        run: |
          ssh -o StrictHostKeyChecking=accept-new -i <(echo "${{ secrets.CI_RUNNER_SSH_KEY }}") \
            root@${{ vars.SERVER_PUBLIC_IP }} "cat > /opt/parapet/.env" <<EOF
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
          GRAFANA_ROOT_URL=${{ vars.GRAFANA_ROOT_URL }}
          EOF

      - name: Deploy app
        run: |
          ssh -o StrictHostKeyChecking=accept-new -i <(echo "${{ secrets.CI_RUNNER_SSH_KEY }}") \
            root@${{ vars.SERVER_PUBLIC_IP }} "bash -euo pipefail -s" <<EOF
            cd /opt/parapet
            git pull origin main
            echo "${{ secrets.GITHUB_TOKEN }}" | docker login ghcr.io -u ${{ github.actor }} --password-stdin
            export IMAGE_TAG=${IMAGE_TAG}
            docker compose pull backend frontend
            docker compose up -d
          EOF

      - name: Health check
        run: |
          sleep 15
          curl --fail https://parapet.digital/api/health
```

Note the two `docker/login-action` and `docker login ghcr.io` steps are **not redundant** — the first authenticates the *runner* to push newly built images; the second, inside the SSH session, authenticates the *box* to pull them. Same underlying ephemeral token, two different machines needing it.

**Heredoc gotcha** (same as dev): the closing `EOF` in both heredoc blocks must sit at the *same indentation as the block's content lines* in the YAML source, not flush-left — `run: |` strips a uniform indentation amount, so matching indentation is what lands the true terminator at column-zero in the executed script.

## 14. First deploy

Push to `main`. The pipeline builds both images, pushes to GHCR, renders and deploys the Caddyfile, writes `.env`, pulls the images on the box, and brings the stack up — no manual `docker compose up` step required, even for a freshly cloned box, since the pipeline handles the entire sequence.

## 15. Grafana

`https://grafana.parapet.digital` — username `admin`, password from `GRAFANA_PASSWORD`.

**Adding Loki as a data source (first time only):** Connections → Data sources → Add data source → Loki → URL `http://loki:3100` → Save & Test.

**Useful log queries:**
```
{container="parapet-backend"} |= "error"
{container="parapet-backend"} |= "pipeline run complete"
```

## 16. Adminer

`https://db.parapet.digital` — two auth layers:
1. **Caddy basicauth** (browser popup) — username `admin`, password is the *plaintext* you generated the hash from in step 10 (from your password manager, not derivable from the hash).
2. **Adminer's own login form** — System `PostgreSQL`, Server `postgres`, Username `postgres`, Password from `.env`'s `POSTGRES_PASSWORD`, Database `parapet`.

**Rotating the Caddy basicauth password:**
```bash
caddy hash-password --plaintext <new-password>
```
Update the `ADMINER_BEARER` secret in the `prod` Environment with the new hash, then push any commit (or re-run the workflow) to redeploy the Caddyfile. Update your password manager with the new plaintext.

**Security notes, unchanged from the original design:**
- Postgres's port (5432) is never exposed to the internet.
- Adminer binds to `127.0.0.1:8081` only — reachable exclusively through Caddy.
- Two independent auth layers protect it.

## 17. Verify

```bash
curl https://parapet.digital/api/health
journalctl -u caddy -f
docker compose logs -f
```

---

## Disaster recovery

A Hetzner snapshot of this box exists. To restore: restore the snapshot, reassign the same public IP if possible (simplifies DNS — otherwise update the A records), confirm Docker/Caddy came back up (`systemctl status docker caddy`), and re-run the deploy workflow to ensure `.env`/`Caddyfile` are current rather than whatever was on disk at snapshot time.

## Known gaps

- CI's deploy key authenticates as `root`, not a scoped deploy user (unlike dev's `ci_runner`). Worth tightening eventually, not yet done.
- No egress restriction on this box at all — by design (see `DEPLOY_DEV.md`'s architecture summary for the reasoning: prod runs reviewed code on `main`, not AI-authored code pre-review, so the isolation dev needs doesn't apply here).
- `db.parapet.digital` is publicly routable (behind two auth layers) rather than access-restricted by network position, unlike dev's Adminer (tailnet-only, no auth layer of its own). Different trade-off, not necessarily a gap — worth revisiting only if this ever feels insufficient.