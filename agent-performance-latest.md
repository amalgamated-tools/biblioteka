# Agent Performance — 2026-04-19
**Run:** 2026-04-19T23:44Z

## Snapshot
- 54 workflows registered, all Copilot, all compiled; ~23 active daily
- 23 open issues (↓ from 50 Apr 18 — accumulation trend REVERSED)
- 19 open PRs (13 Copilot, 5 amalgamated-bot, 1 veverkap)
- Recent 30 PRs: 26 merged = 87% merge rate (up from 84% historical)

## Top Performers
- **Copilot (on-demand)**: 13 open PRs + 16 merged in last 30. High quality. Score: 92/100
- **duplicate-code-detector**: #2383 precise analysis. Score: 88/100
- **discussion-task-miner**: #2337–#2341 actionable tasks mined. Score: 85/100
- **amalgamated-bot (general)**: Docs, perf, test PRs merged cleanly. Score: 83/100

## Agents Needing Improvement
- **daily-doc-updater**: 3 PRs with identical title docs(books)/#2363/#2334/#2355, 2 closed unmerged. Score: 55/100
- **veverkananobot**: 6 issues still open from Apr 18; some addressed by Copilot PRs now. Score: 65/100
- **daily-workflow-updater**: Failed (#2346); fix PR #2376 open. Score: 30/100
- **repo-status**: Still creating daily status as issues (#2379). Score: 45/100

## Critical Issues
### 1. HIGH: daily-doc-updater Duplicate PRs
- #2363 + #2334 + #2355 all "docs(books): document tags field in book detail object"
- Two closed unmerged; one merged (#2363 with expanded scope)
- Root cause: daily-doc-updater running before previous PRs are merged/closed

### 2. HIGH: [aw] Failures (3 open)
- #2346: Daily Workflow Updater failed (new Apr 19) — fix PR #2376 open
- #2328: Detection Runs (persistent, stale)
- #1733: No-Op Runs (very stale, Apr 15 origin)

### 3. MEDIUM: Vague PR Title
- #2329 [Copilot]: "Completing task" — closed unmerged; poor title quality

### 4. MEDIUM: Ephemeral Issues (not Discussions)
- #2379: [repo-status] Daily Status Report as issue (should be discussion)
- Improved: only 1 today vs 6 yesterday

## Good News vs Yesterday
- Issue count: 50 → 23 (↓ -54%) — major improvement ✅
- veverkananobot issues being addressed by Copilot PRs ✅
- PR merge rate held strong: 87% ✅
- Duplicate creation issue-pairs resolved (no new dupes today) ✅

## Discussions Created
"Agent Performance Report — Week of 2026-04-19" in Audits category
