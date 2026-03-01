//go:build windows
// +build windows

package ui

import (
	"fmt"
	"os"

	"golang.org/x/sys/windows"
)

var inMode, outMode uint32

func init() {
	var _ = windows.GetConsoleMode(windows.Stdin, &inMode)
	var _ = windows.GetConsoleMode(windows.Stdout, &outMode)
}

func setConsoleMode() {
	in := inMode | windows.ENABLE_VIRTUAL_TERMINAL_INPUT
	out := outMode | windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING

	if inMode != 0 && inMode != in {
		if err := windows.SetConsoleMode(windows.Stdin, in); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to set console mode: %v\n", err)
		}
	}

	if outMode != 0 && outMode != out {
		if err := windows.SetConsoleMode(windows.Stdout, out); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to set console mode: %v\n", err)
		}
	}
}

func resetConsoleMode() {
	in := inMode | windows.ENABLE_VIRTUAL_TERMINAL_INPUT
	out := outMode | windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING

	if inMode != 0 && inMode != in {
		if err := windows.SetConsoleMode(windows.Stdin, inMode); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to reset console mode: %v\n", err)
		}
	}
	if outMode != 0 && outMode != out {
		if err := windows.SetConsoleMode(windows.Stdout, outMode); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to reset console mode: %v\n", err)
		}
	}
}
