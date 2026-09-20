_VERSION = "0.3.1"

remote_file(
    name = "please_wit",
    binary = True,
    hashes = [
        "eb6ab8496bec4b35473478c1606fe461c414ab5adf1a25a0e6ae407c920080e2",  # linux_amd64
        "d0603e782a4900c14190847988688b5ea8f1a4153f8493577ca1e23df66ad707",  # linux_arm64
        "2a45982b1287f43621a44c2525e878ed34c06eda833c62c9dade62ba22edff05",  # darwin_amd64
        "6ad2bccc853780958e166e1f532b64667bb2cf063088194704512d59d5f0763b",  # darwin_arm64
    ],
    url = f"https://github.com/wcodesoft/please-rules/releases/download/wit-v{_VERSION}/please_wit-{_VERSION}-{CONFIG.OS}_{CONFIG.ARCH}",
    visibility = ["PUBLIC"],
)
