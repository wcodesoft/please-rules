_VERSION = "0.7.1"

remote_file(
    name = "please_ts",
    binary = True,
    hashes = [
    "343e53bf9596bf0959187c3731b6ccb5d25f8dbabc2eb35450fc457a4725ceac",  # linux_amd64
    "2f4b961fa4ecf49e372440c80b19c74401a6c20ad535860581a668992e08b030",  # linux_arm64
    "3a5b28ce96109d92d2f098986271f11f1b3ebd80ea829dbbf505ae72f0a1151e",  # darwin_amd64
    "c444490550ac3af74b70acef7e8d4746e08aeeef517b62faba5381c928cefefa",  # darwin_arm64
],
    url = f"https://github.com/wcodesoft/please-rules/releases/download/ts-v{_VERSION}/please_ts-{_VERSION}-{CONFIG.OS}_{CONFIG.ARCH}",
    visibility = ["PUBLIC"],
)
