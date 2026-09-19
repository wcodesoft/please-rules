_VERSION = "0.2.0"

remote_file(
    name = "please_wit",
    binary = True,
    hashes = [
    "2563090b1ec9eb30c530fec876abf6e7b104376a1319ec6e25dd5c61945c9539",  # linux_amd64
    "30fdbb8a7d6d210f00d516c3994b75fd9d67af93df9bba60ec12a57c9853e48f",  # linux_arm64
    "74abc795e828d4dc4f48aa09e3147134e74a949f1cd8dd8ec9e1a0f8819eb234",  # darwin_amd64
    "c04a60dca0de002f34d43618dd160a31065e03f3f544fae76f465cf578053984",  # darwin_arm64
],
    url = f"https://github.com/wcodesoft/please-rules/releases/download/wit-v{_VERSION}/please_wit-{_VERSION}-{CONFIG.OS}_{CONFIG.ARCH}",
    visibility = ["PUBLIC"],
)
