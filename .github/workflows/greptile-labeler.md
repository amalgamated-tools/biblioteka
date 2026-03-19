---
on:
  pull_request:
    types: [opened, synchronize]
permissions:
      contents: read
      issues: read
      pull-requests: read
engine: copilot
network:
  allowed:
    - defaults
    - go
    - node
tools:
  github:
    toolsets: [default]
safe-outputs:
  add-labels:
  remove-labels:
  add-reviewer:
---

# greptile-labeler

We use a 3rd party PR review app called Greptile. It reviews all PRs and will add comments to files where it suggests changes.

This workflow listens for new PRs and PR updates, and if Greptile has added comments to the PR, it adds a "greptile-changes" label to the PR. This allows us to easily filter for PRs that have been reviewed by Greptile and need attention.

