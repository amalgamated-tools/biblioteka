# Agent Performance — 2026-04-12
**Run:** 2026-04-12T23:44Z

## Snapshot
- 54 workflows registered; 53 Copilot / 1 Codex; all compiled
- PR merge rate (30-PR sample): 60% overall (Copilot 65%, amalgamated-bot 36% in-flight)
- Ecosystem: 8 open issues, healthy
- No-op tracker #1733: 58 comments, working correctly

## Top Performers
- **daily-doc-updater**: 16+ PRs/day, 65% merge rate, highly productive
- **code-simplifier**: 100% merge rate (1 PR today, merged immediately)
- **daily-accessibility-review**: Issues consistently drive merged fixes
- **discussion-task-miner**: Upstream issues (OIDC #1713, kobo #1711) integrated in v0.10.0
- **daily-plan / daily-repo-status / daily-team-status**: Clean, relevant daily reporting

## Underperformers
- **duplicate-code-detector** (CODEX): Hard fail every run — CODEX_API_KEY missing; 36 failed runs
- **unbloat-docs**: 0% merge rate in last batch (PR #1808 open, unmerged)
- **contribution-check**: Still creating report issues even with 'lgtm' results (low signal-to-noise)

## Improvements Since Last Report (Apr 12 morning)
- No new [aw] failure issues created today (5 previous still open)
- contribution-check: still running 4x/day, creating report issues each time
- discussion-task-miner: no new dedup violations observed today

## Behavioral Notes
- 54 workflows, many reactive ones running correctly as skip when no trigger
- Noop issue #1733 receiving regular comments — infrastructure working
- v0.10.0 shipped today with features from agentic discussions

## Discussion Created
Agent Performance Report — Week of 2026-04-12 (second run)
