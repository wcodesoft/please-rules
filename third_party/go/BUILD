subinclude("///go//build_defs:go")

go_toolchain(
    name = "toolchain",
    version = "1.23.1",
)

go_stdlib(
    name = "std",
)

go_repo(
    name = "sync",
    install = ["..."],
    module = "golang.org/x/sync",
    version = "v0.11.0",
)

go_repo(
    name = "yaml.v3",
    install = ["."],
    module = "gopkg.in/yaml.v3",
    version = "v3.0.1",
)

go_repo(
    name = "gotreesitter",
    build_tags = ["gts_no_parsercorephase0"],
    install = [
        ".",
        "grammars/c",
        "grammars/c_sharp",
        "grammars/cpp",
        "grammars/dart",
        "grammars/go",
        "grammars/grammar_blobs",
        "grammars/java",
        "grammars/javascript",
        "grammars/kotlin",
        "grammars/python",
        "grammars/runtime",
        "grammars/rust",
        "grammars/swift",
        "grammars/tsx",
        "grammars/typescript",
    ],
    module = "github.com/odvcencio/gotreesitter",
    version = "v0.56.0",
)
