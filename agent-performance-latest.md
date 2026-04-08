# Agent Performance — 2026-04-08
**Workflow:** agent-performance-analyzer
**Run timestamp:** 2026-04-08T23:44Z

## Ecosystem Snapshot
- 33 workflows tracked, 32 compiled (daily-malicious-code-scan PR #1545 open)
- 30/30 scheduled runs today: 100% success rate
- Open issues: 5 (3 duplicate-code-detector outputs + 2 workflow failures)
- Open agent PRs: 6 (3 doc duplicates, 2 CI fixes, 1 refactor)
- PRs merged today: 0 (vs 6 yesterday)
- Agentic Maintenance ran 11 times — all success

## Top Performers (A-grade)
- agentic-maintenance: A+ (11 runs, 100%)
- daily-accessibility-review: A
- code-simplifier: A
- dead-code-remover: A
- dependabot-burner: A
- daily-security-review: A
- daily-grumpy-reviewer: A
- daily-nitpick-reviewer: A
- daily-documentation-updater: A (sustained improvement from Apr 6 fix)
- duplicate-code-detector: A- (quality issues created but Copilot assignment blocked)

## ⚠️ CRITICAL: update-docs Duplicate PR Recurrence
- 3 identical PRs opened in 2 hours (#1538, #1539, #1547)
- TOCTOU race: multiple Test workflow completions trigger simultaneous runs
- Deduplication check passes before sibling run opens its PR
- All 3 have same title: "docs: document TimeoutState base class..."
- Action: Close #1538 + #1539, keep #1547; fix dedup logic in update-docs

## ⚠️ OPEN: GH_AW_AGENT_TOKEN Copilot Assignment (Day 3)
- Issues: #1551 (today), #1452 (historical)
- daily-workflow-updater and duplicate-code-detector affected
- Copilot fix pipeline degraded — issues sit unassigned

## ⚠️ NEW: merge-conflict-resolver Config Bug
- Issue #1541: push_to_pull_request_branch fails in workflow_dispatch context
- Fix PR #1544 open (Copilot-authored), awaiting merge

## 🟡 MONITORING: daily-workflow-updater Phantom Failure
- Run 24134169785 shows success but created issue #1526
- May be inner agentic step failure masked by outer GHA success
- Assigned to Copilot; expires Apr 15

## 🟡 MONITORING: daily-malicious-code-scan Compilation
- Compiled: No in status tool
- Fix PR #1545 open (Copilot-authored), awaiting merge
- Run succeeded today despite compilation flag

## Resolved Since Yesterday
- PR queue reduction: 14 open Copilot PRs yesterday → 6 today
- All other previously-flagged uncompiled workflows: RESOLVED

## Issues Created This Run
- None (no new issues needed; existing tracking covers all findings)
