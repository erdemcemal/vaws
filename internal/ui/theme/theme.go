// Package theme provides theming support for the UI.
package theme

import (
	"sync"

	"github.com/charmbracelet/lipgloss"
)

// ThemeType represents the theme type.
type ThemeType string

const (
	ThemeAuto  ThemeType = "auto"
	ThemeDark  ThemeType = "dark"
	ThemeLight ThemeType = "light"
)

// Theme holds all color values for the UI.
type Theme struct {
	Name string

	// Primary accent color
	Primary       lipgloss.Color
	PrimaryDim    lipgloss.Color
	PrimaryBright lipgloss.Color

	// Text colors
	Text      lipgloss.Color
	TextMuted lipgloss.Color
	TextDim   lipgloss.Color

	// Background colors
	BgSelected  lipgloss.Color
	BgHighlight lipgloss.Color

	// Status colors
	Error   lipgloss.Color
	Warning lipgloss.Color
	Success lipgloss.Color
	Info    lipgloss.Color

	// Border colors
	Border    lipgloss.Color
	BorderDim lipgloss.Color

	// Special
	Cursor lipgloss.Color
}

// DarkTheme is the default dark theme.
var DarkTheme = Theme{
	Name: "dark",

	Primary:       lipgloss.Color("#7C3AED"),
	PrimaryDim:    lipgloss.Color("#4C1D95"),
	PrimaryBright: lipgloss.Color("#A78BFA"),

	Text:      lipgloss.Color("#E5E7EB"),
	TextMuted: lipgloss.Color("#9CA3AF"),
	TextDim:   lipgloss.Color("#6B7280"),

	BgSelected:  lipgloss.Color("#374151"),
	BgHighlight: lipgloss.Color("#1F2937"),

	Error:   lipgloss.Color("#EF4444"),
	Warning: lipgloss.Color("#F59E0B"),
	Success: lipgloss.Color("#10B981"),
	Info:    lipgloss.Color("#3B82F6"),

	Border:    lipgloss.Color("#374151"),
	BorderDim: lipgloss.Color("#1F2937"),

	Cursor: lipgloss.Color("#7C3AED"),
}

// LightTheme is the light theme for light terminal backgrounds.
var LightTheme = Theme{
	Name: "light",

	Primary:       lipgloss.Color("#6D28D9"),
	PrimaryDim:    lipgloss.Color("#8B5CF6"),
	PrimaryBright: lipgloss.Color("#4C1D95"),

	Text:      lipgloss.Color("#1F2937"),
	TextMuted: lipgloss.Color("#4B5563"),
	TextDim:   lipgloss.Color("#6B7280"),

	BgSelected:  lipgloss.Color("#E5E7EB"),
	BgHighlight: lipgloss.Color("#F3F4F6"),

	Error:   lipgloss.Color("#DC2626"),
	Warning: lipgloss.Color("#D97706"),
	Success: lipgloss.Color("#059669"),
	Info:    lipgloss.Color("#2563EB"),

	Border:    lipgloss.Color("#D1D5DB"),
	BorderDim: lipgloss.Color("#E5E7EB"),

	Cursor: lipgloss.Color("#6D28D9"),
}

var (
	current     = DarkTheme
	currentLock sync.RWMutex
)

// Current returns the currently active theme.
func Current() Theme {
	currentLock.RLock()
	defer currentLock.RUnlock()
	return current
}

// Set sets the current theme.
func Set(t Theme) {
	currentLock.Lock()
	defer currentLock.Unlock()
	current = t
}

// SetByName sets the theme by name.
func SetByName(name ThemeType) {
	switch name {
	case ThemeLight:
		Set(LightTheme)
	case ThemeDark:
		Set(DarkTheme)
	case ThemeAuto:
		Set(Detect())
	default:
		Set(DarkTheme)
	}
}

// IsDark returns true if the current theme is dark.
func IsDark() bool {
	return Current().Name == "dark"
}
