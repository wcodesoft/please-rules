_VERSION = "0.3.2"

remote_file(
    name = "please_swift",
    binary = True,
    hashes = [
    "a14e174ee5a70edae18a1c76bbc18adf8d74e9ae5700391ab98191bb9dd3da4d",  # linux_amd64
    "dfeb2d23158dd7f29f6f510ce854657586b494ff35b7075e96be21351fee5631",  # linux_arm64
    "eb639e05c54359db5905c432abe1b9281be0a74edfe55c1f3a20b1c6aa821b92",  # darwin_amd64
    "28e42baca403d0a285a87cd00cf7d1b74d237c66465075547805146249005835",  # darwin_arm64
],
    url = f"https://github.com/wcodesoft/please-rules/releases/download/swift-v{_VERSION}/please_swift-{_VERSION}-{CONFIG.OS}_{CONFIG.ARCH}",
    visibility = ["PUBLIC"],
)
