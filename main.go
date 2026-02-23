package main

import (
	"fmt"
	"os"

	"hestia/ui"
)

func main() {
	p := ui.NewProgram()
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error running program: %v\n", err)
		os.Exit(1)
	}
}
