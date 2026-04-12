# Contributing

This project is being developed as a portfolio showcase for distributed systems, Kubernetes, and security. The goal is to keep the work easy to review, easy to demo, and easy to connect back to the GitHub roadmap.

## Workflow

Use one issue per branch whenever practical.

- Branch format: `codex/issue-<number>-short-name`
- Examples:
  - `codex/issue-23-env-config`
  - `codex/issue-30-auth-endpoints`

Do not implement directly on `main`.

Keep pull requests small.

- Prefer one issue per PR.
- A PR may cover two tightly related issues if the split would make the work harder to understand.
- If a change grows beyond its issue, stop and split the follow-up into another issue.

## Order Of Work

Follow the roadmap in milestone order unless there is a clear dependency reason not to.

1. Baseline Hardening
2. Authentication and Sessions
3. Saved Lineups and Personal Scoreboard
4. Ranked Evaluation Rules and Global Scoreboard
5. Async Simulation with RabbitMQ
6. Kubernetes Platform Foundation
7. Security and Observability
8. Deferred platform work

## Before You Start

- Pick the issue you are working on.
- Confirm the issue has clear acceptance criteria.
- Create a branch named after the issue.
- Read related epic and child issues before changing code.

## Pull Requests

Every PR should:

- Use the title format `<area>: <short summary>`.
- Match the commit message style for the first line.
- Do not prefix PR titles with `[codex]`.
- Link the related issue with `Closes #<number>` or `Refs #<number>`.
- Explain what changed and why.
- Include verification notes:
  - tests run
  - manual checks performed
  - anything not verified
- Call out config, migration, or deployment impact.

Prefer PRs that are easy to scan.

- Keep unrelated refactors out of feature PRs.
- If behavior changes, update docs in the same PR when reasonable.
- If infra and application changes are mixed, explain the dependency clearly.

## Commit Messages

Use this format for commits:

```text
<area>: <short summary>

<detailed summary, most likely a bullet point list>

#<issue number>
```

Guidelines:

- Keep the first line short and specific.
- Use an area that helps identify the subsystem quickly.
- Good area examples:
  - `frontend`
  - `stats`
  - `simulation`
  - `kubernetes`
  - `auth`
  - `security`
  - `docs`
- Prefer bullet points in the detailed summary when the commit does more than one thing.
- End the message with the related issue number on its own line.

Example:

```text
auth: add session-backed login endpoint

- add login handler and credential validation
- create session after successful authentication
- return consistent auth error responses

#30
```

## Review Standard

Before merging, check:

- The PR clearly satisfies the linked issue.
- Config and secrets are handled safely.
- The change strengthens the Kubernetes/security story rather than obscuring it.
- Tests or manual verification are included.
- Docs are updated when behavior or setup changed.

## Implementation Notes

- Prefer environment-driven configuration over hardcoded values.
- Avoid direct commits to `main`.
- Keep the project runnable locally while improving the Kubernetes path.
- Favor simple, explainable architecture choices over premature platform complexity.
