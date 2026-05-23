# Parapet — Hetzner First Deployment

Manual setup before the GitHub Actions pipeline is active. After this, pushes to `main` handle everything automatically.

---

## Stack

- Frontend: static files served by Caddy from `/opt/parapet/frontend/dist`
- Backend: Docker Compose on `localhost:8080`
- Reverse proxy: Caddy (auto TLS via Let's Encrypt), routes `/api/*` to backend
- CI/CD: GitHub Actions on push to `main`

---

## 1. SSH Key (local machine)

```bash
ssh-keygen -t ed25519 -C "[ssh-key-name]"
# save to /Users/yourusername/.ssh/[your-key-file]

cat ~/.ssh/[your-key-file].pub
# copy output — needed during VPS creation
```

---

## 2. Provision VPS (Hetzner Cloud Console)

- OS: Ubuntu 24.04
- Enable the **Docker** pre-install option
- SSH Key: paste public key from step 1
- Note the assigned IP address

---

## 3. Connect

```bash
ssh -i ~/.ssh/[your-key-file] root@[SERVER-IP]
```

---

## 4. System Updates

```bash
apt update && apt upgrade -y
```

---

## 5. Install Caddy

```bash
apt install -y debian-keyring debian-archive-keyring apt-transport-https curl

curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/gpg.key' | \
  gpg --dearmor -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg

curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt' | \
  tee /etc/apt/sources.list.d/caddy-stable.list

apt update && apt install caddy

systemctl status caddy  # verify: active (running)
```

---

## 6. DNS

Create A records at your DNS provider:

```
Type: A  Name: @       Value: [SERVER-IP]  TTL: 300
Type: A  Name: www     Value: [SERVER-IP]  TTL: 300
Type: A  Name: grafana Value: [SERVER-IP]  TTL: 300
Type: A  Name: db      Value: [SERVER-IP]  TTL: 300
```

Verify propagation:

```bash
dig parapet.digital +short
```

---

## 7. Clone Repository

```bash
git clone https://github.com/yourusername/parapet.git /opt/parapet
cd /opt/parapet
```

---

## 8. Environment Variables

```bash
cp .env.example .env
nano .env
```

---

## 9. Start Backend

```bash
docker compose up --build -d

docker compose ps
curl http://localhost:8080/health
```

---

## 10. Build Frontend (first time only)

Option A — configure GitHub secrets first (step 12), then push to `main`. CI builds and deploys automatically.

Option B — build on the server:

```bash
curl -fsSL https://deb.nodesource.com/setup_20.x | bash -
apt install -y nodejs

cd /opt/parapet/frontend
npm install
npm run build
```

If build fails:

```bash
rm -rf node_modules package-lock.json
npm install
npm run build
```

---

## 11. Configure Caddy

```bash
nano /etc/caddy/Caddyfile
```

```caddy
parapet.digital {
    handle /api/* {
        uri strip_prefix /api
        reverse_proxy localhost:8080
    }
    handle {
        root * /opt/parapet/frontend/dist
        try_files {path} /index.html
        file_server
    }
}

www.parapet.digital {
    redir https://parapet.digital{uri} permanent
}

grafana.parapet.digital {
    reverse_proxy localhost:3000
}
```

DNS must be resolving to the server before Caddy can obtain a TLS cert.

```bash
systemctl restart caddy
systemctl status caddy
```

---

## 12. GitHub Actions Secrets

Repo → Settings → Secrets and variables → Actions:

- `HETZNER_HOST` — server IP
- `HETZNER_SSH_KEY` — contents of `~/.ssh/[your-key-file]` (private key)

Pipeline on push to `main`:
1. Build frontend in CI
2. scp `frontend/dist/` → `/opt/parapet/frontend/dist`
3. `git pull` + `docker compose up -d --build` on server
4. Health check: `GET https://parapet.digital/api/health`

---

## 13. Observability

Grafana is available at https://grafana.parapet.digital
Username: admin
Password: set via GRAFANA_PASSWORD in .env on the server

**Adding Loki as data source in Grafana (first time only):**
1. Go to https://grafana.parapet.digital
2. Left sidebar → Connections → Data sources → Add data source
3. Choose Loki
4. URL: http://loki:3100
5. Click Save & Test — should show green

**Useful log queries:**
```
{container="parapet-backend-1"} |= "error"
{container="parapet-backend-1"} |= "pipeline run complete"
{container="parapet-backend-1"} |= "match"
{container="parapet-backend-1"} |= "socket"
```

### Database UI (Adminer)

Adminer is available at https://db.parapet.digital
It is protected by two layers of authentication:
1. Caddy basic auth — browser username/password popup
2. Adminer login form — requires Postgres credentials

**First time setup on the server:**

**Step 1 — Generate the Caddy basic auth password hash:**
```bash
caddy hash-password --plaintext your-chosen-password
```
This outputs a bcrypt hash like:
```
$2a$14$Zxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx2
```
Copy the entire hash output including the `$` prefix.
Never store the plaintext password in any file — only the hash goes in the Caddyfile.

**Step 2 — Add the Adminer block to `/etc/caddy/Caddyfile`:**
```caddy
db.parapet.digital {
    basicauth {
        admin PASTE_YOUR_HASH_HERE
    }
    reverse_proxy localhost:8081
}
```
Replace `PASTE_YOUR_HASH_HERE` with the bcrypt hash from Step 1.
The username is: `admin`
The password is: whatever plaintext you used in Step 1 — store it in your password manager.

**Step 3 — Restart Caddy:**
```bash
systemctl restart caddy
```

**Step 4 — Add DNS record:**
```
Type: A  Name: db  Value: YOUR_SERVER_IP  TTL: 300
```

**Step 5 — Pull and restart containers:**
```bash
cd /opt/parapet
git pull origin main
docker compose up -d
```

**Step 6 — Verify Adminer is running:**
```bash
docker ps | grep adminer
```

**Logging into Adminer:**

When you visit https://db.parapet.digital your browser shows a username/password popup. Enter:
- Username: `admin`
- Password: the plaintext password you chose in Step 1

After passing Caddy auth, Adminer shows its own login form. Enter:
- System: `PostgreSQL`
- Server: `postgres`
- Username: `postgres`
- Password: your `POSTGRES_PASSWORD` from `.env`
- Database: `parapet`

**If you need to change the Caddy basic auth password:**
1. Generate a new hash: `caddy hash-password --plaintext new-password`
2. Update `/etc/caddy/Caddyfile` with the new hash
3. `systemctl restart caddy`
4. Update your password manager

**Security notes:**
- The Postgres port (5432) is never exposed to the internet
- Adminer only binds to `127.0.0.1:8081` — not reachable from outside
- All traffic goes through Caddy over HTTPS
- Two auth layers protect the database from unauthorized access
- Never expose Adminer without Caddy basic auth in front of it

---

## 14. Verify

```bash
curl https://parapet.digital/api/health

journalctl -u caddy -f     # Caddy logs
docker compose logs -f     # backend logs
```
