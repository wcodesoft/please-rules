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
        found_go = False
        for _, _, files in os.walk(go_dir):
            if any(f.endswith(".go") or f.endswith(".h") or f.endswith(".c") for f in files):
                found_go = True
                break
        self.assertTrue(found_go, f"Expected Go/C files in {go_dir}")

    def test_swift_bindings_generated(self):
        swift_dir = os.environ.get("SWIFT_DIR", "test/wit/swift_bindings")
        self.assertTrue(os.path.isdir(swift_dir), f"Swift output directory {swift_dir} missing")
        files = os.listdir(swift_dir)
        self.assertTrue(any(f.endswith(".swift") for f in files), f"Expected .swift file in {files}")
        self.assertIn("module.modulemap", files, f"Expected module.modulemap in {files}")
        self.assertIn("Structures.swift", files, f"Expected Structures.swift in {files}")
        with open(os.path.join(swift_dir, "Structures.swift"), "r") as f:
            content = f.read()
            self.assertIn("public protocol TwoSum", content)
            self.assertIn("func solve(nums: [Int32], target: Int32) -> [Int32]", content)

    def test_kotlin_bindings_generated(self):
        kt_dir = os.environ.get("KT_DIR", "test/wit/kt_bindings")
        self.assertTrue(os.path.isdir(kt_dir), f"Kotlin output directory {kt_dir} missing")
        files = os.listdir(kt_dir)
        self.assertTrue(any(f.endswith(".kt") for f in files), f"Expected .kt file in {files}")
        self.assertIn("Structures.kt", files, f"Expected Structures.kt in {files}")
        with open(os.path.join(kt_dir, "Structures.kt"), "r") as f:
            content = f.read()
            self.assertIn("package test.structures", content)
            self.assertIn("public interface TwoSum", content)
            self.assertIn("fun solve(nums: List<Int>, target: Int): List<Int>", content)

    def test_ts_bindings_generated(self):
        ts_dir = os.environ.get("TS_DIR", "test/wit/ts_bindings")
        self.assertTrue(os.path.isdir(ts_dir), f"TypeScript output directory {ts_dir} missing")
        files = os.listdir(ts_dir)
        self.assertTrue(any(f.endswith(".d.ts") or f.endswith(".ts") for f in files), f"Expected .d.ts/.ts file in {files}")
        ts_file = next(f for f in files if f.endswith(".d.ts"))
        with open(os.path.join(ts_dir, ts_file), "r") as f:
            content = f.read()
            self.assertIn("export interface TwoSum", content)
            self.assertIn("solve(nums: number[], target: number): number[];", content)

    def test_python_bindings_generated(self):
        py_dir = os.environ.get("PY_DIR", "test/wit/py_bindings")
        self.assertTrue(os.path.isdir(py_dir), f"Python output directory {py_dir} missing")
        files = os.listdir(py_dir)
        self.assertTrue(any(f.endswith(".py") or f.endswith(".pyi") for f in files), f"Expected .py/.pyi file in {files}")
        with open(os.path.join(py_dir, "__init__.py"), "r") as f:
            content = f.read()
            self.assertIn("class TwoSum(Protocol):", content)
            self.assertIn("def solve(self, nums: List[int], target: int) -> List[int]:", content)

if __name__ == "__main__":
    unittest.main()
