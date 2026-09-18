#!/usr/bin/env python3
"""
Release notes generator for please-rules.
Extracts release notes from CHANGELOG.md for a given version and formats
them cleanly for GitHub Releases without automatic changelog or PR links.
"""

import argparse
import os
import re
import sys


def extract_release_notes(changelog_content: str, version: str, title: str) -> str:
    """
    Extracts the release notes section for a given version from changelog content.

    Args:
        changelog_content: Raw string content of CHANGELOG.md.
        version: Target version string (e.g., '0.1.1' or 'v0.1.1').
        title: Display title for the language rules (e.g., 'TypeScript', 'Swift').

    Returns:
        Formatted markdown release notes.
    """
    clean_version = version.lstrip("v")
    notes = ""

    if changelog_content:
        pattern = rf"## \[v?{re.escape(clean_version)}\][^\n]*\n(.*?)(?=\n## \[|\Z)"
        match = re.search(pattern, changelog_content, re.DOTALL)
        if match:
            notes = match.group(1).strip()

    if not notes:
        notes = f"Precompiled hermetic binaries and build definitions for {title}."

    return f"### {title} Rules v{clean_version}\n\n{notes}\n"


def main() -> int:
    parser = argparse.ArgumentParser(
        description="Extract and format release notes from CHANGELOG.md for GitHub Releases."
    )
    parser.add_argument(
        "--changelog",
        default="CHANGELOG.md",
        help="Path to CHANGELOG.md file (default: CHANGELOG.md)",
    )
    parser.add_argument(
        "--version",
        required=True,
        help="Version to extract (e.g., '0.1.1' or 'v0.1.1')",
    )
    parser.add_argument(
        "--title",
        required=True,
        help="Language title (e.g., 'TypeScript', 'Swift', 'Kotlin', 'Rust')",
    )
    parser.add_argument(
        "--out",
        help="Output file path for generated release notes (prints to stdout if omitted)",
    )

    args = parser.parse_args()

    changelog_content = ""
    if os.path.exists(args.changelog):
        with open(args.changelog, "r", encoding="utf-8") as f:
            changelog_content = f.read()

    notes = extract_release_notes(changelog_content, args.version, args.title)

    if args.out:
        out_dir = os.path.dirname(args.out)
        if out_dir:
            os.makedirs(out_dir, exist_ok=True)
        with open(args.out, "w", encoding="utf-8") as f:
            f.write(notes)
    else:
        sys.stdout.write(notes)

    return 0


if __name__ == "__main__":
    sys.exit(main())
