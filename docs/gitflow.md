# Multi-Language Gitflow Architecture

This document describes the multi-language branch architecture and Gitflow for
`please-rules`.

---

## 1. Background & Rationale

Please plugins require a `[PluginDefinition]` in the repository's root
`.plzconfig`:

```ini
[PluginDefinition]
Name = rust
```

Because Please allows only one `[PluginDefinition]` per repository root, hosting
multiple language plugins in a single Git repository creates configuration
collisions if all languages live on the same branch.

To provide clean, dedicated plugins for multiple languages (e.g., Rust, Kotlin,
Python) while keeping all code under the single `please-rules` repository, we
use a **Hub-and-Spoke branch architecture**.

---

## 2. Hub-and-Spoke Architecture

```txt
                 [main] (Shared code ONLY: tools/common/, base CI, licenses)
                   │
         ┌─────────┴─────────┐
         ▼ (merge only)      ▼ (merge only)
      [rust]              [kotlin]         <-- Dedicated language branches
         │                   │
   ┌─────┴─────┐       ┌─────┴─────┐
   ▼           ▼       ▼           ▼
feat/x       feat/y  feat/a      feat/b    <-- Feature branches
```

### Branch Responsibilities

- **`main` (The Hub)**: Contains shared infrastructure, common Go packages
  (`tools/common/`), base GitHub Actions workflows, repository linters, and
  documentation. `main` contains **zero** language-specific build rules or
  compilers.
- **Language Branches (The Spokes - `rust`, `kotlin`, etc.)**: Each language
  maintains a permanent, long-lived branch. On its branch:
  - The root `.plzconfig` sets `[PluginDefinition] Name = <lang>` (e.g. `rust`).
  - Configuration keys use clean, idiomatic names (e.g. `RustcTool`,
    `DefaultEdition`) without awkward cross-language prefixes.
  - Language-specific rules live in `build_defs/<lang>/`.
  - Language-specific tools live in `tools/please_<lang>/`.
  - Language-specific integration tests live in `test/<lang>/`.

---

## 3. The Cardinal Rules of Gitflow

1. **One-Way Downstream Merges**: Commits flow only from `main` to language
   branches (`main` $\to$ `rust`).
2. **Never Merge Language Branches into `main`**: Language branches must never
   be merged back into `main`. Merging a language branch into `main` would
   contaminate other language branches with unrelated rules.
3. **Merge, Never Rebase Published Branches**: Always use `git merge main` to
   pull updates from `main`. **Never rebase** published language branches
   (`rust`, `kotlin`), as rebasing rewrites commit SHAs, breaking consumer build
   caches and revision pins.

---

## 4. Development Workflows

### A. Developing a Language Feature

All language work branches off and merges into the corresponding language
branch:

```bash
# 1. Create a feature branch off the language branch
git checkout rust
git pull origin rust
git checkout -b feat/rust-edition-2024

# 2. Make changes and test
./pleasew test //test/rust/...

# 3. Commit and push
git commit -m "feat(rust): add edition 2024 support"
git push origin feat/rust-edition-2024

# 4. Open a Pull Request targeting `rust` (NOT `main`)
```

### B. Updating Common Infrastructure

When shared packages (e.g. `tools/common/archive`) or base CI workflows need
updates:

```bash
# 1. Make changes on main
git checkout main
git pull origin main
git checkout -b fix/archive-tar-xz

# ... edit tools/common/... ...
git commit -m "fix(common): support .tar.xz extraction"
git push origin fix/archive-tar-xz

# 2. Open PR targeting and merge into `main`

# 3. Automated Downstream Propagation:
# GitHub Actions (.github/workflows/sync-downstream.yml) automatically merges
# 'main' into all active language branches (rust, kotlin, etc.), tests them,
# and pushes to remote.

# Manual Downstream Propagation (if syncing locally or offline):
git checkout rust
git pull origin rust
git merge main -m "chore: sync common infrastructure from main"
git push origin rust

git checkout kotlin
git pull origin kotlin
git merge main -m "chore: sync common infrastructure from main"
git push origin kotlin
```

---

## 5. Release, Tagging & Naming Conventions

### Git Release Tags

Because Git tags share a global namespace within a repository, release tags must
be prefixed with the lowercase language identifier:

```txt
<language>-v<semver>
```

Examples:

- `rust-v0.4.0`
- `rust-v0.4.1`
- `kotlin-v0.1.0`

### GitHub Release Names

GitHub Release titles/names MUST be standardized across all languages with the
capitalized language name in brackets followed by the version:

```txt
[<Language>] v<semver>
```

Examples:

- `[Rust] v0.4.0`
- `[Rust] v0.4.1`
- `[Kotlin] v0.1.0`

### Publishing a Release

Releases are fully automated via GitHub Actions
(`.github/workflows/release.yml`):

1. **Tag and Push**:

   ```bash
   git checkout rust
   git pull origin rust
   git tag rust-v0.4.0
   git push origin rust-v0.4.0
   ```

2. **Automated Release Pipeline**: Pushing the tag triggers the release
   workflow, which:
   - Runs unit tests and binary smoke checks.
   - Cross-compiles static binaries for `linux_amd64`, `linux_arm64`,
     `darwin_amd64`, and `darwin_arm64`.
   - Generates the `checksums.txt` manifest.
   - Creates or updates the GitHub Release with the standardized title
     `[<Language>] v<semver>` and attaches all binaries and checksums.

Alternatively, creating the release via `gh release create` also pushes the tag
and triggers asset generation:

```bash
gh release create rust-v0.4.0 \
  --target rust \
  --title "[Rust] v0.4.0" \
  --notes "..."
```

---

## 6. Consumer Integration

Consumers declare each language plugin by pointing `revision` to the language's
release tag or branch in `plugins/BUILD`:

```starlark
# plugins/BUILD

plugin_repo(
    name = "rust",
    owner = "wcodesoft",
    plugin = "please-rules",
    revision = "rust-v0.4.0",     # or revision = "rust"
)
```

In the consumer's `.plzconfig`:

```ini
[Plugin "rust"]
Target = //plugins:rust
RustcTool = //third_party/rust:toolchain|rustc
```

In the consumer's `BUILD` files:

```starlark
subinclude("///rust//build_defs:rust")

rust_binary(
    name = "my_app",
    srcs = ["main.rs"],
)
```
