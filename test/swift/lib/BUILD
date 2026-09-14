subinclude("//build_defs/swift:swift")

swift_library(
    name = "math",
    srcs = ["Math.swift"],
    module_name = "Math",
    visibility = ["PUBLIC"],
)

swift_test(
    name = "math_test",
    srcs = ["MathTests.swift"],
    deps = [":math"],
)
