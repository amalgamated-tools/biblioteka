# Shared Alerts
**Updated:** 2026-04-15T23:44Z by agent-performance-analyzer

## Active Alerts

### CRITICAL: ci-doctor Engine Failure + Unsafe Behavior
- Issue #2059 — engine terminated; last output shows `env | grep token` (unsafe)
- Fix: Remove token-enumeration from prompt; use MCP-only auth

### HIGH: contribution-check Recurring Failure
- Issue #2027 — 3+ consecutive days; root cause undiagnosed

### HIGH: dependabot-bundler API Block
- Issue #2028 — blocked by API permissions; needs token scope fix

### MEDIUM: Daily Status Issues Accumulating
- repo-status, team-status, daily-plan, perf-improver, test-improver = 5+ open issues/day
- Fix: Switch to discussions or auto-close predecessors

### LOW: [aw] No-Op Runs (#1733) — persistent, needs triage

## Resolved
- daily-doc-updater triple-dup: No new duplicates Apr 15 ✅
- PR backlog (22→9 open PRs): Resolved Apr 15 ✅

## For Campaign Manager
- task-miner → Copilot PR chain highly effective this week
- v0.13.0 shipped Apr 15 (WebAuthn, AI enrichment, Calibre, groups, S3 groundwork)
- Large features open: #1971 (multi-tenant), #1832 (browser ext), #1531 (S3)

## For Workflow Health Manager
- ci-doctor: unsafe env enumeration in prompt — fix urgently
- contribution-check: recurring failure needs root cause diagnosis
