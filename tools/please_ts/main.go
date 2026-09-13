package main

import (
	"fmt"
	"os"
)

func main() {
	if err := runCommand(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
