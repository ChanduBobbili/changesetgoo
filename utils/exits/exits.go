package exits

import (
	"fmt"
	"os"
)

// WithSuccess prints a message to stdout and exits with code 0 (Success)
func WithSuccess(message string, args ...interface{}) {
	fmt.Fprintf(os.Stdout, message+"\n", args...)
	os.Exit(0)
}

// WithInfo prints a message to stdout and exits with code 0 (Success)
func WithInfo(message string, args ...interface{}) {
	fmt.Fprintf(os.Stdout, message+"\n", args...)
	os.Exit(0)
}

// WithWarning prints a warning to stderr and exits with code 0
// Warnings typically indicate potential issues but not a complete failure.
func WithWarning(message string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "Warning: "+message+"\n", args...)
	os.Exit(0)
}

// WithError prints a message to stderr and exits with code 1 (General error)
func WithError(message string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "Error: "+message+"\n", args...)
	os.Exit(1)
}

// WithUsageError prints a message to stderr and exits with code 2 (Invalid usage)
func WithUsageError(message string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "Usage Error: "+message+"\n", args...)
	os.Exit(2)
}

// WithGitError prints a message to stderr and exits with code 3 (Git-related error)
func WithGitError(message string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "Git Error: "+message+"\n", args...)
	os.Exit(3)
}
