_VERSION = "0.5.1"

remote_file(
    name = "please_ts",
    binary = True,
    hashes = [
    "cd9437af62a729923573587c1dfd56080168a780340705e19aaea28f6616e5cc",  # linux_amd64
    "187d53700abf28acacb43a85e08ecfe685c02ce5fcd77af5bbecb0a66194777a",  # linux_arm64
    "2fe96824b4b8c278ae7ef65bac571ba7a575fa099c1016cb74ff77fc572afe83",  # darwin_amd64
    "f9ec8af54a65c60e184e1a539b4d4196af9c91343072bb54213ae907a3de7aee",  # darwin_arm64
],
    url = f"https://github.com/wcodesoft/please-rules/releases/download/ts-v{_VERSION}/please_ts-{_VERSION}-{CONFIG.OS}_{CONFIG.ARCH}",
    visibility = ["PUBLIC"],
)
