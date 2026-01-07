package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"vaws/internal/ui/theme"
)

// Action represents an available action
type Action struct {
	Key      string
	Label    string
	Enabled  bool
	Category string // Group actions by category
}

// ActionBar displays context-sensitive actions like k9s
type ActionBar struct {
	width   int
	actions []Action
}

// NewActionBar creates a new action bar
func NewActionBar() *ActionBar {
	return &ActionBar{
		actions: []Action{},
	}
}

// SetWidth sets the action bar width
func (a *ActionBar) SetWidth(width int) {
	a.width = width
}

// SetActions sets the available actions
func (a *ActionBar) SetActions(actions []Action) {
	a.actions = actions
}

// View renders the action bar
func (a *ActionBar) View() string {
	if len(a.actions) == 0 {
		return ""
	}

	keyStyle := lipgloss.NewStyle().
		Foreground(theme.Primary).
		Bold(true)

	labelStyle := lipgloss.NewStyle().
		Foreground(theme.TextMuted)

	disabledKeyStyle := lipgloss.NewStyle().
		Foreground(theme.TextDim)

	disabledLabelStyle := lipgloss.NewStyle().
		Foreground(theme.TextDim)

	separatorStyle := lipgloss.NewStyle().
		Foreground(theme.Border)

	var parts []string

	for _, action := range a.actions {
		var part string
		if action.Enabled {
			part = keyStyle.Render("<"+action.Key+">") + " " + labelStyle.Render(action.Label)
		} else {
			part = disabledKeyStyle.Render("<"+action.Key+">") + " " + disabledLabelStyle.Render(action.Label)
		}
		parts = append(parts, part)
	}

	content := strings.Join(parts, separatorStyle.Render(" │ "))

	barStyle := lipgloss.NewStyle().
		Background(theme.BgSubtle).
		Foreground(theme.TextMuted).
		Padding(0, 1).
		Width(a.width)

	return barStyle.Render(content)
}

// StackActions returns actions for stack view
func StackActions() []Action {
	return []Action{
		{Key: "enter", Label: "services", Enabled: true},
		{Key: "d", Label: "describe", Enabled: true},
		{Key: "y", Label: "yaml", Enabled: true},
		{Key: "r", Label: "refresh", Enabled: true},
		{Key: "/", Label: "filter", Enabled: true},
		{Key: ":", Label: "command", Enabled: true},
		{Key: "q", Label: "quit", Enabled: true},
	}
}

// ServiceActions returns actions for service view
func ServiceActions() []Action {
	return []Action{
		{Key: "enter", Label: "tasks", Enabled: true},
		{Key: "p", Label: "port-forward", Enabled: true},
		{Key: "l", Label: "logs", Enabled: true},
		{Key: "d", Label: "describe", Enabled: true},
		{Key: "s", Label: "scale", Enabled: false}, // Future feature
		{Key: "/", Label: "filter", Enabled: true},
		{Key: ":", Label: "command", Enabled: true},
		{Key: "esc", Label: "back", Enabled: true},
	}
}

// TaskActions returns actions for task view
func TaskActions() []Action {
	return []Action{
		{Key: "enter", Label: "containers", Enabled: true},
		{Key: "l", Label: "logs", Enabled: true},
		{Key: "s", Label: "shell", Enabled: true},
		{Key: "d", Label: "describe", Enabled: true},
		{Key: "x", Label: "stop", Enabled: true},
		{Key: "/", Label: "filter", Enabled: true},
		{Key: ":", Label: "command", Enabled: true},
		{Key: "esc", Label: "back", Enabled: true},
	}
}

// TunnelActions returns actions for tunnel view
func TunnelActions() []Action {
	return []Action{
		{Key: "p", Label: "new tunnel", Enabled: true},
		{Key: "r", Label: "restart", Enabled: true},
		{Key: "x", Label: "stop", Enabled: true},
		{Key: "c", Label: "clear", Enabled: true},
		{Key: ":", Label: "command", Enabled: true},
		{Key: "esc", Label: "back", Enabled: true},
		{Key: "q", Label: "quit", Enabled: true},
	}
}

// ClusterActions returns actions for cluster view
func ClusterActions() []Action {
	return []Action{
		{Key: "enter", Label: "services", Enabled: true},
		{Key: "d", Label: "describe", Enabled: true},
		{Key: "r", Label: "refresh", Enabled: true},
		{Key: "/", Label: "filter", Enabled: true},
		{Key: ":", Label: "command", Enabled: true},
		{Key: "q", Label: "quit", Enabled: true},
	}
}

// XRayActions returns actions for XRay view
func XRayActions() []Action {
	return []Action{
		{Key: "enter", Label: "expand", Enabled: true},
		{Key: "l", Label: "logs", Enabled: true},
		{Key: "p", Label: "port-forward", Enabled: true},
		{Key: "d", Label: "describe", Enabled: true},
		{Key: ":", Label: "command", Enabled: true},
		{Key: "esc", Label: "back", Enabled: true},
	}
}

// FilterActions returns actions when filtering
func FilterActions() []Action {
	return []Action{
		{Key: "enter", Label: "apply", Enabled: true},
		{Key: "esc", Label: "cancel", Enabled: true},
	}
}

// CommandActions returns actions when command palette is open
func CommandActions() []Action {
	return []Action{
		{Key: "enter", Label: "execute", Enabled: true},
		{Key: "tab", Label: "complete", Enabled: true},
		{Key: "esc", Label: "cancel", Enabled: true},
	}
}
