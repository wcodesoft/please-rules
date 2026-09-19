_VERSION = "0.5.0"

remote_file(
    name = "please_rust",
    url = f"https://github.com/wcodesoft/please-rules/releases/download/rust-v{_VERSION}/please_rust-{_VERSION}-{CONFIG.OS}_{CONFIG.ARCH}",
    hashes = [
    "58e3d697a7772e7db29f3c707142af7c6bb67647b53453083f1e3cad3875d581",  # linux_amd64
    "3ec16945bdf339eb3db7b8876a5fe641ea75eb9921bbf6869d27a06d3855b791",  # linux_arm64
    "dcb1b32736fb987b6fc109c9ec6189023f590b63ffbb66080faede436d579041",  # darwin_amd64
    "745233f8061483eb195a9e4819338f9475f3a5e966fc7d416545fea14d749910",  # darwin_arm64
],
    binary = True,
    visibility = ["PUBLIC"],
)
