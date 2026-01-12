package components

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/lipgloss"
	"vaws/internal/log"
	"vaws/internal/ui/theme"
)

const maxLogEntries = 500

// LogEntry represents a single log entry.
type LogEntry struct {
	Time    time.Time
	Level   string
	Message string
}

// Logs displays log messages in the UI.
type Logs struct {
	mu       sync.RWMutex
	entries  []LogEntry
	width    int
	height   int
	scroll   int
	logger   *log.Logger
}

// NewLogs creates a new Logs component.
func NewLogs(logger *log.Logger) *Logs {
	l := &Logs{
		entries: make([]LogEntry, 0, maxLogEntries),
		logger:  logger,
	}
	// Register this component as the log output
	if logger != nil {
		logger.SetOutput(l)
	}
	return l
}

// Write implements log output interface.
func (l *Logs) Write(level, message string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	entry := LogEntry{
		Time:    time.Now(),
		Level:   level,
		Message: message,
	}

	l.entries = append(l.entries, entry)

	// Trim if too many entries
	if len(l.entries) > maxLogEntries {
		l.entries = l.entries[len(l.entries)-maxLogEntries:]
	}

	// Auto-scroll to bottom
	l.scrollToBottomLocked()
}

// SetSize sets the component dimensions.
func (l *Logs) SetSize(width, height int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.width = width
	l.height = height
}

// ScrollUp scrolls the log view up.
func (l *Logs) ScrollUp() {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.scroll > 0 {
		l.scroll--
	}
}

// ScrollDown scrolls the log view down.
func (l *Logs) ScrollDown() {
	l.mu.Lock()
	defer l.mu.Unlock()
	maxScroll := len(l.entries) - l.height
	if maxScroll < 0 {
		maxScroll = 0
	}
	if l.scroll < maxScroll {
		l.scroll++
	}
}

// ScrollToBottom scrolls to the bottom of the log.
func (l *Logs) ScrollToBottom() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.scrollToBottomLocked()
}

func (l *Logs) scrollToBottomLocked() {
	maxScroll := len(l.entries) - l.height
	if maxScroll < 0 {
		maxScroll = 0
	}
	l.scroll = maxScroll
}

// View renders the logs component.
func (l *Logs) View() string {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if l.width == 0 || l.height == 0 {
		return ""
	}

	st := theme.DefaultStyles()

	// Styles for different log levels
	timeStyle := st.Muted
	infoStyle := lipgloss.NewStyle().Foreground(theme.Success)
	warnStyle := lipgloss.NewStyle().Foreground(theme.Warning)
	errorStyle := lipgloss.NewStyle().Foreground(theme.Error)
	debugStyle := st.Muted

	var lines []string

	// Calculate visible range
	start := l.scroll
	end := start + l.height
	if end > len(l.entries) {
		end = len(l.entries)
	}
	if start > len(l.entries) {
		start = len(l.entries)
	}

	for i := start; i < end; i++ {
		entry := l.entries[i]
		timeStr := entry.Time.Format("15:04:05")

		var levelStyle lipgloss.Style
		switch entry.Level {
		case "INFO":
			levelStyle = infoStyle
		case "WARN":
			levelStyle = warnStyle
		case "ERROR":
			levelStyle = errorStyle
		case "DEBUG":
			levelStyle = debugStyle
		default:
			levelStyle = lipgloss.NewStyle().Foreground(theme.Text)
		}

		// Calculate prefix width for continuation lines
		prefix := fmt.Sprintf("%s %s ",
			timeStyle.Render(timeStr),
			levelStyle.Render(fmt.Sprintf("%-5s", entry.Level)),
		)
		prefixWidth := lipgloss.Width(prefix)
		availableWidth := l.width - prefixWidth
		if availableWidth < 20 {
			availableWidth = 20
		}

		// Wrap message if too long
		wrappedLines := wrapText(entry.Message, availableWidth)

		// First line with prefix
		lines = append(lines, prefix+wrappedLines[0])

		// Continuation lines indented to align with message
		indent := strings.Repeat(" ", prefixWidth)
		for j := 1; j < len(wrappedLines); j++ {
			lines = append(lines, indent+wrappedLines[j])
		}
	}

	// Pad with empty lines if needed
	for len(lines) < l.height {
		lines = append(lines, "")
	}

	return strings.Join(lines, "\n")
}

// Clear clears all log entries.
func (l *Logs) Clear() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.entries = l.entries[:0]
	l.scroll = 0
}
