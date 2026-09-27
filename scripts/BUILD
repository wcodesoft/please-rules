package(default_visibility = ["PUBLIC"])

filegroup(
    name = "scripts",
    srcs = [
        "generate_release_notes.py",
        "strip_swift_toolchain.py",
    ],
)

gentest(
    name = "generate_release_notes_test",
    data = [
        "generate_release_notes.py",
        "generate_release_notes_test.py",
    ],
    no_test_output = True,
    test_cmd = "python3 scripts/generate_release_notes_test.py",
)

gentest(
    name = "strip_swift_toolchain_test",
    data = [
        "strip_swift_toolchain.py",
        "strip_swift_toolchain_test.py",
    ],
    no_test_output = True,
    test_cmd = "python3 -m unittest discover -s scripts -p 'strip_swift_toolchain_test.py'",
)
