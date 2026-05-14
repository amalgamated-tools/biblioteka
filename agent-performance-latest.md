# Agent Performance 2026-05-14

**Run**: agent-performance-analyzer | 2026-05-14T13:23Z

## Summary
- 19 distinct workflows analyzed (from 100 most recent runs)
- Active agents today: 12 distinct names with success
- PR-triggered skip-only agents: Q(22), PR CQR(16), Mergefest(15), CLA(13) — expected; tracked in #2956
- vs prior report (2026-05-13):
  - IMPROVING: Daily Documentation Updater — RECOVERED (1/1 today after failure #2986); over-creation still ongoing
  - ONGOING: Architecture Guardian lock-file issue (#2981) — 5 days unresolved, no run today
  - ONGOING: Daily Testify Expert duplicate smtp (#2977 + #2984) + re-open ssrf (#2991)
  - ONGOING: Code Refiner 0% success (2 cancellations)
  - STABLE: Agentic Maintenance 5/5 (100%)
  - STABLE: Contribution Check 3/3 (100%)
  - RECOVERED: Daily Documentation Healer — 4th consecutive clean day (recommend closing #2972)

## Top Performers
- **Agentic Maintenance**: 5/5 (100%) — consistently top performer
- **Contribution Check**: 3/3 (100%) — STABLE
- **Daily Documentation Healer**: 1/1 — 4th clean day (RECOVERED)
- **Code Simplifier, Dead Code Removal, Daily Malicious Code Scan, Typist, Daily Caveman Optimizer, Go Fan, PR Triage**: 1/1 each

## Underperformers
- **Code Refiner**: 0% success (2 cancelled) — systemic instability; pattern: inconsistency
- **Architecture Guardian**: no run today, #2981 open 5+ days — pattern: under-creation
- **Daily Testify Expert**: duplicate smtp issues + re-opened fixed ssrf — pattern: repetition
- **Daily Documentation Updater**: ~16 PRs/run (#2986) — pattern: over-creation
- **Doc Updater Auto-Merge**: 4/4 skipped — pattern: under-creation (dependency stall)

## Quality Scores (this run)
- Average quality: 63/100 (↑+1 from yesterday's 62)
- Average effectiveness: 58/100 (↑+1 from yesterday's 57)
- PR merge rate: ~84% (stable)

## Behavioral Patterns Detected
- over-creation: 1 (Daily Doc Updater)
- repetition: 1 (Daily Testify Expert)
- under-creation: 2 (Architecture Guardian, Doc Updater Auto-Merge)
- inconsistency: 1 (Code Refiner)
- clean: 14 agents

## Issues Tracked
- #2956: No-Op Runs tracker — OPEN (expected/normal operation)
- #2972: Daily Doc Healer failure — OPEN but RECOVERED (4 clean days, RECOMMEND CLOSE)
- #2981: Architecture Guardian failure — OPEN (ongoing, 5+ days)
- #2984: Testify duplicate smtp — OPEN
- #2986: Daily Doc Updater over-creation — OPEN
- #2991: Testify re-opened fixed ssrf issue — OPEN

## Report Posted
- Comment on #2995 (Issue Group)
