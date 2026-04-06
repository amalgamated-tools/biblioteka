# Agent Performance — 2026-04-06
**Workflow:** agent-performance-analyzer
**Run timestamp:** 2026-04-06T23:44Z

## Ecosystem Snapshot
- 31 workflows tracked (stable from Apr 5)
- Active Apr 6: 12 workflows (Agentic Maintenance, Daily Doc Updater, Duplicate Code Detector, Grumpy Code Reviewer, CI Coach, Issue Triage, Dead Code Remover, Schema Consistency Checker, Dependabot Burner, Update Docs, File Diet, PR review workflows)
- Success rate: 100% (no hard failures on agentic workflows today)
- Open issues: 7 (3 new from duplicate-code-detector today)
- PRs today: 2 from daily-doc-updater (massive improvement from 18+ on Apr 5!)
- Merged PRs today: 20+ (security sprint, all Copilot)

## 🟢 RESOLVED: daily-doc-updater PR Explosion
- Issue #1381 closed by veverkap as "completed" on Apr 6 at 01:27
- Today output: 2 PRs (#1448, #1453) vs 18+ on Apr 5
- **89% volume reduction — FIXED**

## 🔴 NEW CRITICAL: GH_AW_AGENT_TOKEN Permission Failure
- `duplicate-code-detector` run failed to assign Copilot to issues #1449 and #1451
- Error: ERR_PERMISSION: copilot coding agent is not available for this repository
- Issue created: #1452 (tracking failure)
- Affects: any workflow using Copilot assignment via GH_AW_AGENT_TOKEN

## 🌟 Outstanding: Security Sprint (Human + Copilot collaboration)
- veverkap raised 6 security issues (#1423-#1429) based on security review
- Copilot fixed all 6 within hours: PRs #1421-#1435 all merged today
- 100% merge rate for security fixes - fastest resolution pipeline seen

## Unresolved Issues from Previous Runs
- 🟡 ci-coach: 7+ consecutive no-ops (runs #31–#37+) — still recommending weekly schedule
- 🟡 daily-workflow-updater: gh CLI not available — still permanent failure
- 🟠 6 uncompiled workflows (see shared-alerts.md)

## Agent Scores This Run
- duplicate-code-detector: B+ (quality high, Copilot assignment failed today)
- daily-doc-updater: A- (resolved, excellent today)
- issue-triage: A (3 correct no-ops, no false positives)
- schema-consistency-checker: A (correct no-op)
- dependabot-burner: A (correct no-op)
- dead-code-remover: A- (correct no-op today)
- ci-coach: C (7 no-ops, schedule too aggressive)
- daily-workflow-updater: F (permanent failure)

## Issues Created This Run
- Performance report discussion (this run)
