package ui

import (
	"fmt"
	"os/exec"
	"runtime"

	"golang.design/x/clipboard"
)

// clipboardInitialized tracks whether clipboard.Init() succeeded
var clipboardInitialized bool

// initClipboard initializes the clipboard library
func initClipboard() error {
	if clipboardInitialized {
		return nil
	}
	err := clipboard.Init()
	if err == nil {
		clipboardInitialized = true
	}
	return err
}

// copyToClipboard copies text to the system clipboard.
// It first tries the clipboard library, then falls back to system commands.
func copyToClipboard(text string) error {
	// Try the clipboard library first
	if err := initClipboard(); err == nil {
		clipboard.Write(clipboard.FmtText, []byte(text))
		return nil
	}

	// Fallback to system commands
	return copyToClipboardFallback(text)
}

// copyToClipboardFallback uses system commands to copy to clipboard
func copyToClipboardFallback(text string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbcopy")
	case "linux":
		// Try xclip first, then xsel
		if _, err := exec.LookPath("xclip"); err == nil {
			cmd = exec.Command("xclip", "-selection", "clipboard")
		} else if _, err := exec.LookPath("xsel"); err == nil {
			cmd = exec.Command("xsel", "--clipboard", "--input")
		} else {
			return fmt.Errorf("no clipboard tool found (install xclip or xsel)")
		}
	case "windows":
		cmd = exec.Command("clip")
	default:
		return fmt.Errorf("clipboard not supported on %s", runtime.GOOS)
	}

	pipe, err := cmd.StdinPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	_, err = pipe.Write([]byte(text))
	if err != nil {
		return err
	}

	if err := pipe.Close(); err != nil {
		return err
	}

	return cmd.Wait()
}
