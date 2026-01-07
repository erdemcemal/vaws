package components

import "vaws/internal/log"

// Logs is a no-op logs component (logging removed).
type Logs struct{}

// NewLogs creates a new no-op Logs component.
func NewLogs(logger *log.Logger) *Logs {
	return &Logs{}
}

// SetSize sets the size (no-op).
func (l *Logs) SetSize(width, height int) {}

// ScrollUp scrolls up (no-op).
func (l *Logs) ScrollUp() {}

// ScrollDown scrolls down (no-op).
func (l *Logs) ScrollDown() {}

// ScrollToBottom scrolls to bottom (no-op).
func (l *Logs) ScrollToBottom() {}

// View returns empty string.
func (l *Logs) View() string {
	return ""
}
