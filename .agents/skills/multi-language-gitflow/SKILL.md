---
name: multi-language-gitflow
description: >-
  Enforces and guides the multi-language Hub-and-Spoke Gitflow for please-rules.
  Use when developing features, fixing bugs, updating shared tooling, creating
  releases, merging branches, or managing git operations across language
  branches.
---

# Multi-Language Gitflow Skill

This skill provides step-by-step procedures and guardrails for managing the
multi-language Hub-and-Spoke Gitflow in `please-rules`.

---

## The 5 Inviolable Rules

1. **`main` Contains Zero Language Rules**: `main` is strictly for shared
   infrastructure (`tools/common/`), base CI, linters, root documentation, and
   project config templates. Never commit language-specific rules
   (`build_defs/<lang>/`), compiler wrappers (`tools/please_<lang>/`), or tests
   (`test/<lang>/`) directly to `main`.

2. **Strict One-Way Downstream Merging**: Code flows strictly: `main` $\to$
   `<language-branch>` (e.g. `main` $\to$ `rust`). **NEVER merge a language
   branch into `main`**. Merging any language branch into `main` pollutes other
   language branches upon subsequent syncs.

3. **Never Rebase Published Language Branches**: Always use `git merge main`
   when updating language branches with shared infrastructure. **NEVER run
   `git rebase` on permanent language branches** (`rust`, `kotlin`, etc.).
   Rebasing rewrites commit SHAs, breaking consumer Please caches, pinned
   revisions, and Git tags.

4. **Always Prefix Release Tags**: Git tags share a global namespace across the
   repository. Release tags must always follow the `<language>-v<semver>` format
   (e.g. `rust-v0.4.0`, `kotlin-v0.1.0`). Never create bare tags like `v1.0.0`.

5. **Maintain Standard Plugin Definitions on Language Branches**: On each
   language branch:
   - The root `.plzconfig` must set `[PluginDefinition] Name = <lang>` (e.g.
     `Name = rust`).
   - Config keys must be clean and un-prefixed (e.g. `RustcTool`,
     `DefaultEdition`, `Coverage`), matching Please conventions.
   - Tests must reside under `test/<lang>/`.

---

## Standard Procedures

### Procedure 1: Developing a Language Feature or Fix

When writing rules, tools, or tests for a specific language (e.g. Rust):

1. **Verify Base Branch**: Ensure you branch off the respective language branch,
   **not** `main`:

   ```bash
   git checkout rust
   git pull origin rust
   git checkout -b feat/my-rust-feature
   ```

2. **Implement & Test**: Make changes in `build_defs/rust/`,
   `tools/please_rust/`, or `test/rust/`. Run local tests:

   ```bash
   ./pleasew test //...
   ```

3. **Verify Target Branch for PR**: The Pull Request must target `rust` (or the
   respective language branch). **Targeting `main` is strictly prohibited.**

---

### Procedure 2: Modifying Shared Code on `main` & Syncing Downstream

When updating common tooling (`tools/common/`), base GitHub Actions workflows,
or repository-level documentation:

1. **Make Changes on `main`**:

   ```bash
   git checkout main
   git pull origin main
   git checkout -b feat/shared-archive-update

   # ... edit tools/common/... or base CI ...
   ./pleasew test //...
   git commit -m "feat(common): add support for zstd archives"
   git push origin feat/shared-archive-update
   # PR targets and merges into `main`
   ```

2. **Downstream Propagation**:
   - **Automated**: The GitHub Actions workflow
     `.github/workflows/sync-downstream.yml` automatically merges `main` into
     all active language branches, runs tests, and pushes updates upon push to
     `main`.
   - **Manual**: When syncing locally or resolving conflicts offline:
   ```bash
   # Sync Rust branch
   git checkout rust
   git pull origin rust
   git merge main -m "chore: sync shared infrastructure from main"
   ./pleasew test //...
   git push origin rust

   # Repeat for any other active language branches (e.g. kotlin)
   git checkout kotlin
   git pull origin kotlin
   git merge main -m "chore: sync shared infrastructure from main"
   ./pleasew test //...
   git push origin kotlin
   ```

---

### Procedure 3: Creating a Language Release

1. **Ensure Working Tree is Clean & Tests Pass**:

   ```bash
   git checkout rust
   git pull origin rust
   ./pleasew test //...
   ```

2. **Tag the Release**: Tag directly on the language branch using the language
   prefix:

   ```bash
   git tag rust-v0.4.0
   git push origin rust-v0.4.0
   ```

3. **Publish GitHub Release**: Use the standardized title format
   `[<Language>] v<semver>`:

   ```bash
   gh release create rust-v0.4.0 \
     --target rust \
     --title "[Rust] v0.4.0" \
     --notes "..."
   ```

4. **Verify Consumer Declaration**: Consumers consume this release in
   `plugins/BUILD`:
   ```starlark
   plugin_repo(
       name = "rust",
       owner = "wcodesoft",
       plugin = "please-rules",
       revision = "rust-v0.4.0",
   )
   ```

---

### Procedure 4: Bootstrapping a New Language Branch (e.g. Kotlin)

1. **Branch Off `main`**:

   ```bash
   git checkout main
   git pull origin main
   git checkout -b kotlin
   ```

2. **Scaffold the Language Plugin**:
   - In `.plzconfig`: set `[PluginDefinition] Name = kotlin` and define plugin
     configs.
   - In `build_defs/kotlin/`: create `kotlin.build_defs` and `BUILD`.
   - In `test/kotlin/`: create integration tests and test targets.
   - In `docs/kotlin/`: create architecture, rules, and usage docs.

3. **Verify and Push Branch**:
   ```bash
   ./pleasew test //...
   git commit -m "feat(kotlin): bootstrap kotlin ruleset"
   git push -u origin kotlin
   ```

---

## Safety Checklist Before Any Git Operation

Before running any merge, push, or branch creation command, verify:

- [ ] Am I committing language code to a language branch, and NOT `main`?
- [ ] If merging `main` into a language branch, did I use `git merge main` (and
      NOT `git rebase`)?
- [ ] Is this PR targeting the correct language branch (and NOT `main`)?
- [ ] If creating a tag, does it have the `<language>-v` prefix?
- [ ] Does `.plzconfig` retain `[PluginDefinition] Name = <language>` without
      language prefixes on config keys?
