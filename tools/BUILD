_VERSION = "0.6.0"

remote_file(
    name = "please_rust",
    url = f"https://github.com/wcodesoft/please-rules/releases/download/rust-v{_VERSION}/please_rust-{_VERSION}-{CONFIG.OS}_{CONFIG.ARCH}",
    hashes = [
    "2b91fa776bc3302221ad55c174067be0888d16a61daac847bfc6410644956886",  # linux_amd64
    "c11486d43adc6a27a17aa63de85785bc6adfabc69058c84b8a8c6767e29c7036",  # linux_arm64
    "1a5882754cb9f2c631b4745b8ec7325e2af41a7f8cd8d24c846a34cb6c645b35",  # darwin_amd64
    "9f0ce9730a8d183be59e50aca4af834aab5e5f86d4c25ae196e5b517f8b01198",  # darwin_arm64
],
    binary = True,
    visibility = ["PUBLIC"],
)
