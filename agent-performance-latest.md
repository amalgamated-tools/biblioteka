# Agent Performance — 2026-04-13 (Evening)
**Run:** 2026-04-13T23:45Z

## Snapshot
- 54 workflows registered; 53 Copilot / 1 Codex; all compiled
- Agent PR merge rate (28 closed agent PRs): 82%
- Overall PR merge rate (50 sample): 90%
- Open issues: 17 (12 agent-created, 3 human, 2 other)
- Open PRs: 11 (9 from amalgamated-bot, 1 human release PR)
- Ecosystem growth: 24 → 54 workflows since Apr 4 metrics snapshot

## Top Performers
- **code-simplifier**: 100% merge rate; refactoring changes accepted immediately
- **discussion-task-miner**: 3 relevant issues created today (#1842, #1839, #1840) — high quality task extraction
- **repository-quality-improver**: New issue #1884 with thorough cross-dialect analysis
- **tech-content-editorial-board**: Issue #1872 well-structured high-priority analysis
- **daily-doc-updater**: Highly active; ~82% agent PR merge rate; volume leader

## Underperformers
- **duplicate-code-detector** (CODEX): HARD FAIL — CODEX_API_KEY missing; 36+ failures (CRITICAL, ongoing)
- **contribution-check**: Issue #1875 created today for "lgtm" scenario — still low signal-to-noise (MEDIUM, persists since Apr 12)
- **unbloat-docs**: PR #1878 open, previous batch had 0% merge rate (MEDIUM, ongoing)
- **daily-doc-updater** (duplicate PRs): PRs #1865 and #1870 are near-duplicate docs PRs — over-creation signal

## New Observations Since Apr 12
- daily-doc-updater created duplicate docs PRs (#1865 and #1870 both "fix clamping description")
- PR backlog at 10 agent PRs (up from ~5 on Apr 12) — review bandwidth may be limited
- Overall merge rate improved from 84% → 90%
- repo-assist active with CI improvement suggestion (#1827) — positive signal
- 5 open [aw] failure issues remain from Apr 12 (#1753, #1737, #1735, #1730, #1702) — no resolution yet

## Discussion Created
Agent Performance Report — Week of 2026-04-13
