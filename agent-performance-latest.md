# Agent Performance 2026-05-20

**Run**: agent-performance-analyzer | 2026-05-20T13:46Z

## Summary
- 18 distinct workflows analyzed
- Active agents with success today: Agentic Maintenance (5/5), Code Simplifier (1/1), Dead Code Removal (1/1), Daily Malicious Code Scan (1/1), Contribution Check (4/4), Daily Doc Updater (1/1 — RECOVERED), Duplicate Code Detector (1/1)
- PR-triggered skip-only agents: Q, PR CQR, CLA — expected/normal
- vs prior report (2026-05-19):
  - UNCHANGED CRITICAL: Architecture Guardian — now 11+ days offline (May 9-19); issue #3033 OPEN (1d), needs manual fix
  - RECOVERED: Daily Documentation Updater — 1/1 success today; issues #3031 (yesterday's fail) resolved; over-creation issue #2986 still open
  - STABLE FRAGILE: Daily Documentation Healer — 1/1 success again; #3022 still open; await 3+ clean days
  - IMPROVING: Daily Caveman Optimizer — no new PR today (was 2/day yesterday); PR #3028 still unmerged
  - NEW PATTERN: Testify Expert — 3 issues in 3 days (#3021, #3029, #3038); repetition without dedup
  - STABLE: Agentic Maintenance 5/5 (100%), Contribution Check 4/4 (100%)
  - STALE DRAFT: Dead Code Removal PR #3025 — 2+ days as DRAFT

## Pattern Detector Results (2026-05-20)
- over_creation: 0 (improvement: Caveman Optimizer no new PR today)
- repetition: 1 (Testify Expert — 3 consecutive daily issues, same pattern)
- under_creation: 2 (Architecture Guardian, Code Refiner)
- inconsistency: 2 (Daily Doc Healer fragile, Code Refiner cancelled)
- inactive: 1 (Architecture Guardian)
- scope_creep: 0
- clean: 12 agents

## Quality Scores (this run)
- Average quality: 61/100 (↑2 from yesterday)
- Average effectiveness: 57/100 (↑2 from yesterday)
- PR merge rate: ~84% (stable)
- 6 behavioral patterns detected across 4 agents (↓2 from yesterday — Caveman improving)

## Top Performers
- **Agentic Maintenance**: 5/5 (100%) — top performer
- **Contribution Check**: 4/4 (100%)
- **Code Simplifier**: 5-day perfect record

## Underperformers
- **Architecture Guardian**: OFFLINE 11+ days (#3033, 1d open) — CRITICAL, no change
- **Testify Expert**: repetition (#3021, #3029, #3038) — needs dedup guard
- **Daily Doc Healer**: fragile (#3022 still open); do not close until 3+ clean days

## Issues Tracked
- #2956: No-Op Runs tracker — OPEN (expected/normal)
- #2986: Daily Doc Updater over-creation — OPEN (monitor; today's run OK but pattern still exists)
- #2995: Issue Group — OPEN (this report's parent)
- #3022: Daily Doc Healer regression — OPEN (monitor; 2nd clean day today)
- #3025: Dead Code Removal PR — DRAFT 2d (needs merge or promotion)
- #3028: Caveman Optimizer PR — OPEN 2d (needs merge)
- #3029: Testify Expert ssrf — OPEN
- #3033: Architecture Guardian failure — OPEN (CRITICAL)
- #3038: Testify Expert sync_token — OPEN (new today)

## Report Posted
- Comment on #2995 (Issue Group)
