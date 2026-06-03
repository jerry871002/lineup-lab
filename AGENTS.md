# Repository Instructions

## Workflow

- Start new work from the latest `origin/main` unless the user explicitly asks to stack on another branch.
- Use one branch per issue when practical. Prefer branch names like `codex/issue-123-short-name`.
- Keep pull requests small and focused. We usually keep PRs to fewer than three commits.
- For review fixes or small cleanup on an existing PR, prefer amending the relevant commit and force-pushing with `--force-with-lease`.
- Do not include fake issue references. If there is no related issue, omit the issue line from the commit message and PR body.

## Commit Messages

Use this format when there is a related issue:

```text
<area>: <short summary>

- <detailed summary bullet>
- <detailed summary bullet>

#<issue number>
```

If there is no related issue, use the same format without the final issue line.

## Pull Requests

- Link the relevant issue in the PR body with `Closes #<issue>` or `Refs #<issue>`.
- Include verification notes for tests, CI checks, or manual validation.
- After opening or updating a PR, check the automated reviews from Codex and Gemini.
- Not every review comment must be addressed. If a comment is intentionally deferred or rejected, reply on the thread with a clear reason and, when useful, link the tracking issue.
- If a review comment identifies a real bug or missing test, update the PR and rerun the relevant checks before asking for merge.

## Before Coding

- Check for this `AGENTS.md` file and follow the nearest applicable instructions.
- Inspect the relevant code and existing patterns before introducing new abstractions.
- Keep changes scoped to the issue unless the user explicitly expands the task.
