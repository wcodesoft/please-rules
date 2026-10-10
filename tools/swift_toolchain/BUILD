subinclude("//build_defs/swift:swift")

swift_toolchain(
    name = "toolchain",
    visibility = ["PUBLIC"],
)

gentest(
    name = "strip_swift_toolchain_test",
    data = [
        "strip_swift_toolchain.py",
        "strip_swift_toolchain_test.py",
    ],
    no_test_output = True,
    test_cmd = "python3 -m unittest discover -s tools/swift_toolchain -p 'strip_swift_toolchain_test.py'",
)
