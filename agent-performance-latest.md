# Agent Performance 2026-05-16

**Run**: agent-performance-analyzer | 2026-05-16T13:21Z

## Summary
- 19 distinct workflows analyzed (from 50 most recent runs)
- Active agents with success today: Agentic Maintenance (5/5), Code Simplifier (1/1), Contribution Check (2/2), Dead Code Removal (1/1), Daily Documentation Updater (1/1), Daily Documentation Healer (1/1), Daily Malicious Code Scan (1/1), Daily Testify Uber Super Expert (1/1), PR Triage Agent (1/2 so far)
- PR-triggered skip-only agents: Q(10), PR CQR(10), Mergefest(10), CLA(8) — expected; tracked in #2956
- vs prior report (2026-05-15):
  - ONGOING CRITICAL: Architecture Guardian — still 0 runs, issue #2981 now 7+ days unresolved
  - ONGOING HIGH: Code Refiner — 2 cancelled runs (systemic); no issue filed yet
  - ONGOING HIGH: Daily Testify Expert — repetition + scope-creep patterns persist; new issues #3006, #3012, #3015
  - ONGOING MED: Daily Documentation Updater over-creation (~16 PRs/run, #2986)
  - STABLE/IMPROVED: PR Triage Agent — 1/2 success (improving; was 0/1 yesterday)
  - RECOMMEND CLOSE: Daily Documentation Healer — now 6+ clean days; issue #2972 should be closed
  - STABLE: Agentic Maintenance 5/5 (100%)
  - STABLE: Contribution Check 2/2 (100%)
  - LOW: Metrics Collector stale (last update 2026-04-04, 42 days ago)

## Top Performers
- **Agentic Maintenance**: 5/5 (100%) — consistently top performer
- **Contribution Check**: 2/2 (100%) — STABLE
- **Code Simplifier, Dead Code Removal, Daily Malicious Code Scan**: 1/1 each

## Underperformers
- **Architecture Guardian**: 0 runs, #2981 open 7+ days — CRITICAL
- **Code Refiner**: 0% (2 cancels today) — systemic instability
- **Daily Testify Expert**: repetition + scope-creep; issues #3006, #3012, #3015
- **Daily Documentation Updater**: ~16 PRs/run over-creation (#2986)
- **Doc Updater Auto-Merge**: stalled (blocked by PR flood upstream)

## Quality Scores (this run)
- Average quality: 63/100 (stable vs yesterday)
- Average effectiveness: 58/100 (↑1)
- PR merge rate: ~84% (stable)
- 10 behavioral patterns detected across 7 agents

## Behavioral Patterns Detected
- over-creation: 1 (Daily Doc Updater)
- repetition: 1 (Daily Testify Expert)
- scope-creep: 2 (Daily Testify Expert, Doc Updater Auto-Merge)
- under-creation: 1 (Architecture Guardian)
- inconsistency: 3 (Code Refiner, Doc Updater Auto-Merge, PR Triage)
- clean: 9 agents

## Issues Tracked
- #2956: No-Op Runs tracker — OPEN (expected/normal)
- #2972: Daily Doc Healer failure — OPEN but RECOVERED (6+ clean days, RECOMMEND CLOSE)
- #2981: Architecture Guardian failure — OPEN (7+ days, CRITICAL)
- #2986: Daily Doc Updater over-creation — OPEN
- #2995: Issue Group — OPEN (this report's parent)
- #3006/#3012/#3015: Testify Expert outputs (scope-creep pattern)

## Report Posted
- Comment on #2995 (Issue Group)
