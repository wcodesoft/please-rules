_VERSION = "0.3.0"

remote_file(
    name = "please_kotlin",
    url = f"https://github.com/wcodesoft/please-rules/releases/download/kotlin-v{_VERSION}/please_kotlin-{_VERSION}-{CONFIG.OS}_{CONFIG.ARCH}",
    hashes = [
    "572a449d182714243b65e593c6306b671564def6bc46bc31d7644799e0779bbc",  # linux_amd64
    "955cb2182f31d03ced546769cc390c299ba20b65c3d0b5f54f6761b73fbd9bce",  # linux_arm64
    "0eedcb454dd33de3bb69d1857f6ecf7b0fda75a3479af7a15b1b66b725326f90",  # darwin_amd64
    "08e70f69c4d1c0d3a96cd8635ca1b29b144e913a2c8de79cc02d43846b421d1a",  # darwin_arm64
],
    binary = True,
    visibility = ["PUBLIC"],
)
