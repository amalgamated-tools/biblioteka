# Agent Performance — 2026-04-07
**Workflow:** agent-performance-analyzer
**Run timestamp:** 2026-04-07T23:44Z

## Ecosystem Snapshot
- 31 workflows tracked, **all compiled** (previously 6 uncompiled — RESOLVED)
- Scheduled runs today: 19 — all succeeded (100% success rate)
- Open issues: 3 (down from 7 on Apr 6)
- Open agent/Copilot PRs: 16 (14 Copilot + 2 github-actions[bot])
- PRs merged today: 6+ (accessibility fixes, code simplifications, refactoring)

## ✅ RESOLVED: All 6 Uncompiled Workflows Now Compiled
- Previously flagged workflows (artifacts-summary, commit-changes-analyzer,
  duplicate-code-detector, grumpy-reviewer, pr-nitpick-reviewer, weekly-repo-map)
  now all show compiled: Yes
- Status: CLOSED

## ✅ RESOLVED: daily-doc-updater PR explosion
- Resolved Apr 6; today only 1 PR (#1499 Kobo 404 docs) - sustained improvement

## ⚠️ NEW: Copilot PR Queue Backlog (14 open PRs)
- 14 Copilot PRs opened today across multiple themes (refactoring, auth, docs, CI)
- Quality appears high but volume creates review burden
- Recommendation: Set PR review cadence; batch-merge related PRs
- Monitor for stale/merge-conflicting PRs

## 🟡 CARRIED: GH_AW_AGENT_TOKEN Permission Failure
- Issue #1452 still open (from Apr 6 duplicate-code-detector run)
- Today's run adapted gracefully (no Copilot assignment attempted?)
- Action: Verify token has Copilot assignment permission

## 🟡 MONITORING: daily-workflow-updater
- Today ran 4.2 min (positive change from previous failure pattern)
- Previous persistent gh CLI failure; needs 3 more successful runs to clear

## Agent Scores This Run
- code-simplifier: A (13.6 min, quality PRs, #1480 merged via file-diet)
- daily-accessibility-review: A (14.9 min, issue group #1460, fast fix pipeline)
- daily-doc-updater: A (8.1 min, targeted 1 PR #1499, sustained fix)
- dead-code-remover: A (10.6 min, #1480 merged quickly)
- dependabot-burner: A (3.7 min, no excess dependabot PRs)
- duplicate-code-detector: B+ (5.5 min, quality issues, Apr 6 assign failure monitoring)
- schema-consistency-checker: B+ (10.4 min, ran cleanly)
- daily-security-review: B+ (9.2 min, clean - healthy repo or over-cautious)
- daily-workflow-updater: B (4.2 min, apparent improvement - monitoring)
- ci-coach: N/A (not in today's runs — good, schedule improvement may be working)
- daily-nitpick-reviewer: B (9.5 min, ran)
- daily-grumpy-reviewer: B (9.9 min, ran)

## Issues Created This Run
- None (all findings are status updates/monitoring)
- Performance report discussion created
