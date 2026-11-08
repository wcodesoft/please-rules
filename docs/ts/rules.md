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

| Argument             | Type   | Default    | Description                                                                                     |
| :------------------- | :----- | :--------- | :---------------------------------------------------------------------------------------------- |
| `name`               | `str`  | (required) | Target name.                                                                                    |
| `package`            | `str`  | `""`       | NPM package specifier (e.g. `"clsx"`, `"@preact/signals"`). Defaults to `name`.                 |
| `version`            | `str`  | `""`       | NPM package version.                                                                            |
| `hashes`             | `list` | `[]`       | SHA-256 integrity checksums for the downloaded tarball.                                         |
| `url`                | `str`  | `""`       | Custom URL override.                                                                            |
| `deps`               | `list` | `[]`       | Explicit dependencies of this module.                                                           |
| `resolve_transitive` | `bool` | `True`     | Automatically resolve and unpack transitive npm dependencies into an isolated scoped directory. |
| `registry`           | `str`  | `""`       | Optional custom NPM registry URL override.                                                      |

#### Automatic Transitive Resolution & Scoped Import Maps

When `resolve_transitive = True` (the default), `ts_module` automatically
resolves and unpacks all transitive dependencies into an internal `.deps/`
directory. Downstream targets (`ts_library`, `ts_binary`, `ts_test`) synthesize
an ephemeral W3C import map using `scopes`:

- Transitive helpers remain strictly scoped to the parent module without leaking
  into global imports.
- If multiple modules depend on the same singleton or peer dependency, it is
  automatically hoisted.
- If an explicit `ts_module` target is provided in `deps`, it takes precedence
  and overrides any internal transitive version.

---

## `ts_library`

Compiles and type-checks TypeScript / JavaScript files and validates co-located
CSS files into a modular library.

```starlark
ts_library(
    name = "button",
    srcs = [
        "button.css",
        "button.ts",
    ],
    module_name = "@domain/button",
    deps = ["//ui/tokens"],
)
```

### Arguments

| Argument      | Type   | Default    | Description                                                                                                                                 |
| :------------ | :----- | :--------- | :------------------------------------------------------------------------------------------------------------------------------------------ |
| `name`        | `str`  | (required) | Target name.                                                                                                                                |
| `srcs`        | `list` | (required) | List of `.ts`, `.tsx`, `.js`, `.mjs`, or `.css` source files.                                                                               |
| `deps`        | `list` | `[]`       | Dependent libraries or `ts_module` targets.                                                                                                 |
| `module_name` | `str`  | `""`       | Optional module alias for bare specifiers. If omitted, automatically derived from `[plugin "ts"] ModulePrefix` and the Please package path. |
| `flags`       | `list` | `[]`       | Additional flags passed to `deno check`.                                                                                                    |

#### Co-located CSS & Validation

A `ts_library` can include `.css` files directly in `srcs` alongside its `.ts` /
`.tsx` files (or consist solely of `.css` files, such as a design-tokens
library). During compilation:

- All `.css` files are validated for balanced braces, strings, and comments, and
  checked for intra-library selector, `@keyframes`, and `:root` custom property
  collisions.
- TypeScript / JavaScript sources are type-checked with `deno check` (skipped
  automatically when `srcs` contains only `.css` files).
- Both `.ts` and `.css` files are recorded in `ts_metadata.json` (`files` and
  `css_files`) and embedded transitively via `deps`, so downstream `css_bundle`,
  `ts_bundle`, and `ts_browser_test` targets automatically discover all CSS in
  the dependency graph without inverted `used_in` references.

#### Transitive dependencies

A library's output carries its own dependencies: the `ts_library` and
`ts_module` targets it depends on, directly or not, are copied into the output
at their build-output paths. A test, binary or bundle that depends on the
library therefore needs to list **only the library**, not the libraries and
modules the library imports. Listing them as well is harmless.

Embedded files are hard links to the dependency's own output where the file
system allows it, so a module that many libraries depend on is stored once and
the extra disk use is small; otherwise they are copies, and a library close to
the top of a large graph is bigger than its own sources.

#### Subpath Imports & Package Mapping

When a `ts_library` contains multiple files, `please_ts` automatically generates
import map entries for each file under the module prefix, enabling consumers to
import individual files directly without barrel `index.ts` files:

```typescript
import { MetricCard } from "@repo/dashboard/components/MetricCard";
import { Badge } from "@repo/dashboard/components/Badge";
```

Both extensionless (`@module/File`) and explicit extension (`@module/File.ts`)
specifiers are supported.

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

Bundles TypeScript / JavaScript and/or CSS sources into distribution assets.

```starlark
ts_bundle(
    name = "app_bundle",
    main = "index.tsx",
    css_out = "app.min.css",
    deps = [":components"],
    format = "esm",
    minify = True,
)
```

### Arguments

| Argument              | Type   | Default    | Description                                                                                      |
| :-------------------- | :----- | :--------- | :----------------------------------------------------------------------------------------------- |
| `name`                | `str`  | (required) | Target name.                                                                                     |
| `main`                | `str`  | `""`       | Entrypoint source file (required for JS/TS bundles; optional for CSS-only bundles).              |
| `out`                 | `str`  | `""`       | Output bundle path (defaults to `<name>.bundle.js` or `<name>.bundle.css`).                      |
| `css_out`             | `str`  | `""`       | Optional CSS output path; collects, collision-checks, and bundles all CSS from `srcs` and `deps`. |
| `format`              | `str`  | `"esm"`    | Bundle format (`"esm"` or `"iife"`).                                                             |
| `minify`              | `bool` | `False`    | Whether to minify bundled output.                                                                |
| `sourcemap`           | `bool` | `False`    | Whether to generate source maps.                                                                 |
| `allow_css_overrides` | `bool` | `False`    | When `True`, disables strict cross-file CSS selector/variable collision checking.                |
| `bundler_tool`        | `str`  | `""`       | Optional external bundler executable (e.g. esbuild).                                             |

With npm packages (`ts_npm_module` dependencies) the bundle is built by
`deno bundle`, using the pinned esbuild of `esbuild_toolchain`, and
`bundler_tool` is not used. See
[Bundling and compiling](usage.md#bundling-and-compiling).

---

## `css_bundle`

Collects all `.css` files from `srcs` and transitive `ts_library` `deps` in
topological order, verifies that no two distinct files collide on selectors,
`@keyframes`, or `:root` custom properties, and bundles + minifies them using
the pinned `esbuild` binary.

```starlark
css_bundle(
    name = "dashboard_styles",
    srcs = ["dashboard.css"],
    deps = [
        "//ui/button",
        "//ui/card",
    ],
    out = "dashboard.min.css",
)
```

### Arguments

| Argument          | Type   | Default    | Description                                                                       |
| :---------------- | :----- | :--------- | :-------------------------------------------------------------------------------- |
| `name`            | `str`  | (required) | Target name.                                                                      |
| `srcs`            | `list` | `[]`       | Direct `.css` source files for this bundle.                                       |
| `deps`            | `list` | `[]`       | `ts_library` dependencies whose `.css` files are included in topological order.   |
| `main`            | `str`  | `""`       | Optional explicit `.css` entry point.                                             |
| `out`             | `str`  | `""`       | Output stylesheet path (defaults to `<name>.css`).                                |
| `minify`          | `bool` | `True`     | Whether to minify, strip comments/whitespace, and collapse shorthands (default).  |
| `sourcemap`       | `bool` | `False`    | Whether to generate a `.css.map` sourcemap.                                       |
| `allow_overrides` | `bool` | `False`    | When `True`, allows duplicate selectors or `:root` variables across source files. |
| `flags`           | `list` | `[]`       | Additional flags passed to `esbuild`.                                             |

---

## `ts_test`

Runs hermetic unit or simulated DOM tests using Deno test runner or Vitest.

```starlark
# Standard Deno test runner
ts_test(
    name = "calculator_test",
    srcs = ["calculator_test.ts"],
    deps = [":calculator"],
)

# Vitest runner with hermetic toolchain cache
ts_test(
    name = "calculator_vitest_test",
    srcs = ["calculator_vitest_test.ts"],
    deps = [":calculator"],
    runner = "vitest",
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

## `vitest_toolchain`

Assembles a hermetic Vitest toolchain using Deno cache.

The versions of the cache's packages are chosen when it is built, by a window of
time that ends then, and Deno keeps the registry's whole list of versions beside
them. Used on a later day, Deno chooses again, sees newer versions in that list,
and either fetches them into the cache while a build runs (racing with the
targets that copy it) or, in a test run (`--cached-only`), stops with
`npm package not found in cache`. So after `deno cache` the rule runs
`please_ts freeze-cache`, which takes from each package's list the versions
newer than the newest one cached: Deno can then choose only what is there, on
any day. A package with nothing cached (an optional one for another platform)
keeps its list.

```starlark
vitest_toolchain(
    name = "toolchain",
    version = "5.0.1",
)
```

### Arguments

| Argument  | Type  | Default       | Description                                            |
| :-------- | :---- | :------------ | :----------------------------------------------------- |
| `name`    | `str` | `"toolchain"` | Target name.                                           |
| `version` | `str` | `"5.0.1"`     | Vitest version to cache and bundle for test execution. |

---

## `esbuild_toolchain`

Provides the pinned esbuild binary that `deno bundle` runs to bundle npm
packages (Deno would download its own). The `EsbuildTool` config key points at
it (`//tools/esbuild_toolchain:toolchain|esbuild` by default).

```starlark
esbuild_toolchain(
    name = "toolchain",
)
```

### Arguments

| Argument          | Type  | Default       | Description                                                        |
| :---------------- | :---- | :------------ | :----------------------------------------------------------------- |
| `name`            | `str` | `"toolchain"` | Target name.                                                       |
| `version`         | `str` | `"0.25.5"`    | esbuild version; must be the one the Deno version uses.            |
| `esbuild_url`     | `str` | `""`          | Override the download URL (the `@esbuild/<platform>` npm tarball). |
| `esbuild_hash`    | `str` | `""`          | sha256 of that download.                                           |
| `target_platform` | `str` | `""`          | Platform (defaults to the host).                                   |

---

## `browser_toolchain`

Assembles a 100% hermetic headless browser (`chrome-headless-shell`) binary
sysroot.

```starlark
browser_toolchain(
    name = "toolchain",
    version = "154.0.8037.57",
    visibility = ["PUBLIC"],
)
```

### Arguments

| Argument          | Type  | Default           | Description                                                                                       |
| :---------------- | :---- | :---------------- | :------------------------------------------------------------------------------------------------ |
| `name`            | `str` | `"toolchain"`     | Name of the toolchain target.                                                                     |
| `version`         | `str` | `"154.0.8037.57"` | Chromium / chrome-headless-shell release version.                                                 |
| `browser_url`     | `str` | `""`              | Optional custom download URL for the browser zip archive.                                         |
| `browser_hash`    | `str` | `""`              | Optional SHA-256 integrity hash for the archive.                                                  |
| `target_platform` | `str` | `""`              | Optional target platform override (`linux_amd64`, `linux_arm64`, `darwin_amd64`, `darwin_arm64`). |

---

## `ts_browser_test`

Runs hermetic in-browser unit and integration tests with headless browser
binaries via the Chrome DevTools Protocol (CDP) WebSocket interface or Vitest.

```starlark
ts_browser_test(
    name = "counter_browser_test",
    srcs = ["counter_browser_test.ts"],
    browser = "chromium",
    runner = "browser",
    deps = [":counter_component"],
)
```

### Arguments

| Argument                  | Type   | Default      | Description                                                                                 |
| :------------------------ | :----- | :----------- | :------------------------------------------------------------------------------------------ |
| `name`                    | `str`  | (required)   | Target name.                                                                                |
| `srcs`                    | `list` | (required)   | Test source files.                                                                          |
| `browser`                 | `str`  | `"chromium"` | Browser type (`"chromium"`).                                                                |
| `browser_binary`          | `str`  | `""`         | Path or build label for the browser executable (defaults to `BrowserTool` in `.plzconfig`). |
| `runner`                  | `str`  | `"browser"`  | Browser test runner (`"browser"` for hermetic CDP runner, or `"vitest"`).                   |
| `deps`                    | `list` | `[]`         | Dependent libraries.                                                                        |
| `data`                    | `list` | `[]`         | Runtime test data.                                                                          |
| `module_name`             | `str`  | `""`         | Optional module name override.                                                              |
| `flags`                   | `list` | `[]`         | Additional flags.                                                                           |
| `viewport`                | `str`  | `""`         | Emulated window size as `"WIDTHxHEIGHT"`, so media queries resolve the same everywhere.     |
| `snapshots`               | `list` | `[]`         | Golden JSON files for `expectComputedStyles` (see below).                                   |
| `screenshots`             | `list` | `None`       | Golden PNG files for `expectScreenshot`. Passing it, even as `[]`, turns screenshots on.    |
| `pixel_channel_tolerance` | `int`  | `0`          | Largest per-channel (0-255) difference at which two pixels still match.                     |
| `max_diff_ratio`          | `str`  | `"0"`        | Share of pixels allowed to differ, as a string such as `"0.001"`.                           |

### Computed-style snapshots

A browser test can pin how the page is styled, so a CSS change that alters the
rendering fails the build instead of slipping through review. Inside a test:

```ts
declare const expectComputedStyles: (
  name: string,
  selector: string,
  props: string[],
) => void;

expectComputedStyles("card", ".card", ["display", "padding-top", "color"]);
```

This records `getComputedStyle` for every element matching `.card`. Once the
tests finish, the recording is compared with the golden `card.json` (the file
name is the snapshot name; use letters, digits, `_`, `.` and `-`). Each snapshot
is its own test case: a difference fails it and lists the element, property,
golden value and actual value.

A snapshot fails when there is no golden for it, and a golden that no test
recorded fails as stale. Every recording is written to the `<target>.snapshots`
test output (`plz-out/bin/<package>/<target>.snapshots/`), so to accept a
change, copy the file from there over the golden and review the diff.

```starlark
ts_browser_test(
    name = "card_test",
    srcs = ["card_test.ts"],
    snapshots = ["snapshots/card.json"],
    viewport = "800x600",
)
```

Computed styles are deterministic across machines. For what they cannot see
(layout, stacking, clipping, how things actually paint), use screenshots.

### Screenshots

```ts
declare const expectScreenshot: (
  name: string,
  options?: { selector?: string; fullPage?: boolean },
) => Promise<void>;

await expectScreenshot("card", { selector: ".card" });
await expectScreenshot("page", { fullPage: true });
```

`expectScreenshot` captures an element (`selector`), the whole scrollable page
(`fullPage`) or, with no options, the viewport. **Always `await` it:** the
capture happens while the promise is pending, so a test that goes on and changes
the DOM would capture the wrong thing. It waits for web fonts
(`document.fonts.ready`) and two animation frames first.

Goldens are PNG files passed as `screenshots`, one `<name>.png` per screenshot.
A screenshot fails when its size differs from the golden, when more than
`max_diff_ratio` of the pixels differ by more than `pixel_channel_tolerance` in
any channel, when there is no golden, or when a golden was never captured. Every
capture is written to the `<target>.snapshots` test output as `<name>.png`, and
a failing comparison also writes `<name>.diff.png` (differing pixels in red). To
accept a change, copy the capture over the golden and review it.

To record the first goldens, pass `screenshots = []`: the screenshots fail with
"no golden", and the captures are in the test output.

```starlark
ts_browser_test(
    name = "card_test",
    srcs = ["card_test.ts"],
    screenshots = ["screenshots/card.png"],
    viewport = "800x600",
    max_diff_ratio = "0.001",
)
```

Passing `screenshots` also starts the browser with flags that make rendering
reproducible (no font hinting or subpixel positioning, no LCD antialiasing, no
scrollbars, sRGB, device scale 1). They are not used otherwise, so the layout of
other browser tests does not change. Rendering text still depends on the fonts
installed on the machine. For goldens that are stable on every machine and in
CI, load the fonts the page uses with `@font-face` from files you ship, or keep
pixel-checked pages free of text and use computed-style snapshots for the text.
