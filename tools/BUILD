_VERSION = "0.2.1"

remote_file(
    name = "please_ts",
    binary = True,
    hashes = [
    "68273ac8c16b9e59d12843337503851acbebab90984d553299f9112edf5f926f",  # linux_amd64
    "b4319d4c970fd2b803bb054e82d946c44e9f61d2de40e900f70334316853c1fe",  # linux_arm64
    "b05bd880be322266105979a3c186850de7eef75e1499f2468fd8566f5dfe590f",  # darwin_amd64
    "96679892f62bc2c0d1d28cf21a99678906b9ed891729068545d6a065eea17cf8",  # darwin_arm64
],
    url = f"https://github.com/wcodesoft/please-rules/releases/download/ts-v{_VERSION}/please_ts-{_VERSION}-{CONFIG.OS}_{CONFIG.ARCH}",
    visibility = ["PUBLIC"],
)
