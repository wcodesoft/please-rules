_VERSION = "0.7.0"

remote_file(
    name = "please_ts",
    binary = True,
    hashes = [
    "69aac772cf6ffaf5b3c437de97912b8b52bdc7dee839d3970b7c661b2e9d4536",  # linux_amd64
    "2dba97dc193ee4318b91f6a9f2642b7de4812053968881e88f9553937fc8b718",  # linux_arm64
    "4b113a186920ecd1d5bc0c32dd98f555cdb156ce283875397eefcb839aa05dba",  # darwin_amd64
    "1b4b449ed9d2e6fb4a737bfd37e65c656e56e1f28babadb44305f32e70efb2ba",  # darwin_arm64
],
    url = f"https://github.com/wcodesoft/please-rules/releases/download/ts-v{_VERSION}/please_ts-{_VERSION}-{CONFIG.OS}_{CONFIG.ARCH}",
    visibility = ["PUBLIC"],
)
