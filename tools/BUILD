_VERSION = "0.1.0"

remote_file(
    name = "please_wit",
    binary = True,
    hashes = [
    "c51db31c0c9cece32c224cebab68cd187c4e65c9a66d76d21db707c64674bb8a",  # linux_amd64
    "0e374338cb654dd57a7f0dd927a8790c897b3cb51c0ffd48de6a2f85420082d2",  # linux_arm64
    "a1984f16d01edaec81fa72338e28ec4bf3323720e2147a969e1200328617c027",  # darwin_amd64
    "095c1809d049b91c6ab7e34c1cfa7f97651a501f12e57012b0e92d38b424e155",  # darwin_arm64
],
    url = f"https://github.com/wcodesoft/please-rules/releases/download/wit-v{_VERSION}/please_wit-{_VERSION}-{CONFIG.OS}_{CONFIG.ARCH}",
    visibility = ["PUBLIC"],
)
