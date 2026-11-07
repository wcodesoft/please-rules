_VERSION = "0.9.1"

remote_file(
    name = "please_ts",
    binary = True,
    hashes = [
    "d5b334bbf11fbbefd517a12706337c285567122c8c2e1f7efb79f7d96064f8b7",  # linux_amd64
    "8cd9956333058b3a649d3c103b4bb6fe8960715bfc52b703c87e35ba6de4998d",  # linux_arm64
    "d424d8ca1a3356ac3e93c78b33cc47dd64344b104e66579ebdab08e106124384",  # darwin_amd64
    "8aec07057fd6d3b71b78c27326e9e0f0b430116442ce73977b487538fca0a1ea",  # darwin_arm64
],
    url = f"https://github.com/wcodesoft/please-rules/releases/download/ts-v{_VERSION}/please_ts-{_VERSION}-{CONFIG.OS}_{CONFIG.ARCH}",
    visibility = ["PUBLIC"],
)
