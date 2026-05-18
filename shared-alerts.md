# Shared Alerts

## 2026-05-18 (Updated by agent-performance-analyzer)
- ONGOING CRITICAL: Architecture Guardian lock-file out of sync — failed May 9-15, no run May 16/17/18; issue #2981 (7 days unresolved); run `gh aw compile` to fix
- ONGOING HIGH: Code Refiner — 2/2 skipped today, systemic instability; needs workflow config investigation
- NEW REGRESSION: Daily Documentation Healer — failed today (issue #3022) after 7+ clean days; potential cause: Doc Updater ~16 PR flood overwhelming healer; do NOT close #2972
- NEW: Test CI failure today (run 26032697105) — no automated triage; may need manual investigation
- ONGOING MED: Daily Documentation Updater over-creates PRs (~16/run) — needs throttle/dedup; issue #2986; cascading into Doc Updater Auto-Merge stall
- ONGOING: Doc Updater Auto-Merge — stalled 3/3 skipped, blocked by upstream Doc Updater PR flood; will self-resolve if #2986 is fixed
- TESTIFY EXPERT: today's #3021 is new file (NOT duplicate today); prior consecutive-day duplication pattern flagged (add dedup guard)
- MONITOR: PR Triage Agent — 1/2 success; continue monitoring
- EXPECTED: Q(17), PR Code Quality Reviewer(11), Mergefest(10), CLA(10/11) — PR-triggered skips, normal; tracked in #2956
- STABLE: Agentic Maintenance 5/5 (100%) — consistently top performer
- STABLE: Contribution Check 3/3 (100%)
- LOW: Metrics Collector shared memory stale 44 days (last update 2026-04-04) — metrics collector needs re-trigger
