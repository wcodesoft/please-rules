_VERSION = "0.1.3"

remote_file(
    name = "please_swift",
    binary = True,
    hashes = [
    "1927aefa023cec8bc571ac9c4d1c669c00418dbb163500b1c25b986c094bf3a2",  # linux_amd64
    "94cdf72947ec52a1906cf6ef3d00997005c2ce1d3b9a74468b27e07e0b50d81f",  # linux_arm64
    "15b08faff3bcb0a9cc2d40e331015c9691ceedccf3d32948ec6472531aea5827",  # darwin_amd64
    "aa7b64f2982438d8f8e3fff929ab6f01ccffa8d2ea11772a9c857166119502d9",  # darwin_arm64
],
    url = f"https://github.com/wcodesoft/please-rules/releases/download/swift-v{_VERSION}/please_swift-{_VERSION}-{CONFIG.OS}_{CONFIG.ARCH}",
    visibility = ["PUBLIC"],
)
