_VERSION = "0.7.2"

remote_file(
    name = "please_ts",
    binary = True,
    hashes = [
    "05693d54b972f103e05b963467c1e50efee721f1cecc29c4358bb0aed600a15f",  # linux_amd64
    "b666c5a9734d9bfe53b191ee8eae222a5bbded39fef86e411813959a797936dc",  # linux_arm64
    "3e9288e435fe4979fc10c6d36025db70942dd56544606d9ca0070e17e59a1274",  # darwin_amd64
    "158df7a8974970f67c3cf27c3752b5e3a9a838ec1c8139265c23f2c897e3b729",  # darwin_arm64
],
    url = f"https://github.com/wcodesoft/please-rules/releases/download/ts-v{_VERSION}/please_ts-{_VERSION}-{CONFIG.OS}_{CONFIG.ARCH}",
    visibility = ["PUBLIC"],
)
