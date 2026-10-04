# Changelog

All notable changes to the TypeScript / Deno rules plugin will be documented in
this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to
[Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Experimental `ts_npm_module` rule: provides an npm package to Deno's own
  `npm:` resolution, offline. The sha256-pinned tarball is extracted by a build
  step that needs no network access into a slice of a Deno npm cache that
  targets merge into their `DENO_DIR`; no `node_modules`, no lockfile. CommonJS
  packages, subpath imports and packages with only an `exports` map work as Deno
  resolves them. Dependencies are separate `ts_npm_module` targets and the build
  fails if one is missing; targets run with `--cached-only` so a missing module
  fails instead of being downloaded.
- `please_ts unpack --npm-cache` and the `npmcache` package behind it.
- Fixtures for `debug` (CommonJS, transitive dependency), `highlight.js`
  (CommonJS, subpaths) and `@codemirror/legacy-modes` (exports map only).

## [0.6.0] - 2026-10-03

### Added

- `ts_test` keeps the raw lcov report of its coverage tool as a declared test
  output, `plz-out/bin/<package>/<name>.lcov`, with repository-relative paths
  and the function (`FN`/`FNDA`) and branch (`BRDA`) records that Please's
  merged `coverage.xml` and `coverage.json` do not carry. It is written by both
  the Deno and the Vitest runner under `plz cover`, and is empty otherwise.
- `please_ts testrunner --lcov-file` flag.
- Branchy coverage fixtures (`test/ts/lib:branches_test` and
  `branches_vitest_test`) and a CI smoke test asserting the exported functions
  and branches.

### Changed

- The repository `.plzconfig` sets `[cover] fileextension = .ts`.

### Fixed

- `plz cover` now works for `ts_test` targets with `runner = "vitest"`. Vitest's
  lcov report is written to a scratch subdirectory (Vitest empties its reports
  directory, which previously was the test working directory) and converted to
  the Cobertura file Please reads.
- `vitest_toolchain` now also caches `@vitest/coverage-v8` so coverage works
  offline.

### Documentation

- Documented the raw lcov export: location, contents, lifetime and merging.
- Documented the required `[cover]` setting, report location, format and
  contents for TypeScript.
- Added a CI coverage smoke test for the Deno and Vitest runners.

## [0.5.2] - 2026-10-01

### Fixed

- Fixed Vitest failing to resolve nested `module_name` aliases (e.g. a library
  `@scope/app` importing `@scope/app/components/widget`). Generated Vitest
  config now emits ordered, exact-match alias entries so a parent alias no
  longer prefix-captures a nested one.

## [0.5.1] - 2026-09-27

### Fixed

- Fixed timing race condition in `please_ts testrunner` browser test execution
  where CDP client evaluated test scripts before Chromium finished navigating
  from `about:blank` to the test HTML file.
- Explicitly issued `Page.navigate` to local test file URL and polled for
  `file:` protocol readiness and DOM initialization before executing test
  suites.
- Guarded `localStorage` against document access denial exceptions in
  constrained headless CI runner sandboxes.

## [0.5.0] - 2026-09-27

### Added

- Added automatic transitive dependency resolution in `please_ts unpack` and
  `ts_module` via the npm registry (`resolve_transitive = True`), eliminating
  the need to manually declare non-singleton transitive packages in
  `third_party/ts`.
- Synthesized W3C Import Map `scopes` in `please_ts importmap` to encapsulate
  transitive dependencies within private module subpaths (`./.deps/<pkg>`),
  preventing global namespace pollution and supporting conflicting versions of
  shared utilities.
- Implemented explicit target precedence: any top-level explicit `deps` target
  (such as `third_party/ts:react` or `third_party/ts:ms`) automatically
  overrides and redirects private internal module dependencies to the shared
  rule.
- Added automatic singleton and peer dependency hoisting: undeclared peer
  dependencies and singletons are automatically hoisted to the root `imports`
  map.
- Symlinked `.deps` to `node_modules` inside extracted module directories and
  propagated `NODE_PATH` in `please_ts testrunner` to support CommonJS
  `require()` resolution under Deno.
- Added `--unstable-detect-cjs`, `--allow-read`, and `--allow-env` flags to Deno
  test execution and binary compilation for legacy CommonJS interop.

## [0.4.0] - 2026-09-25

### Added

- Added `vitest_toolchain` rule to assemble a hermetic Vitest cache using
  `deno cache npm:vitest@{version}`.
- Added `VitestTool` plugin configuration in `.plzconfig` defaulting to
  `//tools/vitest_toolchain:toolchain`.
- Added support for running `ts_test(..., runner = "vitest")` hermetically
  offline using `--no-remote` and the pre-cached `vitest_toolchain` directory
  (`DENO_DIR`).
- Automatically synthesized `im.Imports["vitest"] = "npm:vitest"` in
  `please_ts compile` when `--vitest-dir` is provided, enabling
  `import { ... } from "vitest"` without manual third-party package definitions.
- Added integration test `//test/ts/lib:calculator_vitest_test` validating
  hermetic Vitest test execution against workspace libraries.

### Changed

- Removed host `$PATH` lookup (`exec.LookPath("vitest")`) in
  `please_ts testrunner`, ensuring all Vitest test runs execute hermetically via
  `deno run --no-remote -A npm:vitest`.
- Filtered out `vitest` and `chai` from Vite's `resolve.alias` synthesis so
  Vitest resolves its own built-in runner modules without collisions, while
  aliasing workspace packages (e.g. `@domain/calculator`).

## [0.3.0] - 2026-09-25

### Added

- Added `browser_toolchain` rule providing hermetic headless Chromium
  (`chrome-headless-shell` v154.0.8037.57) for `linux_amd64`, `linux_arm64`,
  `darwin_amd64`, and `darwin_arm64`.
- Added native standard-library CDP (Chrome DevTools Protocol) WebSocket client
  in Go (`tools/please_ts/testrunner/cdp.go`) to execute in-browser tests
  hermetically without host browser or external runner dependencies.
- Added `runner = "browser"` default for `ts_browser_test` and configured
  `BrowserTool` in `.plzconfig`.
- Extended `please_ts unpack` with `--archive`, `--binary`, and `--symlink` to
  extract `.zip` toolchains and stage binaries without hardcoded shell scripts.
- Added automatic `vitest.config.mjs` synthesis in `runVitest` mapping Please
  import maps into Vite's `resolve.alias` for workspace package resolution.

### Changed

- Upgraded default hermetic Deno toolchain (`DEFAULT_DENO_VERSION`) to `2.9.7`
  across all supported platforms, enabling `node:util.parseEnv` and modern Node
  compatibility for Vitest.

## [0.2.1] - 2026-09-20

### Changed

- Defaulted `ts_bundle` to use `esbuild` via hermetic Deno for bundling,
  minification, and module resolution.
- Synthesize import maps directly into `esbuild` `--alias` flags to accurately
  resolve bare specifiers and subpath package mappings.

### Removed

- Removed naive regex-based `minifyJS` and sequential file concatenation in
  `please_ts bundle` that corrupted code containing URLs and regular
  expressions.

## [0.2.0] - 2026-09-20

### Added

- Support for package-level module mapping in `ts_library` and
  `ts_metadata.json`, allowing individual file imports without barrel index
  files.
- Support for automatic module name derivation in `ts_library`, `ts_binary`,
  `ts_bundle`, `ts_test`, and `ts_browser_test` using configurable
  `ModulePrefix` (`[PluginConfig "module_prefix"]`).

## [0.1.2] - 2026-09-18

### Fixed

- Verified and updated precompiled binary checksums in `tools/BUILD` for
  `please_ts` across `linux_amd64`, `linux_arm64`, `darwin_amd64`, and
  `darwin_arm64`.

## [0.1.1] - 2026-09-17

### Fixed

- Fixed Starlark f-string quote syntax error when passing CLI `flags` in
  `ts_library`, `ts_binary`, `ts_bundle`, `ts_test`, and `ts_browser_test`.
- Standardized CLI flag description and usage from singular `source files` to
  `sources` across `please_ts` subcommands.

## [0.1.0] - 2026-09-13

### Added

- First-class hermetic TypeScript and JavaScript build definitions:
  - `ts_module`: Downloads and extracts npm packages with optional integrity
    verification.
  - `ts_library`: Hermetic type checking via `deno check` and ephemeral import
    maps.
  - `ts_binary`: Standalone self-contained native executable compilation via
    `deno compile`.
  - `ts_bundle`: Web/browser distribution asset bundling.
  - `ts_test`: Sandboxed unit and DOM tests using `deno test` or `vitest` with
    native JUnit XML and Cobertura XML test coverage.
  - `ts_browser_test`: Hermetic in-browser testing using `@vitest/browser` and
    sandboxed browser binaries.
- Hermetic standalone Deno toolchain (`ts_toolchain`) for `linux_amd64`,
  `linux_arm64`, `darwin_amd64`, and `darwin_arm64`.
- `tools/please_ts` orchestrator tool written in Go with commands for `compile`,
  `bundle`, `testrunner`, `unpack`, and `importmap`.
