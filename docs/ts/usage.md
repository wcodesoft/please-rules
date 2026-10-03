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
