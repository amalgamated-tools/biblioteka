# Agent Performance — 2026-04-05
**Workflow:** agent-performance-analyzer
**Run timestamp:** 2026-04-05T02:15Z

## Ecosystem Snapshot
- 28 workflows tracked (24 in metrics + 4 new: agent-performance-analyzer, daily-team-evolution-insights, daily-workflow-updater, dead-code-remover, pr-nitpick-reviewer)
- Active Apr 4: 13 workflows; Apr 3: 8 workflows
- Success rate: 100% (0 failures, up from 96.3% on Mar 31)
- Total safe outputs Apr 4: 18 (issues: 2, prs: 16)
- PR merge rate: 84%, avg merge time: 1.85h
- Open issues: 1 (down from 4 Apr 4, from 10 Apr 3 — excellent resolution velocity)

## CRITICAL: Uncompiled Workflow (Persistent)
- **daily-team-evolution-insights: NOT COMPILED** — second consecutive day; will not execute until fixed
- Escalation: this alert was flagged in yesterday's run and has not been resolved

## Top Performers
- **update-docs**: Exceptional reasoning quality, 0.4h avg merge, contextually identifies stale PRs
- **duplicate-code-detector**: 75% issue closure rate same-day; consistent 3–4 issues/day
- **daily-accessibility-review**: High issue→PR pipeline; 57% same-day closure rate (Mar 31: 4/7)
- **daily-file-diet**: Clean threshold tracking, proper noop behavior (477-line api.ts = watch item)
- **issue-triage**: Fast, accurate; 7/7 correct Mar 31

## Notable Concerns
1. **daily-doc-updater volume spike**: 0 (Mar 31, noop) → 8 PRs (Apr 3) → 16 PRs (Apr 4, doubled!)
   - Risk: PR fatigue for human reviewers; watch for over-creation pattern
2. **ci-coach perpetual no-op**: Runs #31–#34 all noop; 5 prior optimization PRs have addressed all opportunities
   - Recommendation: reduce to weekly schedule to save resources
3. **issue-triage double-fire on #1248**: Same issue triaged twice in 30 min on Apr 4
   - Root cause: likely edited-issue event firing multiple times

## Issues Created This Run
- 1 performance report discussion created (agent-performance-analyzer)

## Trends vs Previous Report
- PR volume from agents: ↑ 100% (8→16 for daily-doc-updater alone)
- Open issues: ↓ 75% (4→1) — healthy
- Agent count: ↑ 4 new workflows added
- Success rate: maintained at 100%

## Coordination Notes for Other Orchestrators
- Campaign Manager: daily-doc-updater volume spike (16 PRs Apr 4) may affect review bandwidth
- Workflow Health Manager: daily-team-evolution-insights uncompiled — needs compilation fix (day 2)
