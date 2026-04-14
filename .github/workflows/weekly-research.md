---
description: |
  This workflow performs research to provide industry insights and competitive analysis.
  Reviews recent code, issues, PRs, industry news, and trends to create comprehensive
  research reports. Covers related products, research papers, market opportunities,
  business analysis, and new ideas. Creates GitHub discussions with findings to inform
  strategic decision-making.

on:
  schedule: weekly on monday
  workflow_dispatch:

permissions: read-all

network: defaults

safe-outputs:
  create-discussion:
    title-prefix: "${{ github.workflow }}"
    category: "ideas"
    max: 3
    close-older-discussions: true
  noop:
    report-as-issue: false    

tools:
  github:
    toolsets: [all]
    min-integrity: none # This workflow is allowed to examine and comment on any issues or PRs
  web-fetch:

timeout-minutes: 15

source: githubnext/agentics/workflows/weekly-research.md@97143ac59cb3a13ef2a77581f929f06719c7402a
---

# Weekly Research

## Job Description

Do a deep research investigation in ${{ github.repository }} repository, and the related industry in general.

- Read selections of the latest code, issues and PRs for this repo.
- Read latest trends and news from the software industry news source on the Web.

Create a new GitHub discussion with title starting with "${{ github.workflow }}" containing a markdown report with

- Interesting news about the area related to this software project.
- Related products and competitive analysis
- Related research papers
- New ideas
- Market opportunities
- Business analysis
- Enjoyable anecdotes

Only a new discussion should be created, no existing discussions should be adjusted.

At the end of the report list write a collapsed section with the following:
- All search queries (web, issues, pulls, content) you used
- All bash commands you executed
- All MCP tools you used


**Important**: If no action is needed after completing your analysis, you **MUST** call the `noop` safe-output tool with a brief explanation. Failing to call any safe-output tool is the most common cause of safe-output workflow failures.

```json
{"noop": {"message": "No action needed: [brief explanation of what was analyzed and why]"}}
```
