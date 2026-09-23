_VERSION = "0.3.5"

remote_file(
    name = "please_kotlin",
    url = f"https://github.com/wcodesoft/please-rules/releases/download/kotlin-v{_VERSION}/please_kotlin-{_VERSION}-{CONFIG.OS}_{CONFIG.ARCH}",
    hashes = [
    "73543bb6c7281f10e80d3ecf9d23ae205c0bd676a5552aef531f0ca909b203b7",  # linux_amd64
    "0cc1f4ee777329217397d61bfe2b54b4e9a46bbe105ea9337de7fff48386c099",  # linux_arm64
    "f2344fb3853a0388bc36efcd2e964f21b60809e71fdc884bc7b40fe404b1324e",  # darwin_amd64
    "070b72ac26e67d967614e680ccff28aa9d3c67099da8d4bcb912f128bf4b1b38",  # darwin_arm64
          ],
    binary = True,
    visibility = ["PUBLIC"],
)

remote_file(
    name = "please_kotlin_wasm",
    url = f"https://github.com/wcodesoft/please-rules/releases/download/kotlin-v{_VERSION}/please_kotlin_wasm-{_VERSION}-{CONFIG.OS}_{CONFIG.ARCH}",
    hashes = [
    "59db999a1be7f7c0f725cdce7e778b7bd9a046f9e7eb14a4a660cb3353d17d8c",  # linux_amd64
    "a14f016b3f82c705e1b28aa117eeb42adb12c3564c59ded126cec75d261e6e9c",  # linux_arm64
    "8b72ab7d5c77dc49fc7c2d1f41adfea8aa10fc327b4e4075d6e5307e35ce0b0d",  # darwin_amd64
    "8e4b249d160423ef4d8404fc5169122f0ff3a6ae348807201a48af20b484d759",  # darwin_arm64
          ],
    binary = True,
    visibility = ["PUBLIC"],
)
