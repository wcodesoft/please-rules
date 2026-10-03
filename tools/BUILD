_VERSION = "0.6.0"

remote_file(
    name = "please_ts",
    binary = True,
    hashes = [
    "e421afca476d2af3ede0ba6e1e9d173d32234b94369a85f999d6d0d9b015ff3e",  # linux_amd64
    "6636be8172c34ab41e7830358d7fa8a135c4ca8c69c883c3d629d028d8f1192e",  # linux_arm64
    "69f1ec973d727548eaf45932e7cfd0217449774423416965a03e5e36a63a0f7e",  # darwin_amd64
    "c25354a8cc736717115de25f7b45583ede18a9672f1f0fc434a2d9dfe0338972",  # darwin_arm64
],
    url = f"https://github.com/wcodesoft/please-rules/releases/download/ts-v{_VERSION}/please_ts-{_VERSION}-{CONFIG.OS}_{CONFIG.ARCH}",
    visibility = ["PUBLIC"],
)
