# Agent Performance 2026-05-23

**Run**: agent-performance-analyzer | 2026-05-23T13:21Z

## Summary
- 14 workflows analyzed (10 active, 4 event-triggered skip-only)
- Active agents successful today: Agentic Maintenance (6/6), Contribution Check (3/3), Code Simplifier (1/1), Daily Caveman Optimizer (1/1), Daily Documentation Healer (1/1), Daily Documentation Updater (1/1), Daily Malicious Code Scan (1/1), Daily Testify Uber Super Expert (1/1), Dead Code Removal (1/1), Duplicate Code Detector (1/1)
- vs prior report (2026-05-22):
  - UNCHANGED CRITICAL: Architecture Guardian — now 14+ days offline; issue #3033 OPEN
  - IMPROVED: Testify Expert — #3056 today has proper title/body (partial recovery from #3046 regression); #3052 regression issue remains OPEN; repetition pattern persists (same pathparser_test.go target)
  - RESOLVED WATCH: Go Fan — no new issue created today (deadline day 2026-05-24); 3 of 5 recent issues now closed; pattern may be settling
  - NEW FLAG: Function Namer — over-creation + scope-creep confirmed (3/3 recent issues auto-closed as wontfix)
  - NEW FLAG: Daily Caveman Optimizer — over-creation (3 PRs this week, 0 merged in current batch)
  - STABLE: Daily Documentation Healer (4+ clean days), Daily Documentation Updater (recovered), Agentic Maintenance (6/6), Contribution Check (3/3)
  - STALE DATA: Metrics Collector last update 2026-04-04 (49 days ago)

## Pattern Detector Results (2026-05-23)
- over_creation: 3 (Go Fan, Daily Caveman Optimizer, Function Namer)
- repetition: 1 (Testify Expert — pathparser_test.go repeated target)
- inconsistency: 2 (Testify Expert — #3046 regression; PR Triage — 50% success rate)
- under_creation: 2 (Architecture Guardian — offline; Code Simplifier — 0 outputs)
- scope_creep: 1 (Function Namer — issues consistently rejected as wontfix)
- clean: 7 agents

## Quality Scores (this run)
- Average quality: 63/100 (↑2 from yesterday — Testify partial recovery)
- Average effectiveness: 61/100 (↑1 from yesterday)
- PR merge rate: ~84% (stable, historical baseline from metrics)
- 9 behavioral patterns detected across 5 agents (↑2 from yesterday — Function Namer + Caveman newly classified)

## Top Performers
- **Agentic Maintenance**: 6/6 (100%) — backbone agent
- **Duplicate Code Detector**: 2 actionable issues this week, both triggered downstream Copilot PRs (#3037, #3055)
- **Contribution Check**: 3/3 clean reports; all structured and closed after review
- **Dead Code Removal**: quality PR (#3025), consistent

## Underperformers
- **Architecture Guardian**: OFFLINE 14+ days (#3033, CRITICAL, still open)
- **Function Namer**: 3/3 recent issues auto-closed as wontfix — scope-creep + over-creation
- **Daily Caveman Optimizer**: 3 PRs this week, 0 merged — over-production
- **Testify Expert**: repetition on pathparser_test.go + prior regression (#3046); partial recovery today
- **Code Simplifier**: 0 outputs despite successful run (2nd consecutive day)

## Issues Tracked
- #2995: Issue Group — OPEN (this report's parent)
- #2956: No-Op Runs tracker — OPEN (expected/normal)
- #2328: Detection Runs — OPEN (expected/normal)
- #3022: Daily Doc Healer — OPEN; still recommend CLOSE (5+ clean days)
- #3031: Daily Doc Updater — OPEN; recommend CLOSE (recovered)
- #3033: Architecture Guardian failure — OPEN (CRITICAL)
- #3045: Contribution Check failure — OPEN (one-off engine failure, monitor)
- #3052: Testify Expert regression (#3046) — OPEN (active investigation)
- #3056: Testify Expert #3056 (pathparser) — today's, quality OK

## Report Posted
- Comment on #2995 (Issue Group)
