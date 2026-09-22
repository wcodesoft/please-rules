_VERSION = "0.3.2"

remote_file(
    name = "please_kotlin",
    url = f"https://github.com/wcodesoft/please-rules/releases/download/kotlin-v{_VERSION}/please_kotlin-{_VERSION}-{CONFIG.OS}_{CONFIG.ARCH}",
    hashes = [
    "afdd506dead34baa688c863ee7e1e652ec37e7a7cf2de2a3b7b2fba77b92cd4d",  # linux_amd64
    "d5eb4256a03dc16d5c68ac96c5978cbf0a9ad7479d3a38a068037d286d0dfb61",  # linux_arm64
    "be38c29316dc7959e98ec2c1a97d98c7640ae04671e211fd4a3e2f6cf64da343",  # darwin_amd64
    "5c4360896d4331374bdb96e44998620449606170712c842ef1cc3855dcf5b687",  # darwin_arm64
          ],
    binary = True,
    visibility = ["PUBLIC"],
)

remote_file(
    name = "please_kotlin_wasm",
    url = f"https://github.com/wcodesoft/please-rules/releases/download/kotlin-v{_VERSION}/please_kotlin_wasm-{_VERSION}-{CONFIG.OS}_{CONFIG.ARCH}",
    hashes = [
    "ecafeb3387d66594cae76b2159e9af59a7b7b56bf1e09f79f961f748e43ffefb",  # linux_amd64
    "3a063a59839446bd3858348dc17d1ff182727c55797b3800ed1cb2766f470556",  # linux_arm64
    "1141342bfb46f9f95eb60c3aa3a951f3efb3fef7eeccb27594fedfaf236901db",  # darwin_amd64
    "a5e83bf0dee9119729d26530cffdcd42670d3e3e2ccd4a16abfba21398587697",  # darwin_arm64
          ],
    binary = True,
    visibility = ["PUBLIC"],
)
