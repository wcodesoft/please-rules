subinclude("//build_defs/kotlin:kotlin")

maven_jar(
    name = "annotations",
    id = "org.jetbrains:annotations:26.0.1",
    sha256 = "2037be378980d3ba9333e97955f3b2cde392aa124d04ca73ce2eee6657199297",
)

maven_jar(
    name = "junit_standalone",
    id = "org.junit.platform:junit-platform-console-standalone:1.11.4",
    sha256 = "b016ef6b1c3454d6d7c2c88ce081dabf289699686af6622d6e4e2e1b54b4a2fc",
    visibility = ["PUBLIC"],
)

kotlin_library(
    name = "lib",
    srcs = ["lib.kt"],
    deps = [":annotations"],
    visibility = ["PUBLIC"],
)

kotlin_test(
    name = "lib_test",
    srcs = ["lib_test.kt"],
    test_class = "test.kotlin.lib.LibTest",
    deps = [
        ":annotations",
        ":junit_standalone",
        ":lib",
    ],
)
