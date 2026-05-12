# Agent Performance 2026-05-12

**Run**: agent-performance-analyzer | 2026-05-12T13:28Z

## Summary
- 19 distinct workflows observed (from 100 most recent runs)
- Active agents today: 13 distinct names with success
- 51/100 runs are skipped/no-op: Q(22), Mergefest(15), PR Code Quality Reviewer(14)
- vs prior report (2026-05-11):
  - REGRESSION: Architecture Guardian failed again — new issue #2981 (was #2966, thought resolved)
  - REGRESSION: Daily Documentation Updater failed — new issue #2986
  - NEW ISSUE: Daily Testify Expert duplicate (#2977 May 11 = #2984 May 12, same file)
  - STABLE: Daily Documentation Healer 2nd clean day (continuing recovery)
  - STABLE: Contribution Check 3/3 (100%) — now marking stable
  - STABLE: Agentic Maintenance 6/6 (100%)

## Top Performers
- **Agentic Maintenance**: 6/6 (100%)
- **Contribution Check**: 3/3 (100%) — STABLE
- **Code Simplifier, Dead Code Removal, Daily Malicious Code Scan, Typist, Daily Caveman Optimizer, Go Fan**: 1/1 each
- **Daily Documentation Healer**: 1/1 (2nd clean day)

## Underperformers
- **Q**: 0/22 (100% skipped) — issue #2956
- **PR Code Quality Reviewer**: 0/14 (100% skipped) — issue #2956
- **Mergefest**: 0/15 (100% skipped) — issue #2956
- **Architecture Guardian**: failed again (#2981) — REGRESSION, fix #2966 did not hold
- **Daily Documentation Updater**: failed today (#2986), AND over-creates PRs (~16/run)
- **Daily Testify Expert**: duplicate issues #2977 + #2984 (same file, consecutive days)
- **Doc Updater Auto-Merge**: 4/4 non-success — blocked?
- **Code Refiner**: 3/3 non-success — skipped?

## Quality Scores (this run)
- Average quality: 61/100 (↓ -1 from yesterday)
- Average effectiveness: 56/100 (↓ -2 from yesterday)
- PR merge rate: ~84% (stable)

## Key Findings
1. HIGH: Architecture Guardian regression — fix didn't hold; issue #2981
2. HIGH: Q/PR CQR/Mergefest — 51 no-op runs; issue #2956 still open
3. MED: Daily Testify Expert duplication — same issue opened two days in a row
4. MED: Daily Doc Updater failed + over-creation pattern
5. LOW: Metrics Collector shared memory stale 38 days

## Issues Tracked
- #2956: No-op runs (Q/PR CQR/Mergefest) — OPEN
- #2972: Daily Doc Healer failure — OPEN, monitoring (2nd clean day)
- #2981: Architecture Guardian failure — OPEN (regression)
- #2984 vs #2977: Testify duplicate — no dedup issue yet
- #2986: Daily Doc Updater failure — OPEN
