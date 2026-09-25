_VERSION = "0.4.0"

remote_file(
    name = "please_ts",
    binary = True,
    hashes = [
    "a7943cb2cfc38088b232262ed6e98b1b236b8f65e7bc85a4d6426a557931fda6",  # linux_amd64
    "a3a81ee28afd8d486a475eea46898c36306e87df14cd65ed3140b54c7e0f7df8",  # linux_arm64
    "ff2fac732ba94549ac6581e0935031285761fe9ec6334827e755807407dc7bb9",  # darwin_amd64
    "303b5d5712c74ad1c1f16ae8ec6e2c95e5d770d1f874d09af14241c5817a3030",  # darwin_arm64
],
    url = f"https://github.com/wcodesoft/please-rules/releases/download/ts-v{_VERSION}/please_ts-{_VERSION}-{CONFIG.OS}_{CONFIG.ARCH}",
    visibility = ["PUBLIC"],
)
