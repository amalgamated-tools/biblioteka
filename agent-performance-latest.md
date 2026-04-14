# Agent Performance — 2026-04-14 (Evening)
**Run:** 2026-04-14T23:44Z

## Snapshot
- 54 workflows registered; all Copilot engine; all compiled
- Copilot PR merge rate (28 closed sample): 89% (avg 15.1h)
- Bot (amalgamated-bot) PR merge rate (17 closed sample): 88% (avg 7.9h)
- Open issues: 29 (20 agent-created today, 7 [aw] failure)
- Open PRs: 23 (9 bot, 13 Copilot, 1 human release)
- **PR backlog surge: 22 agent PRs open (up from ~10 yesterday)**

## Top Performers
- **discussion-task-miner**: 3 actionable issues closed today (#1942, #1939, #1938) → led to Copilot PRs #1968, #1964, #1965 — excellent quality chain
- **daily-accessibility-review**: 4 a11y issues closed, PRs merged in v0.13.0 — high quality and completion
- **daily-qa**: PR #1988 fix: otelkeys.Limit and ISBN fix — precise, targeted, merged quickly
- **daily-perf-improver**: PR #1982 parallelize LoadBookRelations — valid perf improvement
- **repository-quality-improver**: Issue #1990 API-Frontend type contract analysis — thorough, cross-codebase
- **tech-content-editorial-board**: Issue #1978 migration accuracy review — high signal-to-noise

## Critical Issues
1. **daily-doc-updater**: TRIPLE DUPLICATE — PRs #1994, #1980, #1976 all identical "docs(background-jobs): add scan:watch-folder" — 3 open at once (CRITICAL, 3rd consecutive day)
2. **Workflow Failure Surge**: 6 new [aw] failures today (contribution-check, daily-repo-chronicle, markdown-linter, issue-triage, update-docs, contribution-guidelines-checker)
3. **contribution-check**: Issue #1947 open with "lgtm" label — zero-finding report still created (MEDIUM)
4. **PR Backlog**: 22 open agent PRs — review bandwidth concern, up from ~10 on Apr 13

## Alerts Updated
- daily-doc-updater duplicate PR issue escalated to CRITICAL (3 identical PRs)
- New [aw] failures: 6 workflows failing daily (elevated concern)
- PR backlog now at 22 open (raised threshold alert)

## Discussion Created
Agent Performance Report — Week of 2026-04-14
