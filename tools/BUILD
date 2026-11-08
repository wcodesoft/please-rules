_VERSION = "0.9.2"

remote_file(
    name = "please_ts",
    binary = True,
    hashes = [
    "a7c66c1987a6d2ddd1b8d6ea4897652ce64adb5bd37a71073af246ae1f99a43d",  # linux_amd64
    "5d19986e94e41007c16f4564f1ec91eee8efb669a307289e02ca828f456bae16",  # linux_arm64
    "50cba41c9b7881f478168302d5adc756495c1cfec16cd4185ef9bf8eeba0b69b",  # darwin_amd64
    "8253b0fdaeaae786483d2bee60a690dbdcae74996ab41df477166f0bca0d75bf",  # darwin_arm64
],
    url = f"https://github.com/wcodesoft/please-rules/releases/download/ts-v{_VERSION}/please_ts-{_VERSION}-{CONFIG.OS}_{CONFIG.ARCH}",
    visibility = ["PUBLIC"],
)
