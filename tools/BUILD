_VERSION = "0.1.1"

remote_file(
    name = "please_swift",
    binary = True,
    hashes = [
        "4fe75e9eb8c3266165b0ffa97a59438ead308a2a00b0db755d4ac01fc5caea99",  # linux_amd64
        "b2c446b651f8733736c6c23850c9886273c03e147b07ff58bd9fe09d6d9d6f19",  # linux_arm64
        "6157749eae4f053b9f0a0dca2910bb8634ea7858a7d22762a8868405d70e33cc",  # darwin_amd64
        "005f0d6a488f5591fdde1331a152383ef4946ce05d2ed3364597f141de934778",  # darwin_arm64
    ],
    url = f"https://github.com/wcodesoft/please-rules/releases/download/swift-v{_VERSION}/please_swift-{_VERSION}-{CONFIG.OS}_{CONFIG.ARCH}",
    visibility = ["PUBLIC"],
)
