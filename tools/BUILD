_VERSION = "0.3.3"

remote_file(
    name = "please_kotlin",
    url = f"https://github.com/wcodesoft/please-rules/releases/download/kotlin-v{_VERSION}/please_kotlin-{_VERSION}-{CONFIG.OS}_{CONFIG.ARCH}",
    hashes = [
    "2bde09f36d58758c943243e0de5e3d2e11ea00a56121335d1f84e01459bac5e4",  # linux_amd64
    "d35dd816c6fc17e6d3e1259dd67326a592b3473bc7c19b4cdb23c1f537b90d0a",  # linux_arm64
    "d540007b4e06a5f1ea27df8c54b23769d34c3b9f105f26491f63d5a7c42554e4",  # darwin_amd64
    "432f3f0be1bb391777b65f70f2cfb40b772d3e3e99d35df2665a290a014a8c8c",  # darwin_arm64
          ],
    binary = True,
    visibility = ["PUBLIC"],
)

remote_file(
    name = "please_kotlin_wasm",
    url = f"https://github.com/wcodesoft/please-rules/releases/download/kotlin-v{_VERSION}/please_kotlin_wasm-{_VERSION}-{CONFIG.OS}_{CONFIG.ARCH}",
    hashes = [
    "bf5cda3925f82e83a808c626ec209141d7145e395f72a34daac3cb2204ed91dd",  # linux_amd64
    "b3c955f0d42f93b33b6f04d83b328b31b485f23a9e533cae5a9f16ccb1605c61",  # linux_arm64
    "0742a0ce979f590d91f46cbb2f458dad081ef0c4d5c5a74d5b3c630785375a06",  # darwin_amd64
    "5d593da693b53ca11a2bcb5baaf95860db1d11016cc580582ff2b72a12822678",  # darwin_arm64
          ],
    binary = True,
    visibility = ["PUBLIC"],
)
