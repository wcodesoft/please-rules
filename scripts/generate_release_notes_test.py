#!/usr/bin/env python3
"""
Unit tests for generate_release_notes.py.
"""

import os
import subprocess
import sys
import tempfile
import unittest

from generate_release_notes import extract_release_notes

SAMPLE_CHANGELOG = """# Changelog

All notable changes will be documented in this file.

## [0.1.1] - 2026-09-17

### Fixed

- Fixed Starlark f-string quote syntax error when passing CLI `flags`.
- Standardized CLI flag description and usage to `sources`.

## [0.1.0] - 2026-09-13

### Added

- Initial release with hermetic TypeScript rules.
"""


class TestExtractReleaseNotes(unittest.TestCase):
    def test_extract_exact_version(self):
        notes = extract_release_notes(SAMPLE_CHANGELOG, "0.1.1", "TypeScript")
        self.assertIn("### TypeScript Rules v0.1.1", notes)
        self.assertIn("### Fixed", notes)
        self.assertIn("Fixed Starlark f-string quote syntax error", notes)
        self.assertNotIn("## [0.1.0]", notes)
        self.assertNotIn("Initial release with hermetic TypeScript rules", notes)

    def test_extract_version_with_v_prefix(self):
        notes = extract_release_notes(SAMPLE_CHANGELOG, "v0.1.1", "TypeScript")
        self.assertIn("### TypeScript Rules v0.1.1", notes)
        self.assertIn("Fixed Starlark f-string quote syntax error", notes)

    def test_extract_older_version_at_end(self):
        notes = extract_release_notes(SAMPLE_CHANGELOG, "0.1.0", "TypeScript")
        self.assertIn("### TypeScript Rules v0.1.0", notes)
        self.assertIn("### Added", notes)
        self.assertIn("Initial release with hermetic TypeScript rules", notes)
        self.assertNotIn("Fixed Starlark f-string quote syntax error", notes)

    def test_version_not_in_changelog_fallback(self):
        notes = extract_release_notes(SAMPLE_CHANGELOG, "9.9.9", "TypeScript")
        self.assertIn("### TypeScript Rules v9.9.9", notes)
        self.assertIn(
            "Precompiled hermetic binaries and build definitions for TypeScript.",
            notes,
        )

    def test_empty_changelog_fallback(self):
        notes = extract_release_notes("", "0.1.1", "Swift")
        self.assertIn("### Swift Rules v0.1.1", notes)
        self.assertIn(
            "Precompiled hermetic binaries and build definitions for Swift.",
            notes,
        )


class TestCliExecution(unittest.TestCase):
    def setUp(self):
        self.script_path = os.path.join(
            os.path.dirname(__file__), "generate_release_notes.py"
        )

    def test_cli_writes_to_file(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            changelog_file = os.path.join(tmpdir, "CHANGELOG.md")
            with open(changelog_file, "w", encoding="utf-8") as f:
                f.write(SAMPLE_CHANGELOG)

            out_file = os.path.join(tmpdir, "out", "release_notes.md")
            cmd = [
                sys.executable,
                self.script_path,
                "--changelog",
                changelog_file,
                "--version",
                "0.1.1",
                "--title",
                "TypeScript",
                "--out",
                out_file,
            ]
            res = subprocess.run(cmd, capture_output=True, text=True)
            self.assertEqual(res.returncode, 0, msg=res.stderr)
            self.assertTrue(os.path.exists(out_file))

            with open(out_file, "r", encoding="utf-8") as f:
                content = f.read()
            self.assertIn("### TypeScript Rules v0.1.1", content)
            self.assertIn("Fixed Starlark f-string quote syntax error", content)

    def test_cli_missing_changelog_file_uses_fallback(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            nonexistent = os.path.join(tmpdir, "DOES_NOT_EXIST.md")
            cmd = [
                sys.executable,
                self.script_path,
                "--changelog",
                nonexistent,
                "--version",
                "0.1.1",
                "--title",
                "Kotlin",
            ]
            res = subprocess.run(cmd, capture_output=True, text=True)
            self.assertEqual(res.returncode, 0, msg=res.stderr)
            self.assertIn("### Kotlin Rules v0.1.1", res.stdout)
            self.assertIn(
                "Precompiled hermetic binaries and build definitions for Kotlin.",
                res.stdout,
            )


if __name__ == "__main__":
    unittest.main()
