_VERSION = "0.2.0"

remote_file(
    name = "please_swift",
    binary = True,
    hashes = [
    "0c3b59b51e37507437c910087ea2b56afb4f339bd623f8fde7f1ed23f9814cc8",  # linux_amd64
    "06a9ebbec9037d94c48d84597800ca01d22d340d07ece3970034cef09c0e3842",  # linux_arm64
    "b87f4f78da1b852caf7bf0bd14fc75d6a35099c3ae094bcca5308a87713a5c72",  # darwin_amd64
    "7cd2ccae46cf6d9e4c5bbbcf9f37c105e7246a3eae354bb02bdc7e70192fa98d",  # darwin_arm64
],
    url = f"https://github.com/wcodesoft/please-rules/releases/download/swift-v{_VERSION}/please_swift-{_VERSION}-{CONFIG.OS}_{CONFIG.ARCH}",
    visibility = ["PUBLIC"],
)
