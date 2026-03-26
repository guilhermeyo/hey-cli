package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

// confirmFromReader prints a prompt to w and reads confirmation from r.
// Returns true on y/Y/yes/YES input, false otherwise.
func confirmFromReader(prompt string, r io.Reader, w io.Writer) bool {
	fmt.Fprintf(w, "%s [y/N] ", prompt)

	scanner := bufio.NewScanner(r)
	if !scanner.Scan() {
		return false
	}

	answer := strings.TrimSpace(scanner.Text())
	return strings.EqualFold(answer, "y") || strings.EqualFold(answer, "yes")
}

// confirmAction prints a prompt to stderr and waits for y/Y confirmation from stdin.
// Returns false if stdin is not a terminal (requires --yes flag).
// Returns false on empty input (default is N).
func confirmAction(prompt string) bool {
	if !stdinIsTerminal() {
		return false
	}
	return confirmFromReader(prompt, os.Stdin, os.Stderr)
}
