# Agent Performance — 2026-04-05
**Workflow:** agent-performance-analyzer
**Run timestamp:** 2026-04-05T23:44Z

## Ecosystem Snapshot
- 31 workflows tracked (up from 24 on Apr 4; 7 new workflows added)
- New workflows (not yet in metrics): daily-grumpy-reviewer, daily-nitpick-reviewer, daily-security-review, pr-nitpick-reviewer, dependabot-burner
- Active Apr 5: 13+ workflows (estimate)
- Success rate: 100% (all runs passing)
- Open issues: 1 (no-op tracker only)
- 18 agent PRs created Apr 5 (14 merged, 4 open)
- 14 issues created Apr 5 (all 14 closed = 100% same-day resolution!)

## CRITICAL: daily-doc-updater PR Escalation + Duplicates
- Volume trend: 0 (Mar 31) → 8 (Apr 3) → 16 (Apr 4) → 18+ (Apr 5)
- **3 confirmed duplicate PRs today**: #1376≡#1378 (stale kobo_tokens), #1308≡#1352 (icon aria-hidden), #1308≡#1311 (api.test.ts coverage)
- Issue created: see below. This is now CRITICAL (day 3 of escalation)

## RESOLVED: daily-team-evolution-insights NOW COMPILED ✅
- Was Day 3 critical unresolved issue; compiled successfully during this run

## Uncompiled Workflows (6)
- artifacts-summary, commit-changes-analyzer, duplicate-code-detector, grumpy-reviewer, pr-nitpick-reviewer, weekly-repo-map

## ci-coach: 6 consecutive no-op runs (was 4)
- Runs #31-#36 all no action found
- Still recommending schedule reduction to weekly

## daily-workflow-updater: Permanent failure
- gh not installed / not authenticated in agent environment
- Reports noop every run; never executes successfully

## Top Performers
- **daily-accessibility-review**: 8 issues all same-day closed! Exceptional resolution pipeline
- **duplicate-code-detector**: 3 issues today, all auto-closed = 100% same-day
- **code-simplifier**: Merged PRs with good code quality; 100% acceptance rate
- **dead-code-remover**: First PR (#1333) merged, clean results
- **issue-triage**: Fast accurate triaging, 100% success rate

## Issues Created This Run
- 1 performance report discussion (agent-performance-analyzer)

## Coordination Notes for Other Orchestrators
- Campaign Manager: daily-doc-updater volume CRITICAL (18 PRs Apr 5, 3 duplicates) — review bandwidth impact
- Workflow Health Manager: 6 uncompiled workflows; daily-workflow-updater should be disabled or fixed
