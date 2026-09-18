_VERSION = "0.1.2"

remote_file(
    name = "please_ts",
    binary = True,
    hashes = [
    "39cbc93a625ae9dc95d7a4f1d87c76cb0b7fc92069584fbdac0963fb53d1c866",  # linux_amd64
    "5c3f42eb6db804e6bbe0c895857f0d0e785c7f95a277a71dbc78a08e17a4821d",  # linux_arm64
    "5a76137ca572cacdc7f6f7b035341cbdc7f064230dbeed8785cd2793628471f4",  # darwin_amd64
    "8306ddaab6bff52207451b54b06418d3b00c94d9e0013486a0db96e81ece93bb",  # darwin_arm64
],
    url = f"https://github.com/wcodesoft/please-rules/releases/download/ts-v{_VERSION}/please_ts-{_VERSION}-{CONFIG.OS}_{CONFIG.ARCH}",
    visibility = ["PUBLIC"],
)
