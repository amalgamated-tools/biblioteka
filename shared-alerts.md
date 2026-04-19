# Shared Alerts
**Updated:** 2026-04-19T23:44Z by agent-performance-analyzer

## Active Alerts

### HIGH: daily-doc-updater Duplicate PRs (Persisting)
- #2363 (merged) + #2355 (closed) + #2334 (closed) — all same docs(books) change
- Pattern: doc-updater creates new PR before previous is merged
- Fix: Add `skip-if-match: is:pr is:open in:title "docs(books)"` guard

### HIGH: [aw] Failure Cluster (3 active, ↓ from 5 yesterday)
- **daily-workflow-updater** (#2346): NEW Apr 19 — fix PR #2376 open
- **Detection Runs** (#2328): Persistent/stale
- **No-Op Runs** (#1733): Very stale — candidate for closure
- Resolved since Apr 18: sergo, dependabot-bundler failures ✅

### MEDIUM: Draft PRs Awaiting Review (3)
- #2386: docs(database) update index names — DRAFT
- #2384: refactor(db) deduplicate reading list ownership — DRAFT
- #2382: docs INITIAL_ADMIN bootstrap — DRAFT (duplicate of #2377?)

### MEDIUM: repo-status Using Issues (not Discussions)
- #2379: [repo-status] Daily Status as issue (should be discussion)
- Improvement: only 1 today vs 6 yesterday ✅

### LOW: veverkananobot Issues from Apr 18 (6 open)
- Being addressed by Copilot PRs (#2371, #2372, #2370, #2369, #2373)
- Pipeline is working; expect resolution within 24h

## Resolved Since Apr 18
- Issue accumulation reversed: 50 → 23 open ✅
- Duplicate issue pairs (same-file dupes): 0 new today ✅  
- sergo + dependabot-bundler [aw] failures closed ✅
- PR velocity: 87% merge rate ✅

## For Campaign Manager
- Strong execution: 26/30 recent PRs merged (87%)
- Major features: LLM config UI (#2331 merged), Apply All (#2330 merged)
- 3 draft PRs blocking: review needed for #2384, #2386, #2382
- Task miner pipeline producing actionable issues → Copilot PRs

## For Workflow Health Manager
- daily-workflow-updater still failing (#2346) — fix PR #2376 in review
- daily-doc-updater needs skip-if-match guard to prevent duplicate PRs
- #1733 No-Op Runs very stale — recommend closing as resolved
