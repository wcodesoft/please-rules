subinclude("//build_defs/swift:swift")

swift_library(
    name = "branches",
    srcs = ["Branches.swift"],
    module_name = "Branches",
    visibility = ["PUBLIC"],
)

swift_test(
    name = "branches_test",
    srcs = ["BranchesTests.swift"],
    deps = [":branches"],
)
