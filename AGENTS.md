# Workspace Rules for AI Agents (`please-rules`)

This repository employs a **Hub-and-Spoke multi-language Gitflow**. All AI
agents operating in this codebase MUST strictly adhere to the following rules:

---

## 1. Branching & Gitflow Rules

1. **`main` is strictly for common / shared code**:
   - `main` contains **ZERO** language-specific rules (`build_defs/<lang>/`),
     compiler tools (`tools/please_<lang>/`), or integration tests
     (`test/<lang>/`).
   - Only shared packages (`tools/common/`), root documentation, repository
     linters, and base CI belong on `main`.

2. **Language branches are permanent and isolated**:
   - Every language lives on its own dedicated long-lived branch (`rust`,
     `kotlin`, `python`, etc.).
   - Feature branches for a language must branch off `<language>` and PR target
     `<language>` (e.g. `feat/rust-edition` $\to$ PR into `rust`).
   - **NEVER target `main` with language-specific PRs or commits.**

3. **Strict One-Way Downstream Merges (`main` $\to$ `<language>`)**:
   - **NEVER merge a language branch into `main`**.
   - Propagate shared infrastructure updates downstream by running
     `git merge main` inside the language branch.
   - **NEVER rebase published language branches** (`git rebase` is forbidden on
     `rust`, `kotlin`, etc.). Rebasing rewrites commit SHAs, breaking consumer
     hashes and cached revisions.

4. **Release Tagging**:
   - Release tags MUST always include the language prefix:
     `<language>-v<semver>` (e.g. `rust-v0.4.0`, `kotlin-v0.1.0`). Never create
     bare tags like `v1.0.0`.

---

## 2. Plugin & Configuration Rules

1. **Plugin Definition Naming**:
   - On a language branch, `.plzconfig` must set
     `[PluginDefinition] Name = <lang>` (e.g. `Name = rust`).
   - Plugin config keys must be standard and un-prefixed on language branches
     (`RustcTool`, `DefaultEdition`, `Coverage`).

2. **Test Layout**:
   - All tests for a language must reside under `test/<lang>/` (e.g.
     `test/rust/bin`, `test/rust/lib`, `test/rust/toolchain`).
   - Never put unnamespaced language tests directly in `test/`.

3. **Portability of Build Definitions**:
   - Build definition files (`.build_defs`) within `build_defs/<lang>/` should
     use repo-relative subincludes (e.g.
     `subinclude("//build_defs/rust:constants")`), avoiding hardcoded subrepo
     aliases like `///rust` so they remain portable.

---

## 3. Reference Skill

For step-by-step procedures on branching, syncing, releases, and safety
checklists, activate the `multi-language-gitflow` skill located at:
[.agents/skills/multi-language-gitflow/SKILL.md](file:///.agents/skills/multi-language-gitflow/SKILL.md)
