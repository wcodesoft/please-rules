_VERSION = "0.1.2"

remote_file(
    name = "please_swift",
    binary = True,
    hashes = [
    "b75982a31d2b115129139d543c60ef6b1cd37336e40dacff29c91b495b4acb34",  # linux_amd64
    "3b7961b0194814863e8719ae7a38e342717f31ab799401a138776f0c117903fa",  # linux_arm64
    "394fedf992fb558d230989ae2243bbefbf057fcbbc27d5e2b2ae008cb922bba0",  # darwin_amd64
    "40c90c068d117521065b6be2fc46ef497c7e83bf3ccfe083a4cb67ac6f074a11",  # darwin_arm64
],
    url = f"https://github.com/wcodesoft/please-rules/releases/download/swift-v{_VERSION}/please_swift-{_VERSION}-{CONFIG.OS}_{CONFIG.ARCH}",
    visibility = ["PUBLIC"],
)
