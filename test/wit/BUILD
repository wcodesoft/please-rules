subinclude("//build_defs/wit:wit")

wit_library(
    name = "structures_wit",
    srcs = ["fixtures/two_sum.wit"],
    package = "test:structures",
)

rust_wit_bindgen(
    name = "rust_bindings",
    wit = ":structures_wit",
    worlds = ["two-sum-world"],
)

go_wit_bindgen(
    name = "go_bindings",
    wit = ":structures_wit",
    worlds = ["two-sum-world"],
)

cc_wit_bindgen(
    name = "cpp_bindings",
    wit = ":structures_wit",
    worlds = ["two-sum-world"],
)

swift_wit_bindgen(
    name = "swift_bindings",
    wit = ":structures_wit",
)

kt_wit_bindgen(
    name = "kt_bindings",
    wit = ":structures_wit",
)

ts_wit_bindgen(
    name = "ts_bindings",
    wit = ":structures_wit",
)

python_wit_bindgen(
    name = "py_bindings",
    wit = ":structures_wit",
)

gentest(
    name = "bindings_test",
    data = [
        "verify_bindings_test.py",
        ":cpp_bindings",
        ":go_bindings",
        ":kt_bindings",
        ":py_bindings",
        ":rust_bindings",
        ":swift_bindings",
        ":ts_bindings",
    ],
    labels = ["wit"],
    no_test_output = True,
    test_cmd = """
export RUST_DIR="test/wit/rust_bindings"
export CPP_DIR="test/wit/cpp_bindings"
export GO_DIR="test/wit/go_bindings"
export SWIFT_DIR="test/wit/swift_bindings"
export KT_DIR="test/wit/kt_bindings"
export TS_DIR="test/wit/ts_bindings"
export PY_DIR="test/wit/py_bindings"

python3 test/wit/verify_bindings_test.py
""",
)
