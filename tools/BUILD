_VERSION = "0.5.0"

remote_file(
    name = "please_kotlin",
    url = f"https://github.com/wcodesoft/please-rules/releases/download/kotlin-v{_VERSION}/please_kotlin-{_VERSION}-{CONFIG.OS}_{CONFIG.ARCH}",
    hashes = [
    "da0e5aa8e364b0f2084256378858b3a2ae3b8a7cc478f0a0c68adc977479e060",  # linux_amd64
    "2f403f6d5536917efebc44df625135d3fc9368aecb0bf952d4353d9c645c8c74",  # linux_arm64
    "54f0cdf1d3c14a13af15823d87d0c5bc1ab3bfb89bc7b17415b5a2f30791cb96",  # darwin_amd64
    "98608ecfb8949ce7b5127591365f02423cc812d1d09c8b0dd2e356572bf8e86c",  # darwin_arm64
          ],
    binary = True,
    visibility = ["PUBLIC"],
)
