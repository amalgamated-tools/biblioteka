# Agent Performance — 2026-04-09
**Workflow:** agent-performance-analyzer
**Run timestamp:** 2026-04-09T23:44Z

## Ecosystem Snapshot
- 24 workflows tracked (31 in Mar-23 inventory; 7 inactive/manual)
- 13/13 scheduled daily runs: 100% success rate (Apr 4 baseline)
- Active open issues carried from Apr 8: 5 tracked
- Open agent PRs awaiting merge: ~4-6 (fix PRs #1544, #1545 + update-docs duplicates)
- PR merge rate: 84% (1.85h avg merge time) — ecosystem healthy
- Average agent quality score: 82.2/100

## Top Performers (A-grade)
- agentic-maintenance: A+ (11 runs/day, 100%, infrastructure backbone)
- daily-accessibility-review: A (2 issues/day, consistent, targeted)
- code-simplifier: A (PRs created, 100% success)
- dead-code-remover: A (sustained)
- daily-security-review: A (sustained)
- daily-grumpy-reviewer: A (sustained)
- daily-nitpick-reviewer: A (sustained)
- daily-documentation-updater: A (sustained improvement since Apr 6 fix)
- ci-coach: A- (noop when no issues - correct behavior)
- schema-consistency-checker: A- (2 actionable issues/day)
- daily-semgrep-scan: A- (100% success, appropriate noop)

## ⚠️ CRITICAL (Day 2): update-docs Duplicate PR Race Condition
- 3 identical PRs (#1538, #1539, #1547) still open/unresolved
- Root cause: TOCTOU race — concurrent Test completions trigger simultaneous runs
- No fix applied yet; dedup logic inadequate for race window
- Action: Implement lock-based or post-create dedup check

## ⚠️ MEDIUM (Day 4): GH_AW_AGENT_TOKEN Copilot Assignment
- Day 4 without resolution
- Issues #1452 (old) + #1551 (Apr 8) both open
- Copilot auto-fix pipeline degraded; affects duplicate-code-detector + daily-workflow-updater
- Action: Verify org plan + GH_AW_AGENT_TOKEN scope

## ⚠️ PENDING: merge-conflict-resolver Fix PR #1544
- Config bug: push_to_pull_request_branch fails in workflow_dispatch
- Fix open, awaiting merge

## ⚠️ PENDING: daily-malicious-code-scan Fix PR #1545
- Compilation flag issue; runs succeed but compiled=No in status
- Fix open, awaiting merge

## 🟡 MONITORING: daily-workflow-updater Phantom Failure
- Issue #1526, Copilot-assigned, expires Apr 15
- Run 24134169785 shows outer GHA success but inner step failure

## 🟡 NEW: daily-repo-chronicle Zero Runs
- Zero runs observed in Apr 3-4 tracking window
- Schedule: daily — should be running; possible silent failure
- No issue created yet; recommend investigation

## Issues Created This Run
- None (existing tracking issues cover active findings)
- Discussion: Created weekly performance report

## Performance Trend
- Apr 3→Apr 4: 100% success rate stable
- daily-doc-updater: 8 issues Apr 3 → 16 PRs Apr 4 → fixed Apr 6 (oscillation pattern)
- PR queue: 14 open (Apr 7) → ~6 open (Apr 8) — significant improvement
