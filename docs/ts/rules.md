# TypeScript Rules Reference

Comprehensive reference for all TypeScript build definitions provided by
`please-rules`.

---

## `ts_toolchain`

Assembles a 100% hermetic Deno compiler and runtime sysroot.

```starlark
ts_toolchain(
    name = "toolchain",
    version = "2.2.3",
    visibility = ["PUBLIC"],
)
```

### Arguments

| Argument          | Type  | Default       | Description                                              |
| :---------------- | :---- | :------------ | :------------------------------------------------------- |
| `name`            | `str` | `"toolchain"` | Name of the toolchain target.                            |
| `version`         | `str` | `"2.2.3"`     | Deno release version.                                    |
| `deno_url`        | `str` | `""`          | Optional custom download URL for the Deno archive.       |
| `deno_hash`       | `str` | `""`          | Optional SHA-256 hash for the archive.                   |
| `target_platform` | `str` | `""`          | Optional target platform override (`linux_amd64`, etc.). |

---

## `ts_module`

Downloads an npm package tarball, extracts it, and creates metadata for
downstream targets.

```starlark
ts_module(
    name = "clsx",
    package = "clsx",
    version = "2.1.1",
    hashes = ["6320104b491ca645120e8f99589ce663ff1fb337ba80066719a011bd95320303"],
)
```

### Arguments

| Argument  | Type   | Default    | Description                                                                     |
| :-------- | :----- | :--------- | :------------------------------------------------------------------------------ |
| `name`    | `str`  | (required) | Target name.                                                                    |
| `package` | `str`  | `""`       | NPM package specifier (e.g. `"clsx"`, `"@preact/signals"`). Defaults to `name`. |
| `version` | `str`  | `""`       | NPM package version.                                                            |
| `hashes`  | `list` | `[]`       | SHA-256 integrity checksums for the downloaded tarball.                         |
| `url`     | `str`  | `""`       | Custom URL override.                                                            |
| `deps`    | `list` | `[]`       | Dependencies of this module.                                                    |

---

## `ts_library`

Compiles and type-checks TypeScript / JavaScript files into a modular library.

```starlark
ts_library(
    name = "calculator",
    srcs = ["calculator.ts"],
    module_name = "@domain/calculator",
    deps = [":utils"],
)
```

### Arguments

| Argument      | Type   | Default    | Description                                           |
| :------------ | :----- | :--------- | :---------------------------------------------------- |
| `name`        | `str`  | (required) | Target name.                                          |
| `srcs`        | `list` | (required) | List of `.ts`, `.tsx`, `.js`, or `.mjs` source files. |
| `deps`        | `list` | `[]`       | Dependent libraries or `ts_module` targets.           |
| `module_name` | `str`  | `""`       | Module alias for bare import specifiers.              |
| `flags`       | `list` | `[]`       | Additional flags passed to `deno check`.              |

---

## `ts_binary`

Compiles a TypeScript program into a standalone native executable via
`deno compile`.

```starlark
ts_binary(
    name = "app",
    main = "main.ts",
    deps = [":calculator"],
)
```

### Arguments

| Argument | Type   | Default    | Description                                              |
| :------- | :----- | :--------- | :------------------------------------------------------- |
| `name`   | `str`  | (required) | Target name.                                             |
| `main`   | `str`  | (required) | Entrypoint source file.                                  |
| `srcs`   | `list` | `[]`       | Additional source files.                                 |
| `deps`   | `list` | `[]`       | Dependent libraries.                                     |
| `flags`  | `list` | `[]`       | Flags passed to `deno compile` (e.g. `["--allow-net"]`). |

---

## `ts_bundle`

Bundles TypeScript / JavaScript sources into a single distribution asset.

```starlark
ts_bundle(
    name = "app_bundle",
    main = "index.tsx",
    deps = [":components"],
    format = "esm",
    minify = True,
)
```

### Arguments

| Argument       | Type   | Default    | Description                                          |
| :------------- | :----- | :--------- | :--------------------------------------------------- |
| `name`         | `str`  | (required) | Target name.                                         |
| `main`         | `str`  | (required) | Entrypoint source file.                              |
| `out`          | `str`  | `""`       | Output bundle path (defaults to `<name>.bundle.js`). |
| `format`       | `str`  | `"esm"`    | Bundle format (`"esm"` or `"iife"`).                 |
| `minify`       | `bool` | `False`    | Whether to minify bundled output.                    |
| `sourcemap`    | `bool` | `False`    | Whether to generate source maps.                     |
| `bundler_tool` | `str`  | `""`       | Optional external bundler executable (e.g. esbuild). |

---

## `ts_test`

Runs hermetic unit or simulated DOM tests using Deno test runner or Vitest.

```starlark
ts_test(
    name = "calculator_test",
    srcs = ["calculator_test.ts"],
    deps = [":calculator"],
)
```

### Arguments

| Argument | Type   | Default    | Description                           |
| :------- | :----- | :--------- | :------------------------------------ |
| `name`   | `str`  | (required) | Target name.                          |
| `srcs`   | `list` | (required) | Test source files.                    |
| `deps`   | `list` | `[]`       | Dependent libraries.                  |
| `data`   | `list` | `[]`       | Runtime test data.                    |
| `runner` | `str`  | `"deno"`   | Test runner (`"deno"` or `"vitest"`). |

---

## `ts_browser_test`

Runs hermetic in-browser tests with headless browser binaries.

```starlark
ts_browser_test(
    name = "counter_browser_test",
    srcs = ["counter.browser.test.tsx"],
    browser = "chromium",
    deps = [":counter"],
    data = ["//tools/playwright:chromium"],
)
```
