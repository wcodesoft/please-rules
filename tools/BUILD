_VERSION = "0.6.1"

remote_file(
    name = "please_rust",
    url = f"https://github.com/wcodesoft/please-rules/releases/download/rust-v{_VERSION}/please_rust-{_VERSION}-{CONFIG.OS}_{CONFIG.ARCH}",
    hashes = [
    "7f8e89a7800204223da3abc4fffaa6b19ae9caab5a18916382edbdd5a0fad459",  # linux_amd64
    "9d8d934e028ab04fdf4b2f95efbdc27b6704007a990de4dbfbe716c4fafa3041",  # linux_arm64
    "8481b9586203b840f80105af8e10601bde8d408cf067de28045cc50503259b7f",  # darwin_amd64
    "dd45df3d3fde10458c46323e02548fa2b18733d9a1b1c5edc9a866521be2fb7f",  # darwin_arm64
],
    binary = True,
    visibility = ["PUBLIC"],
)
