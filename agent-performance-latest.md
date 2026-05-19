# Agent Performance 2026-05-19

**Run**: agent-performance-analyzer | 2026-05-19T13:53Z

## Summary
- 17 distinct workflows observed
- Active agents with success today: Agentic Maintenance (6/6), Code Simplifier (1/1), Contribution Check (3/3), Dead Code Removal (1/1), Daily Malicious Code Scan (1/1), Daily Testify Expert (1/1), Daily Documentation Healer (1/1 — RECOVERED), Duplicate Code Detector (1/1), Typist (1/1), Go Fan (1/1), Caveman Optimizer (1/1)
- PR-triggered skip-only agents: Q(17), PR CQR(11), Mergefest, CLA — expected; tracked in #2956
- vs prior report (2026-05-18):
  - ESCALATED CRITICAL: Architecture Guardian — now 10+ days offline (was 7 days), issue #2981 now 8+ days unresolved
  - ONGOING HIGH: Code Refiner — 2/2 cancelled today; 7+ day systemic cancellation pattern
  - NEW FAILURE: Daily Documentation Updater — FAILED today (issue #3031); ALSO ongoing over-creation (#2986); dual problem
  - RECOVERED: Daily Documentation Healer — 1/1 success today (was failed yesterday #3022); issues #2972 and #3022 still open
  - NEW PATTERN: Daily Caveman Optimizer — over_creation + repetition (PR #3018 unmerged + PR #3028 created today)
  - STABLE: Agentic Maintenance 6/6 (100%)
  - STABLE: Contribution Check 3/3 (100%)
  - IMPROVING: PR Triage Agent — 1/2 (50%, stable improving)

## Pattern Detector Results (2026-05-19)
- over_creation: 2 (Daily Doc Updater, Daily Caveman Optimizer)
- repetition: 2 (Daily Testify Expert, Daily Caveman Optimizer)
- under_creation: 2 (Architecture Guardian, Code Refiner)
- inconsistency: 3 (Code Refiner, Daily Doc Healer, PR Triage)
- inactive: 1 (Architecture Guardian)
- scope_creep: 0
- clean: 10 agents

## Quality Scores (this run)
- Average quality: 59/100 (↓2 from yesterday)
- Average effectiveness: 55/100 (↓2 from yesterday)
- PR merge rate: ~84% (stable baseline)
- 8 behavioral patterns detected across 5 agents (↑1 agent from yesterday — Caveman Optimizer newly flagged)

## Top Performers
- **Agentic Maintenance**: 6/6 (100%) — top performer
- **Contribution Check**: 3/3 (100%)
- **Dead Code Removal**: PR #3025 (8 functions removed)

## Underperformers
- **Architecture Guardian**: OFFLINE 10+ days (#2981, 8+ days open) — CRITICAL ESCALATION
- **Daily Documentation Updater**: NEWLY FAILING (#3031) + over-creation (#2986) — dual failure
- **Code Refiner**: 0 outputs (2/2 cancelled) — systemic
- **Daily Caveman Optimizer**: 2 unmerged PRs stacking (#3018 + #3028) — NEW FLAG

## Issues Tracked
- #2956: No-Op Runs tracker — OPEN (expected/normal)
- #2972: Daily Doc Healer prior failure — OPEN; monitor stability (3+ clean days before closing)
- #2981: Architecture Guardian failure — OPEN (8+ days, CRITICAL)
- #2986: Daily Doc Updater over-creation — OPEN
- #2995: Issue Group — OPEN (this report's parent)
- #3019: Testify Expert pathparser (prior duplicate) — OPEN
- #3021: Testify Expert enrich_goodreads — OPEN
- #3022: Daily Doc Healer regression — OPEN
- #3031: Daily Doc Updater NEW FAILURE — OPEN

## Report Posted
- Comment on #2995 (Issue Group)
