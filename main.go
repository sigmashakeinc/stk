package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
)

// version is the STK release version. Release builds set it with
// -ldflags "-X main.version=<tag>".
var version = "dev"

const usageText = `stk — Sigma Token Killer: wraps a command and prints token-reduced output.

Usage:
  stk [--raw] [--max <n>] <command> [args...]
  stk --version | version

Flags (must precede <command>; stk stops flag parsing at the first non-stk token):
  --raw          Passthrough mode: no capture/reduction; also enabled by STK_DISABLE=1
  --max <n>      Override max output lines (default 200; also STK_MAX_LINES)
  --version      Print version and exit

STK does NOT register a silent global command-rewrite hook.
`

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	// Fast-path: version subcommand.
	if len(args) == 1 && args[0] == "version" {
		fmt.Printf("stk %s\n", version)
		return 0
	}

	raw := os.Getenv("STK_DISABLE") == "1"
	maxLines := DefaultMaxLines

	// Honor STK_MAX_LINES env (flag wins later if provided).
	if v := os.Getenv("STK_MAX_LINES"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			maxLines = n
		}
	}

	// Parse only leading stk flags; stop at the first unrecognised token.
	i := 0
	for i < len(args) {
		switch args[i] {
		case "--version":
			fmt.Printf("stk %s\n", version)
			return 0
		case "--raw":
			raw = true
			i++
		case "--max":
			if i+1 >= len(args) {
				fmt.Fprintf(os.Stderr, "stk: --max requires an argument\n")
				return 1
			}
			n, err := strconv.Atoi(args[i+1])
			if err != nil || n <= 0 {
				fmt.Fprintf(os.Stderr, "stk: --max value must be a positive integer, got %q\n", args[i+1])
				return 1
			}
			maxLines = n
			i += 2
		case "--help", "-h":
			fmt.Print(usageText)
			return 0
		default:
			// First non-stk token — everything from here is the wrapped command.
			goto doneFlags
		}
	}
doneFlags:

	if i >= len(args) {
		fmt.Print(usageText)
		return 1
	}

	cmdName := args[i]
	cmdArgs := args[i+1:]

	cmd := exec.Command(cmdName, cmdArgs...)

	if raw {
		// Passthrough: wire directly to our stdout/stderr so the child sees a
		// proper TTY (if any) and STK adds zero overhead.
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		if err := cmd.Run(); err != nil {
			return exitCode(err)
		}
		return 0
	}

	// Capture mode: combined stdout+stderr into one buffer.
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	cmd.Stdin = os.Stdin

	runErr := cmd.Run()

	origBytes := buf.Len()
	reduced := Reduce(buf.String(), maxLines)
	fmt.Print(reduced)
	// Ensure output ends with a newline when the reduced content is non-empty
	// and doesn't already end with one (mirrors what the terminal would show).
	if len(reduced) > 0 && reduced[len(reduced)-1] != '\n' {
		fmt.Println()
	}
	appendSavings(origBytes, len(reduced))

	if runErr != nil {
		return exitCode(runErr)
	}
	return 0
}

// exitCode extracts the exit status from a cmd.Run() error.
// Returns 127 if the command could not be started, otherwise the child's code.
func exitCode(err error) int {
	if err == nil {
		return 0
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}
	// Command not found / permission denied / etc.
	fmt.Fprintf(os.Stderr, "stk: %v\n", err)
	return 127
}
