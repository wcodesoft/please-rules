_VERSION = "0.3.0"

remote_file(
    name = "please_swift",
    binary = True,
    hashes = [
    "d39ae9f9957369216548e435972ff4b3a9db7f41b5ed6934ec47ba5cd29dfc55",  # linux_amd64
    "289c4ed439a0d0a5335475148782f04ec8480e3f80d949b2b482a95e8e8ccf8b",  # linux_arm64
    "aba332f68e913189a3bad30dd2e64e409e0c60c9ffcba2176eee629a5a75edbf",  # darwin_amd64
    "11c1a4bedf6af44f394ceeaf424a6558b7d3bce2e16a120714af88a0b7e11d98",  # darwin_arm64
],
    url = f"https://github.com/wcodesoft/please-rules/releases/download/swift-v{_VERSION}/please_swift-{_VERSION}-{CONFIG.OS}_{CONFIG.ARCH}",
    visibility = ["PUBLIC"],
)
