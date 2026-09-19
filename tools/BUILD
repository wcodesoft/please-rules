_VERSION = "0.2.0"

remote_file(
    name = "please_kotlin",
    url = f"https://github.com/wcodesoft/please-rules/releases/download/kotlin-v{_VERSION}/please_kotlin-{_VERSION}-{CONFIG.OS}_{CONFIG.ARCH}",
    hashes = [
    "e94c1215fd9382121f8f8a307c13731e8c0a71e8f0dc6237aba0cae16e4dc3d4",  # linux_amd64
    "074591ee6bdb32a0b0ea93f9d672c03a61ef278d2bb12801806649bc165a717b",  # linux_arm64
    "b595269b32b209102ddba707009848a7ff04d852b05969d078e1ec59c793f645",  # darwin_amd64
    "693aa158a73a0bd2a70f29804cd8a07d645133d151eb6ba156bb279caeadeb44",  # darwin_arm64
],
    binary = True,
    visibility = ["PUBLIC"],
)
