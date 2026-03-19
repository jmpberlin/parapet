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
Type: A  Name: @    Value: [SERVER-IP]  TTL: 300
Type: A  Name: www  Value: [SERVER-IP]  TTL: 300
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

## 13. Verify

```bash
curl https://parapet.digital/api/health

journalctl -u caddy -f     # Caddy logs
docker compose logs -f     # backend logs
```