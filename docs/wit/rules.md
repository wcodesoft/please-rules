# WIT Rule Reference

The `///wit` ruleset provides rules for defining WIT packages and generating
cross-language bindings and interface contracts.

---

## `wit_library`

Collects and validates `.wit` files into a consumable WIT package.

```starlark
subinclude("///wit//build_defs:wit")

wit_library(
    name = "structures_wit",
    srcs = glob(["*.wit"]),
    package = "contract:structures",
)
```

### Arguments

| Name         | Type   | Default  | Description                                               |
| :----------- | :----- | :------- | :-------------------------------------------------------- |
| `name`       | `str`  | Required | Name of the target.                                       |
| `srcs`       | `list` | Required | List of `.wit` source files.                              |
| `package`    | `str`  | `""`     | Optional package identifier (e.g. `contract:structures`). |
| `deps`       | `list` | `[]`     | Dependency `wit_library` targets.                         |
| `visibility` | `list` | `None`   | Target visibility.                                        |

---

## `kt_wit_bindgen`

Generates idiomatic Kotlin interfaces directly from the WIT AST. Requires
**zero** external binaries.

```starlark
kt_wit_bindgen(
    name = "structures_kotlin",
    wit = ":structures_wit",
    package = "contract.structures",
)
```

### Generated Code Example

For a WIT interface:

```wit
interface disjoint-set {
    find-root: func(element: s32) -> s32;
    union-sets: func(a: s32, b: s32) -> bool;
}
```

`kt_wit_bindgen` generates:

```kotlin
package contract.structures

public interface DisjointSet {
    fun findRoot(element: Int): Int
    fun unionSets(a: Int, b: Int): Boolean
}
```

---

## `swift_wit_bindgen`

Generates Swift protocols and `module.modulemap` directly from the WIT AST.
Requires **zero** external binaries.

```starlark
swift_wit_bindgen(
    name = "structures_swift",
    wit = ":structures_wit",
)
```

### Generated Code Example

```swift
import Foundation

public protocol DisjointSet {
    func findRoot(element: Int32) -> Int32
    func unionSets(a: Int32, b: Int32) -> Bool
}
```

---

## `ts_wit_bindgen`

Generates TypeScript interface definitions (`.d.ts`) directly from the WIT AST.
Requires **zero** external binaries.

```starlark
ts_wit_bindgen(
    name = "structures_ts",
    wit = ":structures_wit",
)
```

### Generated Code Example

```typescript
export interface DisjointSet {
  findRoot(element: number): number;
  unionSets(a: number, b: number): boolean;
}
```

---

## `python_wit_bindgen`

Generates Python `Protocol` interfaces and `.pyi` type stubs directly from the
WIT AST. Requires **zero** external binaries.

```starlark
python_wit_bindgen(
    name = "structures_py",
    wit = ":structures_wit",
)
```

### Generated Code Example

```python
from typing import Protocol

class DisjointSet(Protocol):
    def find_root(self, element: int) -> int: ...
    def union_sets(self, a: int, b: int) -> bool: ...
```

---

## `rust_wit_bindgen`

Generates Rust traits, structs, and enums directly from the WIT AST. Requires
**zero** external binaries.

```starlark
rust_wit_bindgen(
    name = "structures_rust",
    wit = ":structures_wit",
)
```

### Generated Code Example

```rust
pub trait DisjointSet {
    fn find_root(&mut self, element: i32) -> i32;
    fn union_sets(&mut self, a: i32, b: i32) -> bool;
}
```

---

## `go_wit_bindgen`

Generates Go interfaces and struct types directly from the WIT AST. Requires
**zero** external binaries.

```starlark
go_wit_bindgen(
    name = "structures_go",
    wit = ":structures_wit",
)
```

### Generated Code Example

```go
type DisjointSet interface {
    FindRoot(element int32) int32
    UnionSets(a int32, b int32) bool
}
```

---

## `cc_wit_bindgen`

Generates C++ abstract base classes, headers (`.h`), and companions (`.cpp`)
directly from the WIT AST. Requires **zero** external binaries.

```starlark
cc_wit_bindgen(
    name = "structures_cc",
    wit = ":structures_wit",
)
```

### Generated Code Example

```cpp
class DisjointSet {
public:
    virtual ~DisjointSet() = default;
    virtual int32_t find_root(int32_t element) = 0;
    virtual bool union_sets(int32_t a, int32_t b) = 0;
};
```
