_VERSION = "0.3.5"

remote_file(
    name = "please_kotlin",
    url = f"https://github.com/wcodesoft/please-rules/releases/download/kotlin-v{_VERSION}/please_kotlin-{_VERSION}-{CONFIG.OS}_{CONFIG.ARCH}",
    hashes = [
    "73543bb6c7281f10e80d3ecf9d23ae205c0bd676a5552aef531f0ca909b203b7",  # linux_amd64
    "0cc1f4ee777329217397d61bfe2b54b4e9a46bbe105ea9337de7fff48386c099",  # linux_arm64
    "f2344fb3853a0388bc36efcd2e964f21b60809e71fdc884bc7b40fe404b1324e",  # darwin_amd64
    "070b72ac26e67d967614e680ccff28aa9d3c67099da8d4bcb912f128bf4b1b38",  # darwin_arm64
          ],
    binary = True,
    visibility = ["PUBLIC"],
)
