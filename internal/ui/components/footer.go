package components

import (
	"strings"

	"vaws/internal/ui/theme"
)

// KeyBinding represents a key binding for display in the footer.
type KeyBinding struct {
	Key  string
	Desc string
}

// Footer renders the application footer with key bindings.
type Footer struct {
	width    int
	bindings []KeyBinding
}

// NewFooter creates a new Footer component.
func NewFooter() *Footer {
	return &Footer{}
}

// SetWidth sets the footer width.
func (f *Footer) SetWidth(width int) {
	f.width = width
}

// SetBindings sets the key bindings to display.
func (f *Footer) SetBindings(bindings []KeyBinding) {
	f.bindings = bindings
}

// View renders the footer.
func (f *Footer) View() string {
	s := theme.DefaultStyles()

	divider := s.StatusDivider.Render(" | ")

	var parts []string
	for _, b := range f.bindings {
		part := s.StatusKey.Render(b.Key) + " " + s.StatusValue.Render(b.Desc)
		parts = append(parts, part)
	}

	content := strings.Join(parts, divider)

	return s.StatusBar.Width(f.width).Render(content)
}

// DefaultBindings returns the default key bindings for the main view.
func DefaultBindings() []KeyBinding {
	return []KeyBinding{
		{Key: "↑↓", Desc: "navigate"},
		{Key: "→", Desc: "select"},
		{Key: ":", Desc: "command"},
		{Key: "/", Desc: "filter"},
		{Key: "a", Desc: "auto-refresh"},
		{Key: "t", Desc: "tunnels"},
		{Key: "q", Desc: "quit"},
	}
}

// ServiceBindings returns key bindings for the services view.
func ServiceBindings() []KeyBinding {
	return []KeyBinding{
		{Key: "↑↓", Desc: "navigate"},
		{Key: "←", Desc: "back"},
		{Key: "p", Desc: "port forward"},
		{Key: ":", Desc: "command"},
		{Key: "/", Desc: "filter"},
		{Key: "l", Desc: "logs"},
		{Key: "q", Desc: "quit"},
	}
}

// FilterBindings returns the key bindings for filter mode.
func FilterBindings() []KeyBinding {
	return []KeyBinding{
		{Key: "enter", Desc: "apply filter"},
		{Key: "esc", Desc: "cancel"},
	}
}

// PortInputBindings returns the key bindings for port input mode.
func PortInputBindings() []KeyBinding {
	return []KeyBinding{
		{Key: "enter", Desc: "start tunnel"},
		{Key: "esc", Desc: "cancel"},
	}
}

// TunnelBindings returns key bindings for tunnels view.
func TunnelBindings() []KeyBinding {
	return []KeyBinding{
		{Key: "↑↓", Desc: "navigate"},
		{Key: "←", Desc: "back"},
		{Key: "p", Desc: "new"},
		{Key: "x", Desc: "stop"},
		{Key: "r", Desc: "restart"},
		{Key: ":", Desc: "command"},
		{Key: "q", Desc: "quit"},
	}
}
