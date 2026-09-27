_VERSION = "0.5.0"

remote_file(
    name = "please_ts",
    binary = True,
    hashes = [
    "95f84a5c3f8835223922e402429155fa3e6fc3260d262e2e6286c5c7bbfbdd82",  # linux_amd64
    "497ec358eb0f917013d969b5e7b5c19cb6c060160faaeebac5c713ccc7f2e5ea",  # linux_arm64
    "0fc275388fa8e87b8937114a13d89d9e003e5994eff65e8fa98987b903c61a25",  # darwin_amd64
    "9438876a057215d982e5beb9c0a24bb00d6b04e10c2be3d0ab1a5caf886314c7",  # darwin_arm64
],
    url = f"https://github.com/wcodesoft/please-rules/releases/download/ts-v{_VERSION}/please_ts-{_VERSION}-{CONFIG.OS}_{CONFIG.ARCH}",
    visibility = ["PUBLIC"],
)
