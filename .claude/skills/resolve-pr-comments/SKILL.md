---
name: resolve-pr-comments
description: Go through all review comments on the current branch's PR, check if each is still valid, fix valid issues, reply to every comment, and resolve all review threads via the GitHub API.
user-invocable: true
allowed-tools: Read Edit Write Glob Grep Bash Agent
---

# Resolve PR Review Comments

This skill reviews all comments on the pull request associated with the current branch, determines whether each comment is still valid, fixes valid issues, replies to every comment, and resolves all review threads.

## Step 1: Identify the PR

Determine the PR number for the current branch:

```
gh pr list --head <current-branch> --json number,title
```

If no PR is found, inform the user and stop.

## Step 2: Fetch all review threads

Use the GitHub GraphQL API to get all review threads (both resolved and unresolved), including their comments, resolution status, file paths, and line numbers:

```
gh api graphql -f query='
  query($owner: String!, $repo: String!, $pr: Int!) {
    repository(owner: $owner, name: $repo) {
      pullRequest(number: $pr) {
        reviewThreads(first: 100) {
          nodes {
            id
            isResolved
            comments(first: 50) {
              nodes {
                body
                author { login }
                path
                line
              }
            }
          }
        }
      }
    }
  }
'
```

Extract the repository owner and name from the git remote URL.

## Step 3: Process each unresolved thread

For each **unresolved** review thread:

1. **Read the comment** to understand what the reviewer is requesting.
2. **Read the referenced file and line** to check the current state of the code.
3. **Determine validity**:
   - If the issue described in the comment **still exists** in the current code, it is **valid** -- fix it.
   - If the issue has **already been addressed** (by a prior commit or another fix), it is **no longer valid**.
4. **Fix valid issues**:
   - Make the necessary code changes.
   - Run `make lint` to verify changes pass linting.
   - Stage and commit the fix with an appropriate conventional commit message.
5. **Reply to the comment** using the GitHub API:
   - For **fixed** issues: reply explaining what was changed.
   - For **already resolved** issues: reply stating the issue was already addressed and explain why.

```
gh api repos/{owner}/{repo}/pulls/{pr}/comments/{comment_id}/replies -f body='<reply>'
```

6. **Resolve the thread** using the GraphQL API:

```
gh api graphql -f query='
  mutation($threadId: ID!) {
    resolveReviewThread(input: {threadId: $threadId}) {
      thread { isResolved }
    }
  }
' -f threadId='<thread_id>'
```

## Step 4: Handle already-resolved threads

For threads that are **already resolved**, skip them -- do not reply or take action.

## Step 5: Push changes

After all comments have been processed:

1. If any code fixes were made, push the changes to the remote branch.
2. Provide a summary table of all threads processed, showing:
   - The comment summary
   - Whether it was fixed or already resolved
   - The action taken

## Important notes

- Always read the current state of the code before deciding if a comment is valid. Do not assume based on the comment alone.
- Group related fixes into a single commit when possible.
- Run `make lint` after making changes to ensure they pass linting before committing.
- Use conventional commit messages (e.g., `fix(handlers): ...`, `refactor(db): ...`).
- Do not include a Co-Authored-By trailer in commit messages.
- When replying to comments, be concise and specific about what was changed or why the issue no longer applies.
