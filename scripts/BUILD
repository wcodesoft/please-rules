package(default_visibility = ["PUBLIC"])

filegroup(
    name = "scripts",
    srcs = ["generate_release_notes.py"],
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
