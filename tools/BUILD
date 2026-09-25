_VERSION = "0.3.0"

remote_file(
    name = "please_ts",
    binary = True,
    hashes = [
    "3b86bbc51f6f1a843464dbddb33ab6cf0d7141e7d76e1afb8e39a143148a6a6f",  # linux_amd64
    "13b8113e8b4f59bdade0a9827dab32f608ad4014f47a03d3eefad075c739d536",  # linux_arm64
    "f5e2599c732d96a2366ffaa169c58394fbfdcc770d39f8e96a63d35a03245ccd",  # darwin_amd64
    "93dd134f565246cd3c2e84e37183cdc1c4282aeb7ea5beee5d20c2d812815eed",  # darwin_arm64
],
    url = f"https://github.com/wcodesoft/please-rules/releases/download/ts-v{_VERSION}/please_ts-{_VERSION}-{CONFIG.OS}_{CONFIG.ARCH}",
    visibility = ["PUBLIC"],
)
