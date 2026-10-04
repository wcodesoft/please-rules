# TypeScript Rules Usage & Guides

This guide walks through creating, testing, and bundling TypeScript applications
in Please.

---

## 1. Creating Libraries and Aliases

Declare modular libraries with explicit `module_name` attributes so downstream
targets can import using bare specifiers:

```starlark
# src/math/BUILD
ts_library(
    name = "math",
    srcs = ["calculator.ts"],
    module_name = "@app/math",
    visibility = ["PUBLIC"],
)
```

Inside `calculator.ts`:

```typescript
export function add(a: number, b: number): number {
  return a + b;
}
```

Consumers can import either via alias or relative paths:

```typescript
import { add } from "@app/math";
```

### Direct Subpath Imports Without Barrel Files

When your library contains multiple component files, consumers can import
individual files directly without needing an `index.ts` barrel file:

```starlark
ts_library(
    name = "common",
    srcs = [
        "Badge.ts",
        "MetricCard.ts",
        "Modal.ts",
    ],
    module_name = "@repo/dashboard/components/common",
)
```

Downstream targets can import individual components directly:

```typescript
import { MetricCard } from "@repo/dashboard/components/common/MetricCard";
import { Badge } from "@repo/dashboard/components/common/Badge";
```

### Automatic Module Name Derivation

You can omit `module_name` on `ts_library` targets. If
`[Plugin "ts"] ModulePrefix = @repo` is set in `.plzconfig`, the module name is
automatically derived:

- If target `name` matches the directory name or is `lib`:
  `{ModulePrefix}/{package}` (e.g. `@repo/dashboard/components/common`).
- Otherwise: `{ModulePrefix}/{package}/{name}`.

```starlark
# In src/components/BUILD
ts_library(
    name = "components",
    srcs = ["Button.tsx", "Input.tsx"],
    # module_name is automatically derived as @repo/src/components
)
```

---

## 2. Using NPM Packages with `ts_module`

Declare third-party packages in `third_party/js/BUILD`:

```starlark
# third_party/js/BUILD
ts_module(
    name = "clsx",
    package = "clsx",
    version = "2.1.1",
    hashes = ["6320104b491ca645120e8f99589ce663ff1fb337ba80066719a011bd95320303"],
    visibility = ["PUBLIC"],
)
```

In your application library:

```starlark
ts_library(
    name = "components",
    srcs = ["Button.tsx"],
    deps = ["//third_party/js:clsx"],
)
```

---

## 3. Running Unit Tests

Define tests with `ts_test`:

```starlark
ts_test(
    name = "calculator_test",
    srcs = ["calculator_test.ts"],
    deps = [":math"],
)
```

Inside `calculator_test.ts`:

```typescript
import { add } from "@app/math";

Deno.test("add calculates properly", () => {
  if (add(2, 2) !== 4) {
    throw new Error("expected 2 + 2 = 4");
  }
});
```

Run tests via Please:

```bash
./pleasew test //src/math:calculator_test
```

---

## 4. Compiling Standalone Binaries

Compile CLI tools into standalone executables that run without Deno installed:

```starlark
ts_binary(
    name = "cli",
    main = "cli.ts",
    deps = [":math"],
    flags = ["--allow-read"],
)
```

Run the built executable:

```bash
./pleasew run //src/cli:cli
```

---

## Code Coverage (`plz cover`)

### Required `[cover]` setting

Please only reports coverage for files whose extension is listed under `[cover]`
in your `.plzconfig`. The TypeScript plugin cannot set this for you; without it
the report is silently `Total coverage: No data`, or lists files of other
languages instead of yours.

```ini
[cover]
fileextension = .ts
```

If your repository mixes languages, add one `fileextension` line per extension.

### Running

```bash
./pleasew cover //src/...
```

The Deno runner (the default) uses `deno test --coverage`; the Vitest runner
(`runner = "vitest"`) uses the V8 provider (`@vitest/coverage-v8`), which
`vitest_toolchain` caches alongside Vitest so coverage also works offline.

### Report output

- **Location**: Please merges the per-test results into
  `plz-out/log/coverage.xml` (Cobertura XML) and `plz-out/log/coverage.json`
  (per-file line markers plus per-file and per-directory percentages).
- **Contents**: line coverage only. Branch coverage and per-function data are
  not emitted, so consumers cannot join coverage with per-function complexity
  from this report.
- **Format**: both runners write Cobertura XML to `$COVERAGE_FILE`, converted
  from lcov. Vitest writes its lcov report to a scratch directory (Vitest
  empties its reports directory, so it cannot be the test working directory).
- **Vitest note**: when coverage is active the runner writes a `package.json`
  declaring `vitest` and `@vitest/coverage-v8` in the test working directory (if
  none exists) so Deno can resolve the provider.

### Raw lcov export (functions and branches)

Please reduces every coverage report to per-line data. Each `ts_test` also keeps
the coverage tool's own lcov report, which has the function (`FN`/`FNDA`) and
branch (`BRDA`) records that `coverage.xml` and `coverage.json` do not:

- **Location**: `plz-out/bin/<package>/<target name>.lcov`, for example
  `plz-out/bin/test/ts/lib/branches_test.lcov`. It is declared as a test output,
  so it is written next to the test binary.
- **Content**: lcov with repository-relative paths. Both the Deno and the Vitest
  runner provide lines, functions and branches. The file is empty unless the
  test ran under `plz cover`.
- **Lifetime**: the file reflects the last test run that actually executed. Read
  it right after `./pleasew cover --rerun <target>`; do not assume it is
  recreated when Please reuses a cached result.
- **Merging**: reports from several targets can be merged with the shared
  `tools/common/lcov` package. Do not merge branch records of the Deno and
  Vitest runners for the same file: they number the arms of a branch
  differently.

---

## npm Packages Through Deno's Own Resolution (`ts_npm_module`, experimental)

`ts_module` unpacks a package and describes it with a single entry file, which
cannot express CommonJS packages, subpath imports (`highlight.js/lib/core`) or
packages that only have an `exports` map. `ts_npm_module` takes a different
route: Deno resolves the package itself through an `npm:` specifier, and the
rule only has to make the package available **offline**:

```starlark
ts_npm_module(
    name = "ms",
    hashes = ["f6616e15e530ed552f9daa2d3ce71963947c6bc7c98c9b64fd3e673fd02622c6"],
    version = "2.1.3",
)

ts_npm_module(
    name = "debug",
    hashes = ["c803a8ca9b835b7a75f7150ee52f7f640675515bafbe8f5da78fcc0ae12914ac"],
    version = "4.3.7",
    deps = [":ms"],  # every dependency is its own ts_npm_module
)

ts_library(
    name = "humanize",
    srcs = ["humanize.ts"],
    module_name = "@app/humanize",
    deps = [":debug"],
)

ts_test(
    name = "humanize_test",
    srcs = ["humanize_test.ts"],
    deps = [":debug", ":humanize"],  # list the npm module itself, as with ts_module
)
```

- **Hermetic**: the tarball is pinned by `hashes` (sha256, checked by Please).
  The build step runs in the sandbox, without network access, and extracts it
  into a slice of a Deno npm cache. There is no `node_modules` directory and no
  lockfile; the BUILD files are the lock.
- **Dependencies are explicit**: nothing is resolved at build time. If a package
  depends on something that is not in `deps`, the build fails and names it. Each
  module bundles the packages of its dependencies, so a target lists only the
  modules it imports.
- **Resolution is Deno's**: `exports` maps, CommonJS, subpaths and a package's
  own types work as they do for `npm:` specifiers. Imports use the plain package
  name, for example `import createDebug from "debug"`.
- **Offline is enforced**: targets whose import map has npm modules run Deno
  with `--cached-only`. A module missing from a target's `deps` fails at once
  with `npm package not found in cache`; Deno does not download it.
- **Not wired yet**: `ts_bundle` and `ts_binary`, the Vitest and browser
  runners, and a helper that prints the declarations for a package and its
  dependencies. The cache layout (`registry.json`, including its
  `_deno.packumentFormat` key) is internal to Deno, so it is tied to the Deno
  version the plugin pins.

The fixtures in `test/ts/npm_cache` cover a CommonJS package with a transitive
dependency (`debug`), a CommonJS package imported through subpaths
(`highlight.js`), and an ES module package that only has an `exports` map and no
`main` (`@codemirror/legacy-modes`, with its eleven dependencies).
