subinclude("//build_defs/kotlin:kotlin")

kotlin_toolchain(
    name = "toolchain",
    visibility = ["PUBLIC"],
)

kotlin_library(
    name = "hermetic_lib",
    srcs = ["lib.kt"],
    visibility = ["PUBLIC"],
)

kotlin_binary(
    name = "hermetic_bin",
    srcs = ["main.kt"],
    deps = [":hermetic_lib"],
)

kotlin_test(
    name = "hermetic_test",
    srcs = ["hermetic_test.kt"],
    test_class = "test.kotlin.toolchain.HermeticTest",
    deps = [
        ":hermetic_lib",
        "//test/kotlin/lib:junit_standalone",
    ],
)
