#!/usr/bin/env python3
"""
strip_swift_toolchain.py

Pulls official Swift toolchain archives, strips non-compiler / non-runtime assets,
and produces a minimal Swift toolchain distribution bundle (~150-190MB xz / ~300MB gz)
suitable for hermetic Please builds.

Retained components:
- bin/swiftc, bin/swift, bin/swift-frontend, bin/swift-driver, bin/swift-demangle, bin/swift-plugin-server
- bin/clang, bin/clang++, bin/clang-17, bin/ld.lld, bin/lld, bin/*.cfg
- bin/llvm-profdata, bin/llvm-cov (for test coverage)
- lib/swift/linux (Swift stdlib, Glibc, Foundation, Synchronization, Regex, Testing, XCTest)
- lib/swift/host (compiler host libraries required by swift-frontend)
- lib/swift/shims, lib/swift/dispatch, lib/swift/Block, lib/swift/CoreFoundation, lib/swift/_FoundationCShims, lib/swift/_foundation_unicode
- lib/swift_static (static libraries and modules for static linking)
- lib/clang (compiler builtin headers)
- lib/libIndexStore.so*, lib/libswiftDemangle.so*, lib/libswiftGenericMetadataBuilder.so*
- libexec/swift/linux/swift-backtrace*
- LICENSE.txt & copyright notices

Stripped components:
- LLDB debugger & server (bin/lldb*, lib/liblldb*, bin/repl_swift, local/lib/python*/dist-packages/lldb)
- Swift Package Manager (bin/swift-build*, bin/swift-package*, bin/swift-run, bin/swift-test, bin/swift-sdk, bin/swift-experimental-sdk, lib/swift/pm, share/pm)
- DocC documentation compiler (bin/docc, share/docc)
- Language server & tools (bin/sourcekit-lsp, bin/clangd, lib/libsourcekitdInProc.so, include/SourceKit)
- Python 3 bindings / runtime (lib/python3*, local/lib/python3*)
- Bare-metal embedded toolchain (lib/swift/embedded, lib/swift_static/embedded)
- LLVM LTO libraries (lib/libLTO.so*)
- LLVM archiver tools not required by Please (bin/llvm-ar, bin/llvm-ranlib)
- Diagnostics, manual pages, SDK module lists, documentation (share/man, share/doc, bin/sdk-module-lists, bin/swift-api-*, bin/swift-symbolgraph-extract, bin/swift-format, bin/swift-help)
"""

import argparse
import hashlib
import os
import shutil
import subprocess
import sys
import tarfile
import tempfile
import urllib.request

OFFICIAL_SWIFT_RELEASES = {
    "6.0.3": {
        "linux_amd64": {
            "url": "https://download.swift.org/swift-6.0.3-release/ubuntu2204/swift-6.0.3-RELEASE/swift-6.0.3-RELEASE-ubuntu22.04.tar.gz",
            "strip_prefix": "swift-6.0.3-RELEASE-ubuntu22.04/usr",
        },
        "linux_arm64": {
            "url": "https://download.swift.org/swift-6.0.3-release/ubuntu2204-aarch64/swift-6.0.3-RELEASE/swift-6.0.3-RELEASE-ubuntu22.04-aarch64.tar.gz",
            "strip_prefix": "swift-6.0.3-RELEASE-ubuntu22.04-aarch64/usr",
        },
    },
    "6.3.3": {
        "linux_amd64": {
            "url": "https://download.swift.org/swift-6.3.3-release/ubuntu2204/swift-6.3.3-RELEASE/swift-6.3.3-RELEASE-ubuntu22.04.tar.gz",
            "strip_prefix": "swift-6.3.3-RELEASE-ubuntu22.04/usr",
        },
        "linux_arm64": {
            "url": "https://download.swift.org/swift-6.3.3-release/ubuntu2204-aarch64/swift-6.3.3-RELEASE/swift-6.3.3-RELEASE-ubuntu22.04-aarch64.tar.gz",
            "strip_prefix": "swift-6.3.3-RELEASE-ubuntu22.04-aarch64/usr",
        },
    },
}

# Directories and patterns to remove relative to the extracted root (usr)
STRIP_PATTERNS = [
    # LLDB
    "bin/lldb*",
    "lib/liblldb*",
    "bin/repl_swift",
    "local/lib/python*",
    "lib/python*",
    # SPM
    "bin/swift-build*",
    "bin/swift-package*",
    "bin/swift-run",
    "bin/swift-test",
    "bin/swift-sdk",
    "bin/swift-experimental-sdk",
    "lib/swift/pm",
    "share/pm",
    # DocC
    "bin/docc",
    "share/docc",
    # SourceKit & LSP
    "bin/sourcekit-lsp",
    "bin/clangd",
    "lib/libsourcekitdInProc.so*",
    "include/SourceKit",
    # Embedded bare-metal
    "lib/swift/embedded",
    "lib/swift_static/embedded",
    # LTO
    "lib/libLTO*",
    # Unneeded utilities & SDK tools
    "bin/llvm-ar",
    "bin/llvm-ranlib",
    "bin/sdk-module-lists",
    "bin/swift-api-*",
    "bin/swift-symbolgraph-extract",
    "bin/swift-build-sdk-interfaces",
    "bin/swift-format",
    "bin/swift-help",
    "bin/plutil",
    # Docs & Man
    "share/man",
    "share/doc",
    "lib/swift/FrameworkABIBaseline",
]


def compute_sha256(file_path: str) -> str:
    h = hashlib.sha256()
    with open(file_path, "rb") as f:
        while chunk := f.read(65536):
            h.update(chunk)
    return h.hexdigest()


def download_file(url: str, dest_path: str):
    print(f"Downloading {url} to {dest_path}...")
    urllib.request.urlretrieve(url, dest_path)
    print("Download complete.")


def strip_directory(root_dir: str):
    print(f"Stripping non-compiler assets in {root_dir}...")
    import glob

    for pattern in STRIP_PATTERNS:
        matches = glob.glob(os.path.join(root_dir, pattern))
        for match in matches:
            if os.path.islink(match) or os.path.isfile(match):
                os.remove(match)
            elif os.path.isdir(match):
                shutil.rmtree(match)

    # Ensure LICENSE.txt is preserved at top level
    license_src = os.path.join(root_dir, "share", "swift", "LICENSE.txt")
    license_dst = os.path.join(root_dir, "LICENSE.txt")
    if os.path.exists(license_src) and not os.path.exists(license_dst):
        shutil.copy2(license_src, license_dst)

    # Clean up empty directories
    for root, dirs, files in os.walk(root_dir, topdown=False):
        for dir_name in dirs:
            dir_path = os.path.join(root, dir_name)
            try:
                if not os.listdir(dir_path):
                    os.rmdir(dir_path)
            except OSError:
                pass


def package_tarball(source_dir: str, output_tarball: str):
    print(f"Packaging {output_tarball} from {source_dir}...")
    # Package flat directly from source_dir
    cmd = ["tar", "-czf", output_tarball, "-C", source_dir, "."]
    subprocess.check_call(cmd)
    size_mb = os.path.getsize(output_tarball) / (1024 * 1024)
    sha256 = compute_sha256(output_tarball)
    print(f"Created {output_tarball}: {size_mb:.1f} MB, SHA-256: {sha256}")
    return sha256


def process_platform(version: str, platform: str, output_dir: str, archive_path: str = None) -> dict:
    info = OFFICIAL_SWIFT_RELEASES.get(version, {}).get(platform)
    if not info:
        raise ValueError(f"Unknown version {version} or platform {platform}")

    url = info["url"]
    strip_prefix = info.get("strip_prefix", "")

    with tempfile.TemporaryDirectory() as temp_dir:
        download_target = archive_path
        if not download_target:
            download_target = os.path.join(temp_dir, os.path.basename(url))
            download_file(url, download_target)

        extract_dir = os.path.join(temp_dir, "extracted")
        os.makedirs(extract_dir, exist_ok=True)

        print(f"Extracting {download_target}...")
        subprocess.check_call(["tar", "-xzf", download_target, "-C", extract_dir])

        root_dir = os.path.join(extract_dir, strip_prefix) if strip_prefix else extract_dir
        if not os.path.isdir(root_dir):
            raise FileNotFoundError(f"Expected extracted directory {root_dir} not found")

        strip_directory(root_dir)

        os.makedirs(output_dir, exist_ok=True)
        out_filename = f"swift-{version}-minimal-{platform}.tar.gz"
        out_path = os.path.join(output_dir, out_filename)

        sha256 = package_tarball(root_dir, out_path)
        return {
            "platform": platform,
            "version": version,
            "file": out_filename,
            "path": out_path,
            "sha256": sha256,
        }


def main():
    parser = argparse.ArgumentParser(description="Strip official Swift toolchains into minimal distributions")
    parser.add_argument("--version", default="6.0.3", help="Swift version (default: 6.0.3)")
    parser.add_argument("--platform", choices=["linux_amd64", "linux_arm64", "all"], default="linux_amd64", help="Platform to process")
    parser.add_argument("--archive", help="Optional local path to pre-downloaded official tarball")
    parser.add_argument("--output-dir", default="./dist", help="Output directory for minimal bundles")
    args = parser.parse_args()

    platforms = ["linux_amd64", "linux_arm64"] if args.platform == "all" else [args.platform]
    results = []

    for plat in platforms:
        archive = args.archive if len(platforms) == 1 else None
        res = process_platform(args.version, plat, args.output_dir, archive)
        results.append(res)

    print("\nSummary:")
    for res in results:
        print(f"  {res['platform']}: {res['file']} -> {res['sha256']}")


if __name__ == "__main__":
    main()
