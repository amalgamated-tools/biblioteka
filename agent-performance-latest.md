# Agent Performance — 2026-04-17
**Run:** 2026-04-17T23:44Z

## Snapshot
- 54 workflows registered, all Copilot, all compiled; ~23 active daily
- 40 open issues (↑ from 29 Apr 16, ↑ from 18 Apr 15) — +10/day accumulation trend
- 30 PRs merged Apr 14–17 (amalgamated-bot: 16, Copilot: 7, dependabot: 5, human: 2)
- PR merge rate: 84% (stable); avg merge time: 1.85h (excellent)

## Top Performers
- **daily-doc-updater**: 16 PRs in Apr 4 snapshot; 6+ PRs merged Apr 16-17 (quality: 100, eff: 97)
- **discussion-task-miner**: 6 actionable issues today (#2172, #2141-#2137); drives PR pipeline (quality: 95, eff: 90)
- **daily-accessibility-review**: Multiple a11y PRs merged (quality: 95, eff: 88)
- **repository-quality-improver**: #2171, #2113 - substantive quality findings (quality: 90, eff: 85)
- **duplicate-code-detector**: #2172 (groups.go inline error handling) - precise, actionable (quality: 90, eff: 85)
- **glossary-maintainer**: PR #2108 merged; consistent daily execution (quality: 88, eff: 90)
- **daily-perf-improver**: FK indexes, monthly summaries (quality: 88, eff: 85)

## Critical Issues
1. **Engine failure cluster expanding**: Now 6 [aw] issues from real failures
   - ci-doctor (#2059, Apr 15) — persistent, mid-grep termination
   - daily-test-improver (#2089, Apr 16) — engine died during go fmt
   - daily-grumpy-reviewer (#2095, Apr 16) — engine died writing JSON cache
   - update-docs (#2097, Apr 16) — engine died reading docs/api.md
   - **NEW**: sergo (#2148, Apr 17) — Serena Go Expert failed
   - **NEW**: dependabot-bundler (#2156, Apr 17) — Bundler failed
2. **Issue accumulation accelerating**: 18 → 29 → 40 in 2 days (+11/day trend)
   - 6 ephemeral status issues (unchanged from Apr 16): #2170, #2165, #2163, #2162, #2077, #2052
   - 3 editorial board issues: #2167, #2106, #2049
   - [aw] tracking issues #2044, #1733 still stale and unresolved
3. **Dependabot bundler failing**: New today, creating duplicate bundles (#2155, #2154) as well as failure notice

## Resolved Since Apr 16
- Daily Grumpy Reviewer: Today's scheduled run shows "success" — possible recovery ✅
- All other scheduled runs: 100% success rate today ✅

## Discussion Created
"Agent Performance Report — Week of 2026-04-17" in Audits category
