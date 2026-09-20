package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	subcmd := os.Args[1]
	var err error

	switch subcmd {
	case "compile", "wasm":
		err = handleCompile(os.Args[2:])
	case "--help", "-h", "help":
		printUsage()
		os.Exit(0)
	default:
		// If first arg starts with -, treat as compile flags directly
		if strings.HasPrefix(subcmd, "-") {
			err = handleCompile(os.Args[1:])
		} else {
			fmt.Fprintf(os.Stderr, "unknown subcommand: %s\n", subcmd)
			printUsage()
			os.Exit(1)
		}
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, "Usage: please_kotlin_wasm <compile|wasm> [options]\n\n")
	fmt.Fprintf(os.Stderr, "Options:\n")
	fmt.Fprintf(os.Stderr, "  --out <path>          Output .wasm binary path or destination directory\n")
	fmt.Fprintf(os.Stderr, "  --srcs <paths>        Comma-separated source files or directories\n")
	fmt.Fprintf(os.Stderr, "  --deps <paths>        Comma-separated dependency klibs or directories\n")
	fmt.Fprintf(os.Stderr, "  --libraries <paths>   Paths to Kotlin wasm standard libraries (.klib)\n")
	fmt.Fprintf(os.Stderr, "  --kotlinc-wasm <path> Path to kotlinc-wasm compiler executable\n")
	fmt.Fprintf(os.Stderr, "  --target <target>     WebAssembly target: wasm-js or wasm-wasi (default: wasm-js)\n")
	fmt.Fprintf(os.Stderr, "  --main <call|noCall>  Whether to invoke main() on instantiation (default: noCall)\n")
	fmt.Fprintf(os.Stderr, "  --module-name <name>  WebAssembly module name\n")
	fmt.Fprintf(os.Stderr, "  --wit <path>          Path to WebAssembly Interface Types (.wit) definition\n")
	fmt.Fprintf(os.Stderr, "  --impl <class>        Implementation class name for WIT export bridge (default: <Interface>Impl)\n")
	fmt.Fprintf(os.Stderr, "  --flags <flags>       Additional compiler flags\n")
}

func handleCompile(args []string) error {
	cmd := flag.NewFlagSet("compile", flag.ContinueOnError)
	out := cmd.String("out", "", "Output .wasm path or directory")
	srcsFlag := cmd.String("srcs", "", "Comma-separated source files or directories")
	depsFlag := cmd.String("deps", "", "Comma-separated dependency klibs or directories")
	librariesFlag := cmd.String("libraries", "", "Comma-separated klib paths")
	kotlincWasm := cmd.String("kotlinc-wasm", "", "Path to kotlinc-wasm executable")
	target := cmd.String("target", "wasm-js", "Target WebAssembly platform (wasm-js or wasm-wasi)")
	mainMode := cmd.String("main", "noCall", "Main function invocation mode (call or noCall)")
	moduleName := cmd.String("module-name", "", "Module name")
	witFlag := cmd.String("wit", "", "Path to WebAssembly Interface Types (.wit) definition")
	implFlag := cmd.String("impl", "", "Implementation class name for WIT export bridge")
	flags := cmd.String("flags", "", "Additional kotlinc-wasm flags")

	if err := cmd.Parse(args); err != nil {
		return err
	}

	srcs := ExpandCommaSeparated([]string{*srcsFlag})
	srcs = append(srcs, cmd.Args()...)

	deps := ExpandCommaSeparated([]string{*depsFlag})
	libs := ExpandCommaSeparated([]string{*librariesFlag})

	var extraFlags []string
	if *flags != "" {
		extraFlags = strings.Fields(*flags)
	}

	opts := WasmOptions{
		KotlincWasm: *kotlincWasm,
		Out:         *out,
		Srcs:        srcs,
		Deps:        deps,
		Libraries:   libs,
		Main:        *mainMode,
		Target:      *target,
		ModuleName:  *moduleName,
		Wit:         *witFlag,
		Impl:        *implFlag,
		Flags:       extraFlags,
	}

	return CompileWasm(opts)
}
