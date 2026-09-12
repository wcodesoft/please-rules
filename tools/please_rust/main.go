package main

import (
	"fmt"
	"io"
	"os"
)

func printUsage(w io.Writer) {
	fmt.Fprintf(w, "Usage: plz_rust <command> [arguments]\n\n")
	fmt.Fprintf(w, "Commands:\n")
	fmt.Fprintf(w, "  compile       Compile Rust code with rustc\n")
	fmt.Fprintf(w, "  download      Download a crate from crates.io and verify its checksum\n")
	fmt.Fprintf(w, "  hash          Print the SHA-256 of a crate tarball from crates.io\n")
	fmt.Fprintf(w, "  compile-c     Compile C sources into a static archive\n")
	fmt.Fprintf(w, "  test-runner   Execute test binary and output JUnit results\n")
}

func main() {
	if err := runCLI(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
