# Hermetic Mutation Testing Reports

Code coverage (`plz cover`) measures which lines, functions, and branches ran
during a test suite, but not whether the tests actually verify the behavior of
those lines. Mutation testing seeds small, deterministic syntax changes
(**mutants**) into executable function bodies and runs the test target against
each mutant:

- If a test fails, the mutant is **killed**.
- If all tests still pass, the mutant **survived** (exposing a test gap).

Every mutation test target writes a machine-readable JSON report alongside
Please's standard `test.results` JUnit XML output. This page is the contract for
the shared mutation testing packages (`//tools/common/mutation`,
`//tools/common/treesitter`, `//tools/common/mutate`, and
`//tools/common/mutationcheck`) and downstream language rules.

---

## Where the files are

Each mutation test target writes one JSON report, declared as a `test_outputs`
entry:

```text
plz-out/bin/<package>/<target name>.mutation.json
```

| Rule / Recipe        | Language   | Example                                                          |
| :------------------- | :--------- | :--------------------------------------------------------------- |
| `ts_mutate_test`     | TypeScript | `plz-out/bin/test/ts/lib/calculator_mutate_test.mutation.json`   |
| `kotlin_mutate_test` | Kotlin     | `plz-out/bin/test/kotlin/branches/branches_mutate.mutation.json` |
| `rust_mutate_test`   | Rust       | `plz-out/bin/test/rust/branches/branches_mutate.mutation.json`   |
| `swift_mutate_test`  | Swift      | `plz-out/bin/test/swift/branches/branches_mutate.mutation.json`  |
| `mutate` (`go_test`) | Go         | `plz-out/bin/pkg/lib_mutate_test.mutation.json`                  |
| `mutate` (`python`)  | Python     | `plz-out/bin/pkg/lib_mutate_test.mutation.json`                  |

All mutation test rules are tagged with `labels = ["mutate", "manual"]` by
default so a standard `./pleasew test` remains fast and only
`./pleasew test --include mutate` (or an explicit target invocation) runs
mutation testing.

---

## Report Schema (`.mutation.json`)

```json
{
  "schema_version": 1,
  "language": "go",
  "target": "//pkg:lib_mutate_test",
  "min_score": 80,
  "total": 3,
  "killed": 2,
  "survived": 1,
  "timeout": 0,
  "unviable": 0,
  "score": 66.67,
  "mutants": [
    {
      "id": "m1",
      "file": "pkg/lib.go",
      "line": 12,
      "col": 10,
      "end_line": 12,
      "end_col": 11,
      "mutator": "arithmetic",
      "function": "Add",
      "original": "+",
      "replacement": "-",
      "status": "killed",
      "duration_ms": 142
    }
  ]
}
```

### Statuses and Score Formula

| Status     | Meaning                                                                 |
| :--------- | :---------------------------------------------------------------------- |
| `killed`   | At least one test failed when the mutant was applied                    |
| `survived` | All tests passed with the mutant applied                                |
| `timeout`  | The mutant caused the test suite to exceed the per-mutant timeout       |
| `unviable` | The mutant failed to compile or load and is excluded from the score     |
| `skipped`  | The mutant was listed (`-list`) or filtered out without test evaluation |

$$\text{score} = \frac{\text{killed} + \text{timeout}}{\text{killed} + \text{survived} + \text{timeout}} \times 100$$

If no mutants are evaluated (`total == 0`), `score` is `100.0`.

---

## Multi-Language Tree-sitter Mutator (`//tools/common/treesitter` & `//tools/common/mutate`)

Instead of relying on external language-specific CLI wrappers or regex text
substitution, `please-rules` uses pure-Go Tree-sitter grammars
(`github.com/odvcencio/gotreesitter`) in `//tools/common/treesitter` to parse
concrete syntax trees (CSTs) for **Go**, **Rust**, **TypeScript/JavaScript**
(`.ts`, `.tsx`, `.js`, `.jsx`), **Kotlin** (`.kt`, `.kts`), **Swift**
(`.swift`), and **Python** (`.py`).

### Structural Guarantees

1. **Function bodies only**: `LanguageRules.Function` locates executable
   function, method, and closure bodies. Package headers, `import` / `use`
   clauses, type definitions, struct/interface/protocol declarations, and
   annotations/attributes are never mutated.
2. **In-file test exclusion**: `LanguageRules.Test` skips in-file test blocks
   and functions (`#[test]`, `#[cfg(test)]`, `Deno.test`,
   `describe`/`it`/`test`, `@Test`, `func Test...`, `def test_...`).
3. **Exact byte-span splicing**: Each candidate `Site` records its exact
   `[StartByte:EndByte]` span and 1-based line/column coordinates, ensuring
   single-mutation isolation and byte-for-byte restoration after each run.

### Mutator Categories

| Category          | Mutations                                                                |
| :---------------- | :----------------------------------------------------------------------- |
| `arithmetic`      | `+` <-> `-`, `*` <-> `/`, `%` -> `*` (skips string literals)             |
| `comparison`      | `<` -> `<=`, `>=`; `<=` -> `<`, `>`; `>` -> `>=`, `<=`; `>=` -> `>`, `<` |
| `equality`        | `==` <-> `!=`, `===` <-> `!==`                                           |
| `logical`         | `&&` <-> `                                                               |     | `, `and`<->`or` |
| `boolean_literal` | `true` <-> `false`, `True` <-> `False`                                   |
| `unary`           | `!x` -> `x`, `not x` -> `x`, `++` <-> `--`                               |
| `assignment`      | `+=` <-> `-=`, `*=` <-> `/=`                                             |

---

## Running `//tools/common/mutate`

```bash
./pleasew build //tools/common/mutate

plz-out/bin/tools/common/mutate/mutate \
  -srcs pkg/calc.go \
  -target //pkg:calc_mutate_test \
  -test-cmd "go test ./pkg/..." \
  -out plz-out/bin/pkg/calc_mutate_test.mutation.json \
  -results-file test.results \
  -min-score 80
```

Use `-list` to inspect all discovered mutants for any supported language without
executing tests:

```bash
plz-out/bin/tools/common/mutate/mutate -srcs pkg/calc.rs -list
```

---

## Checking a Report in CI (`//tools/common/mutationcheck`)

`mutationcheck` (`//tools/common/mutationcheck`) asserts facts about a
`.mutation.json` report in CI smoke tests, analogous to `lcovcheck`:

```bash
./pleasew build //tools/common/mutationcheck
M=plz-out/bin/tools/common/mutationcheck/mutationcheck

$M -report plz-out/bin/pkg/calc_mutate_test.mutation.json \
   -lang go -min-score 60 -min-total 3 -min-killed 2 \
   -path pkg/calc.go -line 12=killed -function Clamp=survived
```

Every failed assertion is listed and the process exits with status `1`.
