_VERSION = "0.1.0"

remote_file(
    name = "please_ts",
    binary = True,
    hashes = [
        "0d5a048fde46877cb70d931f7229c684f42228d649843017432e4e24f10876cb",  # linux_amd64
        "d8051a700600bbcce564d416d7355b415b7abc48d3bf888b6c50d43aeba7990e",  # linux_arm64
        "8de1361e1a107819e4db58a96e98c3ba829798173005b05193bd93924929c72b",  # darwin_amd64
        "e4e918cd4d6b6e6907605605879403535c6885f1fb92a183b600a07a6726408a",  # darwin_arm64
    ],
    url = f"https://github.com/wcodesoft/please-rules/releases/download/ts-v{_VERSION}/please_ts-{_VERSION}-{CONFIG.OS}_{CONFIG.ARCH}",
    visibility = ["PUBLIC"],
)
