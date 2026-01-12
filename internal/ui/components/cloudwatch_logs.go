package components

import (
	"fmt"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"vaws/internal/model"
	"vaws/internal/ui/theme"
)

const (
	maxCloudWatchEntries       = 1000
	cloudWatchPollInterval     = 5 * time.Second
	cloudWatchSpinnerInterval  = 100 * time.Millisecond
)

// CloudWatchLogsTickMsg signals time to fetch new logs.
type CloudWatchLogsTickMsg time.Time

// CloudWatchSpinnerTickMsg signals time to update spinner animation.
type CloudWatchSpinnerTickMsg time.Time

// CloudWatchLogsPanel displays CloudWatch log entries with container tabs.
type CloudWatchLogsPanel struct {
	mu           sync.RWMutex
	entries      []model.CloudWatchLogEntry
	containers   []model.ContainerLogConfig
	selectedTab  int
	width        int
	height       int
	scroll       int
	autoScroll   bool
	streaming    bool
	spinnerFrame int
	serviceName  string
	taskID       string
}

// NewCloudWatchLogsPanel creates a new CloudWatch logs panel.
func NewCloudWatchLogsPanel() *CloudWatchLogsPanel {
	return &CloudWatchLogsPanel{
		entries:    make([]model.CloudWatchLogEntry, 0, maxCloudWatchEntries),
		autoScroll: true,
	}
}

// SetContainers sets the container tabs.
func (p *CloudWatchLogsPanel) SetContainers(configs []model.ContainerLogConfig) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.containers = configs
	if p.selectedTab >= len(configs) {
		p.selectedTab = 0
	}
}

// SetEntries sets log entries.
func (p *CloudWatchLogsPanel) SetEntries(entries []model.CloudWatchLogEntry) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.entries = entries
	if len(p.entries) > maxCloudWatchEntries {
		p.entries = p.entries[len(p.entries)-maxCloudWatchEntries:]
	}
	if p.autoScroll {
		p.scrollToBottomLocked()
	}
}

// AppendEntries adds new entries (for incremental updates).
func (p *CloudWatchLogsPanel) AppendEntries(entries []model.CloudWatchLogEntry) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.entries = append(p.entries, entries...)
	if len(p.entries) > maxCloudWatchEntries {
		p.entries = p.entries[len(p.entries)-maxCloudWatchEntries:]
	}
	if p.autoScroll {
		p.scrollToBottomLocked()
	}
}

// SelectedContainer returns the currently selected container config.
func (p *CloudWatchLogsPanel) SelectedContainer() *model.ContainerLogConfig {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.selectedTab < len(p.containers) {
		return &p.containers[p.selectedTab]
	}
	return nil
}

// SelectedContainerIndex returns the index of the selected container.
func (p *CloudWatchLogsPanel) SelectedContainerIndex() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.selectedTab
}

// SelectNextTab moves to next container tab.
func (p *CloudWatchLogsPanel) SelectNextTab() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.containers) > 0 {
		p.selectedTab = (p.selectedTab + 1) % len(p.containers)
	}
}

// SelectPrevTab moves to previous container tab.
func (p *CloudWatchLogsPanel) SelectPrevTab() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.containers) > 0 {
		p.selectedTab = (p.selectedTab - 1 + len(p.containers)) % len(p.containers)
	}
}

// SetContext sets service/task context info.
func (p *CloudWatchLogsPanel) SetContext(serviceName, taskID string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.serviceName = serviceName
	p.taskID = taskID
}

// SetSize sets panel dimensions.
func (p *CloudWatchLogsPanel) SetSize(width, height int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.width = width
	p.height = height
}

// SetStreaming sets streaming state.
func (p *CloudWatchLogsPanel) SetStreaming(streaming bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.streaming = streaming
}

// IsStreaming returns whether streaming is active.
func (p *CloudWatchLogsPanel) IsStreaming() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.streaming
}

// AdvanceSpinner advances the spinner animation frame.
func (p *CloudWatchLogsPanel) AdvanceSpinner() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.spinnerFrame = (p.spinnerFrame + 1) % len(spinnerFrames)
}

// ScrollUp scrolls log view up.
func (p *CloudWatchLogsPanel) ScrollUp() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.autoScroll = false
	if p.scroll > 0 {
		p.scroll--
	}
}

// ScrollDown scrolls log view down.
func (p *CloudWatchLogsPanel) ScrollDown() {
	p.mu.Lock()
	defer p.mu.Unlock()
	maxScroll := p.maxScrollLocked()
	if p.scroll < maxScroll {
		p.scroll++
	}
	if p.scroll >= maxScroll {
		p.autoScroll = true
	}
}

// ScrollToBottom scrolls to newest logs and enables auto-scroll.
func (p *CloudWatchLogsPanel) ScrollToBottom() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.autoScroll = true
	p.scrollToBottomLocked()
}

func (p *CloudWatchLogsPanel) scrollToBottomLocked() {
	p.scroll = p.maxScrollLocked()
}

func (p *CloudWatchLogsPanel) maxScrollLocked() int {
	// Account for header lines (title + tabs + spacing)
	headerLines := 4
	if len(p.containers) <= 1 {
		headerLines = 3
	}
	visibleLines := p.height - headerLines
	if visibleLines < 1 {
		visibleLines = 1
	}

	filteredCount := p.filteredEntriesCountLocked()
	maxScroll := filteredCount - visibleLines
	if maxScroll < 0 {
		maxScroll = 0
	}
	return maxScroll
}

func (p *CloudWatchLogsPanel) filteredEntriesCountLocked() int {
	if len(p.containers) == 0 || p.selectedTab >= len(p.containers) {
		return len(p.entries)
	}

	selectedStream := p.containers[p.selectedTab].LogStreamName
	count := 0
	for _, e := range p.entries {
		if e.LogStreamName == selectedStream {
			count++
		}
	}
	return count
}

// Clear clears all entries.
func (p *CloudWatchLogsPanel) Clear() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.entries = p.entries[:0]
	p.scroll = 0
}

// TickCmd returns command for polling interval.
func (p *CloudWatchLogsPanel) TickCmd() tea.Cmd {
	return tea.Tick(cloudWatchPollInterval, func(t time.Time) tea.Msg {
		return CloudWatchLogsTickMsg(t)
	})
}

// SpinnerTickCmd returns command for spinner animation interval.
func (p *CloudWatchLogsPanel) SpinnerTickCmd() tea.Cmd {
	return tea.Tick(cloudWatchSpinnerInterval, func(t time.Time) tea.Msg {
		return CloudWatchSpinnerTickMsg(t)
	})
}

// View renders the CloudWatch logs panel.
func (p *CloudWatchLogsPanel) View() string {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.width == 0 || p.height == 0 {
		return ""
	}

	var b strings.Builder

	// Title with streaming indicator
	titleStyle := lipgloss.NewStyle().
		Foreground(theme.Primary).
		Bold(true)

	title := "CloudWatch Logs"
	if p.serviceName != "" {
		title = fmt.Sprintf("CloudWatch Logs - %s", p.serviceName)
	}

	if p.streaming {
		streamingStyle := lipgloss.NewStyle().Foreground(theme.Success)
		spinnerChar := spinnerFrames[p.spinnerFrame]
		title += streamingStyle.Render(fmt.Sprintf(" %s STREAMING", spinnerChar))
	}

	b.WriteString(titleStyle.Render(title))
	b.WriteString("\n")

	// Container tabs (if multiple containers)
	if len(p.containers) > 1 {
		b.WriteString(p.renderTabsLocked())
		b.WriteString("\n")
	} else if len(p.containers) == 1 {
		containerStyle := lipgloss.NewStyle().Foreground(theme.TextMuted)
		b.WriteString(containerStyle.Render("Container: " + p.containers[0].ContainerName))
		b.WriteString("\n")
	}

	b.WriteString("\n")

	// Log entries
	st := theme.DefaultStyles()

	// Filter entries for selected container
	var filteredEntries []model.CloudWatchLogEntry
	if len(p.containers) > 0 && p.selectedTab < len(p.containers) {
		selectedStream := p.containers[p.selectedTab].LogStreamName
		for _, e := range p.entries {
			if e.LogStreamName == selectedStream {
				filteredEntries = append(filteredEntries, e)
			}
		}
	} else {
		filteredEntries = p.entries
	}

	if len(filteredEntries) == 0 {
		b.WriteString(st.Muted.Render("No log entries. Waiting for logs..."))
	} else {
		// Calculate visible range
		headerLines := 4
		if len(p.containers) <= 1 {
			headerLines = 3
		}
		maxVisible := p.height - headerLines - 1 // -1 for scroll indicator
		if maxVisible < 1 {
			maxVisible = 1
		}

		start := p.scroll
		end := start + maxVisible
		if end > len(filteredEntries) {
			end = len(filteredEntries)
		}
		if start > len(filteredEntries) {
			start = len(filteredEntries)
		}

		timeStyle := st.Muted

		for i := start; i < end; i++ {
			entry := filteredEntries[i]
			timeStr := entry.Timestamp.Format("15:04:05.000")
			message := strings.TrimSpace(entry.Message)

			// Calculate available width for message (after timestamp)
			timestampWidth := lipgloss.Width(timeStr) + 1 // +1 for space
			availableWidth := p.width - 6 - timestampWidth // -6 for padding

			if availableWidth < 20 {
				availableWidth = 20
			}

			// Wrap long messages
			wrappedLines := wrapText(message, availableWidth)

			// First line includes timestamp
			line := fmt.Sprintf("%s %s", timeStyle.Render(timeStr), wrappedLines[0])
			b.WriteString(line)

			// Continuation lines are indented to align with message
			indent := strings.Repeat(" ", timestampWidth)
			for j := 1; j < len(wrappedLines); j++ {
				b.WriteString("\n")
				b.WriteString(indent + wrappedLines[j])
			}

			if i < end-1 {
				b.WriteString("\n")
			}
		}

		// Scroll indicator
		if len(filteredEntries) > maxVisible {
			b.WriteString("\n")
			scrollStyle := lipgloss.NewStyle().Foreground(theme.TextDim)
			autoScrollIndicator := ""
			if p.autoScroll {
				autoScrollIndicator = " [AUTO-SCROLL]"
			}
			scrollInfo := fmt.Sprintf("  %d/%d%s", min(p.scroll+maxVisible, len(filteredEntries)), len(filteredEntries), autoScrollIndicator)
			b.WriteString(scrollStyle.Render(scrollInfo))
		}
	}

	containerStyle := lipgloss.NewStyle().
		Padding(1, 2).
		Width(p.width)

	return containerStyle.Render(b.String())
}

func (p *CloudWatchLogsPanel) renderTabsLocked() string {
	tabStyle := lipgloss.NewStyle().
		Padding(0, 1).
		Foreground(theme.TextMuted)

	activeTabStyle := lipgloss.NewStyle().
		Padding(0, 1).
		Foreground(theme.Primary).
		Bold(true).
		Background(theme.BgHighlight)

	var tabs []string
	for i, c := range p.containers {
		name := c.ContainerName
		if len(name) > 15 {
			name = name[:12] + "..."
		}

		if i == p.selectedTab {
			tabs = append(tabs, activeTabStyle.Render(name))
		} else {
			tabs = append(tabs, tabStyle.Render(name))
		}
	}

	hintStyle := lipgloss.NewStyle().Foreground(theme.TextDim)
	tabsLine := strings.Join(tabs, " | ")
	return tabsLine + hintStyle.Render("  (Tab/Shift+Tab)")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// wrapText wraps text to fit within maxWidth characters.
func wrapText(text string, maxWidth int) []string {
	if maxWidth <= 0 {
		maxWidth = 80
	}

	if len(text) <= maxWidth {
		return []string{text}
	}

	var lines []string
	for len(text) > 0 {
		if len(text) <= maxWidth {
			lines = append(lines, text)
			break
		}

		// Find a good break point (prefer space)
		breakPoint := maxWidth
		for i := maxWidth; i > maxWidth/2; i-- {
			if text[i] == ' ' {
				breakPoint = i
				break
			}
		}

		lines = append(lines, text[:breakPoint])
		text = strings.TrimLeft(text[breakPoint:], " ")
	}

	if len(lines) == 0 {
		return []string{text}
	}

	return lines
}
