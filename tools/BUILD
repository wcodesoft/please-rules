_VERSION = "0.4.0"

remote_file(
    name = "please_kotlin",
    url = f"https://github.com/wcodesoft/please-rules/releases/download/kotlin-v{_VERSION}/please_kotlin-{_VERSION}-{CONFIG.OS}_{CONFIG.ARCH}",
    hashes = [
    "5d74a551fa44d6d6c560a86b03e9ff4a6106a77c0b5da248fad0e19b1f51f279",  # linux_amd64
    "55ad6fcaacbec0afc359b5ecbd8bf0e24d57291163e8d56e72a0ceb9e3a7cfb4",  # linux_arm64
    "c22a60313907e177e0ffcc9c3aa39ba0089f07fd00799ed38252ed86a9394298",  # darwin_amd64
    "45fe02fb28b52c18a191b68bbe5285f1c0ad0ee16b4667d2890b87e9a2f69449",  # darwin_arm64
          ],
    binary = True,
    visibility = ["PUBLIC"],
)

remote_file(
    name = "please_kotlin_wasm",
    url = f"https://github.com/wcodesoft/please-rules/releases/download/kotlin-v{_VERSION}/please_kotlin_wasm-{_VERSION}-{CONFIG.OS}_{CONFIG.ARCH}",
    hashes = [
    "06d811d9969a4a6b9043da136f43411aa97644ae582d1a5c4ff45b8f1ea17e5f",  # linux_amd64
    "63e2a15c4220b6cebcf3c01e2b564055b9e8178c4778aa7d45228f9ecfafc310",  # linux_arm64
    "b1899bc93844ff8527466ba28c38d25539e563d162725fb1f01989fd4c3278ab",  # darwin_amd64
    "9f697ddd9d80541ea73c15175be1d8f6b45da26d11d35fb0ca5fb60d63530e64",  # darwin_arm64
          ],
    binary = True,
    visibility = ["PUBLIC"],
)
