#!/usr/bin/env python3
"""Unit tests for update_release_pins.py."""

import hashlib
import os
import subprocess
import sys
import tempfile
import unittest

import update_release_pins as pins
from update_release_pins import PinError

# tools/BUILD of a rule set with one tool.
ONE_TOOL = '''_VERSION = "0.5.0"

remote_file(
    name = "please_rust",
    url = f"https://example.com/releases/download/rust-v{_VERSION}/please_rust-{_VERSION}-{CONFIG.OS}_{CONFIG.ARCH}",
    hashes = [
    "2b91fa776bc3302221ad55c174067be0888d16a61daac847bfc6410644956886",  # linux_amd64
    "c11486d43adc6a27a17aa63de85785bc6adfabc69058c84b8a8c6767e29c7036",  # linux_arm64
    "1a5882754cb9f2c631b4745b8ec7325e2af41a7f8cd8d24c846a34cb6c645b35",  # darwin_amd64
    "9f0ce9730a8d183be59e50aca4af834aab5e5f86d4c25ae196e5b517f8b01198",  # darwin_arm64
],
    binary = True,
    visibility = ["PUBLIC"],
)
'''

# tools/BUILD of the Kotlin rules: two tools, each with its own hashes, oddly indented.
TWO_TOOLS = '''_VERSION = "0.3.5"

remote_file(
    name = "please_kotlin",
    url = f"https://example.com/kotlin-v{_VERSION}/please_kotlin-{_VERSION}-{CONFIG.OS}_{CONFIG.ARCH}",
    hashes = [
    "5d74a551fa44d6d6c560a86b03e9ff4a6106a77c0b5da248fad0e19b1f51f279",  # linux_amd64
    "55ad6fcaacbec0afc359b5ecbd8bf0e24d57291163e8d56e72a0ceb9e3a7cfb4",  # linux_arm64
    "c22a60313907e177e0ffcc9c3aa39ba0089f07fd00799ed38252ed86a9394298",  # darwin_amd64
    "45fe02fb28b52c18a191b68bbe5285f1c0ad0ee16b4667d2890b87e9a2f69449",  # darwin_arm64
          ],
    binary = True,
    visibility = ["PUBLIC"],
)

remote_file(
    name = "please_kotlin_wasm",
    url = f"https://example.com/kotlin-v{_VERSION}/please_kotlin_wasm-{_VERSION}-{CONFIG.OS}_{CONFIG.ARCH}",
    hashes = [
    "06d811d9969a4a6b9043da136f43411aa97644ae582d1a5c4ff45b8f1ea17e5f",  # linux_amd64
    "63e2a15c4220b6cebcf3c01e2b564055b9e8178c4778aa7d45228f9ecfafc310",  # linux_arm64
    "b1899bc93844ff8527466ba28c38d25539e563d162725fb1f01989fd4c3278ab",  # darwin_amd64
    "9f697ddd9d80541ea73c15175be1d8f6b45da26d11d35fb0ca5fb60d63530e64",  # darwin_arm64
          ],
    binary = True,
    visibility = ["PUBLIC"],
)
'''


def fake_hashes(tag):
    return {p: hashlib.sha256(f"{tag}-{p}".encode()).hexdigest() for p in pins.PLATFORMS}


class UpdateTest(unittest.TestCase):
    def test_updates_the_version_and_hashes_of_one_tool(self):
        new = fake_hashes("rust")
        out = pins.update(ONE_TOOL, "0.6.0", {"please_rust": new})
        version, targets = pins.read_pins(out)
        self.assertEqual(version, "0.6.0")
        self.assertEqual(targets, {"please_rust": new})
        # Everything else is untouched.
        self.assertIn('url = f"https://example.com/releases/download/rust-v{_VERSION}', out)
        self.assertIn("binary = True,", out)
        self.assertTrue(out.endswith(")\n"))

    def test_two_tools_get_their_own_hashes(self):
        kotlin, wasm = fake_hashes("kotlin"), fake_hashes("wasm")
        out = pins.update(TWO_TOOLS, "0.4.0", {"please_kotlin": kotlin, "please_kotlin_wasm": wasm})
        version, targets = pins.read_pins(out)
        self.assertEqual(version, "0.4.0")
        self.assertEqual(targets["please_kotlin"], kotlin)
        self.assertEqual(targets["please_kotlin_wasm"], wasm)

    def test_updating_one_of_two_tools_leaves_the_other_alone(self):
        _, before = pins.read_pins(TWO_TOOLS)
        wasm = fake_hashes("wasm")
        out = pins.update(TWO_TOOLS, "0.4.0", {"please_kotlin_wasm": wasm})
        _, after = pins.read_pins(out)
        self.assertEqual(after["please_kotlin"], before["please_kotlin"])
        self.assertEqual(after["please_kotlin_wasm"], wasm)

    def test_a_tool_whose_name_is_a_prefix_of_another_is_not_confused(self):
        # please_kotlin must not match the please_kotlin_wasm block (a pattern on the name would).
        kotlin = fake_hashes("only-kotlin")
        out = pins.update(TWO_TOOLS, "0.4.0", {"please_kotlin": kotlin})
        _, after = pins.read_pins(out)
        _, before = pins.read_pins(TWO_TOOLS)
        self.assertEqual(after["please_kotlin"], kotlin)
        self.assertEqual(after["please_kotlin_wasm"], before["please_kotlin_wasm"])

    def test_is_idempotent(self):
        new = fake_hashes("rust")
        once = pins.update(ONE_TOOL, "0.6.0", {"please_rust": new})
        twice = pins.update(once, "0.6.0", {"please_rust": new})
        self.assertEqual(once, twice)

    def test_the_hash_lines_keep_their_platform_comments(self):
        out = pins.update(ONE_TOOL, "0.6.0", {"please_rust": fake_hashes("rust")})
        for platform in pins.PLATFORMS:
            self.assertEqual(sum(1 for line in out.split("\n") if line.endswith(f"# {platform}")), 1)


class ErrorTest(unittest.TestCase):
    def test_unknown_target_is_an_error(self):
        with self.assertRaisesRegex(PinError, 'exactly one remote_file named "nope"'):
            pins.update(ONE_TOOL, "0.6.0", {"nope": fake_hashes("x")})

    def test_duplicate_target_is_an_error(self):
        with self.assertRaisesRegex(PinError, "found 2"):
            pins.update(ONE_TOOL + "\n" + ONE_TOOL.split("\n\n", 1)[1], "0.6.0", {"please_rust": fake_hashes("x")})

    def test_missing_or_duplicate_version_is_an_error(self):
        no_version = "\n".join(line for line in ONE_TOOL.split("\n") if not line.startswith("_VERSION"))
        with self.assertRaisesRegex(PinError, "exactly one _VERSION, found 0"):
            pins.update(no_version, "0.6.0", {"please_rust": fake_hashes("x")})
        with self.assertRaisesRegex(PinError, "exactly one _VERSION, found 2"):
            pins.update('_VERSION = "1"\n' + ONE_TOOL, "0.6.0", {"please_rust": fake_hashes("x")})

    def test_a_block_without_hashes_is_an_error(self):
        broken = ONE_TOOL.replace("hashes = [", "digests = [")
        with self.assertRaisesRegex(PinError, 'exactly one "hashes = \\["'):
            pins.update(broken, "0.6.0", {"please_rust": fake_hashes("x")})

    def test_an_unclosed_list_or_block_is_an_error(self):
        with self.assertRaises(PinError):
            pins.update(ONE_TOOL.replace("],\n", ""), "0.6.0", {"please_rust": fake_hashes("x")})
        with self.assertRaisesRegex(PinError, "unterminated"):
            pins.update(ONE_TOOL.rstrip("\n").rstrip(")"), "0.6.0", {"please_rust": fake_hashes("x")})

    def test_hashes_for_other_platforms_are_an_error(self):
        windows = ONE_TOOL.replace("# darwin_arm64", "# windows_amd64")
        with self.assertRaisesRegex(PinError, "expected"):
            pins.update(windows, "0.6.0", {"please_rust": fake_hashes("x")})

    def test_a_hash_line_without_a_platform_comment_is_an_error(self):
        undocumented = ONE_TOOL.replace("  # linux_arm64", "")
        with self.assertRaisesRegex(PinError, "no platform comment|expected"):
            pins.update(undocumented, "0.6.0", {"please_rust": fake_hashes("x")})


class CommandLineTest(unittest.TestCase):
    def setUp(self):
        self.dir = tempfile.TemporaryDirectory()
        self.addCleanup(self.dir.cleanup)
        self.build = os.path.join(self.dir.name, "BUILD")
        with open(self.build, "w") as f:
            f.write(TWO_TOOLS)

    def make_binaries(self, name, version, platforms=pins.PLATFORMS):
        digests = {}
        for platform in platforms:
            path = os.path.join(self.dir.name, f"{name}-{version}-{platform}")
            content = f"{name}:{platform}".encode()
            with open(path, "wb") as f:
                f.write(content)
            digests[platform] = hashlib.sha256(content).hexdigest()
        return os.path.join(self.dir.name, f"{name}-{version}"), digests

    def run_script(self, *args):
        script = os.path.join(os.path.dirname(os.path.abspath(__file__)), "update_release_pins.py")
        return subprocess.run([sys.executable, script, *args], capture_output=True, text=True)

    def test_hashes_are_computed_from_the_binaries(self):
        kotlin_prefix, kotlin = self.make_binaries("please_kotlin", "0.4.0")
        wasm_prefix, wasm = self.make_binaries("please_kotlin_wasm", "0.4.0")
        result = self.run_script("--build", self.build, "--version", "v0.4.0",
                                 "--target", f"please_kotlin={kotlin_prefix}",
                                 "--target", f"please_kotlin_wasm={wasm_prefix}")
        self.assertEqual(result.returncode, 0, result.stderr)
        with open(self.build) as f:
            version, targets = pins.read_pins(f.read())
        self.assertEqual(version, "0.4.0")
        self.assertEqual(targets, {"please_kotlin": kotlin, "please_kotlin_wasm": wasm})

    def test_a_missing_binary_fails_and_leaves_the_file_alone(self):
        prefix, _ = self.make_binaries("please_kotlin", "0.4.0", platforms=pins.PLATFORMS[:3])
        result = self.run_script("--build", self.build, "--version", "0.4.0", "--target", f"please_kotlin={prefix}")
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("binary not found", result.stderr)
        with open(self.build) as f:
            self.assertEqual(f.read(), TWO_TOOLS)

    def test_a_format_problem_fails_and_leaves_the_file_alone(self):
        with open(self.build, "w") as f:
            f.write(TWO_TOOLS.replace("hashes = [", "digests = [", 1))
        prefix, _ = self.make_binaries("please_kotlin", "0.4.0")
        result = self.run_script("--build", self.build, "--version", "0.4.0", "--target", f"please_kotlin={prefix}")
        self.assertNotEqual(result.returncode, 0)
        self.assertIn('exactly one "hashes = ["', result.stderr)

    def test_bad_target_arguments_are_rejected(self):
        for target in ("please_kotlin", "=prefix", "name="):
            result = self.run_script("--build", self.build, "--version", "0.4.0", "--target", target)
            self.assertNotEqual(result.returncode, 0, target)
        result = self.run_script("--build", self.build, "--version", "0.4.0", "--target", "a=x", "--target", "a=y")
        self.assertNotEqual(result.returncode, 0)


if __name__ == "__main__":
    unittest.main()
