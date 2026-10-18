_VERSION = "0.3.1"

remote_file(
    name = "please_swift",
    binary = True,
    hashes = [
    "563cc488bdbecfc6ef2360db2775e3c1f35e5162fdd8622d116d9b38049976cb",  # linux_amd64
    "2c4125df3e0b7f4612a3e54093b4e16629dcb4f426faa641a8c47aa3f6c055e6",  # linux_arm64
    "cdffd36df41f0e5e7b16e612ee8bc201dac802faa3abe4a7ef79cc1cc748ba56",  # darwin_amd64
    "1acde750ccda7a71a10de969ea882080e1a39eabaec5e05350e3a16d515cfd11",  # darwin_arm64
],
    url = f"https://github.com/wcodesoft/please-rules/releases/download/swift-v{_VERSION}/please_swift-{_VERSION}-{CONFIG.OS}_{CONFIG.ARCH}",
    visibility = ["PUBLIC"],
)
