# Agent Performance — 2026-04-12
**Run:** 2026-04-12T15:35Z

## Snapshot
- 54 workflows; 53 Copilot / 1 Codex; 53/54 compiled; PR merge rate 84%
- 5 open [aw] failures: arborist #1753, triage #1737, this analyzer #1735, code-metrics #1730, markdown-linter #1702
- Noop fix PRs #1635/#1636 still awaiting merge

## Top Performers
- **daily-doc-updater**: 16 PRs/day, high merge rate
- **repo-assist**: Batched 4 a11y fixes in #1700; priority-high CI/bug issues
- **discussion-task-miner**: Issues drive real implementations (OIDC #1713, kobo #1711, telemetry #1712 all merged in v0.10.0)
- **daily-accessibility-review**: Issues → accessibility fixes merged

## Underperformers
- **duplicate-code-detector**: Hard fail daily (missing CODEX_API_KEY)
- **issue-arborist, agentic-triage**: Hard failing
- **weekly-issue-summary**: Not compiled

## Behavioral Issues
- **contribution-check**: 4 issues on Apr 12 (no skip guard for 0-result runs)
- **discussion-task-miner**: Issue #1751 duplicated 3 days running — no dedup

## Discussion Created: Agent Performance Report — Week of 2026-04-12
