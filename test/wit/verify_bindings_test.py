#!/usr/bin/env python3
"""Integration tests verifying generated bindings across all languages."""

import os
import unittest

class TestWitBindings(unittest.TestCase):
    def test_rust_bindings_generated(self):
        rust_dir = os.environ.get("RUST_DIR", "test/wit/rust_bindings")
        self.assertTrue(os.path.isdir(rust_dir), f"Rust output directory {rust_dir} missing")
        files = os.listdir(rust_dir)
        self.assertTrue(any(f.endswith(".rs") for f in files), f"Expected .rs file in {files}")

    def test_cpp_bindings_generated(self):
        cpp_dir = os.environ.get("CPP_DIR", "test/wit/cpp_bindings")
        self.assertTrue(os.path.isdir(cpp_dir), f"C++ output directory {cpp_dir} missing")
        files = os.listdir(cpp_dir)
        self.assertTrue(any(f.endswith(".h") or f.endswith(".cpp") for f in files), f"Expected .h/.cpp file in {files}")

    def test_go_bindings_generated(self):
        go_dir = os.environ.get("GO_DIR", "test/wit/go_bindings")
        self.assertTrue(os.path.isdir(go_dir), f"Go output directory {go_dir} missing")
        files = os.listdir(go_dir)
        self.assertTrue(any(f.endswith(".go") or f.endswith(".h") or f.endswith(".c") for f in files), f"Expected Go/C files in {files}")

    def test_swift_bindings_generated(self):
        swift_dir = os.environ.get("SWIFT_DIR", "test/wit/swift_bindings")
        self.assertTrue(os.path.isdir(swift_dir), f"Swift output directory {swift_dir} missing")
        files = os.listdir(swift_dir)
        self.assertTrue(any(f.endswith(".swift") for f in files), f"Expected .swift file in {files}")

    def test_kotlin_bindings_generated(self):
        kt_dir = os.environ.get("KT_DIR", "test/wit/kt_bindings")
        self.assertTrue(os.path.isdir(kt_dir), f"Kotlin output directory {kt_dir} missing")
        files = os.listdir(kt_dir)
        self.assertTrue(any(f.endswith(".kt") for f in files), f"Expected .kt file in {files}")

    def test_ts_bindings_generated(self):
        ts_dir = os.environ.get("TS_DIR", "test/wit/ts_bindings")
        self.assertTrue(os.path.isdir(ts_dir), f"TypeScript output directory {ts_dir} missing")
        files = os.listdir(ts_dir)
        self.assertTrue(any(f.endswith(".d.ts") or f.endswith(".ts") for f in files), f"Expected .d.ts/.ts file in {files}")

    def test_python_bindings_generated(self):
        py_dir = os.environ.get("PY_DIR", "test/wit/py_bindings")
        self.assertTrue(os.path.isdir(py_dir), f"Python output directory {py_dir} missing")
        files = os.listdir(py_dir)
        self.assertTrue(any(f.endswith(".py") or f.endswith(".pyi") for f in files), f"Expected .py/.pyi file in {files}")

if __name__ == "__main__":
    unittest.main()
