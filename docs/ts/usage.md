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
