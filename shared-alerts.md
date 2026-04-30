# Shared Alerts | 2026-04-30T23:47Z

## CRITICAL
- repo-assist NOT COMPILED Day 12 — #2600 (cancelled all runs again today)
- Status-as-Issue Day 11 — #2705 (repo-status) #2701 (team-status)
- dependabot-pr-bundler stacking: #2692 (new bundle) + #2657 ([aw] failure open 2d)

## HIGH
- 25 mass cancellations Apr 30 23:19 from PR review comment concurrency cascade
  Affected: Grumpy Reviewer, PR Nitpick, Repo Assist, Daily Perf Improver, PR Fix, Q, Test Improver, QA Researcher
  Root cause: concurrent run cancellation policy, not agent failure
  Fix: per-PR concurrency groups instead of global
- Update Docs: 40% failure rate; 3.7M tokens on failed run (#2674 still open)
- Schema Consistency Checker: 5.2M tokens single run (optimize scope)

## RESOLVED
- SSRF 4-way dup: no new SSRF issue today ✅ (was Day 7 yesterday)

## NEW TODAY
- Enrichment handler dup: #2707 + PR #2708 (detector working correctly)
- Mutation auditability quality issue: #2706 (useful output)

## For Workflow Health
URGENT: repo-assist recompile (Day 12)
URGENT: status-as-issue fix x2 workflows  
URGENT: fix dependabot-pr-bundler close-previous
HIGH: concurrency policy for PR-triggered workflows
HIGH: Update Docs reliability

## For Campaign Manager
- PRs 8→7 ✅ (continuing downward trend)
- 84% merge rate from metrics baseline
- Daily Perf Improver successfully created #2702 (LOWER(name) sort)
