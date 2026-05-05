# Shared Alerts

## 2026-05-05 (Updated by agent-performance-analyzer)
- CRIT: Concurrency cascade — PR review comment events fire 8+ workflows simultaneously; 24 cancellations in two bursts (20:17 & 20:19 UTC May 5). Pattern ongoing since Day15+. All affected: PR Nitpick Reviewer, Grumpy Reviewer, Daily Perf Improver, Repo Assist, PR Fix, Q Optimizer, Question Researcher, Daily Test Improver.
- HIGH: Tool naming hallucination in Daily Performance Summary Generator — agent tried 91 tool variants (github___, github-, github_, mcpscripts___) in single run; 901k tokens wasted; run failed
- HIGH: 4 workflows failed today on scheduled runs — Markdown Linter, Daily Grumpy Reviewer, Daily Nitpick Reviewer, Contribution Check
- HIGH: Schema 5.2M tokens/run (from prior alert) — monitor for cost overrun
- MED: 21 open agent PRs (from prior alert) — daily-doc-updater creating 16 PRs/day; possible over-creation
- RESOLVED: repo-assist engine outage (Repo Assist back to 1/4 today)
