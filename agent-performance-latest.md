# Agent Performance 2026-05-22

**Run**: agent-performance-analyzer | 2026-05-22T13:40Z

## Summary
- 17 workflows analyzed (12 active, 5 event-triggered skip-only)
- Active agents with success today: Agentic Maintenance (4/4), Contribution Check (2/2), Code Simplifier (1/1), Typist (1/1), Dead Code Removal (1/1), Daily Doc Updater (1/1), Daily Malicious Code Scan (1/1), PR Triage (1/1), Scheduled (1/1)
- vs prior report (2026-05-21):
  - UNCHANGED CRITICAL: Architecture Guardian — now 13+ days offline; issue #3033 OPEN
  - STABLE: Daily Documentation Healer — 4+ clean days; #3022 should be CLOSED
  - STABLE: Daily Documentation Updater — 4+ clean days, fully recovered
  - ESCALATING: Testify Expert — 5th consecutive daily issue (#3046 today); #3046 body="test body" = quality REGRESSION; new issue created
  - CONTINUING WATCH: Go Fan — 3rd consecutive daily issue (#3049 today); deadline 2026-05-24 (2 days remaining)
  - STABLE: Agentic Maintenance 4/4, Contribution Check 2/2

## Pattern Detector Results (2026-05-22)
- over_creation: 2 (Testify Expert, Go Fan)
- repetition: 1 (Testify Expert — 5 consecutive issues)
- inconsistency: 1 (Testify Expert — #3046 body="test body" regression)
- under_creation: 1 (Code Simplifier — 0 outputs)
- inactive: 1 (Architecture Guardian)
- scope_creep: 0
- clean: 13 agents

## Quality Scores (this run)
- Average quality: 61/100 (↓2 from yesterday — Testify regression)
- Average effectiveness: 60/100 (↑1 from yesterday)
- PR merge rate: ~84% (stable)
- 6 behavioral patterns detected across 4 agents (↑1 from yesterday)

## Top Performers
- **Agentic Maintenance**: 4/4 (100%) — backbone agent
- **Dead Code Removal + Daily Doc Updater**: consistent clean PRs
- **Duplicate Code Detector + Function Namer**: high-quality analytical issues

## Underperformers
- **Architecture Guardian**: OFFLINE 13+ days (#3033, still open) — CRITICAL, no change
- **Testify Expert**: quality regression (#3046 body="test body") + 5th consecutive issue; new issue created
- **Go Fan**: watch — 3rd consecutive day; 2026-05-24 deadline (2 days)
- **Code Simplifier**: under-creation — 0 outputs despite successful run

## Issues Tracked
- #2995: Issue Group — OPEN (this report's parent)
- #2956: No-Op Runs tracker — OPEN (expected/normal)
- #3022: Daily Doc Healer — OPEN; RECOMMEND CLOSE (4+ clean days)
- #3033: Architecture Guardian failure — OPEN (CRITICAL)
- #3042: Testify Expert pathparser — OPEN
- #3046: Testify Expert "test body" — OPEN (quality regression)
- #3049: Go Fan golang.org/x/sync — OPEN (watch)
- New issue created for #3046 Testify regression

## Report Posted
- Comment on #2995 (Issue Group) — aw_N8hs3NYC
