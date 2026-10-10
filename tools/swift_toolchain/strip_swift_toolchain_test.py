#!/usr/bin/env python3
import os
import tempfile
import unittest
from strip_swift_toolchain import strip_directory, compute_sha256

class TestStripSwiftToolchain(unittest.TestCase):
    def test_strip_directory(self):
        with tempfile.TemporaryDirectory() as td:
            # Create simulated structure
            bin_dir = os.path.join(td, "bin")
            lib_dir = os.path.join(td, "lib")
            share_dir = os.path.join(td, "share", "swift")
            spm_dir = os.path.join(td, "lib", "swift", "pm")
            os.makedirs(bin_dir)
            os.makedirs(lib_dir)
            os.makedirs(share_dir)
            os.makedirs(spm_dir)

            # Files that should be preserved
            with open(os.path.join(bin_dir, "swiftc"), "w") as f:
                f.write("#!/bin/sh\n")
            with open(os.path.join(share_dir, "LICENSE.txt"), "w") as f:
                f.write("License test\n")

            # Files that should be stripped
            with open(os.path.join(bin_dir, "lldb"), "w") as f:
                f.write("lldb")
            with open(os.path.join(bin_dir, "docc"), "w") as f:
                f.write("docc")
            with open(os.path.join(bin_dir, "swift-build"), "w") as f:
                f.write("swift-build")
            with open(os.path.join(lib_dir, "liblldb.so"), "w") as f:
                f.write("liblldb")
            with open(os.path.join(spm_dir, "PackageModel.swiftmodule"), "w") as f:
                f.write("spm")

            strip_directory(td)

            # Verify kept files
            self.assertTrue(os.path.exists(os.path.join(bin_dir, "swiftc")))
            self.assertTrue(os.path.exists(os.path.join(td, "LICENSE.txt")))

            # Verify removed files
            self.assertFalse(os.path.exists(os.path.join(bin_dir, "lldb")))
            self.assertFalse(os.path.exists(os.path.join(bin_dir, "docc")))
            self.assertFalse(os.path.exists(os.path.join(bin_dir, "swift-build")))
            self.assertFalse(os.path.exists(os.path.join(lib_dir, "liblldb.so")))
            self.assertFalse(os.path.exists(spm_dir))

    def test_compute_sha256(self):
        with tempfile.NamedTemporaryFile() as tf:
            tf.write(b"hello world")
            tf.flush()
            expected = "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9"
            self.assertEqual(compute_sha256(tf.name), expected)

if __name__ == "__main__":
    unittest.main()
