---
description: Daily issues report as a themed HTML dashboard with CSS-only charts, SVG tickers, OGP images, sitemaps, and GitHub Pages deployment
on:
  schedule: daily
  workflow_dispatch:
permissions:
  contents: read
  actions: read
  issues: read
  pull-requests: read
  pages: write
  id-token: write
engine: copilot
strict: true
tracker-id: daily-issues-report
features:
  dangerous-permissions-write: true
tools:
  github:
    lockdown: false
    toolsets: [default]
  bash:
    - "*"
safe-outputs:
  upload-asset:
timeout-minutes: 30
imports:
  - shared/mood.md
  - shared/jqschema.md
  - shared/issues-data-fetch.md
  - shared/reporting.md
source: github/gh-aw/.github/workflows/daily-issues-report.md@852cb06ad52958b402ed982b69957ffc57ca0619

steps:
  - name: Setup report site directory
    run: |
      mkdir -p /tmp/gh-aw/report-site

  - name: Upload Pages artifact
    if: always()
    uses: actions/upload-pages-artifact@v4
    with:
      path: /tmp/gh-aw/report-site

  - name: Deploy to GitHub Pages
    if: always()
    id: pages-deployment
    uses: actions/deploy-pages@v4
---

{{#runtime-import? .github/shared-instructions.md}}

# Daily Issues Report — Themed HTML Dashboard

You are an expert front-end engineer and data analyst. Your job is to generate a complete, self-contained static HTML dashboard that visualizes repository issue metrics — **using only HTML, inline CSS, and inline SVG**. No JavaScript. No Python. No external assets or CDN links.

## Mission

Generate a daily report analyzing up to 1000 issues from the repository:
1. Process issues data with `bash` and `jq`
2. Build a themed HTML dashboard with **CSS-only charts** (bar, donut, sparkline)
3. Generate an **SVG ticker** strip summarizing key metrics
4. Produce an **OGP image** (SVG) for social-media previews
5. Write a **sitemap.xml**
6. Write every file to `/tmp/gh-aw/report-site/` for GitHub Pages deployment

## Current Context

- **Repository**: ${{ github.repository }}
- **Run ID**: ${{ github.run_id }}
- **Date**: Generated daily

---

## Phase 1: Load and Prepare Data

The issues data has been pre-fetched and is available at `/tmp/gh-aw/issues-data/issues.json`.

1. **Verify the data**:
   ```bash
   jq 'length' /tmp/gh-aw/issues-data/issues.json
   ```

2. **Extract metrics with `jq`** into `/tmp/gh-aw/report-site/data.json`:

   ```bash
   jq -c '{
     total:        length,
     open:         [.[] | select(.state=="OPEN")]  | length,
     closed:       [.[] | select(.state=="CLOSED")] | length,
     opened_7d:    [.[] | select(.createdAt > (now - 7*86400  | todate))] | length,
     opened_30d:   [.[] | select(.createdAt > (now - 30*86400 | todate))] | length,
     stale:        [.[] | select(.state=="OPEN") | select(.updatedAt < (now - 30*86400 | todate))] | length,
     no_labels:    [.[] | select(.state=="OPEN") | select((.labels | length)==0)] | length,
     no_assignees: [.[] | select(.state=="OPEN") | select((.assignees | length)==0)] | length,
     top_labels:   [.[] | .labels[]?.name] | group_by(.) | map({name:.[0], count:length}) | sort_by(-.count) | .[0:10],
     top_authors:  [.[] | .author.login] | group_by(.) | map({login:.[0], count:length}) | sort_by(-.count) | .[0:10],
     clusters:     [.[] | {
       cluster: (
         if   (.title | test("bug|fix|error";"i"))           then "Bug Reports"
         elif (.title | test("feat|enhancement|request";"i")) then "Feature Requests"
         elif (.title | test("doc|readme";"i"))               then "Documentation"
         elif (.title | test("test";"i"))                     then "Testing"
         elif (.title | test("refactor|cleanup";"i"))         then "Refactoring"
         elif (.title | test("secur|vuln";"i"))               then "Security"
         elif (.title | test("perf|slow";"i"))                then "Performance"
         else "Other" end
       )
     }] | group_by(.cluster) | map({name:.[0].cluster, count:length}) | sort_by(-.count),
     daily_opened: [.[] | .createdAt[:10]] | group_by(.) | map({date:.[0], count:length}) | sort_by(.date) | .[-30:],
     daily_closed: [.[] | select(.closedAt != null) | .closedAt[:10]] | group_by(.) | map({date:.[0], count:length}) | sort_by(.date) | .[-30:]
   }' /tmp/gh-aw/issues-data/issues.json > /tmp/gh-aw/report-site/data.json
   ```

   Adapt the `jq` filter as needed if any fields are missing. Gracefully handle nulls.

---

## Phase 2: Generate Themed HTML Dashboard

Write `/tmp/gh-aw/report-site/index.html` — a **single self-contained HTML file** with all CSS inlined in a `<style>` block. No `<script>` tags. No external resources.

### Design System

Use CSS custom properties on `:root` for a **dark professional theme**:

| Token | Value | Purpose |
|---|---|---|
| `--bg` | `#0d1117` | Page background |
| `--surface` | `#161b22` | Card background |
| `--border` | `#30363d` | Card/section borders |
| `--text` | `#e6edf3` | Primary text |
| `--text-muted` | `#7d8590` | Secondary text |
| `--accent` | `#58a6ff` | Links, highlights |
| `--green` | `#3fb950` | Positive / open metrics |
| `--red` | `#f85149` | Negative / attention metrics |
| `--purple` | `#bc8cff` | Cluster / category accent |
| `--orange` | `#d29922` | Warning / stale metrics |
| `--chart-1` through `--chart-8` | distinct hues | Chart segment colors |

Include a `@media (prefers-color-scheme: light)` override that remaps these tokens to a light palette.

### Layout

- Responsive CSS Grid: single column on mobile, 2-col on tablet, 3-col on desktop.
- `max-width: 1200px; margin: 0 auto;` wrapper.
- Sticky header bar with repo name, date, and a mini SVG ticker (see Phase 3).

### Required Sections (top → bottom)

1. **Hero / Summary Cards** — a grid of metric cards:
   - Total Issues (open / closed split shown as a stacked bar inside the card)
   - Issues Opened Last 7 Days
   - Issues Opened Last 30 Days
   - Stale Issues (30+ days no activity)
   - Issues Without Labels
   - Issues Without Assignees

2. **Issue Activity — Last 30 Days** (CSS-only area/line chart)
   - One SVG `<polyline>` for "opened per day" and another for "closed per day".
   - Use the `daily_opened` and `daily_closed` arrays from `data.json`.
   - The SVG viewBox should be e.g. `0 0 600 200`; scale data points to fit.
   - Add axis labels (first and last date) and a subtle grid with `<line>` elements.
   - Include a legend below the chart.

3. **Issue Clusters by Theme** (CSS-only horizontal bar chart)
   - Each cluster is a `<div>` row: label on the left, colored bar whose `width` is a CSS `calc()` percentage of the maximum cluster count, count on the right.
   - Use `--chart-N` colors for each bar.
   - Sort descending by count.

4. **Top Labels** — styled HTML `<table>` with label name (color-coded badge) and count.

5. **Top Authors** — styled HTML `<table>` with avatar placeholder (first-letter circle) and count.

6. **Issues Needing Attention**
   - Collapsible `<details>` sections for:
     - Stale issues (list up to 20)
     - Unlabeled issues (list up to 20)
   - Each item links to the issue URL.

7. **Recommendations** — 3–5 bullet-point insights derived from the data.

8. **Footer** — generation timestamp, repository link, workflow run link.

### CSS-Only Chart Techniques

**Horizontal bar chart (clusters, labels)**:
```css
.bar {
  height: 28px;
  border-radius: 4px;
  /* width set via inline style: style="width: 72%" */
  transition: width 0.3s ease;
}
```

**Donut chart (open vs closed)**:
```css
.donut {
  width: 120px; height: 120px;
  border-radius: 50%;
  /* Percentages injected as inline style */
  background: conic-gradient(
    var(--green) 0% var(--open-pct),
    var(--red)   var(--open-pct) 100%
  );
  mask: radial-gradient(circle, transparent 55%, black 56%);
  -webkit-mask: radial-gradient(circle, transparent 55%, black 56%);
}
```

**Sparkline (activity trend)**:
Use an inline `<svg>` with `<polyline>` — no JS needed. Compute the `points` attribute from the daily arrays.

### OGP Meta Tags

Add these `<meta>` tags in `<head>`:

```html
<meta property="og:title"       content="Daily Issues Report — YYYY-MM-DD — OWNER/REPO" />
<meta property="og:description" content="X open issues, Y closed, Z opened this week" />
<meta property="og:image"       content="og-image.svg" />
<meta property="og:type"        content="website" />
<meta property="og:url"         content="https://OWNER.github.io/REPO/" />
<meta name="twitter:card"       content="summary_large_image" />
```

Replace placeholders with actual computed values from the metrics.

---

## Phase 3: Generate SVG Ticker

Write `/tmp/gh-aw/report-site/ticker.svg` — a standalone SVG that displays a scrolling marquee of key stats. This ticker is also embedded inline in the HTML header.

The ticker should be a horizontal strip (e.g. `viewBox="0 0 1200 40"`) containing:
- Metric labels and values as `<text>` elements spaced evenly
- A `<animateTransform>` for a smooth left-scroll loop (CSS `@keyframes` inside `<style>` within the SVG is also acceptable)
- Metrics to show: Total Issues · Open · Closed · Opened 7d · Stale · Unlabeled

Example structure:
```xml
<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1200 40">
  <style>
    .ticker-text { font: 600 14px system-ui, sans-serif; fill: #e6edf3; }
    .ticker-label { fill: #7d8590; }
    @keyframes scroll { from { transform: translateX(0); } to { transform: translateX(-50%); } }
    .ticker-group { animation: scroll 20s linear infinite; }
  </style>
  <g class="ticker-group">
    <!-- duplicated content for seamless loop -->
    <text x="0"   y="26" class="ticker-label">Total</text>
    <text x="40"  y="26" class="ticker-text">542</text>
    <text x="100" y="26" class="ticker-label">Open</text>
    <text x="140" y="26" class="ticker-text">128</text>
    <!-- ... repeat, then duplicate the whole set offset by half-width ... -->
  </g>
</svg>
```

---

## Phase 4: Generate OGP Image

Write `/tmp/gh-aw/report-site/og-image.svg` — a 1200 × 630 SVG suitable for social-media previews.

Design:
- Dark background (`#0d1117`), rounded rect border (`#30363d`).
- Large title: "Daily Issues Report".
- Subtitle: repository name and date.
- Three large metric boxes in a row: **Open**, **Closed**, **Opened 7d**, each with a big number and a label.
- A mini bar chart showing the top 5 clusters.
- Small footer text: "Generated by GitHub Actions".

Upload this SVG as an asset using the `upload asset` tool and use the returned URL for the `og:image` meta tag.

---

## Phase 5: Generate Sitemap

Write `/tmp/gh-aw/report-site/sitemap.xml`:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url>
    <loc>https://OWNER.github.io/REPO/</loc>
    <lastmod>YYYY-MM-DD</lastmod>
    <changefreq>daily</changefreq>
    <priority>1.0</priority>
  </url>
</urlset>
```

Replace `OWNER`, `REPO`, and `YYYY-MM-DD` with actual values from context.

---

## Phase 6: Upload OGP Image Asset

Use the `upload asset` tool to upload `/tmp/gh-aw/report-site/og-image.svg` (or render it to PNG first if the upload tool only accepts images). Collect the returned URL and patch the `og:image` meta tag in `index.html` to use the absolute URL.

---

## File Manifest

When finished, `/tmp/gh-aw/report-site/` must contain at minimum:

| File | Purpose |
|------|---------|
| `index.html` | Full dashboard (single self-contained file, all CSS inlined) |
| `og-image.svg` | Open Graph preview image (1200 × 630) |
| `ticker.svg` | Standalone scrolling ticker strip |
| `sitemap.xml` | Sitemap for search engines |
| `data.json` | Raw metrics extracted from issues (for transparency) |

The post-agent workflow steps will automatically upload this directory as a GitHub Pages artifact and deploy it.

---

## Important Guidelines

### Data Quality
- Handle missing fields gracefully (null checks in `jq` filters)
- Validate date formats before processing
- Skip malformed issues rather than failing
- If the issues data file is empty or missing, generate a placeholder page

### HTML Quality
- Valid HTML5 (`<!DOCTYPE html>`)
- All CSS in a single `<style>` block — no inline `style` attributes except for data-driven values (bar widths, conic-gradient percentages, polyline points)
- No `<script>` tags anywhere
- Accessible: proper heading hierarchy, `alt` text on SVGs, ARIA labels on charts, sufficient color contrast
- Responsive: readable on mobile (320px) through desktop (1440px+)

### Chart Quality
- CSS-only bar charts with smooth `border-radius` and subtle hover states (`:hover` for bar highlight)
- Donut chart via `conic-gradient` with mask for the center hole
- SVG polyline sparklines with clean axis labels
- Consistent color palette using the design tokens above
- All charts must be readable without JavaScript

### SVG Quality
- Valid SVG 1.1 namespace
- Text elements use `system-ui, sans-serif` for cross-platform rendering
- Animations use CSS `@keyframes` inside `<style>` (not SMIL) for broader compatibility
- OGP image must render correctly at 1200 × 630 in social-media previews

---

## Success Criteria

A successful run will:
- ✅ Extract and process all available issues data with `jq`
- ✅ Generate `index.html` with a professional dark-themed dashboard
- ✅ Include CSS-only bar charts, donut chart, and SVG sparkline — no JS
- ✅ Generate `ticker.svg` with scrolling metrics
- ✅ Generate `og-image.svg` (1200 × 630) for social previews
- ✅ Include OGP `<meta>` tags in `index.html`
- ✅ Generate `sitemap.xml` with correct URLs
- ✅ Write all files to `/tmp/gh-aw/report-site/`
- ✅ Upload the OGP image as an asset and update the meta tag with the absolute URL

Begin now. Extract the data, build the HTML, generate the SVGs, write the sitemap, and place everything in `/tmp/gh-aw/report-site/`.
