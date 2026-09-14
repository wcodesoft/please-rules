_VERSION = "0.1.0"

remote_file(
    name = "please_swift",
    binary = True,
    hashes = [
        "5e7932a4615b44706272c5e11e340c4c3c331b18305e94ded0a5ceb4feedd03e",  # linux_amd64
        "5cea1918bfd760ca7977206b3bcc77e7095d013c84fdbf55b3e9331c1e894691",  # linux_arm64
        "93319ebd0f765a32954d0211527c78d38bf377a0e581ce82249dab17d0ce9bdf",  # darwin_amd64
        "6f346e26300c1e46f37d17e14f537df42daa8992c82a59a058761e37418432fa",  # darwin_arm64
    ],
    url = f"https://github.com/wcodesoft/please-rules/releases/download/swift-v{_VERSION}/please_swift-{_VERSION}-{CONFIG.OS}_{CONFIG.ARCH}",
    visibility = ["PUBLIC"],
)
