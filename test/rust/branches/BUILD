subinclude("//build_defs/rust:rust")

rust_library(
    name = "branches",
    srcs = ["branches.rs"],
    crate_name = "mybranches",
    visibility = ["PUBLIC"],
)

rust_test(
    name = "branches_test",
    srcs = ["branches_test.rs"],
    crate_name = "mybranches_test",
    deps = [":branches"],
)
