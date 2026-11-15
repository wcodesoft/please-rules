_VERSION = "0.11.1"

remote_file(
    name = "please_ts",
    binary = True,
    hashes = [
    "9cd1a274edccd30da463adf6e9c7a6677a2823b83b0fe5ac339d5763abf1b123",  # linux_amd64
    "212208567a4b67fb2d0a5111c446efb22cdbc0979c8192a837c4223703e6ba7c",  # linux_arm64
    "ba0e9f3ba71044042dcffd2e407e29517e2c5fed51e4e1dc499687307c089953",  # darwin_amd64
    "69ba91b089a018b137bd7c734bea825dbc33818f8e4a45efe1ad8ff875b21da7",  # darwin_arm64
],
    url = f"https://github.com/wcodesoft/please-rules/releases/download/ts-v{_VERSION}/please_ts-{_VERSION}-{CONFIG.OS}_{CONFIG.ARCH}",
    visibility = ["PUBLIC"],
)
