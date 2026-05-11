# Agent Performance 2026-05-11

**Run**: agent-performance-analyzer | 2026-05-11T13:50Z

## Summary
- 18 distinct workflows analyzed (from 100 most recent runs)
- 34 success / 65 skipped / 1 in-progress
- 64 no-op/skipped runs: Q(22), PR Code Quality Reviewer(16), Mergefest(14), CLA(12)
- vs prior report (2026-05-10):
  - IMPROVEMENT: Daily Documentation Healer recovered (1/1 success today, was failing)
  - IMPROVEMENT: Architecture Guardian issue #2966 now CLOSED
  - STABLE: Contribution Check 3/3 (100%) — sustained
  - ONGOING: No-op cluster (Q/PR Code Quality Reviewer/Mergefest) unchanged

## Top Performers
- **Agentic Maintenance**: 6/6 (100%) — most-run agent, perfect record
- **Contribution Check**: 3/3 (100%) — sustained recovery from 0/3 last week
- **Code Simplifier**: 1/1 (100%)
- **Dead Code Removal Agent**: 1/1 (100%)
- **Daily Malicious Code Scan Agent**: 1/1 (100%)
- **Daily Testify Uber Super Expert**: 1/1 — created issue #2977
- **Go Fan**: 1/1 — created issue #2978

## Underperformers
- **Q**: 0/22 (100% skipped) — under-creation, issue #2956
- **PR Code Quality Reviewer**: 0/16 (100% skipped) — under-creation, issue #2956
- **Mergefest**: 0/14 (100% skipped) — under-creation, issue #2956
- **Architecture Guardian**: issue #2966 CLOSED; not seen in today's window — monitor
- **Daily Documentation Updater**: over-creation (~16 PRs/run) — needs throttle
- **Daily Documentation Healer**: recovered today, but issue #2972 still open — monitor

## Quality Scores (this run)
- Average quality: 62/100
- Average effectiveness: 58/100
- PR merge rate: 84% (ecosystem healthy)

## Key Findings
1. ONGOING HIGH: 64/100 runs are no-ops; Q/PR CQR/Mergefest share broken trigger (#2956)
2. IMPROVEMENT: Architecture Guardian #2966 closed — verify next run
3. IMPROVEMENT: Daily Documentation Healer recovered today
4. ONGOING MED: Daily Doc Updater over-creates (~16 PRs/run)
5. LOW: Metrics Collector shared memory still stale (2026-04-04) — 37 days

## Discussion Created
- Yes — "Agent Performance Report — Week of 2026-05-11"

## Issues Created
- None new (tracking via #2956, #2972; #2966 closed)
