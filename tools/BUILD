_VERSION = "0.3.1"

remote_file(
    name = "please_kotlin",
    url = f"https://github.com/wcodesoft/please-rules/releases/download/kotlin-v{_VERSION}/please_kotlin-{_VERSION}-{CONFIG.OS}_{CONFIG.ARCH}",
    hashes = [
        "7e679bbca6d1bd802225ca3c685897cd59bb97599c106efc3406ac3ffd534671",  # linux_amd64
        "3ebd358ae37cbcb5a8ef9d221a06fa62ec964029ba2ed70cf5c663ab82d30baf",  # linux_arm64
        "5ee5379be8c420971e06abb7bf1584e1c023a50d3530c5b96f2a669bc1d0e320",  # darwin_amd64
        "37f775724152e145a3994d3b1382f68153f8f54c4957d14934fa39e9e6cd96fb",  # darwin_arm64
    ],
    binary = True,
    visibility = ["PUBLIC"],
)

remote_file(
    name = "please_kotlin_wasm",
    url = f"https://github.com/wcodesoft/please-rules/releases/download/kotlin-v{_VERSION}/please_kotlin_wasm-{_VERSION}-{CONFIG.OS}_{CONFIG.ARCH}",
    hashes = [
        "196f71871a4aee9866cc0f0ddeebf6be6714e12c0cd72ec5cc967971fa9c2317",  # linux_amd64
        "d8f5f49a7db50a3af92c38adc611785470251352db64c9c44eee35b94e14d230",  # linux_arm64
        "f9bac4df158a398015e2c250d8b1cd125f9fad53cf11a24b79470f3d918c895a",  # darwin_amd64
        "963f20343dd2980f1e9bf3b2da15418e134bd5f7225cdccf40f38747e97a0bae",  # darwin_arm64
    ],
    binary = True,
    visibility = ["PUBLIC"],
)
