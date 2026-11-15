_VERSION = "0.11.0"

remote_file(
    name = "please_ts",
    binary = True,
    hashes = [
    "2ff10ecb6f1546a6ffad6a43cd517854319347259e0163669164d30fda96b2cc",  # linux_amd64
    "58cc384c01fe695a000b9c610fec88995c22884dccebcae00882524bedbc2d02",  # linux_arm64
    "ccdb36a0eadae1cd2fece228433f2859936b8327532c3c6963d98d536d2ac22a",  # darwin_amd64
    "a96c825da4cb064c7c074d1b076bacfc2081516c7160374f271e1eb4f7fc8e23",  # darwin_arm64
],
    url = f"https://github.com/wcodesoft/please-rules/releases/download/ts-v{_VERSION}/please_ts-{_VERSION}-{CONFIG.OS}_{CONFIG.ARCH}",
    visibility = ["PUBLIC"],
)
