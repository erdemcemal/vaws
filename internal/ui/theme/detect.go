package theme

import (
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

// Detect attempts to detect whether the terminal is using a light or dark theme.
// Returns DarkTheme if detection fails (safe default for most terminals).
func Detect() Theme {
	// 1. Check COLORFGBG environment variable
	// Format: "fg;bg" where higher bg values indicate light background
	if colorfgbg := os.Getenv("COLORFGBG"); colorfgbg != "" {
		if isLightFromCOLORFGBG(colorfgbg) {
			return LightTheme
		}
		return DarkTheme
	}

	// 2. Check terminal-specific environment variables
	if isLightTerminal() {
		return LightTheme
	}

	// 3. On macOS, try to detect system appearance
	if runtime.GOOS == "darwin" {
		if isMacOSLightMode() {
			return LightTheme
		}
	}

	// 4. Default to dark theme (most common for developers)
	return DarkTheme
}

// isLightFromCOLORFGBG parses the COLORFGBG environment variable.
// Format is typically "fg;bg" or "fg;bg;extra"
// Background values: 0-6 and 8 are dark, 7 and 9-15 are light
func isLightFromCOLORFGBG(value string) bool {
	parts := strings.Split(value, ";")
	if len(parts) < 2 {
		return false
	}

	// Get background color (second part)
	bg, err := strconv.Atoi(parts[1])
	if err != nil {
		return false
	}

	// Standard ANSI: 0-6 and 8 are dark, 7 (white) and 9-15 are light
	// 7 = white, 15 = bright white
	return bg == 7 || (bg >= 9 && bg <= 15)
}

// isLightTerminal checks for terminal-specific indicators of light theme.
func isLightTerminal() bool {
	// Check for iTerm2 with light profile hints
	if profile := os.Getenv("ITERM_PROFILE"); profile != "" {
		lower := strings.ToLower(profile)
		if strings.Contains(lower, "light") {
			return true
		}
	}

	// Check for common light theme indicators in TERM_PROGRAM
	if termProgram := os.Getenv("TERM_PROGRAM"); termProgram != "" {
		// Apple Terminal defaults vary, but we can check other hints
		lower := strings.ToLower(termProgram)
		if strings.Contains(lower, "apple_terminal") {
			// Apple Terminal - check additional hints
			if bg := os.Getenv("__CFBundleIdentifier"); bg != "" {
				// Could add more specific checks here
			}
		}
	}

	// Check for VS Code terminal theme hints
	if vscodeTheme := os.Getenv("VSCODE_THEME_KIND"); vscodeTheme != "" {
		if strings.Contains(strings.ToLower(vscodeTheme), "light") {
			return true
		}
	}

	// Check for explicit light mode environment variable (custom)
	if themeHint := os.Getenv("TERMINAL_THEME"); themeHint != "" {
		if strings.ToLower(themeHint) == "light" {
			return true
		}
	}

	return false
}

// isMacOSLightMode checks if macOS is in light mode using AppleScript.
func isMacOSLightMode() bool {
	// Use defaults command to check dark mode setting
	cmd := exec.Command("defaults", "read", "-g", "AppleInterfaceStyle")
	output, err := cmd.Output()
	if err != nil {
		// If the command fails or key doesn't exist, system is in light mode
		// (AppleInterfaceStyle only exists when dark mode is enabled)
		return true
	}

	// If output contains "Dark", system is in dark mode
	return !strings.Contains(strings.ToLower(string(output)), "dark")
}
