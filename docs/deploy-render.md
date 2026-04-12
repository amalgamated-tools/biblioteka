# Deploy to Render

This guide walks through deploying Biblioteka on [Render](https://render.com), a fully managed cloud platform with automatic TLS, persistent disks, and a straightforward dashboard.

## Prerequisites

- A [Render account](https://dashboard.render.com/register)
- A GitHub account (to use the Blueprint deploy path) — or just the Render dashboard

## Overview

Biblioteka on Render uses:

- **Web Service** — the Biblioteka container pulled from GHCR
- **Persistent Disk** — a Render managed disk mounted at `/data` for the SQLite database
- **Redis** — a Render managed Redis instance for the background job queue

## Option A — Blueprint (Infrastructure as Code)

The fastest way to deploy is with a Render Blueprint. A Blueprint is a `render.yaml` file that defines all services at once.

### 1. Create `render.yaml`

Place this file in the root of a repository you control (or a fork of Biblioteka):

```yaml
services:
  - type: web
    name: biblioteka
    runtime: image
    image:
      url: ghcr.io/amalgamated-tools/biblioteka:latest
    healthCheckPath: /api/health
    disk:
      name: biblioteka-data
      mountPath: /data
      sizeGB: 3
    envVars:
      - key: PORT
        value: 8080
      - key: SECURE_COOKIES
        value: "true"
      - key: LOG_LEVEL
        value: info
      - key: LOG_FORMAT
        value: json
      - key: REDIS_URL
        fromService:
          type: redis
          name: biblioteka-redis
          property: connectionString
      - key: JWT_SECRET
        generateValue: true
      - key: TRUSTED_PROXIES
        value: "10.0.0.0/8,172.16.0.0/12"

  - type: redis
    name: biblioteka-redis
    plan: free
    maxmemoryPolicy: noeviction
```

### 2. Push to GitHub

Commit `render.yaml` to a GitHub repository and push.

### 3. Connect to Render

1. Go to [dashboard.render.com](https://dashboard.render.com) → **New** → **Blueprint**.
2. Connect your GitHub repository.
3. Render detects `render.yaml` and shows a preview of the services to create.
4. Click **Apply** — Render provisions the web service, disk, and Redis in one step.

### 4. Open your app

Once the deploy finishes, click the generated URL (e.g., `https://biblioteka.onrender.com`). The first account you register becomes the admin.

### Disabling public sign-up

After creating your account, add `DISABLE_SIGNUP=true` in **Dashboard → biblioteka → Environment** and redeploy.

---

## Option B — Manual Dashboard Setup

If you prefer to set everything up through the Render dashboard without a `render.yaml`:

### 1. Create a Redis instance

1. **New** → **Redis**.
2. Name it `biblioteka-redis`.
3. Set **Max Memory Policy** to `noeviction`.
4. Choose a region and the **Free** plan.
5. Click **Create Redis** and note the **Internal Redis URL** shown in the service overview.

### 2. Create the web service

1. **New** → **Web Service**.
2. Select **Deploy an existing image**.
3. Enter the image URL: `ghcr.io/amalgamated-tools/biblioteka:latest`
4. Name: `biblioteka`
5. Region: same as your Redis instance.
6. Plan: **Free** (or Starter for always-on availability; see [note on free-tier sleep](#free-tier-sleep) below).

### 3. Configure environment variables

In the **Environment** tab of the web service, add:

| Key | Value |
|-----|-------|
| `PORT` | `8080` |
| `SECURE_COOKIES` | `true` |
| `LOG_LEVEL` | `info` |
| `LOG_FORMAT` | `json` |
| `REDIS_URL` | *(paste the Internal Redis URL from step 1)* |
| `JWT_SECRET` | *(click **Generate** for a random value)* |
| `TRUSTED_PROXIES` | `10.0.0.0/8,172.16.0.0/12` |

### 4. Add a persistent disk

1. Go to the **Disks** tab of the web service.
2. Click **Add Disk**.
3. **Mount Path:** `/data`
4. **Size:** 3 GB (adjust to your needs).

### 5. Deploy

Click **Create Web Service**. Render pulls the image, attaches the disk, and starts the container.

---

## Updating

Render automatically re-deploys when it detects a new version of the image (if **Auto-Deploy** is enabled). To trigger a manual redeploy:

1. Go to **Dashboard → biblioteka**.
2. Click **Manual Deploy** → **Deploy latest commit**.

Alternatively, to pin a specific image tag, edit the image URL in **Settings → Image URL** and save.

## Custom Domain

1. Go to **Dashboard → biblioteka → Settings → Custom Domains**.
2. Add your domain.
3. Render provides a `CNAME` target — add it to your DNS.
4. Render provisions a TLS certificate automatically.

## Logs

Live logs are available in **Dashboard → biblioteka → Logs**. You can filter by keyword or level.

## Backup (SQLite)

Render persistent disks are not directly accessible from outside the container. To back up your SQLite database, use the Render Shell (available on paid plans) or the [Render API](https://render.com/docs/api) to run a backup command. Alternatively, consider the PostgreSQL backend for a database you can snapshot via the Render PostgreSQL add-on.

### Using PostgreSQL instead of SQLite

For a more durable setup on Render, replace the persistent disk with a Render PostgreSQL instance:

1. **New** → **PostgreSQL** → create an instance named `biblioteka-db`.
2. Remove the `disk` block from `render.yaml` (or skip step 4 above).
3. Set the `DATABASE_URL` environment variable to the **Internal Database URL** provided by the PostgreSQL service.

Render automatically backs up PostgreSQL databases on paid plans.

## Free-Tier Sleep

Render free-tier web services spin down after 15 minutes of inactivity and take ~30 seconds to wake up. For a personal library accessed throughout the day this is usually acceptable. Upgrade to the **Starter** plan ($7/month) if you want always-on availability.

## Cost

| Component | Free tier | Notes |
|-----------|-----------|-------|
| Web Service | Free (sleeps after inactivity) | Starter plan ($7/mo) for always-on |
| Persistent Disk | $0.25/GB/month | Billed from the first GB |
| Redis | Free (25 MB) | Sufficient for the job queue |

A minimal always-on setup (Starter web + 3 GB disk + free Redis) costs around **$8–9/month**.
