subinclude("//build_defs/kotlin:kotlin")

kotlin_binary(
    name = "bin",
    srcs = ["main.kt"],
    deps = ["//test/kotlin/lib:lib"],
    visibility = ["PUBLIC"],
)

gentest(
    name = "bin_test",
    data = [":bin"],
    no_test_output = True,
    test_cmd = "$(location :bin)",
)
