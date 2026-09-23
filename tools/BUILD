_VERSION = "0.3.4"

remote_file(
    name = "please_kotlin",
    url = f"https://github.com/wcodesoft/please-rules/releases/download/kotlin-v{_VERSION}/please_kotlin-{_VERSION}-{CONFIG.OS}_{CONFIG.ARCH}",
    hashes = [
    "87d77b32c269fb8115071c534c06b4f66a9afb382abda85abe7ab38c2fd1f1f5",  # linux_amd64
    "88131843e90a467e1346aaf62acf91b9fc949070723e73a9562ad972f18cdf08",  # linux_arm64
    "d321dd6967d0d70ced3b308ac6e0f44b630ae0d614eecc3abbf73fd85074479f",  # darwin_amd64
    "9309014dd4b64686dda1edfc51e752f1f7114e48eb66c92414fef4f85ec217be",  # darwin_arm64
          ],
    binary = True,
    visibility = ["PUBLIC"],
)

remote_file(
    name = "please_kotlin_wasm",
    url = f"https://github.com/wcodesoft/please-rules/releases/download/kotlin-v{_VERSION}/please_kotlin_wasm-{_VERSION}-{CONFIG.OS}_{CONFIG.ARCH}",
    hashes = [
    "6c0a099c4e6129370cf8e3449e1eb2384b3d69f7575a9c4cf4e45c730f1124b9",  # linux_amd64
    "51efda58b17495c83fba9d09c3ea9e4636265277ffa94cd2149d375d48dc37b8",  # linux_arm64
    "d9a534b092e11e9bae39b6a3d9da4d7b4928e6dc24503bb4214b7452986a36b8",  # darwin_amd64
    "860863988484da25fea208a4654a1293946df50cd983b5675eb1a69198248db7",  # darwin_arm64
          ],
    binary = True,
    visibility = ["PUBLIC"],
)
