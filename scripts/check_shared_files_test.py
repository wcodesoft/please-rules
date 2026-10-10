#!/usr/bin/env python3
import os
import subprocess
import tempfile
import unittest

import check_shared_files as guard


def run(cwd, *args):
    env = dict(
        os.environ,
        GIT_AUTHOR_NAME="t",
        GIT_AUTHOR_EMAIL="t@t",
        GIT_COMMITTER_NAME="t",
        GIT_COMMITTER_EMAIL="t@t",
    )
    subprocess.run(["git", *args], cwd=cwd, env=env, check=True, capture_output=True)


def write(root, path, text):
    full = os.path.join(root, path)
    os.makedirs(os.path.dirname(full), exist_ok=True)
    with open(full, "w") as f:
        f.write(text)


class IsShared(unittest.TestCase):
    def test_files_and_directories_main_owns(self):
        for path in (
            ".github/workflows/release.yml",
            ".github/actions/format-check/action.yml",
            "scripts/BUILD",
            "scripts/swift/anything.py",
            "tools/common/lcovcheck/main.go",
            "AGENTS.md",
        ):
            self.assertTrue(guard.is_shared(path), path)

    def test_what_a_branch_owns_or_appends_to(self):
        for path in (
            ".github/workflows/ci-swift.yml",
            ".github/workflows/build-swift-minimal.yml",
            "tools/swift_toolchain/BUILD",
            "tools/please_swift/main.go",
            "build_defs/swift/swift.build_defs",
            "test/swift/lib/BUILD",
            "docs/README.md",
            "docs/swift/README.md",
            ".plzconfig",
            "plugins/BUILD",
            "CHANGELOG.md",
            "scriptsx/other.py",
            "tools/commonplace/x",
        ):
            self.assertFalse(guard.is_shared(path), path)


class Violations(unittest.TestCase):
    def blobs(self, head, main):
        return (lambda p: head.get(p)), (lambda p: main.get(p))

    def test_a_shared_file_that_differs_is_a_violation(self):
        h, m = self.blobs({"scripts/BUILD": "a"}, {"scripts/BUILD": "b"})
        self.assertEqual(guard.violations(["scripts/BUILD"], h, m), ["scripts/BUILD"])

    def test_a_shared_file_equal_to_mains_is_a_sync(self):
        h, m = self.blobs({"scripts/BUILD": "b"}, {"scripts/BUILD": "b"})
        self.assertEqual(guard.violations(["scripts/BUILD"], h, m), [])

    def test_a_shared_file_added_on_the_branch_is_a_violation(self):
        h, m = self.blobs({"scripts/strip.py": "a"}, {})
        self.assertEqual(guard.violations(["scripts/strip.py"], h, m), ["scripts/strip.py"])

    def test_a_shared_file_deleted_on_the_branch_is_a_violation(self):
        h, m = self.blobs({}, {"scripts/BUILD": "b"})
        self.assertEqual(guard.violations(["scripts/BUILD"], h, m), ["scripts/BUILD"])

    def test_a_file_deleted_on_both_sides_is_the_same(self):
        h, m = self.blobs({}, {})
        self.assertEqual(guard.violations(["scripts/old.py"], h, m), [])

    def test_files_the_branch_owns_are_never_checked(self):
        h, m = self.blobs({"tools/please_swift/x.go": "a"}, {})
        self.assertEqual(guard.violations(["tools/please_swift/x.go"], h, m), [])

    def test_the_result_is_sorted(self):
        h, m = self.blobs({"scripts/b": "1", "scripts/a": "1"}, {})
        self.assertEqual(guard.violations(["scripts/b", "scripts/a"], h, m), ["scripts/a", "scripts/b"])


class InARepository(unittest.TestCase):
    """The same rule through real git: a main, a language branch and what each does."""

    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.dir = self.tmp.name
        run(self.dir, "init", "-q", "-b", "main")
        write(self.dir, "scripts/BUILD", "shared 1\n")
        write(self.dir, "docs/README.md", "index\n")
        run(self.dir, "add", "-A")
        run(self.dir, "commit", "-q", "-m", "base")
        run(self.dir, "branch", "swift")

    def tearDown(self):
        self.tmp.cleanup()

    def commit(self, message):
        run(self.dir, "add", "-A")
        run(self.dir, "commit", "-q", "-m", message)

    def check(self):
        cwd = os.getcwd()
        os.chdir(self.dir)
        try:
            return guard.main(["--base", "swift", "--main", "main"])
        finally:
            os.chdir(cwd)

    def feature(self):
        run(self.dir, "checkout", "-q", "-b", "feature", "swift")

    def test_a_feature_that_touches_only_its_own_files_passes(self):
        self.feature()
        write(self.dir, "tools/please_swift/main.go", "package main\n")
        write(self.dir, "docs/README.md", "index with a swift link\n")
        self.commit("swift work")
        self.assertEqual(self.check(), 0)

    def test_a_feature_that_edits_a_shared_file_fails(self):
        self.feature()
        write(self.dir, "scripts/BUILD", "shared 1, plus swift\n")
        self.commit("edit shared")
        self.assertEqual(self.check(), 1)

    def test_a_feature_that_adds_a_file_to_a_shared_directory_fails(self):
        self.feature()
        write(self.dir, "scripts/strip_swift_toolchain.py", "print()\n")
        self.commit("add to shared")
        self.assertEqual(self.check(), 1)

    def test_a_sync_of_main_passes_even_though_it_changes_shared_files(self):
        run(self.dir, "checkout", "-q", "main")
        write(self.dir, "scripts/BUILD", "shared 2\n")
        self.commit("main changes a shared file")
        run(self.dir, "checkout", "-q", "-b", "sync", "swift")
        run(self.dir, "merge", "-q", "--no-edit", "main")
        self.assertEqual(self.check(), 0)

    def test_a_branch_that_only_has_an_older_copy_of_a_shared_file_passes(self):
        # main moved on after the branch was cut, and the PR does not touch the file.
        run(self.dir, "checkout", "-q", "main")
        write(self.dir, "scripts/BUILD", "shared 2\n")
        self.commit("main moves on")
        self.feature()
        write(self.dir, "tools/please_swift/main.go", "package main\n")
        self.commit("swift work")
        self.assertEqual(self.check(), 0)


if __name__ == "__main__":
    unittest.main()
