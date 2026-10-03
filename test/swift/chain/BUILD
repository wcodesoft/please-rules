subinclude("//build_defs/swift:swift")

swift_library(
    name = "base",
    srcs = ["Base.swift"],
    module_name = "Base",
)

swift_library(
    name = "mid",
    srcs = ["Mid.swift"],
    module_name = "Mid",
    deps = [":base"],
)

swift_test(
    name = "mid_test",
    srcs = ["MidTests.swift"],
    deps = [":mid"],
)

swift_binary(
    name = "mid_bin",
    main = "main.swift",
    deps = [":mid"],
)
