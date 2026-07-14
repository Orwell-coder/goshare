//go:build !windows

package client

// getTerminalWidth returns a fixed width for non-Windows platforms.
// On Linux/macOS, terminal width detection would use golang.org/x/term.
func getTerminalWidth() (int, bool) {
	return 80, false
}
