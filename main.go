package main

import (
	"fmt"
	"os"

	"tui/tui"
)

func main() {
	p := tui.NewProgram()
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error running program: %v\n", err)
		os.Exit(1)
	}
}
