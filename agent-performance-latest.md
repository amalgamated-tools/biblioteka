# Agent Performance 2026-05-15

**Run**: agent-performance-analyzer | 2026-05-15T13:24Z

## Summary
- 19 distinct workflows analyzed (from 80 most recent runs)
- Active agents with success today: Agentic Maintenance (4/4), Code Simplifier (1/1), Contribution Check (2/2), Dead Code Removal (1/1), Typist (1/1)
- PR-triggered skip-only agents: Q(14), PR CQR(11), Mergefest(11), CLA(11) — expected; tracked in #2956
- vs prior report (2026-05-14):
  - ONGOING CRITICAL: Architecture Guardian — still no runs, issue #2981 now 6 days unresolved
  - ONGOING HIGH: Code Refiner — 0% success (3 cancels in window), no issue filed yet
  - ONGOING HIGH: Daily Testify Expert — repetition + scope-creep patterns confirmed
  - ONGOING MED: Daily Documentation Updater over-creation (~16 PRs/run, #2986)
  - NEW: PR Triage Agent — 1 failure in window, 1 in-progress (50% failure rate, low volume)
  - NEW: Auto-label docs-only PRs — 1 failure in window (50%, low volume)
  - OPEN: #2972 Daily Doc Healer — still OPEN despite 4+ clean days (recommend close)
  - STABLE: Agentic Maintenance 4/4 (100%)
  - STABLE: Contribution Check 2/2 (100%)

## Top Performers
- **Agentic Maintenance**: 4/4 (100%) — consistently top performer
- **Contribution Check**: 2/2 (100%) — STABLE
- **Code Simplifier, Dead Code Removal, Typist**: 1/1 each

## Underperformers
- **Architecture Guardian**: 0 runs, #2981 open 6 days — CRITICAL
- **Code Refiner**: 0% (3 cancels) — systemic instability
- **Daily Testify Expert**: repetition (dup smtp #2977+#2984) + scope-creep (re-open #2991)
- **Daily Documentation Updater**: ~16 PRs/run over-creation (#2986)
- **Doc Updater Auto-Merge**: stalled (blocked by PR flood upstream)
- **PR Triage Agent**: 1 failure (50%, needs monitoring)

## Quality Scores (this run)
- Average quality: 63/100 (stable vs yesterday)
- Average effectiveness: 57/100 (↓1 due to PR Triage Agent failure)
- PR merge rate: ~84% (stable)
- 11 behavioral patterns detected across 8 agents

## Behavioral Patterns Detected
- over-creation: 1 (Daily Doc Updater)
- repetition: 1 (Daily Testify Expert)
- scope-creep: 2 (Daily Testify Expert, Doc Updater Auto-Merge)
- under-creation: 2 (Architecture Guardian, Daily Testify Expert)
- inconsistency: 5 (Code Refiner, Doc Updater Auto-Merge, PR Triage, CLA, Auto-label)
- clean: 8 agents

## Issues Tracked
- #2956: No-Op Runs tracker — OPEN (expected/normal)
- #2972: Daily Doc Healer failure — OPEN but RECOVERED (5+ clean days, RECOMMEND CLOSE)
- #2981: Architecture Guardian failure — OPEN (6+ days, CRITICAL)
- #2984: Testify dup smtp — CLOSED
- #2986: Daily Doc Updater over-creation — OPEN
- #2991: Testify re-opened ssrf — CLOSED
- #2995: Issue Group — OPEN (this report's parent)

## Report Posted
- Comment on #2995 (Issue Group)
