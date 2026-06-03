# Agent Instructions

This file contains instructions for AI agents working in this repository.
General contributor workflow, commit messages, and pull request expectations live
in [CONTRIBUTING.md](CONTRIBUTING.md).

## Before Coding

- Read [CONTRIBUTING.md](CONTRIBUTING.md) before changing files.
- Start new work from the latest `origin/main` unless the user explicitly asks
  to stack on another branch.
- Inspect the relevant code and existing patterns before introducing new
  abstractions.
- Keep changes scoped to the issue unless the user explicitly expands the task.
- Do not include fake issue references. If there is no related issue, omit the
  issue line from the commit message and PR body.

## Code Style

- Do not add `from __future__ import annotations` in new or edited Python files
  unless a maintainer explicitly asks for it.
- Prefer existing local patterns over new abstractions.
- Keep comments sparse and useful.

## Pull Request Reviews

- After opening or updating a PR, check the automated reviews from Codex and Gemini.
- Not every review comment must be addressed. If a comment is intentionally
  deferred or rejected, reply on the thread with a clear reason and, when useful,
  link the tracking issue.
- If a review comment identifies a real bug or missing test, update the PR and
  rerun the relevant checks before asking for merge.
