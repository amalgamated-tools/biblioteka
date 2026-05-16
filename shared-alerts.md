# Shared Alerts

## 2026-05-16 (Updated by agent-performance-analyzer)
- ONGOING CRITICAL: Architecture Guardian lock-file out of sync — failed May 9, May 11, May 13, no run May 14/15/16; issue #2981; run `gh aw compile` to fix (7+ days unresolved)
- ONGOING HIGH: Code Refiner — 0% success (cancellations persistent), systemic instability; no issue filed yet — recommend creating one
- ONGOING MED: Daily Documentation Updater over-creates PRs (~16/run) — needs throttle/dedup; issue #2986; cascading into Doc Updater Auto-Merge stall
- ONGOING MED: Daily Testify Expert — scope-creep pattern (issues #3006, #3012, #3015 created today); needs dedup guard and closed-issue state check
- ONGOING: Doc Updater Auto-Merge — stalled, blocked by upstream Doc Updater PR flood; will self-resolve if #2986 is fixed
- MONITOR: PR Triage Agent — improving (1/2 success); continue monitoring next 3 runs
- EXPECTED: Q(10+), PR Code Quality Reviewer(10+), Mergefest(10+), CLA(8+) — PR-triggered skips, normal; tracked in #2956
- RECOMMEND CLOSE: Daily Documentation Healer — 6+ consecutive clean days; issue #2972 should be closed now
- STABLE: Agentic Maintenance 5/5 (100%) — consistently top performer
- STABLE: Contribution Check 2/2 (100%) — no alert needed
- LOW: Metrics Collector shared memory stale 42 days (last update 2026-04-04) — metrics collector needs re-trigger
