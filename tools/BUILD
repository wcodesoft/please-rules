_VERSION = "0.3.0"

remote_file(
    name = "please_wit",
    binary = True,
    hashes = [
    "29618322173bf0b948e46085ebf4c56d77559ac74d7bf796057812f61c04a5c1",  # linux_amd64
    "ffae15c0057602f5c1a38a9465e6644513c2b7b255bde5b9635f3a1e6b1347c9",  # linux_arm64
    "776d430aad1dae9d869acfaca880132fffc1df470aebebb8a7d6be450919c8d1",  # darwin_amd64
    "a155690404ae8c82b1a4dea6c9f699f6855d69e5a4178365a2172aa0c69d89f0",  # darwin_arm64
],
    url = f"https://github.com/wcodesoft/please-rules/releases/download/wit-v{_VERSION}/please_wit-{_VERSION}-{CONFIG.OS}_{CONFIG.ARCH}",
    visibility = ["PUBLIC"],
)
