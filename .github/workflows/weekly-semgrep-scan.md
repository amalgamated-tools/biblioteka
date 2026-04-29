---
description: Weekly Semgrep security scan for SQL injection and other vulnerabilities
name: Weekly Semgrep Scan
imports:
  - shared/security-analysis-base.md
  - shared/mcp/semgrep.md
  - shared/observability-otlp.md
on:
  schedule: weekly on monday around 21:00
  workflow_dispatch:
permissions:
  contents: read
  issues: read
  pull-requests: read
  security-events: read
safe-outputs:
  create-code-scanning-alert:
    driver: "Semgrep Security Scanner"
<<<<<<< current (local changes)
  noop:
    report-as-issue: false
source: github/gh-aw/.github/workflows/daily-semgrep-scan.md@525b5b77a444146979ba1759b2a23d72934bc6fc
||||||| base (original)
source: github/gh-aw/.github/workflows/daily-semgrep-scan.md@525b5b77a444146979ba1759b2a23d72934bc6fc
=======

tools:
  cli-proxy: true

source: github/gh-aw/.github/workflows/daily-semgrep-scan.md@7f977f17bd6948b45209fab4719566b435f8ecc5
>>>>>>> new (upstream)
---

Scan the repository for SQL injection vulnerabilities using Semgrep.

{{#import shared/noop-reminder.md}}
