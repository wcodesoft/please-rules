package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"tools/please_wit/generate"
	"tools/please_wit/toolchain"
)

func handleToolchain(args []string) error {
	cmd := flag.NewFlagSet("toolchain", flag.ContinueOnError)
	out := cmd.String("out", "", "Output directory for toolchain")
	witBindgenTar := cmd.String("wit-bindgen-tar", "", "Path to wit-bindgen tar.gz")
	wasmToolsTar := cmd.String("wasm-tools-tar", "", "Path to wasm-tools tar.gz")

	if err := cmd.Parse(args); err != nil {
		return err
	}

	opts := toolchain.Options{
		Out:           *out,
		WitBindgenTar: *witBindgenTar,
		WasmToolsTar:  *wasmToolsTar,
	}
	return toolchain.Run(opts)
}

func handleGenerate(args []string) error {
	cmd := flag.NewFlagSet("generate", flag.ContinueOnError)
	bindgen := cmd.String("bindgen", "wit-bindgen", "Path to wit-bindgen binary")
	lang := cmd.String("lang", "", "Target language (rust, go, cpp, swift, kotlin, ts, python)")
	out := cmd.String("out", "", "Output directory for generated bindings")
	srcsFlag := cmd.String("srcs", "", "Space or comma-separated WIT source paths or directory")
	worldsFlag := cmd.String("worlds", "", "Comma-separated world names")
	flagsFlag := cmd.String("flags", "", "Additional flags to pass to wit-bindgen")
	packageFlag := cmd.String("package", "", "WIT package name or namespace (e.g. babel:structures)")
	companionFilename := cmd.String("companion-filename", "", "Custom filename for companion file")
	moduleName := cmd.String("module-name", "", "Module name for Swift module.modulemap")

	if err := cmd.Parse(args); err != nil {
		return err
	}

	var srcs []string
	if *srcsFlag != "" {
		for _, s := range strings.Fields(*srcsFlag) {
			for _, part := range strings.Split(s, ",") {
				if trimmed := strings.TrimSpace(part); trimmed != "" {
					srcs = append(srcs, trimmed)
				}
			}
		}
	}
	srcs = append(srcs, cmd.Args()...)

	var worlds []string
	if *worldsFlag != "" {
		for _, w := range strings.Split(*worldsFlag, ",") {
			if trimmed := strings.TrimSpace(w); trimmed != "" {
				worlds = append(worlds, trimmed)
			}
		}
	}

	var extraFlags []string
	if *flagsFlag != "" {
		extraFlags = strings.Fields(*flagsFlag)
	}

	opts := generate.Options{
		Bindgen:           *bindgen,
		Lang:              *lang,
		Out:               *out,
		Srcs:              srcs,
		Worlds:            worlds,
		Flags:             extraFlags,
		Package:           *packageFlag,
		CompanionFilename: *companionFilename,
		ModuleName:        *moduleName,
	}
	return generate.Run(opts)
}

func handlePackage(args []string) error {
	cmd := flag.NewFlagSet("package", flag.ContinueOnError)
	out := cmd.String("out", "", "Output directory for WIT package")
	srcsFlag := cmd.String("srcs", "", "Space or comma-separated WIT source paths")

	if err := cmd.Parse(args); err != nil {
		return err
	}

	if *out == "" {
		return fmt.Errorf("--out is required")
	}

	if err := os.MkdirAll(*out, 0755); err != nil {
		return err
	}

	var srcs []string
	if *srcsFlag != "" {
		for _, s := range strings.Fields(*srcsFlag) {
			for _, part := range strings.Split(s, ",") {
				if trimmed := strings.TrimSpace(part); trimmed != "" {
					srcs = append(srcs, trimmed)
				}
			}
		}
	}
	srcs = append(srcs, cmd.Args()...)

	for _, src := range srcs {
		fi, err := os.Stat(src)
		if err != nil {
			continue
		}
		if fi.IsDir() {
			entries, err := os.ReadDir(src)
			if err != nil {
				continue
			}
			for _, e := range entries {
				if !e.IsDir() && strings.HasSuffix(e.Name(), ".wit") {
					data, err := os.ReadFile(filepath.Join(src, e.Name()))
					if err == nil {
						_ = os.WriteFile(filepath.Join(*out, e.Name()), data, 0644)
					}
				}
			}
		} else {
			data, err := os.ReadFile(src)
			if err == nil {
				_ = os.WriteFile(filepath.Join(*out, filepath.Base(src)), data, 0644)
			}
		}
	}

	return nil
}

func printUsage(w io.Writer) {
	fmt.Fprintf(w, "please_wit - Orchestrator tool for Please WIT rules\n\n")
	fmt.Fprintf(w, "Usage:\n")
	fmt.Fprintf(w, "  please_wit toolchain [options]\n")
	fmt.Fprintf(w, "  please_wit generate [options] [-- sources...]\n")
	fmt.Fprintf(w, "  please_wit package [options] [-- sources...]\n")
}

func runCommand(args []string) error {
	if len(args) < 1 {
		printUsage(os.Stderr)
		return fmt.Errorf("no subcommand specified")
	}

	switch args[0] {
	case "toolchain":
		return handleToolchain(args[1:])
	case "generate":
		return handleGenerate(args[1:])
	case "package":
		return handlePackage(args[1:])
	case "help", "--help", "-h":
		printUsage(os.Stdout)
		return nil
	default:
		return fmt.Errorf("unknown subcommand %q", args[0])
	}
}
