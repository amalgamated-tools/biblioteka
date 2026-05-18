# Agent Performance 2026-05-18

**Run**: agent-performance-analyzer | 2026-05-18T14:07Z

## Summary
- 17 distinct workflows observed (from 80 most recent runs)
- Active agents with success today: Agentic Maintenance (5/5), Code Simplifier (1/1), Contribution Check (3/3), Dead Code Removal (1/1), Daily Malicious Code Scan (1/1), Daily Testify Expert (1/1), Daily Documentation Updater (1/1)
- PR-triggered skip-only agents: Q(17), PR CQR(11), Mergefest(10), CLA(10/11) — expected; tracked in #2956
- vs prior report (2026-05-17):
  - ONGOING CRITICAL: Architecture Guardian — 5+ consecutive failures, issue #2981 now 7 days unresolved
  - ONGOING HIGH: Code Refiner — 2/2 skipped today; systemic instability continues
  - NEW REGRESSION: Daily Documentation Healer — failed today (issue #3022) after 7+ clean days; prior recommendation to close #2972 now on hold
  - NEW: Test CI failure today (run 26032697105)
  - ONGOING MED: Daily Documentation Updater over-creation (~16 PRs/run, #2986 open)
  - STABLE GOOD: Testify Expert — today's #3021 is NEW file (enrich_goodreads_test.go), NOT a duplicate today; prior duplication pattern remains flagged
  - IMPROVING: PR Triage Agent — 1/2 success (stable improving)
  - STABLE: Agentic Maintenance 5/5 (100%)
  - STABLE: Contribution Check 3/3 (100%)
  - LOW: Metrics Collector stale (last update 2026-04-04, 44 days ago)

## Top Performers
- **Agentic Maintenance**: 5/5 (100%) — consistently top performer
- **Contribution Check**: 3/3 (100%) — STABLE
- **Code Simplifier, Dead Code Removal, Daily Malicious Code Scan**: 1/1 each

## Underperformers
- **Architecture Guardian**: 0 runs, #2981 open 7 days — CRITICAL
- **Code Refiner**: 0 outputs (2/2 skipped) — systemic instability
- **Daily Documentation Healer**: REGRESSION — failed today (#3022); prior recovery streak broken
- **Daily Documentation Updater**: ~16 PRs/run over-creation (#2986)

## Quality Scores (this run)
- Average quality: 61/100 (↓2)
- Average effectiveness: 57/100 (↓2)
- PR merge rate: ~84% (stable)
- 7 behavioral patterns detected across 6 agents

## Behavioral Patterns Detected
- over-creation: 1 (Daily Doc Updater)
- repetition: 1 historical (Daily Testify Expert — NOT today, but prior consecutive-day duplication confirmed)
- scope-creep: 1 (Doc Healer self-reporting)
- under-creation/inactive: 3 (Architecture Guardian, Code Refiner, Doc Auto-Merge stalled)
- inconsistency: 4 (Architecture Guardian, Doc Healer, Testify Expert, PR Triage)
- clean: 9 agents

## Issues Tracked
- #2956: No-Op Runs tracker — OPEN (expected/normal)
- #2972: Daily Doc Healer prior failure — OPEN; do NOT close (regression today)
- #2981: Architecture Guardian failure — OPEN (7 days, CRITICAL)
- #2986: Daily Doc Updater over-creation — OPEN
- #2995: Issue Group — OPEN (this report's parent)
- #3019: Testify Expert pathparser (yesterday's duplicate) — OPEN
- #3021: Testify Expert enrich_goodreads (today, new file) — OPEN
- #3022: Daily Doc Healer regression — OPEN (NEW TODAY)

## Report Posted
- Comment on #2995 (Issue Group)
