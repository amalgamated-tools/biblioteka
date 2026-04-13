# Deploy to Railway

This guide walks through deploying Biblioteka on [Railway](https://railway.app), a developer platform with simple UI-driven deployments, persistent volumes, and a built-in Redis add-on.

## Prerequisites

- A [Railway account](https://railway.app/login)
- The [Railway CLI](https://docs.railway.app/guides/cli) (optional but useful for local management)

## Overview

Biblioteka on Railway uses:

- **Service** — the Biblioteka container pulled from GHCR
- **Volume** — a Railway persistent volume mounted at `/data` for the SQLite database
- **Redis** — a Railway managed Redis service for the background job queue

!!! note "Volume support"
    Persistent volumes are available on the **Hobby plan** ($5/month) and above. The free Trial tier does not support volumes. If you are on the Trial tier, consider the [PostgreSQL backend](#using-postgresql-instead-of-sqlite) below.

## Step 1 — Create a new project

1. Go to [railway.app/new](https://railway.app/new).
2. Click **Empty project**.

## Step 2 — Add a Redis service

1. Inside the project, click **+ New** → **Database** → **Add Redis**.
2. Railway provisions a Redis instance and exposes a `REDIS_URL` variable automatically.

## Step 3 — Add the Biblioteka service

1. Click **+ New** → **Docker Image**.
2. Enter the image: `ghcr.io/amalgamated-tools/biblioteka:latest`
3. Click **Deploy**.

## Step 4 — Configure environment variables

Go to the Biblioteka service → **Variables** tab. Add the following:

| Variable | Value |
|----------|-------|
| `PORT` | `8080` |
| `SECURE_COOKIES` | `true` |
| `LOG_LEVEL` | `info` |
| `LOG_FORMAT` | `json` |
| `JWT_SECRET` | *(click the magic wand to generate a random value)* |
| `REDIS_URL` | `${{Redis.REDIS_URL}}` |
| `TRUSTED_PROXIES` | `10.0.0.0/8,172.16.0.0/12` |

The `${{Redis.REDIS_URL}}` reference is a Railway [reference variable](https://docs.railway.app/guides/variables#referencing-another-services-variable) that automatically pulls the connection string from your Redis service — replace `Redis` with the name of your Redis service if you renamed it.

## Step 5 — Attach a persistent volume

1. Go to the Biblioteka service → **Volumes** tab.
2. Click **+ New Volume**.
3. Set the **Mount Path** to `/data`.
4. Click **Create**.

## Step 6 — Set the health check and port

1. Go to **Settings** → **Networking**.
2. Click **Add Public Domain** to get a generated Railway domain (e.g., `biblioteka-production.up.railway.app`).
3. Under **Health Check**, set the path to `/api/health`.

## Step 7 — Redeploy

After setting the variables and volume, trigger a redeploy:

1. Go to the Biblioteka service → **Deployments**.
2. Click **Redeploy** on the latest deployment.

Once the deploy finishes, open the generated domain. The first account you register becomes the admin.

## Disabling public sign-up

After creating your account, add `DISABLE_SIGNUP=true` in the **Variables** tab and redeploy.

---

## Updating

Railway can automatically redeploy when a new Docker image is available:

1. Go to the service → **Settings** → **Image**.
2. Enable **Auto-deploy on image update** if offered, or simply click **Redeploy** in the **Deployments** tab whenever you want to pull a newer `latest` tag.

To pin a specific version, update the image tag (e.g., `ghcr.io/amalgamated-tools/biblioteka:v0.1.0`) in **Settings → Image**.

## Custom Domain

1. Go to the service → **Settings** → **Networking** → **Custom Domain**.
2. Add your domain.
3. Add the `CNAME` record shown to your DNS provider.
4. Railway provisions a TLS certificate automatically.

## Logs

Live logs are available in **Dashboard → service → Logs**. Use the search bar to filter by level or keyword.

## Using PostgreSQL instead of SQLite

For a more resilient setup (or if volumes are not available on your plan):

1. Add a PostgreSQL service: **+ New** → **Database** → **Add PostgreSQL**.
2. In the Biblioteka service variables, add:

   | Variable | Value |
   |----------|-------|
   | `DATABASE_URL` | `${{Postgres.DATABASE_URL}}` |

3. Remove or leave the volume unattached — Biblioteka will use PostgreSQL automatically when `DATABASE_URL` is set.
4. Redeploy.

Railway's PostgreSQL service includes daily automated backups on the Hobby plan.

## Backup (SQLite)

If you are using SQLite with a volume, you can back up the database file using the Railway **Shell** tab (available on paid plans) to run commands directly inside the container:

```bash
cp /data/biblioteka.db /data/biblioteka-$(date +%Y%m%d).db.bak
```

> **Note:** `railway run` executes commands on your **local** machine with Railway environment variables injected — it does not run inside the container. Use the Shell tab for direct container access.

## Cost

| Component | Hobby plan |
|-----------|-----------|
| Web service | ~$5–10/month (based on usage) |
| Persistent volume | $0.25/GB/month |
| Redis | ~$1–3/month (based on usage) |
| PostgreSQL | ~$1–5/month (based on usage) |

Railway bills based on resource consumption. A lightly used personal library typically costs **$5–10/month** on the Hobby plan. See [Railway pricing](https://railway.app/pricing) for current rates.

## Troubleshooting

| Problem | Solution |
|---------|----------|
| Container exits immediately | Check the **Logs** tab — unreachable Redis can cause a startup failure. If `JWT_SECRET` is missing, the server logs a warning and starts with a random secret instead; set a strong `JWT_SECRET` in production so sessions remain valid across restarts |
| `REDIS_URL` shows as empty | Verify the reference variable uses the correct service name (e.g., `${{Redis.REDIS_URL}}`) |
| Volume not mounted | Go to **Volumes** and confirm the mount path is exactly `/data` |
| App unreachable | Confirm a public domain is configured under **Settings → Networking** and the port is `8080` |
