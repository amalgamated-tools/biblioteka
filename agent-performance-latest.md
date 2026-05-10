# Agent Performance 2026-05-10

**Run**: agent-performance-analyzer | 2026-05-10T13:20Z

## Summary
- 11 distinct workflows active in current window (2026-05-10)
- 60 runs sampled: Agentic Maintenance 100%, Code Simplifier 100%, Contribution Check 100%, Dead Code Removal 100%
- 51 no-op/skipped runs: Q(14), PR Code Quality Reviewer(14), Mergefest(13), CLA Assistant(10)
- vs prior report (2026-05-09): Contribution Check now passing (was 0/3 failures) — IMPROVEMENT
- New failures: Architecture Guardian (#2966), Daily Documentation Healer (#2972)

## Top Performers
- **Agentic Maintenance**: 3/3 (100%) — consistent, reliable
- **Code Simplifier**: 1/1 (100%)
- **Contribution Check**: 1/1 (100%) — IMPROVED from last week's 0/3
- **Dead Code Removal Agent**: 1/1 (100%)
- **Daily Documentation Updater**: 1/1 (100%) — but over-creation persists

## Underperformers
- **Mergefest**: 0/13 (100% skipped) — 0 output, consuming Actions minutes
- **PR Code Quality Reviewer**: 0/14 (100% skipped) — same issue
- **Q**: 0/14 (100% skipped) — same issue
- **CLA Assistant**: 0/10 (100% skipped) — trigger-based, may be expected
- **Architecture Guardian**: 0/1 (100% failure) — open issue #2966
- **Daily Documentation Healer**: 0/1 (100% failure) — open issue #2972

## Key Findings
1. IMPROVEMENT: Contribution Check now passing (was critical failure last week)
2. HIGH: 2 new workflow failures — Architecture Guardian, Daily Documentation Healer
3. ONGOING - HIGH: 51/60 sampled runs produce zero output (no-op waste)
4. ONGOING - MED: Daily Doc Updater over-creation pattern persists
5. LOW: Metrics data in shared memory is stale (last updated 2026-04-04); Metrics Collector needs attention

## Discussion Created
- Yes (this run)

## Issues Created
- None new (tracking via existing #2956, #2966, #2972)
