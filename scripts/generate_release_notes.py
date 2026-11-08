#!/usr/bin/env python3
"""
Release notes generator for please-rules.
Extracts release notes from CHANGELOG.md for a given version and formats
them cleanly for GitHub Releases without automatic changelog or PR links.
"""

import argparse
import os
import sys


def _is_list_item(line: str) -> bool:
    """Returns True if line begins with a Markdown unordered (-, *, +) or ordered (1.) list marker."""
    stripped = line.lstrip(" \t")
    if len(stripped) >= 2 and stripped[0] in "-*+" and stripped[1] in " \t":
        return True
    dot = stripped.find(".")
    if dot > 0 and stripped[:dot].isdigit() and len(stripped) > dot + 1 and stripped[dot + 1] in " \t":
        return True
    return False


def unwrap_markdown(text: str) -> str:
    """
    Unwraps hard line breaks within paragraphs and bullet list items so that
    release notes flow naturally across the full width of GitHub Releases,
    while preserving headings, code blocks, blank lines, and nested list indentation.
    """
    lines = text.split("\n")
    out = []
    in_code_block = False

    current_block = []
    current_type = None  # "list", "para", None

    def flush():
        nonlocal current_block, current_type
        if not current_block:
            return
        if current_type == "list":
            first_line = current_block[0]
            rest = [l.strip() for l in current_block[1:]]
            if rest:
                out.append(first_line + " " + " ".join(rest))
            else:
                out.append(first_line)
        elif current_type == "para":
            out.append(" ".join(l.strip() for l in current_block))
        else:
            out.extend(current_block)
        current_block = []
        current_type = None

    for line in lines:
        if line.strip().startswith("```"):
            flush()
            in_code_block = not in_code_block
            out.append(line)
            continue

        if in_code_block:
            out.append(line)
            continue

        stripped = line.strip()
        if not stripped:
            flush()
            out.append("")
            continue

        if line.startswith("#"):
            flush()
            out.append(line)
            continue

        if _is_list_item(line):
            flush()
            current_type = "list"
            current_block = [line]
        elif current_type == "list":
            current_block.append(line)
        else:
            if current_type != "para":
                flush()
                current_type = "para"
            current_block.append(line)

    flush()
    return "\n".join(out)


def _extract_version_section(changelog_content: str, clean_version: str) -> str:
    """Scans changelog_content line by line for the ## [version] section."""
    target_headers = (f"## [{clean_version}]", f"## [v{clean_version}]")
    in_section = False
    in_code_block = False
    collected = []

    for line in changelog_content.splitlines():
        if line.strip().startswith("```"):
            in_code_block = not in_code_block

        if not in_code_block and line.startswith("## ["):
            if in_section:
                break
            if any(
                line == h or line.startswith(h + " ") or line.startswith(h + "\t")
                for h in target_headers
            ):
                in_section = True
                continue

        if in_section:
            collected.append(line)

    return "\n".join(collected).strip()


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
        notes = _extract_version_section(changelog_content, clean_version)

    if not notes:
        notes = f"Precompiled hermetic binaries and build definitions for {title}."
    else:
        notes = unwrap_markdown(notes)

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
