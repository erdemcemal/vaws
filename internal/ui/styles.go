package ui

import (
	"github.com/charmbracelet/lipgloss"

	"vaws/internal/model"
	"vaws/internal/ui/theme"
)

// GetStyles returns all UI styles based on the current theme.
func GetStyles() StyleSet {
	t := theme.Current()

	return StyleSet{
		// Layout
		App:    lipgloss.NewStyle(),
		Header: lipgloss.NewStyle().Bold(true).Padding(0, 1).Background(t.Primary).Foreground(lipgloss.Color("#FFFFFF")),
		Footer: lipgloss.NewStyle().Padding(0, 1).Foreground(t.TextMuted),

		// List
		List:         lipgloss.NewStyle().Padding(1, 2),
		ListItem:     lipgloss.NewStyle().PaddingLeft(2).Foreground(t.Text),
		ListSelected: lipgloss.NewStyle().PaddingLeft(2).Bold(true).Foreground(lipgloss.Color("#FFFFFF")).Background(t.BgSelected),
		ListTitle:    lipgloss.NewStyle().Bold(true).Foreground(t.Primary).MarginBottom(1),

		// Details pane
		Details:      lipgloss.NewStyle().Padding(1, 2).Border(lipgloss.RoundedBorder()).BorderForeground(t.Border),
		DetailsTitle: lipgloss.NewStyle().Bold(true).Foreground(t.Primary).MarginBottom(1),
		DetailsLabel: lipgloss.NewStyle().Foreground(t.TextMuted).Width(16),
		DetailsValue: lipgloss.NewStyle().Foreground(t.Text),

		// Logs panel
		Logs:      lipgloss.NewStyle().Padding(0, 1).Border(lipgloss.RoundedBorder()).BorderForeground(t.Border),
		LogsTitle: lipgloss.NewStyle().Bold(true).Foreground(t.TextMuted),

		// Status indicators
		StatusHealthy:    lipgloss.NewStyle().Foreground(t.Success),
		StatusWarning:    lipgloss.NewStyle().Foreground(t.Warning),
		StatusError:      lipgloss.NewStyle().Foreground(t.Error),
		StatusInProgress: lipgloss.NewStyle().Foreground(t.Warning),

		// Text
		Title:     lipgloss.NewStyle().Bold(true).Foreground(t.Text),
		Subtitle:  lipgloss.NewStyle().Foreground(t.TextMuted),
		Muted:     lipgloss.NewStyle().Foreground(t.TextMuted),
		Bold:      lipgloss.NewStyle().Bold(true).Foreground(t.Text),
		Key:       lipgloss.NewStyle().Bold(true).Foreground(t.Primary),
		Highlight: lipgloss.NewStyle().Background(t.BgHighlight),

		// Input
		FilterInput: lipgloss.NewStyle().Foreground(t.Text),
		FilterLabel: lipgloss.NewStyle().Foreground(t.TextMuted),

		// Spinner/loading
		Spinner: lipgloss.NewStyle().Foreground(t.Primary),

		// Theme reference
		Theme: t,
	}
}

// StyleSet holds all UI styles.
type StyleSet struct {
	// Layout
	App    lipgloss.Style
	Header lipgloss.Style
	Footer lipgloss.Style

	// List
	List         lipgloss.Style
	ListItem     lipgloss.Style
	ListSelected lipgloss.Style
	ListTitle    lipgloss.Style

	// Details pane
	Details      lipgloss.Style
	DetailsTitle lipgloss.Style
	DetailsLabel lipgloss.Style
	DetailsValue lipgloss.Style

	// Logs panel
	Logs      lipgloss.Style
	LogsTitle lipgloss.Style

	// Status indicators
	StatusHealthy    lipgloss.Style
	StatusWarning    lipgloss.Style
	StatusError      lipgloss.Style
	StatusInProgress lipgloss.Style

	// Text
	Title     lipgloss.Style
	Subtitle  lipgloss.Style
	Muted     lipgloss.Style
	Bold      lipgloss.Style
	Key       lipgloss.Style
	Highlight lipgloss.Style

	// Input
	FilterInput lipgloss.Style
	FilterLabel lipgloss.Style

	// Spinner/loading
	Spinner lipgloss.Style

	// Theme reference for direct color access
	Theme theme.Theme
}

// Styles is kept for backward compatibility, initialized with dark theme.
// Components should migrate to using GetStyles() for theme-aware styling.
var Styles = GetStyles()

// RefreshStyles updates the global Styles variable with current theme.
// Call this after changing the theme.
func RefreshStyles() {
	Styles = GetStyles()
}

// StatusStyle returns the appropriate style for a stack status.
func StatusStyle(status string) lipgloss.Style {
	s := GetStyles()
	switch {
	case contains(status, "COMPLETE") && !contains(status, "ROLLBACK"):
		return s.StatusHealthy
	case contains(status, "IN_PROGRESS"):
		return s.StatusInProgress
	case contains(status, "FAILED") || contains(status, "ROLLBACK"):
		return s.StatusError
	default:
		return s.Muted
	}
}

// ServiceStatusStyle returns the appropriate style for a service health status.
func ServiceStatusStyle(running, desired int) lipgloss.Style {
	s := GetStyles()
	switch {
	case running == desired && desired > 0:
		return s.StatusHealthy
	case running > 0:
		return s.StatusWarning
	default:
		return s.StatusError
	}
}

// FunctionStatusStyle returns the appropriate style for a Lambda function state.
func FunctionStatusStyle(state model.FunctionState) lipgloss.Style {
	s := GetStyles()
	switch state {
	case model.FunctionStateActive:
		return s.StatusHealthy
	case model.FunctionStatePending:
		return s.StatusInProgress
	case model.FunctionStateInactive:
		return s.StatusWarning
	case model.FunctionStateFailed:
		return s.StatusError
	default:
		return s.Muted
	}
}

// TableStatusStyle returns the appropriate style for a DynamoDB table status.
func TableStatusStyle(status model.TableStatus) lipgloss.Style {
	s := GetStyles()
	switch status {
	case model.TableStatusActive:
		return s.StatusHealthy
	case model.TableStatusCreating, model.TableStatusUpdating:
		return s.StatusInProgress
	case model.TableStatusDeleting:
		return s.StatusWarning
	case model.TableStatusArchiving, model.TableStatusArchived:
		return s.StatusWarning
	case model.TableStatusInaccessible:
		return s.StatusError
	default:
		return s.Muted
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr) >= 0
}

func findSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
