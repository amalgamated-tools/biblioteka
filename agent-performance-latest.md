# Agent Performance 2026-05-13

**Run**: agent-performance-analyzer | 2026-05-13T13:36Z

## Summary
- 26 distinct workflows observed (from 200 most recent runs)
- Active agents today: 12 distinct names with success
- 51/200 no-op/skipped: Q(21), Mergefest(15), PR CQR(15) — PR-triggered, correctly skip; tracked in #2956
- vs prior report (2026-05-12):
  - RECOVERED: Daily Documentation Healer — 3rd consecutive clean day → marking RECOVERED
  - REGRESSION: Testify Expert reopened previously auto-closed ssrf file (#2991 vs #2960)
  - ONGOING: Architecture Guardian lock-file issue (#2981) — 4 days unresolved
  - ONGOING: Daily Doc Updater over-creation (~16 PRs/run) + failure (#2986)
  - STABLE: Agentic Maintenance 6/6 (100%)
  - STABLE: Contribution Check 3/3 (100%)

## Top Performers
- **Agentic Maintenance**: 6/6 (100%) — consistently top performer
- **Contribution Check**: 3/3 (100%) — STABLE
- **Code Simplifier, Dead Code Removal, Daily Malicious Code Scan, Typist, Daily Caveman Optimizer, Go Fan, Daily Doc Healer**: 1/1 each
- **Daily Testify Expert**: 1/1 success (but dedup concern)

## Underperformers
- **Architecture Guardian**: lock file out of sync, failed May 9 + May 11 (#2981 open) — 4 days unresolved
- **Daily Documentation Updater**: failed (#2986) + over-creates ~16 PRs/run
- **Daily Testify Expert**: re-opened previously auto-closed issue (#2991 vs #2960 for ssrf)
- **Code Refiner**: 2/3 cancelled, 0% success — unstable execution
- **Doc Updater Auto-Merge**: 4/4 skipped — likely blocked (no PRs to merge?)
- **Q/PR CQR/Mergefest**: PR-triggered, 100% skip — expected behavior, tracked in #2956

## Quality Scores (this run)
- Average quality: 62/100 (↑+1 from yesterday's 61)
- Average effectiveness: 57/100 (↑+1 from yesterday's 56)
- PR merge rate: ~84% (stable)

## Key Findings
1. HIGH: Architecture Guardian lock file regression — 4 days unresolved (#2981)
2. MED: Daily Testify Expert re-opened previously fixed issue (#2991 = regression of #2960)
3. MED: Daily Doc Updater failure (#2986) + over-creation (~16 PRs/run)
4. LOW: Code Refiner unstable (2 cancellations)
5. RECOVERED: Daily Documentation Healer — 3rd clean day

## Issues Tracked
- #2956: No-Op Runs tracker — OPEN (expected/normal operation)
- #2972: Daily Doc Healer failure — OPEN but RECOVERED (3 clean days)
- #2981: Architecture Guardian failure — OPEN (ongoing, 4 days)
- #2984: Testify duplicate smtp — OPEN
- #2986: Daily Doc Updater failure — OPEN
- #2991: Testify re-opened fixed ssrf issue — OPEN (new today)
