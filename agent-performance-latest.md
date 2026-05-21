# Agent Performance 2026-05-21

**Run**: agent-performance-analyzer | 2026-05-21T13:54Z

## Summary
- 16 workflows analyzed (12 active, 4 event-triggered skip-only)
- Active agents with success today: Agentic Maintenance (4/4), Contribution Check (3/3), Code Simplifier (1/1), Typist (1/1), Dead Code Removal (1/1), Daily Doc Updater (1/1), Daily Malicious Code Scan (1/1), Daily Doc Healer (1/1), Daily Testify Expert (1/1), Go Fan (1/1), PR Triage (1/1)
- vs prior report (2026-05-20):
  - UNCHANGED CRITICAL: Architecture Guardian — now 12+ days offline; issue #3033 OPEN
  - MILESTONE: Daily Documentation Healer — 3rd clean day → NOW STABLE; recommend closing #3022
  - FULLY RECOVERED: Daily Documentation Updater — 3rd clean day; #2986 already CLOSED
  - CONTINUING: Testify Expert — 4th consecutive daily issue (#3042); all 4 open (#3021, #3029, #3038, #3042)
  - NEW WATCH: Go Fan — 2nd consecutive daily issue (#3043 today, #3039 yesterday closed); monitor through 2026-05-24
  - STABLE: Agentic Maintenance 4/4, Contribution Check 3/3

## Pattern Detector Results (2026-05-21)
- over_creation: 2 (Agentic Maintenance — run freq; Testify Expert — output accumulation)
- repetition: 2 (Testify Expert — 4 consecutive issues; Go Fan — 2 consecutive, watch)
- under_creation: 3 (Architecture Guardian, Code Simplifier — 0 PRs, Agentic Maintenance — 0 outputs)
- inconsistency: 0
- inactive: 1 (Architecture Guardian)
- scope_creep: 0
- clean: 9 agents

## Quality Scores (this run)
- Average quality: 63/100 (↑2 from yesterday)
- Average effectiveness: 59/100 (↑2 from yesterday)
- PR merge rate: ~84% (stable, no new agent PRs today)
- 5 behavioral patterns detected across 4 agents (↓1 from yesterday)

## Top Performers
- **Agentic Maintenance**: 4/4 (100%) — backbone agent
- **Contribution Check**: 3/3 (100%)
- **Dead Code Removal + Daily Malicious Code Scan**: consistent clean execution

## Underperformers
- **Architecture Guardian**: OFFLINE 12+ days (#3033, still open) — CRITICAL, no change
- **Testify Expert**: repetition — 4 open issues (#3021, #3029, #3038, #3042); needs dedup guard
- **Go Fan**: watch — 2 consecutive days; 2026-05-24 deadline for escalation

## Issues Tracked
- #2956: No-Op Runs tracker — OPEN (expected/normal)
- #2995: Issue Group — OPEN (this report's parent)
- #3022: Daily Doc Healer — OPEN; recommend CLOSE (3 clean days achieved 2026-05-21)
- #3033: Architecture Guardian failure — OPEN (CRITICAL)
- #3038: Testify Expert sync_token — OPEN
- #3042: Testify Expert pathparser — OPEN (new today)
- #3043: Go Fan jackc/pgx — OPEN (new today; watch)

## Report Posted
- Comment on #2995 (Issue Group)
