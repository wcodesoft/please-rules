_VERSION = "0.1.1"

remote_file(
    name = "please_swift",
    binary = True,
    hashes = [
        "c2ab05fa9feb38e373632a05ba5988a462c404dd47fbfa79bafcec8247b795d3",  # linux_amd64
        "0b54a558333b9e4f92cd534ddcbdebea861093afed8b6e4e2bbeb4e209119b11",  # linux_arm64
        "0b7c9eaaf06b39a248a15a5bbb2da8fd23fdfc3951221d0584de9db16df3d416",  # darwin_amd64
        "14ebfb048a634b9c1abb048d0ed8ba55fab0d99e1c7d0d56759b592aeab0859e",  # darwin_arm64
    ],
    url = f"https://github.com/wcodesoft/please-rules/releases/download/swift-v{_VERSION}/please_swift-{_VERSION}-{CONFIG.OS}_{CONFIG.ARCH}",
    visibility = ["PUBLIC"],
)
