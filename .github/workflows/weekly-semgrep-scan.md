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
  security-events: write
safe-outputs:
  create-code-scanning-alert:
    driver: "Semgrep Security Scanner"
  noop:
source: github/gh-aw/.github/workflows/daily-semgrep-scan.md@525b5b77a444146979ba1759b2a23d72934bc6fc
---

Scan the repository for SQL injection vulnerabilities using Semgrep.

**Important**: If no action is needed after completing your analysis, you **MUST** call the `noop` safe-output tool with a brief explanation. Failing to call any safe-output tool is the most common cause of safe-output workflow failures.

```json
{"noop": {"message": "No action needed: [brief explanation of what was analyzed and why]"}}
```
