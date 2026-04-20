# Agent Performance — 2026-04-20
**Run:** 2026-04-20T23:44Z

## Snapshot
- 55 workflows registered, all Copilot; repo-assist NOT compiled (⚠️)
- 27 open issues (↑ from 23 Apr 19)
- 6 open PRs (3 drafts, 2 ready, 1 release) — ↓ from ~19 Apr 19 ✅ (major improvement)
- Recent PR merge rate: 18/20 = 90% for high-numbered recent PRs
- [aw] Failures: 5 active (↑ from 3 Apr 19) — 2 new failures

## Top Performers
- **glossary-maintainer**: #2427 clean PR, right output type. Score: 90/100
- **discussion-task-miner**: #2400,2399,2398,2397,2396 — 5 actionable task issues. Score: 85/100
- **repo-quality-improver**: #2429 detailed Swagger analysis. Score: 78/100
- **tech-content-editorial-board**: #2422 PR + #2421 editorial issue. Score: 78/100
- **daily-doc-updater**: No new duplicate PRs today (improved from 3 dupes Apr 19). Score: 72/100

## Agents Needing Improvement
- **repo-assist**: NOT COMPILED + FAILED (#2389). Score: 15/100 — CRITICAL
- **daily-accessibility-review**: FAILED (#2390). Score: 30/100
- **issue-arborist**: FAILED (#2405). Score: 30/100
- **repo-status / daily-team-status / daily-plan / daily-repo-chronicle / daily-weekly-repo-map**: Creating issues for status content (wrong output type). Score: 45-50/100
- **portfolio-analyst**: #2424 + #2404 both "Monthly Activity 2026-04" — possible duplication. Score: 50/100

## Critical Issues
### 1. CRITICAL: repo-assist NOT COMPILED + FAILED
- #2389: [aw] Repo Assist failed
- Workflow not compiled — lock file may be stale
- Requires immediate fix

### 2. HIGH: [aw] Failure Cluster — 5 active (↑ from 3)
- NEW: #2405 Issue Arborist failed
- NEW: #2389 Repo Assist failed (not compiled)
- PERSISTENT: #2390 Daily Accessibility Review failed
- STALE: #2328 Detection Runs
- VERY STALE: #1733 No-Op Runs (candidate for closure)

### 3. HIGH: Status Content as Issues
- 5 agents creating status reports as issues (wrong output type):
  #2428 (repo-status), #2420 (team-status), #2419 (daily-plan), #2417 (repo-chronicle), #2416 (repo-map)
- Previous report noted this for repo-status only (1); now spread to 5 agents

### 4. MEDIUM: Draft PRs Accumulating (3 open)
- #2418, #2423, #2425 all DRAFT from amalgamated-bot
- Drafts may not receive review/merge attention

### 5. MEDIUM: Monthly Activity Duplication
- #2424 "perf: Monthly Activity 2026-04" + #2404 "test: Monthly Activity 2026-04" 
- Two issues for same period from different agents (daily-qa?)

## Improvements vs Yesterday
- PR count: ~19 → 6 (↓ -68%) — major reduction, most merged ✅
- PR merge rate: ~87% → ~90% ✅
- daily-doc-updater: 0 duplicate PRs today (vs 3 yesterday) ✅
- veverkananobot issues addressed/closed ✅

## Discussions Created
"Agent Performance Report — Week of 2026-04-20" in Audits category
