---
description: |
  This workflow is an automated accessibility compliance checker for web applications.
  Reviews websites against WCAG 2.2 guidelines using Playwright browser automation.
  Identifies accessibility issues and creates GitHub issues with detailed
  findings and remediation recommendations. Helps maintain accessibility standards
  continuously throughout the development cycle.

on:
  schedule: every 24h
  workflow_dispatch:

permissions: read-all

network:
  allowed:
  - defaults
  - node
  - python
  - go

safe-outputs:
  mentions: false
  allowed-github-references: []
  create-issue:
    title-prefix: "fix(accessibility): "
    labels: [a11y, automated-analysis]
    assignees: [copilot]
    max: 15
    expires: 7d
    group: true
    close-older-issues: true
  noop:
    report-as-issue: false

tools:
  playwright:
  web-fetch:
  github:
    toolsets: [all]

timeout-minutes: 15

steps:
  - name: Checkout repository
    uses: actions/checkout@de0fac2e4500dabe0009e67214ff5f5447ce83dd
    with:
      fetch-depth: 0
      persist-credentials: false    

  - name: Set up Node.js
    uses: actions/setup-node@53b83947a5a98c8d113130e565377fae1a50d02f
    with:
      node-version: "22"

  - name: Install pnpm
    uses: pnpm/action-setup@fc06bc1257f339d1d5d8b3a19a8cae5388b55320
    with:
      version: 10.32.1

  - name: Install frontend dependencies
    working-directory: ./frontend
    run: pnpm install --frozen-lockfile

  - name: Build frontend
    working-directory: ./frontend
    run: pnpm run build

  - name: Set up Go
    uses: actions/setup-go@4b73464bb391d4059bd26b0524d20df3927bd417
    with:
      go-version: 1.26.2
      cache: true
      cache-dependency-path: go.sum

  - name: Build Go binary
    run: go build -o biblioteka ./cmd/server    

  - name: Build and run app in background
    run: |
      echo "Building and running the app in background..."
      PORT=3000 JWT_SECRET=github-actions ./biblioteka -mode server &
      echo "Waiting for server to be ready..."
      server_ready=false
      for i in $(seq 1 30); do
        echo "Checking if server is up (attempt $i)..."
        if curl -sf http://localhost:3000/api/health; then
          server_ready=true
          break
        fi
        sleep 1
      done
      if [ "$server_ready" = false ]; then
        echo "Error: server did not become ready at http://localhost:3000/api/health within 30 seconds."
        exit 1
      fi
source: githubnext/agentics/workflows/daily-accessibility-review.md@5423b1a98cf7ee7bf7837e434903c3e1d15d7a07
engine: copilot
---

# Daily Accessibility Review

Your name is ${{ github.workflow }}.  Your job is to review a website for accessibility best
practices.  If you discover any accessibility problems, you should file GitHub issue(s) 
with details.

Our team uses the Web Content Accessibility Guidelines (WCAG) 2.2.  You may 
refer to these as necessary by browsing to https://www.w3.org/TR/WCAG22/ using
the WebFetch tool.  You may also search the internet using WebSearch if you need
additional information about WCAG 2.2.

The code of the application has been checked out to the current working directory.

Steps:

1. Use the Playwright MCP tool to browse to `localhost:3000`. Review the website for accessibility problems by navigating around, clicking
  links, pressing keys, taking snapshots and/or screenshots to review, etc. using the appropriate Playwright MCP commands.

2. Review the source code of the application to look for accessibility issues in the code.  Use the Grep, LS, Read, etc. tools.

3. Use the GitHub MCP tool to create issues for any accessibility problems you find.  Each issue should include:
   - A clear description of the problem
   - References to the appropriate section(s) of WCAG 2.2 that are violated
   - Any relevant code snippets that illustrate the issue
   - Recommendations for how to fix the issue, if possible
   - Screenshots or snapshots that illustrate the issue, if possible
   - A notice that the PR should include a reference to the issue number, e.g., "Fixes #123"