---
description: Daily Semgrep security scan for SQL injection and other vulnerabilities
name: Daily Semgrep Scan
imports:
  - shared/mood.md
  - shared/mcp/semgrep.md
on:
  schedule: daily
  workflow_dispatch:
permissions:
  contents: read
  issues: read
  pull-requests: read
  security-events: read
safe-outputs:
  create-code-scanning-alert:
    driver: "Semgrep Security Scanner"
source: github/gh-aw/.github/workflows/daily-semgrep-scan.md@852cb06ad52958b402ed982b69957ffc57ca0619
---

Scan the repository for SQL injection vulnerabilities using Semgrep.
