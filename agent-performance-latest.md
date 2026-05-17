# Agent Performance 2026-05-17

**Run**: agent-performance-analyzer | 2026-05-17T13:21Z

## Summary
- 14 distinct workflows observed (from 80 most recent runs)
- Active agents with success today: Agentic Maintenance (5/5), Code Simplifier (1/1), Contribution Check (3/3), Dead Code Removal (1/1), Daily Documentation Updater (1/1), Daily Documentation Healer (1/1), Daily Malicious Code Scan (1/1), Daily Testify Uber Super Expert (1/1)
- PR-triggered skip-only agents: Q(18), PR CQR(17), Mergefest(15), CLA(13) — expected; tracked in #2956
- vs prior report (2026-05-16):
  - ONGOING CRITICAL: Architecture Guardian — still 0 runs, issue #2981 now 8+ days unresolved (was opened 2026-05-11)
  - ONGOING HIGH: Code Refiner — absent entirely from runs (was cancelling); systemic instability continues
  - CRITICAL NEW: Daily Testify Expert — created #3015 (2026-05-16) AND #3019 (2026-05-17) for SAME file (internal/pathparser/pathparser_test.go) — confirmed duplication; existing issues #3006, #3012 unresolved
  - ONGOING MED: Daily Documentation Updater over-creation (~16 PRs/run, #2986 open)
  - IMPROVING: PR Triage Agent — 1/2 success (stable at improving trajectory from 0/1)
  - RECOMMEND CLOSE: Daily Documentation Healer — 7+ consecutive clean days; issue #2972 should be closed
  - STABLE: Agentic Maintenance 5/5 (100%)
  - STABLE: Contribution Check 3/3 (100%)
  - LOW: Metrics Collector stale (last update 2026-04-04, 43 days ago)

## Top Performers
- **Agentic Maintenance**: 5/5 (100%) — consistently top performer
- **Contribution Check**: 3/3 (100%) — STABLE
- **Code Simplifier, Dead Code Removal, Daily Malicious Code Scan, Daily Doc Healer**: 1/1 each

## Underperformers
- **Architecture Guardian**: 0 runs, #2981 open 8 days — CRITICAL
- **Code Refiner**: 0 runs (previously cancelling) — systemic instability
- **Daily Testify Expert**: confirmed duplication (same file, consecutive days); issues #3006, #3012, #3015, #3019
- **Daily Documentation Updater**: ~16 PRs/run over-creation (#2986)

## Quality Scores (this run)
- Average quality: 63/100 (stable)
- Average effectiveness: 59/100 (↑1)
- PR merge rate: ~84% (stable)
- 10+ behavioral patterns detected across 6 agents

## Behavioral Patterns Detected
- over-creation: 1 (Daily Doc Updater)
- repetition/duplication: 2 (Daily Testify Expert — same file on consecutive days)
- scope-creep: 1 (Doc Updater Auto-Merge)
- under-creation/inactive: 2 (Architecture Guardian, Code Refiner)
- inconsistency: 1 (PR Triage)
- clean: 9 agents

## Issues Tracked
- #2956: No-Op Runs tracker — OPEN (expected/normal)
- #2972: Daily Doc Healer failure — OPEN but RECOVERED (7+ clean days, RECOMMEND CLOSE)
- #2981: Architecture Guardian failure — OPEN (8 days, CRITICAL)
- #2986: Daily Doc Updater over-creation — OPEN
- #2995: Issue Group — OPEN (this report's parent)
- #3006/#3012/#3015/#3019: Testify Expert outputs (confirmed duplication pattern)

## Report Posted
- Comment on #2995 (Issue Group)
