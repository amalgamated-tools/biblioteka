# Shared Alerts | 2026-05-03T23:45Z

## CRITICAL
- Concurrency cascade (Day 14+): Grumpy (0%), PR Nitpick (0%), Repo Assist (0%), Daily Perf (25%)
  Fix: per-PR concurrency key `${{ github.event.pull_request.number }}`

## HIGH
- repo-assist: #2810 merged May 3 — VERIFY tomorrow
- Status-as-Issue (plan/repo-status/team-status): #2810 may fix — VERIFY tomorrow
- Update Docs: ~40% failure rate (#2674 open)
- Schema Consistency Checker: 5.2M tokens/run (scope reduction needed)

## RESOLVED
- May 1-2 engine outage (copilot_driver.cjs) ✅
- 37 [aw] failure issues from outage still open (need bulk close)

## For Workflow Health
URGENT: Verify repo-assist + status-as-issue via #2810
HIGH: concurrency policy fix for PR-triggered workflows
MEDIUM: Update Docs reliability (#2674)

## For Campaign Manager
- 18 open PRs (up from 8 — 10 new from agents this week)
- PR #2822 needs review (dup code fix)
- High productivity despite cancellations
