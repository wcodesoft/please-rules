_VERSION = "0.9.0"

remote_file(
    name = "please_ts",
    binary = True,
    hashes = [
    "f6960e5d0f93d783effec7e9ee328d729512fa2f6182cdc944513c2a8c17a94f",  # linux_amd64
    "4bede4d7328bde8c733c25988e6be5b08dd18b20818e2c040d2ae46b772ee7b5",  # linux_arm64
    "cc46bddbd4c35710ecc6c1b675cfd32f268848a5668c9d7a805d6006221da438",  # darwin_amd64
    "a68bd3e47643c442d195d9b611fad126f5bfbfec3d74b82aed08790747c67e67",  # darwin_arm64
],
    url = f"https://github.com/wcodesoft/please-rules/releases/download/ts-v{_VERSION}/please_ts-{_VERSION}-{CONFIG.OS}_{CONFIG.ARCH}",
    visibility = ["PUBLIC"],
)
