_VERSION = "0.1.3"

remote_file(
    name = "please_kotlin",
    url = f"https://github.com/wcodesoft/please-rules/releases/download/kotlin-v{_VERSION}/please_kotlin-{_VERSION}-{CONFIG.OS}_{CONFIG.ARCH}",
    hashes = [
        "fd5d32631fac33cab06c52867070a1b5d69f7f1f40e46b8fe96a4a1816178e04",  # linux_amd64
        "9ed0f455b99ccd58e35539bfbffd36e3f6db14fa2900faa8abb2081153f1475b",  # linux_arm64
        "cb17007daf263a2adce84d4a53bd331045e7546ad64af0dcce34363119bba364",  # darwin_amd64
        "e2b2d2eaff7b856e51229acb9553c67be9ffb08f8001f62f4b72cbf3eb3b871a",  # darwin_arm64
    ],
    binary = True,
    visibility = ["PUBLIC"],
)
