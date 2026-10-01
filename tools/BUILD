_VERSION = "0.5.2"

remote_file(
    name = "please_ts",
    binary = True,
    hashes = [
    "8c5a5121e897b476ef1deba464ffa33e20659799f513ba3c39fcf9d4f29ab040",  # linux_amd64
    "bceba6315551f81a1da2468212807dec34a03229eeb146b3851d0ed169646c43",  # linux_arm64
    "305f5f1d1244ca967586259547750d238bbad635180c54e8a4ed89b1bf522c3d",  # darwin_amd64
    "383434d1771e500c745ba769e8590d53dd4ddb185a236ee21b9e1f3a359b83d0",  # darwin_arm64
],
    url = f"https://github.com/wcodesoft/please-rules/releases/download/ts-v{_VERSION}/please_ts-{_VERSION}-{CONFIG.OS}_{CONFIG.ARCH}",
    visibility = ["PUBLIC"],
)
