# Agent Performance — 2026-04-10
**Run:** 2026-04-10T23:44Z

## Snapshot
- 19 scheduled workflows; 68% success rate today (2 hard fails + 4 noop-omission fails)
- PR merge rate: 84% — healthy
- Top performers: agentic-maintenance (A+), daily-doc-updater (A), daily-grumpy-reviewer (A)

## CRITICAL: Noop Omission Epidemic (Apr 10, Day 1)
- 4 agents completed but forgot to call noop: daily-nitpick-reviewer (24 turns, 2.45M tokens!), dead-code-remover (17t, 734k), daily-file-diet (10t, 431k), code-simplifier
- Copilot fix PRs: #1635, #1636 (partial) — need merge + extend to remaining 2 agents
- Fix: Add mandatory noop instruction to ALL reviewer/analyzer prompts

## CRITICAL: Duplicate Code Detector Hard Fail
- Error: CODEX_API_KEY / OPENAI_API_KEY not configured — fails daily
- Fix: Configure secret in repo/org settings (5 min operational fix)

## MEDIUM: Daily Accessibility Review Crash (Run 51, Apr 10)
- Orphan process termination; was succeeding runs 47-50 (Apr 6-9)
- Investigate crash/OOM cause

## Pending Merges: #1635, #1636 (Copilot noop fixes)

## Resolved This Period
- update-docs duplicate PRs: RESOLVED (1 targeted PR today)
- Copilot PR backlog: RESOLVED (~5 open, down from 14)
