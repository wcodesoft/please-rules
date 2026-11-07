_VERSION = "0.8.0"

remote_file(
    name = "please_ts",
    binary = True,
    hashes = [
    "c215fd154d4183d1adf8d6740dc148e69d6248550d77f6c69ec56a9444c33335",  # linux_amd64
    "f4c72debbf2fcae29aa272e4826495e597e3eba7d926d2007e5d302185120cea",  # linux_arm64
    "db5562287ab55741ae8d0f26225ad92cad6e770d269671b2cadd59dfa426ff52",  # darwin_amd64
    "4599bb0f2ae851cea1f184ae89c6403f5ba3c1e06589e9d663418929d09de34a",  # darwin_arm64
],
    url = f"https://github.com/wcodesoft/please-rules/releases/download/ts-v{_VERSION}/please_ts-{_VERSION}-{CONFIG.OS}_{CONFIG.ARCH}",
    visibility = ["PUBLIC"],
)
