_VERSION = "0.4.1"

remote_file(
    name = "please_rust",
    url = f"https://github.com/wcodesoft/please-rules/releases/download/rust-v{_VERSION}/please_rust-{_VERSION}-{CONFIG.OS}_{CONFIG.ARCH}",
    hashes = [
        "e816f67c123af8ebc36e3e7e0faace7715a511e353c61ad2516e2c2fe672baf0",  # linux_amd64
        "7dcbd5c095e8ebfec05c6b9838ce5d662327030e7f2e8039fe9bca332d5c26a3",  # linux_arm64
        "f5ee1b73bec712b4c05db99271c3fd8934c8ec19805734bcc18a07ff21ec9bf9",  # darwin_amd64
        "2ba15f8441985152023bdbe5f0f6862fb08d3771275c9d4caaecb4af95aee3ee",  # darwin_arm64
    ],
    binary = True,
    visibility = ["PUBLIC"],
)
