---
name: cleaning-up-stale-branches
description: Use when asked to clean up, prune, audit, or delete stale/merged/unneeded git branches, locally or on GitHub, or when "git push --delete" fails with "dst refspec matches more than one"
---

# Cleaning Up Stale Branches

## Overview

Classify every branch before deleting anything. Two traps make naive cleanup wrong in this repo: **squash-merged branches look unmerged to git**, and **release branch names (vX.Y.Z) collide with release tags**, so bare-name remote deletion either errors or risks the wrong ref.

## Process

### 1. Gather state

```bash
git fetch --prune
git branch -a -vv                  # locals + tracking status ("gone" = remote deleted)
gh pr list --state open --json number,headRefName   # NEVER delete a branch with an open PR
git worktree list                  # branches checked out elsewhere can't/shouldn't be deleted
```

### 2. Classify remote branches

```bash
git branch -r --merged origin/main     # safe: fully merged
git branch -r --no-merged origin/main  # needs PR-history check, NOT automatically unsafe
```

For each **unmerged** branch, check its PR history — squash merges don't show as merged in git:

```bash
gh pr list --head "<branch>" --state all --json number,state,title
```

| PR state | Classification |
|----------|---------------|
| `MERGED` | Squash-merged — content is in main, safe to delete |
| `CLOSED` | Abandoned/superseded — safe to delete |
| No PR at all | Potentially unique work — list it for the user, do not delete without explicit approval |

### 3. Confirm with the user

Present the grouped plan (merged / squash-merged / closed-PR / unique-work) and get explicit approval before any remote deletion. Bulk remote deletion will be blocked by the permission classifier without it.

### 4. Delete remote branches — always `refs/heads/`

```bash
git push origin --delete refs/heads/v0.5.4 refs/heads/fix-docker-semver-tags ...
```

Bare names fail with `error: dst refspec v0.5.4 matches more than one` because every release has a same-named tag. Never resolve that by deleting the tag — tags are the published releases.

### 5. Delete local branches

```bash
git branch -d <merged-branches>        # -d for branches git sees as merged
git branch -D <squash-merged-branches> # -D only AFTER step 2 confirmed PR merged/closed
```

Skip any branch checked out in another worktree (e.g. Conductor workspaces under `~/conductor/workspaces/`) — let the owning tool remove its worktree first.

### 6. Verify

```bash
git fetch --prune && git branch -a
git ls-remote --tags origin | tail -5   # confirm release tags intact
```

## Common Mistakes

| Mistake | Reality |
|---------|---------|
| Trusting `--no-merged` as "has unmerged work" | Squash-merged PRs always show unmerged. Check `gh pr list --head` first. |
| `git push origin --delete v0.5.4` (bare name) | Collides with the release tag. Use `refs/heads/v0.5.4`. |
| Deleting a no-PR branch because it's old | Old ≠ landed. Only the user can write that work off. |
| `git branch -D` to silence "not fully merged" | The warning is the safety net. Verify the PR state instead. |
| Deleting/recreating tags to fix ambiguity | Tags are published releases. Never touch them during branch cleanup. |
