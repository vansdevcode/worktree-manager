package ui

import (
	"fmt"
	"os"
)

// Color codes
const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorCyan   = "\033[36m"
)

// Info prints an informational message in blue
func Info(format string, args ...interface{}) {
	fmt.Fprintf(os.Stdout, ColorBlue+format+ColorReset+"\n", args...)
}

// Success prints a success message in green
func Success(format string, args ...interface{}) {
	fmt.Fprintf(os.Stdout, ColorGreen+format+ColorReset+"\n", args...)
}

// Warning prints a warning message in yellow
func Warning(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, ColorYellow+format+ColorReset+"\n", args...)
}

// Error prints an error message in red
func Error(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, ColorRed+"Error: "+format+ColorReset+"\n", args...)
}

// Plain prints a message without color
func Plain(format string, args ...interface{}) {
	fmt.Fprintf(os.Stdout, format+"\n", args...)
}
