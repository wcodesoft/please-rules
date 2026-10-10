#!/usr/bin/env python3
"""Fails when a language branch edits a file that main owns.

main holds the shared infrastructure and every language branch gets it with `git merge
main`. A shared file that a branch edits as well is what makes that merge conflict, so
a change to one belongs on main. A pull request into a language branch passes when
every shared file it changes is identical to main's version, which is what a sync of
main looks like, and fails when a shared file is edited, added or deleted on the branch.

Usage (in a checkout of the pull request, with both refs fetched):

    check_shared_files.py --base origin/swift --main origin/main
"""

import argparse
import subprocess
import sys

# Paths main owns: a file, or a directory when the entry ends with a slash.
SHARED = (
    ".github/workflows/release.yml",
    ".github/workflows/ci-main.yml",
    ".github/workflows/sync-downstream.yml",
    ".github/actions/",
    "scripts/",
    "tools/common/",
    ".agents/",
    "AGENTS.md",
    ".prettierrc",
    ".markdownlint.json",
)


def is_shared(path):
    """Whether a path is one main owns."""
    return any(path == s or (s.endswith("/") and path.startswith(s)) for s in SHARED)


def violations(changed, head_blob, main_blob):
    """The shared files among `changed` whose content differs from main's.

    head_blob and main_blob map a path to its blob id, or None when the file does not
    exist on that side (a file deleted on both sides is the same, not a difference).
    """
    return sorted(p for p in changed if is_shared(p) and head_blob(p) != main_blob(p))


def git(*args):
    return subprocess.run(
        ["git", *args], capture_output=True, text=True, check=False
    )


def blob_at(ref):
    """A function giving the blob id of a path at a ref, or None if it is not there."""

    def blob(path):
        out = git("rev-parse", "--verify", "--quiet", f"{ref}:{path}")
        return out.stdout.strip() if out.returncode == 0 else None

    return blob


def changed_files(base):
    """The files the pull request changes: what HEAD has beyond the base."""
    out = git("diff", "--name-only", f"{base}...HEAD")
    if out.returncode != 0:
        raise SystemExit(f"could not diff {base}...HEAD: {out.stderr.strip()}")
    return [line for line in out.stdout.splitlines() if line]


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__.split("\n")[0])
    parser.add_argument("--base", required=True, help="the branch the PR targets")
    parser.add_argument("--main", default="origin/main", help="the ref of main")
    args = parser.parse_args(argv)

    bad = violations(changed_files(args.base), blob_at("HEAD"), blob_at(args.main))
    if not bad:
        print("No shared file is edited on this branch.")
        return 0
    print("::error::Shared files cannot be edited on a language branch.")
    print("These files are owned by main and differ from main's version:")
    for path in bad:
        print(f"  {path}")
    print()
    print("Change them on main (a pull request into main) and let the sync bring the")
    print("change here, or, if this branch needs something of its own, keep it in a")
    print("path that main does not own. See docs/ci.md.")
    return 1


if __name__ == "__main__":
    sys.exit(main())
