# Agent Performance — 2026-04-18
**Run:** 2026-04-18T23:44Z

## Snapshot
- 54 workflows registered, all Copilot, all compiled; ~23 active daily
- 50 open issues (↑ from 40 Apr 17, ↑ from 29 Apr 16) — +10/day accumulation trend continues
- 28/30 PRs merged today (Apr 18) = 93% same-day merge rate (up from 84% historical)
- Copilot: 21 PRs, amalgamated-bot: 7, veverkap: 2

## Top Performers
- **veverkananobot (daily-nitpick/grumpy-reviewer)**: 23 open issues today; specific conventional commit format; Score: 85/100 (high quality, but duplicate creation pattern detected)
- **Copilot (on-demand)**: 21/28 merged PRs today; feat+fix+docs mix; Score: 95/100
- **daily-doc-updater (amalgamated-bot)**: #2237, #2236, #2203, #2190 merged; consistent doc updates; Score: 90/100
- **duplicate-code-detector (amalgamated-bot)**: #2280 - precise structural analysis; Score: 88/100
- **discussion-task-miner (amalgamated-bot)**: #2220, #2219, #2218 actionable tasks; Score: 87/100
- **repo-assist (amalgamated-bot)**: #2225 perf PR merged; Score: 87/100
- **daily-perf-improver**: Monthly rollup useful; Score: 82/100

## Critical Issues

### 1. CRITICAL: veverkananobot Duplicate Issue Creation
Three confirmed duplicate pairs in open issues today:
- Auth.svelte console.error: #2250 AND #2263 (identical fix)
- Recommendations.svelte console.error: #2249 AND #2262 (identical fix)
- book_annotations dead code: #2242 AND #2259 (same fix)
- ISO 8601 timestamps: #2243-#2246 (per-file granular) vs #2260 (rollup)
- WORSE: PR #2199 "fix(frontend): remove redundant console.error calls" was merged today — yet issues #2249 and #2262 still open requesting the same fix. Agent not checking merged PRs.
- Root cause: daily-nitpick-reviewer AND daily-grumpy-reviewer both running → double-creating similar issues; no dedup check for already-merged PRs.

### 2. HIGH: [aw] Failure Issues — Stagnant (6 open)
- #2264 NEW: [aw] Agentic Triage failed (Apr 18)
- #2239: [aw] CI Failure Doctor failed (persistent)
- #2235: [aw] Update Docs failed (persistent)
- #2233: [aw] Contribution Guidelines Checker failed (persistent)
- #2214, #1733: Detection Runs / No-Op Runs (stale, never resolved)

### 3. HIGH: Issue Accumulation Accelerating
- Open issues: 18 → 29 → 40 → 50 in 3 days (+10/day)
- 6 ephemeral status issues still open (daily-plan, repo-status, team-status, perf-improver monthly)
- Fix: status reports should switch to discussions, not issues

## PRs not merged today
- #2230: docs(claude): correct delete helper location — still open (duplicate of #2231?)
- #2207: feat(frontend): add Reading Groups API client module — still open

## Discussion Created
"Agent Performance Report — Week of 2026-04-18" in Audits category
