_VERSION = "0.1.1"

remote_file(
    name = "please_ts",
    binary = True,
    hashes = [
    "6cf79399bad5793d3671c365f0e0ed09118949c0e91b328a16f1f8daf18436c6",  # linux_amd64
    "dcf80d5c370c2bebab45a6e7d67677c1980331c7a0ea039bf2320433d0316149",  # linux_arm64
    "26c043f0ae1cbe6f616e9d37e7f07b25b0645d9dd84e2d44468d305eb157038e",  # darwin_amd64
    "6f5802b6a38850885daef8fb5c465c16359dc68d94688922d106cc7fdb6ee59b",  # darwin_arm64
],
    url = f"https://github.com/wcodesoft/please-rules/releases/download/ts-v{_VERSION}/please_ts-{_VERSION}-{CONFIG.OS}_{CONFIG.ARCH}",
    visibility = ["PUBLIC"],
)
