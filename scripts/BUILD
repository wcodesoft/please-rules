package(default_visibility = ["PUBLIC"])

filegroup(
    name = "scripts",
    srcs = [
        "check_shared_files.py",
        "generate_release_notes.py",
        "update_release_pins.py",
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
    name = "update_release_pins_test",
    data = [
        "update_release_pins.py",
        "update_release_pins_test.py",
    ],
    no_test_output = True,
    test_cmd = "python3 scripts/update_release_pins_test.py",
)

gentest(
    name = "check_shared_files_test",
    data = [
        "check_shared_files.py",
        "check_shared_files_test.py",
    ],
    no_test_output = True,
    test_cmd = "python3 scripts/check_shared_files_test.py",
)

# throwaway: a rust-only edit of a file main owns, to see the guard fail
