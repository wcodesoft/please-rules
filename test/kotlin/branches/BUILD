subinclude("//build_defs/kotlin:kotlin")

kotlin_library(
    name = "branches",
    srcs = ["branches.kt"],
    visibility = ["PUBLIC"],
)

kotlin_test(
    name = "branches_test",
    srcs = ["branches_test.kt"],
    test_class = "test.kotlin.branches.BranchesTest",
    deps = [
        ":branches",
        "//test/kotlin/lib:junit_standalone",
    ],
)
