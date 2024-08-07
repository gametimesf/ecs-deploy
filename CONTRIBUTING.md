# Contributing

## Table of Contents

[Dependencies](#dependencies)  
[Branching](#branching)  
[Commits](#commits)  
[Fixing bad commit messages](#fixing-bad-commit-messages)  
[Pull Requests](#pull-requests)

## Dependencies

Before you begin, make sure you have installed the required dependencies, which can be found in the [README](README.md).

## Branching

This repository uses [trunk-based
development](https://www.atlassian.com/continuous-delivery/continuous-integration/trunk-based-development),
in which small, short-lived feature branches are created off of and regularly
merged back into the `master` branch, which is the only branch that should be
long-lived.

Be sure to pull in locally the most recent changes from the remote `master`
branch before creating a new branch.

When creating a new branch, avoid using special characters such as `/`.

## Commits

Ensure your commits follow [Conventional
Commit](https://www.conventionalcommits.org/en/v1.0.0/) guidelines. You can find
all the available prefixes allowed in this repository in the [.commitlintrc.js
file](.commitlintrc.js) (defined in `validTypes`).

Pre-commit will fail if a commit with a bad message is made locally.

### Fixing bad commit messages

If the check on your PR fails due to a bad commit message, you can fix your commits with an
interactive rebase:

```bash
# checkout your branch
git checkout <branch-name>

# rebase interactively from this branch to the default branch
git rebase -i master
```

You'll get something that looks like this:

```
pick 1a2b3c4 adjust the env vars
pick 2b3c4d5 debug
pick 3c4d5e6 cache frequently requested listings

# Rebase 3c4d5e6..1a2b3c4 onto 3c4d5e6 (3 commands)
#
# Commands:
# p, pick <commit> = use commit
# r, reword <commit> = use commit, but edit the commit message
# e, edit <commit> = use commit, but stop for amending
# s, squash <commit> = use commit, but meld into previous commit
# f, fixup <commit> = like "squash", but discard this commit's log message
# x, exec <command> = run command (the rest of the line) using shell
# b, break = stop here (continue rebase later with 'git rebase --continue')
# d, drop <commit> = remove commit
```

The included instructional comment (it's always there) is very helpful. You
might adjust your commits to look like this:

```
r 1a2b3c4 adjust the env vars
f 2b3c4d5 debug
r 3c4d5e6 cache frequently requested listings
```

Then save and quit. Notice the "r" next to commits we want to reword, and "f"
next to the useless debug commit that will effectively be hidden. You'll be
prompted to edit the commit messages for the commits you marked with "r". In
this case they should be something like this:

```
fix: adjust the env vars

feat: cache frequently requested listings
```

Once the rebase is complete, you'll need to force push your branch to the remote
(because we changed git history):

```bash
git push -f
```

## Pull Requests

Any permanent changes (additions, removals, and/or modifications) to
resources/services for any environment, should be reflected in the
code and merged into the `master` branch following a pull request.

Please fill out the PR template before requesting a code review. When a PR is
opened, github actions will automatically run all required checks.

Pull requests require at least 2 approvals (at least one must be a @gametimesf/platform member) in order to be merged.
