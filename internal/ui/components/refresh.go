package components

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"vaws/internal/ui/theme"
)

// Default refresh interval
const DefaultRefreshInterval = 10 * time.Second

// AutoRefreshTickMsg is sent on each auto-refresh tick
type AutoRefreshTickMsg time.Time

// RefreshIndicator shows the auto-refresh status
type RefreshIndicator struct {
	enabled      bool
	interval     time.Duration
	lastRefresh  time.Time
	refreshing   bool
	frame        int
	spinnerChars []string
}

// NewRefreshIndicator creates a new refresh indicator
func NewRefreshIndicator() *RefreshIndicator {
	return &RefreshIndicator{
		enabled:      true,
		interval:     DefaultRefreshInterval,
		lastRefresh:  time.Now(),
		refreshing:   false,
		frame:        0,
		spinnerChars: []string{"◴", "◷", "◶", "◵"},
	}
}

// SetEnabled enables or disables auto-refresh
func (r *RefreshIndicator) SetEnabled(enabled bool) {
	r.enabled = enabled
}

// IsEnabled returns whether auto-refresh is enabled
func (r *RefreshIndicator) IsEnabled() bool {
	return r.enabled
}

// SetInterval sets the refresh interval
func (r *RefreshIndicator) SetInterval(interval time.Duration) {
	r.interval = interval
}

// SetRefreshing sets the refreshing state
func (r *RefreshIndicator) SetRefreshing(refreshing bool) {
	r.refreshing = refreshing
	if !refreshing {
		r.lastRefresh = time.Now()
	}
}

// IsRefreshing returns whether a refresh is in progress
func (r *RefreshIndicator) IsRefreshing() bool {
	return r.refreshing
}

// Tick advances the spinner animation
func (r *RefreshIndicator) Tick() {
	r.frame = (r.frame + 1) % len(r.spinnerChars)
}

// TickCmd returns a command for auto-refresh timing
func (r *RefreshIndicator) TickCmd() tea.Cmd {
	if !r.enabled {
		return nil
	}
	return tea.Tick(r.interval, func(t time.Time) tea.Msg {
		return AutoRefreshTickMsg(t)
	})
}

// TimeSinceRefresh returns time since last refresh
func (r *RefreshIndicator) TimeSinceRefresh() time.Duration {
	return time.Since(r.lastRefresh)
}

// View renders the refresh indicator
func (r *RefreshIndicator) View() string {
	if !r.enabled {
		return lipgloss.NewStyle().
			Foreground(theme.TextDim).
			Render("⏸")
	}

	if r.refreshing {
		spinnerStyle := lipgloss.NewStyle().
			Foreground(theme.Primary)
		return spinnerStyle.Render(r.spinnerChars[r.frame])
	}

	// Show time since last refresh
	elapsed := r.TimeSinceRefresh()
	var indicator string
	if elapsed < 5*time.Second {
		indicator = "●" // Just refreshed
	} else if elapsed < r.interval/2 {
		indicator = "◐"
	} else {
		indicator = "○"
	}

	style := lipgloss.NewStyle().Foreground(theme.Success)
	return style.Render(indicator)
}

// StatusView returns a more detailed status for the header
func (r *RefreshIndicator) StatusView() string {
	if !r.enabled {
		return lipgloss.NewStyle().
			Foreground(theme.TextDim).
			Render("auto-refresh off")
	}

	if r.refreshing {
		spinnerStyle := lipgloss.NewStyle().Foreground(theme.Primary)
		return spinnerStyle.Render(r.spinnerChars[r.frame] + " refreshing...")
	}

	elapsed := r.TimeSinceRefresh()
	seconds := int(elapsed.Seconds())

	var text string
	if seconds < 1 {
		text = "just now"
	} else if seconds < 60 {
		text = fmt.Sprintf("%ds ago", seconds)
	} else {
		text = fmt.Sprintf("%dm ago", seconds/60)
	}

	style := lipgloss.NewStyle().Foreground(theme.TextMuted)
	return style.Render(text)
}
