# Shared Alerts

## 2026-05-15 (Updated by agent-performance-analyzer)
- ONGOING CRITICAL: Architecture Guardian lock-file out of sync — failed May 9, May 11, May 13, no run May 14 or May 15; issue #2981; run `gh aw compile` to fix (6+ days unresolved)
- ONGOING HIGH: Code Refiner — 0% success (3 cancellations in window), systemic instability; no issue filed yet — recommend creating one
- ONGOING MED: Daily Documentation Updater over-creates PRs (~16/run) — needs throttle/dedup; issue #2986; cascading into Doc Updater Auto-Merge stall
- ONGOING MED: Daily Testify Expert — duplicate smtp issues (#2977/#2984), re-opened fixed ssrf (#2991); needs dedup guard and closed-issue state check
- ONGOING: Doc Updater Auto-Merge — stalled (4 queued), blocked by upstream Doc Updater PR flood; will self-resolve if #2986 is fixed
- NEW MONITOR: PR Triage Agent — 1 failure in window (50% rate, low sample); watch next 3 runs
- NEW MONITOR: Auto-label docs-only PRs — 1 failure in window (50% rate, low sample); watch next 5 runs
- EXPECTED: Q(14), PR Code Quality Reviewer(11), Mergefest(11), CLA(11) — PR-triggered skips, normal; tracked in #2956
- RECOMMEND CLOSE: Daily Documentation Healer — 5+ consecutive clean days; issue #2972 should be closed
- STABLE: Agentic Maintenance 4/4 (100%) — consistently top performer
- STABLE: Contribution Check 2/2 (100%) — no alert needed
- LOW: Metrics Collector shared memory stale 40+ days (last update 2026-04-04) — metrics collector may need re-trigger
