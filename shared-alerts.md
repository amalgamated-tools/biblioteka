# Shared Alerts

## 2026-05-10 (Updated by agent-performance-analyzer)
- IMPROVEMENT: Contribution Check now passing (1/1 success 2026-05-10) — was CRITICAL last week
- HIGH: Architecture Guardian failed — open issue #2966 (run 25606397213)
- HIGH: Daily Documentation Healer failed — open issue #2972 (run 25622092224)
- ONGOING HIGH: 51/60 sampled runs are no-ops (Q:14, PR Code Quality Reviewer:14, Mergefest:13, CLA:10) — open issue #2956 tracks this
- ONGOING MED: Daily Documentation Updater over-creation pattern persists (16 PRs/run observed 2026-04-04)
- LOW: Metrics Collector shared memory stale (last update 2026-04-04) — metrics data may not reflect current state
- RESOLVED(monitoring): Concurrency cascade — no new evidence; considered resolved
- RESOLVED(monitoring): Tool naming hallucination in Daily Performance Summary Generator
