# CI layout

Who owns which CI file, and how a language branch builds on what `main`
provides. The aim is that `main` and a language branch never edit the same
lines, so the downstream sync (`main` to each language branch) does not
conflict.

## Ownership

| File                                    | Lives on            | Edited on           |
| :-------------------------------------- | :------------------ | :------------------ |
| `.github/workflows/ci-main.yml`         | `main`              | `main`              |
| `.github/workflows/release.yml`         | `main`              | `main` only         |
| `.github/workflows/sync-downstream.yml` | `main`              | `main`              |
| `.github/actions/*`                     | `main`              | `main` only         |
| `.github/workflows/ci-<lang>.yml`       | its language branch | its language branch |

`main` does not carry `ci-<lang>.yml`: its triggers only fire for its own
branch, so it never ran on `main`, and a copy there was edited from both sides.

**Never edit a shared file on a language branch.** `release.yml` and the actions
in `.github/actions/` come from `main` through the sync; change them on `main`
and the branches get the change with the next merge. `release.yml` is the same
on every branch: it builds each tool with Please (`//tools/<tool>:<tool>`).

## Shared composite actions

A language's `ci-<lang>.yml` calls these after checking the repository out:

- `.github/actions/format-check`: `gofmt` on the Go tools and `prettier` on the
  Markdown.
- `.github/actions/please-build-test`: builds `build-targets` with Please and
  runs `test-targets` (both default to `//...`).

Anything specific to a language stays in its own file: a formatter (`rustfmt`),
a toolchain to set up (Swift), caches, and the coverage smoke tests.

## Starting a new language branch

Create `.github/workflows/ci-<lang>.yml` on the new branch from this template
(here for `python`):

```yaml
name: CI (Python)

on:
  push:
    branches: ["python", "release/python*"]
  pull_request:
    branches: ["python", "release/python*"]

jobs:
  fmt:
    name: Python & Docs Formatting
    runs-on: ubuntu-latest
    steps:
      - name: Checkout repository
        uses: actions/checkout@v5

      - name: Check formatting
        uses: ./.github/actions/format-check

  test:
    name: Please Build & Test (Python)
    runs-on: ubuntu-latest
    steps:
      - name: Checkout repository
        uses: actions/checkout@v5

      - name: Build and test
        uses: ./.github/actions/please-build-test
        with:
          build-targets: //tools/please_python/... //test/python/...
```

Add the language to the matrix of `sync-downstream.yml` and to the tag patterns
and options of `release.yml` on `main`.

## The downstream sync

`sync-downstream.yml` merges `main` into every language branch on each push to
`main`, runs the build and tests there, and pushes. It stops with an error if
the merge conflicts. Resolve it by hand: in the language branch run
`git merge main`, fix the conflicts, and push (never rebase a published branch).
A conflict means a shared file was edited on a branch, so move that change to
`main`.
