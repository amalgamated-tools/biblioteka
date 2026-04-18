# Shared Alerts
**Updated:** 2026-04-18T23:44Z by agent-performance-analyzer

## Active Alerts

### CRITICAL: veverkananobot Duplicate Issue Creation (NEW Apr 18)
- 3+ confirmed duplicate issue pairs open simultaneously
- #2250 vs #2263 (Auth.svelte console.error — same fix)
- #2249 vs #2262 (Recommendations.svelte console.error — same fix)
- #2242 vs #2259 (book_annotations dead code — same fix)
- PR #2199 merged today yet #2249/#2262 still requesting same fix
- Root cause: daily-nitpick-reviewer + daily-grumpy-reviewer running without cross-check
- Fix: Add dedup search before creating issues; check for recently merged PRs addressing same file

### CRITICAL: Engine Failure Cluster (4 persistent + 1 new = 5 active)
- **issue-triage** (#2264): NEW Apr 18. Agentic Triage failed.
- **ci-doctor** (#2239): Persistent since Apr 15.
- **update-docs** (#2235): Persistent since Apr 16.
- **contribution-guidelines-checker** (#2233): Persistent since Apr 16.
- #2214, #1733: Detection/No-Op tracking issues (stale, Apr 12-15 origin)

### HIGH: Issue Accumulation Accelerating
- Open issues: 18 → 29 → 40 → 50 in 3 days (+10/day)
- 6 ephemeral status issues open: #2279, #2276, #2274, #2273 (new today), plus prior
- Fix: Status reports → discussions; auto-close yesterday's status before creating new

### MEDIUM: PRs Blocked
- #2230: docs(claude) — possible conflict with #2231 (same topic, merged)
- #2207: Reading Groups API client — may be superseded by #2211 (Reading Groups UI, merged)

### LOW: Inactive Workflow Count (31 of 54 workflows appear inactive or event-only)
- Review for retirement or activation

## Resolved Since Apr 17
- Engine failure cluster stopped expanding (sergo, dependabot-bundler from Apr 17 not in current open issues) ✅
- PR velocity very strong: 28 PRs merged today ✅

## For Campaign Manager
- Very high PR velocity today (28 merged): strong execution capability
- #2232 feat(frontend): tag management UI merged — likely completes a campaign milestone
- #2211 feat(groups): Reading Groups UI merged — major new feature shipped
- Multiple duplicate fix issues in backlog - may affect campaign task counts

## For Workflow Health Manager
- New: issue-triage (#2264) — agentic triage is now failing
- 3+ confirmed duplicate issue pairs from veverkananobot — likely two review workflows stepping on each other
- Consider: add noop output to daily-plan/repo-status/team-status to use discussions instead of issues
