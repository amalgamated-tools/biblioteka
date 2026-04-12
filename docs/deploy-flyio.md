# Deploy to Fly.io

This guide walks through deploying Biblioteka on [Fly.io](https://fly.io), a platform that runs your app close to your users with automatic TLS, persistent volumes, and a generous free tier.

## Prerequisites

- A [Fly.io account](https://fly.io/app/sign-up)
- The [flyctl CLI](https://fly.io/docs/flyctl/install/) installed and authenticated (`fly auth login`)

## Overview

Biblioteka runs as a single Docker container on Fly.io. The setup is:

- **App** — the Biblioteka container using the pre-built GHCR image
- **Volume** — a Fly.io persistent volume mounted at `/data` for the SQLite database
- **Redis** — an Upstash Redis instance (provisioned via the `fly` CLI) for the background job queue

## Step 1 — Create the app

Choose a unique app name (shown as `your-biblioteka` in examples below):

```bash
fly apps create your-biblioteka
```

## Step 2 — Create a persistent volume

The SQLite database is stored in `/data` inside the container. Create a volume in the same region you plan to deploy to (replace `lax` with your preferred [region code](https://fly.io/docs/reference/regions/)):

```bash
fly volumes create biblioteka_data \
  --app your-biblioteka \
  --region lax \
  --size 3
```

`--size` is the volume size in GB. 3 GB is comfortable for most personal libraries; increase it if you plan to store many large PDFs.

## Step 3 — Add Upstash Redis

Biblioteka requires Redis for background job processing. Provision a free Upstash Redis instance:

```bash
fly ext upstash redis create \
  --app your-biblioteka \
  --name your-biblioteka-redis
```

This automatically sets the `REDIS_URL` secret on your app.

## Step 4 — Set secrets

```bash
# Required: a strong random secret for signing JWTs
fly secrets set \
  --app your-biblioteka \
  JWT_SECRET=$(openssl rand -hex 32)
```

Optionally, disable public sign-up after you create your first account:

```bash
fly secrets set --app your-biblioteka DISABLE_SIGNUP=true
```

## Step 5 — Create `fly.toml`

Create a `fly.toml` file in your working directory with the following content (replace `your-biblioteka` with your app name and update the `primary_region` to match your volume's region):

```toml
app = "your-biblioteka"
primary_region = "lax"

[build]
  image = "ghcr.io/amalgamated-tools/biblioteka:latest"

[env]
  PORT = "8080"
  SECURE_COOKIES = "true"
  LOG_LEVEL = "info"
  LOG_FORMAT = "json"
  TRUSTED_PROXIES = "10.0.0.0/8,172.16.0.0/12"

[mounts]
  source = "biblioteka_data"
  destination = "/data"

[[services]]
  internal_port = 8080
  protocol = "tcp"

  [[services.ports]]
    port = 80
    handlers = ["http"]
    force_https = true

  [[services.ports]]
    port = 443
    handlers = ["tls", "http"]

  [[services.http_checks]]
    interval = "15s"
    timeout = "5s"
    grace_period = "10s"
    method = "GET"
    path = "/api/health"
```

## Step 6 — Deploy

```bash
fly deploy --app your-biblioteka
```

Fly.io builds the deployment, attaches the volume, and starts the container. Once complete, open the app:

```bash
fly open --app your-biblioteka
```

The first account you register becomes the admin.

## Updating

To deploy a newer version of Biblioteka, pull the latest image and redeploy:

```bash
fly deploy --app your-biblioteka
```

Migrations run automatically on startup — no manual steps needed.

## Custom Domain

To point your own domain at the app:

1. Add a CNAME record pointing your domain to `your-biblioteka.fly.dev`.
2. Add the certificate:
   ```bash
   fly certs add books.example.com --app your-biblioteka
   ```

Fly.io provisions a Let's Encrypt certificate automatically. See the [Fly.io custom domains documentation](https://fly.io/docs/networking/custom-domain/) for detailed steps.

## Viewing Logs

```bash
fly logs --app your-biblioteka
```

## SSH Access

```bash
fly ssh console --app your-biblioteka
```

From inside the container you can inspect the SQLite database:

```bash
ls /data/
```

## Scaling

For a personal library, a single `shared-cpu-1x` machine with 256 MB RAM is sufficient. To change the machine size:

```bash
fly scale vm shared-cpu-1x --memory 512 --app your-biblioteka
```

## Backup (SQLite)

Download the database file from the running container:

```bash
fly ssh sftp get /data/biblioteka.db ./biblioteka.db.bak --app your-biblioteka
```

Or use `fly ssh console` and `fly sftp` for a full interactive session.

## Cost

The [Fly.io free tier](https://fly.io/docs/about/pricing/) includes three shared-CPU VMs and 3 GB of persistent storage — enough to run Biblioteka at no cost. Upstash Redis is also free up to 10,000 daily commands.

## Troubleshooting

| Problem | Solution |
|---------|----------|
| App starts but shows 502 | Check `fly logs` — the container may still be starting; the health check grace period is 10 s |
| `REDIS_URL` not set | Run `fly ext upstash redis list` to confirm the Redis instance was created; re-run step 3 if missing |
| Volume not attached | Run `fly volumes list --app your-biblioteka` and verify the volume exists in the same region as your app |
| JWT errors after redeploy | Clear browser cookies and log in again — JWT secrets changed between deploys invalidate existing tokens |
