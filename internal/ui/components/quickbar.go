package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"vaws/internal/ui/theme"
)

// QuickKey represents a quick access key binding.
type QuickKey struct {
	Key      string // e.g., "1", "2", ":"
	Label    string // e.g., "ECS", "Lambda", "command"
	Active   bool   // Is this the current view?
	Disabled bool   // Is this key disabled?
}

// QuickBar renders a footer with quick access keys and mode indicators.
//
// Example:
//
//	[1]ECS  [2]Lambda  [3]SQS  [4]API  [5]Stacks  │  :command  /filter  ?help
type QuickBar struct {
	width       int
	resourceKeys []QuickKey
	actionKeys   []QuickKey
	mode         string // Current mode: "", "filter", "command"
	filterText   string // Current filter text (if in filter mode)
}

// NewQuickBar creates a new QuickBar component.
func NewQuickBar() *QuickBar {
	return &QuickBar{
		resourceKeys: []QuickKey{
			{Key: "1", Label: "ECS"},
			{Key: "2", Label: "Lambda"},
			{Key: "3", Label: "SQS"},
			{Key: "4", Label: "API"},
			{Key: "5", Label: "Stacks"},
		},
		actionKeys: []QuickKey{
			{Key: ":", Label: "command"},
			{Key: "/", Label: "filter"},
			{Key: "?", Label: "help"},
			{Key: "q", Label: "quit"},
		},
	}
}

// SetWidth sets the quick bar width.
func (q *QuickBar) SetWidth(width int) {
	q.width = width
}

// SetActiveResource sets which resource key is currently active.
func (q *QuickBar) SetActiveResource(key string) {
	for i := range q.resourceKeys {
		q.resourceKeys[i].Active = q.resourceKeys[i].Key == key
	}
}

// SetActiveResourceByLabel sets active resource by label name.
func (q *QuickBar) SetActiveResourceByLabel(label string) {
	for i := range q.resourceKeys {
		q.resourceKeys[i].Active = q.resourceKeys[i].Label == label
	}
}

// SetMode sets the current mode (empty, "filter", or "command").
func (q *QuickBar) SetMode(mode string) {
	q.mode = mode
}

// SetFilterText sets the current filter text.
func (q *QuickBar) SetFilterText(text string) {
	q.filterText = text
}

// ClearActive clears all active states.
func (q *QuickBar) ClearActive() {
	for i := range q.resourceKeys {
		q.resourceKeys[i].Active = false
	}
}

// SetContextActions sets context-specific action keys based on current view.
// These appear before the standard action keys.
func (q *QuickBar) SetContextActions(actions []QuickKey) {
	// Start with context actions, then add standard actions
	standardActions := []QuickKey{
		{Key: ":", Label: "command"},
		{Key: "/", Label: "filter"},
		{Key: "?", Label: "help"},
		{Key: "q", Label: "quit"},
	}
	q.actionKeys = append(actions, standardActions...)
}

// ClearContextActions resets to default action keys.
func (q *QuickBar) ClearContextActions() {
	q.actionKeys = []QuickKey{
		{Key: ":", Label: "command"},
		{Key: "/", Label: "filter"},
		{Key: "?", Label: "help"},
		{Key: "q", Label: "quit"},
	}
}

// View renders the quick bar.
func (q *QuickBar) View() string {
	bgStyle := lipgloss.NewStyle().
		Background(theme.BgSubtle).
		Width(q.width)

	keyStyle := lipgloss.NewStyle().
		Foreground(theme.Warning).
		Bold(true)

	labelStyle := lipgloss.NewStyle().
		Foreground(theme.Text)

	activeKeyStyle := lipgloss.NewStyle().
		Foreground(theme.Primary).
		Bold(true)

	activeLabelStyle := lipgloss.NewStyle().
		Foreground(theme.Primary).
		Bold(true)

	dimKeyStyle := lipgloss.NewStyle().
		Foreground(theme.TextDim)

	dimLabelStyle := lipgloss.NewStyle().
		Foreground(theme.TextDim)

	separatorStyle := lipgloss.NewStyle().
		Foreground(theme.Border)

	filterStyle := lipgloss.NewStyle().
		Foreground(theme.Info)

	separator := separatorStyle.Render(" │ ")

	// Handle special modes
	if q.mode == "filter" {
		filterPrompt := filterStyle.Render("Filter: " + q.filterText + "█")
		hint := dimLabelStyle.Render("  (Enter to apply, Esc to cancel)")
		content := filterPrompt + hint
		return bgStyle.Padding(0, 1).Render(content)
	}

	if q.mode == "command" {
		cmdPrompt := filterStyle.Render(": ")
		hint := dimLabelStyle.Render("  (Tab to complete, Enter to run, Esc to cancel)")
		content := cmdPrompt + hint
		return bgStyle.Padding(0, 1).Render(content)
	}

	// Normal mode - show resource keys and action keys
	var parts []string

	// Resource keys
	var resourceParts []string
	for _, rk := range q.resourceKeys {
		var keyStr, labelStr string
		if rk.Active {
			keyStr = activeKeyStyle.Render("[" + rk.Key + "]")
			labelStr = activeLabelStyle.Render(rk.Label)
		} else if rk.Disabled {
			keyStr = dimKeyStyle.Render("[" + rk.Key + "]")
			labelStr = dimLabelStyle.Render(rk.Label)
		} else {
			keyStr = keyStyle.Render("[" + rk.Key + "]")
			labelStr = labelStyle.Render(rk.Label)
		}
		resourceParts = append(resourceParts, keyStr+labelStr)
	}
	parts = append(parts, strings.Join(resourceParts, "  "))

	// Action keys
	var actionParts []string
	for _, ak := range q.actionKeys {
		keyStr := keyStyle.Render(ak.Key)
		labelStr := dimLabelStyle.Render(ak.Label)
		actionParts = append(actionParts, keyStr+labelStr)
	}
	parts = append(parts, strings.Join(actionParts, "  "))

	content := strings.Join(parts, separator)

	return bgStyle.Padding(0, 1).Render(content)
}

// DefaultResourceKeys returns the default resource quick keys.
func DefaultResourceKeys() []QuickKey {
	return []QuickKey{
		{Key: "1", Label: "ECS"},
		{Key: "2", Label: "Lambda"},
		{Key: "3", Label: "SQS"},
		{Key: "4", Label: "API"},
		{Key: "5", Label: "Stacks"},
	}
}
