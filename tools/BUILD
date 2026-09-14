_VERSION = "0.1.0"

remote_file(
    name = "please_swift",
    binary = True,
    hashes = [
        "202a2a0fe83dbe71be5e0cc614728828375a7b3d14ac1ae7bb702f9a07131993",  # linux_amd64
        "a145c37ee1c4644d31b1f6a51b5abbc5690bafeb813588e086ab2ff651335423",  # linux_arm64
        "d15ca2091b6d92378304eaf116c61975d24286511f7331ffe3739ba97478323d",  # darwin_amd64
        "c1a655d1c111d562f599c8e952920599951b602a833668670b85b8dd21dcb756",  # darwin_arm64
    ],
    url = f"https://github.com/wcodesoft/please-rules/releases/download/swift-v{_VERSION}/please_swift-{_VERSION}-{CONFIG.OS}_{CONFIG.ARCH}",
    visibility = ["PUBLIC"],
)
