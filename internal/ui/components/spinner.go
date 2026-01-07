package components

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"vaws/internal/ui/theme"
)

// Spinner frames for animation
var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// Alternative spinner styles:
// var spinnerFrames = []string{"◐", "◓", "◑", "◒"}
// var spinnerFrames = []string{"⣾", "⣽", "⣻", "⢿", "⡿", "⣟", "⣯", "⣷"}
// var spinnerFrames = []string{"▁", "▂", "▃", "▄", "▅", "▆", "▇", "█", "▇", "▆", "▅", "▄", "▃", "▂"}

// SpinnerTickMsg is sent on each spinner frame update.
type SpinnerTickMsg time.Time

// Spinner is an animated loading spinner.
type Spinner struct {
	frame    int
	interval time.Duration
}

// NewSpinner creates a new Spinner.
func NewSpinner() *Spinner {
	return &Spinner{
		frame:    0,
		interval: 80 * time.Millisecond,
	}
}

// Tick advances the spinner to the next frame.
func (s *Spinner) Tick() {
	s.frame = (s.frame + 1) % len(spinnerFrames)
}

// View returns the current spinner frame.
func (s *Spinner) View() string {
	spinnerStyle := lipgloss.NewStyle().Foreground(theme.Primary)
	return spinnerStyle.Render(spinnerFrames[s.frame])
}

// TickCmd returns a command that sends SpinnerTickMsg at the spinner's interval.
func (s *Spinner) TickCmd() tea.Cmd {
	return tea.Tick(s.interval, func(t time.Time) tea.Msg {
		return SpinnerTickMsg(t)
	})
}

// SpinnerWithText renders a spinner with accompanying text.
func SpinnerWithText(spinner *Spinner, text string) string {
	textStyle := lipgloss.NewStyle().Foreground(theme.TextMuted)
	return spinner.View() + " " + textStyle.Render(text)
}
