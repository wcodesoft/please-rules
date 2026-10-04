#!/usr/bin/env python3
"""Pin a release in tools/BUILD: the version and the sha256 of each platform binary.

Each language rule set downloads its Go helper tool from a GitHub release through
`remote_file` targets in tools/BUILD. After a release has been built, this script rewrites
`_VERSION` and the `hashes` of those targets, computing the hashes from the binaries
themselves.

The file is edited by structure, not by pattern: it finds the `remote_file(...)` block of
each target by its name, replaces the lines of its `hashes = [...]` list, and then reads
the file back and checks the result. Anything unexpected (a missing or duplicated target,
a hash line without a platform comment, a platform that is not built) is an error, so a
change in the format of tools/BUILD stops the release instead of shipping stale pins.

Usage:
    update_release_pins.py --build tools/BUILD --version 0.6.1 \\
        --target please_rust=dist/please_rust-0.6.1
    # looks for dist/please_rust-0.6.1-<platform> for each platform

A target is NAME=PREFIX; give --target once per tool (the Kotlin rules have two).
"""

import argparse
import hashlib
import os
import sys

PLATFORMS = ("linux_amd64", "linux_arm64", "darwin_amd64", "darwin_arm64")
HASH_INDENT = "    "


class PinError(Exception):
    """tools/BUILD or the built binaries are not in the expected shape."""


def sha256_of(path):
    digest = hashlib.sha256()
    with open(path, "rb") as f:
        for chunk in iter(lambda: f.read(1 << 20), b""):
            digest.update(chunk)
    return digest.hexdigest()


def hashes_for(prefix):
    """The sha256 of <prefix>-<platform> for every platform."""
    result = {}
    for platform in PLATFORMS:
        path = f"{prefix}-{platform}"
        if not os.path.isfile(path):
            raise PinError(f"binary not found: {path}")
        result[platform] = sha256_of(path)
    return result


def find_blocks(lines, name):
    """Return (start, end) line indexes of the remote_file block whose name is `name`.

    A block runs from a line `remote_file(` to the next line that is exactly `)`.
    """
    found = []
    i = 0
    while i < len(lines):
        if lines[i].strip() == "remote_file(":
            end = i + 1
            while end < len(lines) and lines[end].rstrip() != ")":
                end += 1
            if end == len(lines):
                raise PinError(f"unterminated remote_file block at line {i + 1}")
            if any(line.strip() == f'name = "{name}",' for line in lines[i:end]):
                found.append((i, end))
            i = end
        i += 1
    if len(found) != 1:
        raise PinError(f'expected exactly one remote_file named "{name}", found {len(found)}')
    return found[0]


def find_hash_list(lines, start, end, name):
    """Return (open, close) line indexes of the `hashes = [` list inside a block."""
    opens = [i for i in range(start, end) if lines[i].strip() == "hashes = ["]
    if len(opens) != 1:
        raise PinError(f'expected exactly one "hashes = [" in "{name}", found {len(opens)}')
    close = next((i for i in range(opens[0] + 1, end) if lines[i].strip() in ("],", "]")), None)
    if close is None:
        raise PinError(f'the hashes list of "{name}" is not closed')
    return opens[0], close


def read_pins(text):
    """Parse tools/BUILD into (version, {target: {platform: hash}}), strictly."""
    lines = text.split("\n")
    versions = [
        line.split("=", 1)[1].strip().strip('"')
        for line in lines
        if line.startswith("_VERSION") and "=" in line
    ]
    if len(versions) != 1:
        raise PinError(f"expected exactly one _VERSION, found {len(versions)}")

    targets = {}
    i = 0
    while i < len(lines):
        if lines[i].strip() == "remote_file(":
            end = i + 1
            while end < len(lines) and lines[end].rstrip() != ")":
                end += 1
            names = [
                line.strip()[len('name = "'):-2]
                for line in lines[i:end]
                if line.strip().startswith('name = "') and line.strip().endswith('",')
            ]
            if len(names) != 1:
                raise PinError(f"remote_file at line {i + 1} has no single name")
            opened, closed = find_hash_list(lines, i, end, names[0])
            pins = {}
            for line in lines[opened + 1:closed]:
                entry, _, platform = line.partition("#")
                value = entry.strip().rstrip(",").strip('"')
                platform = platform.strip()
                if not value or not platform:
                    raise PinError(f'a hash of "{names[0]}" has no platform comment: {line!r}')
                pins[platform] = value
            targets[names[0]] = pins
            i = end
        i += 1
    return versions[0], targets


def update(text, version, wanted):
    """Return text with _VERSION and the hashes of each wanted target replaced.

    `wanted` maps a target name to {platform: sha256}.
    """
    lines = text.split("\n")

    version_lines = [i for i, line in enumerate(lines) if line.startswith("_VERSION")]
    if len(version_lines) != 1:
        raise PinError(f"expected exactly one _VERSION, found {len(version_lines)}")
    lines[version_lines[0]] = f'_VERSION = "{version}"'

    # Edit from the bottom up so that earlier line indexes stay valid.
    spans = []
    for name in wanted:
        start, end = find_blocks(lines, name)
        spans.append((find_hash_list(lines, start, end, name), name))
    for (opened, closed), name in sorted(spans, key=lambda s: s[0][0], reverse=True):
        old = {}
        for line in lines[opened + 1:closed]:
            _, _, platform = line.partition("#")
            old[platform.strip()] = line
        if set(old) != set(PLATFORMS):
            raise PinError(
                f'the hashes of "{name}" are for {sorted(old)}, expected {sorted(PLATFORMS)}'
            )
        lines[opened + 1:closed] = [
            f'{HASH_INDENT}"{wanted[name][platform]}",  # {platform}' for platform in PLATFORMS
        ]

    result = "\n".join(lines)

    # Read the file back: what was written is what was asked for.
    got_version, got = read_pins(result)
    if got_version != version:
        raise PinError(f"_VERSION is {got_version} after the update, expected {version}")
    for name, pins in wanted.items():
        if got.get(name) != pins:
            raise PinError(f'the hashes of "{name}" are not what was written')
    return result


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument("--build", default="tools/BUILD", help="the BUILD file to update")
    parser.add_argument("--version", required=True, help="release version, without a leading v")
    parser.add_argument("--target", action="append", required=True, metavar="NAME=PREFIX",
                        help="a remote_file target and the path prefix of its binaries")
    args = parser.parse_args(argv)

    wanted = {}
    for spec in args.target:
        name, sep, prefix = spec.partition("=")
        if not sep or not name or not prefix:
            parser.error(f"--target must be NAME=PREFIX, got {spec!r}")
        if name in wanted:
            parser.error(f"target {name} given twice")
        wanted[name] = hashes_for(prefix)

    with open(args.build) as f:
        text = f.read()
    updated = update(text, args.version.lstrip("v"), wanted)
    with open(args.build, "w") as f:
        f.write(updated)
    for name in wanted:
        print(f"pinned {name} {args.version.lstrip('v')} in {args.build}")


if __name__ == "__main__":
    try:
        main()
    except PinError as err:
        print(f"error: {err}", file=sys.stderr)
        sys.exit(1)
