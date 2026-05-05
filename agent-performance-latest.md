# Agent Performance 2026-05-05

**Run**: agent-performance-analyzer | 2026-05-05T23:45Z

## Summary
- 55 registered workflows (30 active in 7 days)
- 60 runs: 31 success (52%), 5 failed (8%), 24 cancelled (40%)
- Vs baseline (2026-04-04): success rate dropped from 100% → 52%

## Top Performers
- **Agentic Triage**: 7/7 success (100%)
- **Contribution Guidelines Checker**: 7/7 success (100%)
- **Daily single-run workflows**: ~20 workflows at 100% on scheduled runs

## Underperformers
- **PR Nitpick Reviewer 🔍**: 0/5 success (100% cancelled)
- **Grumpy Code Reviewer 🔥**: 0/4 (100% cancelled)
- **Daily Perf Improver**: 1/5 (80% cancelled)
- **Daily Performance Summary Generator**: 1/1 failed; 901k tokens, 91 tool types — tool naming confusion

## Key Issues
1. **CRIT - Concurrency cascade**: 24 cancellations from PR review comment storms (20:17-20:19 on May 5); 8 workflows fired simultaneously per event
2. **HIGH - Recurring failures**: Markdown Linter, Daily Grumpy Reviewer, Daily Nitpick Reviewer, Contribution Check all failed today
3. **HIGH - Tool naming hallucination**: Daily Perf Summary Generator tried github___, github-, github_ variants (91 tool types); failed despite 901k tokens
4. **MED - Zero outputs**: Many workflows run but produce no safe outputs — requires investigation

## Discussion Created
- Yes (this run)

## Issues Created
- None this run (pattern matches existing CRIT alert)
