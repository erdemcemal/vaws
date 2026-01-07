package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"vaws/internal/model"
	"vaws/internal/ui/theme"
)

// SQSTable displays SQS queues in a simple table format.
type SQSTable struct {
	width   int
	height  int
	queues  []model.Queue
	cursor  int
	loading bool
	err     error
	spinner *Spinner
}

// NewSQSTable creates a new SQSTable.
func NewSQSTable() *SQSTable {
	return &SQSTable{
		spinner: NewSpinner(),
	}
}

// SetSize sets the table dimensions.
func (t *SQSTable) SetSize(width, height int) {
	t.width = width
	t.height = height
}

// SetQueues sets the queue list.
func (t *SQSTable) SetQueues(queues []model.Queue) {
	t.queues = queues
	if t.cursor >= len(queues) {
		t.cursor = max(0, len(queues)-1)
	}
}

// SetLoading sets the loading state.
func (t *SQSTable) SetLoading(loading bool) {
	t.loading = loading
}

// SetError sets the error state.
func (t *SQSTable) SetError(err error) {
	t.err = err
}

// Spinner returns the spinner for loading animation.
func (t *SQSTable) Spinner() *Spinner {
	return t.spinner
}

// Cursor returns the current cursor position.
func (t *SQSTable) Cursor() int {
	return t.cursor
}

// SelectedQueue returns the currently selected queue.
func (t *SQSTable) SelectedQueue() *model.Queue {
	if t.cursor >= 0 && t.cursor < len(t.queues) {
		return &t.queues[t.cursor]
	}
	return nil
}

// Up moves the cursor up.
func (t *SQSTable) Up() {
	if t.cursor > 0 {
		t.cursor--
	}
}

// Down moves the cursor down.
func (t *SQSTable) Down() {
	if t.cursor < len(t.queues)-1 {
		t.cursor++
	}
}

// Top moves the cursor to the top.
func (t *SQSTable) Top() {
	t.cursor = 0
}

// Bottom moves the cursor to the bottom.
func (t *SQSTable) Bottom() {
	if len(t.queues) > 0 {
		t.cursor = len(t.queues) - 1
	}
}

// QueueCount returns the number of queues.
func (t *SQSTable) QueueCount() int {
	return len(t.queues)
}

// View renders the SQS table.
func (t *SQSTable) View() string {
	if t.loading {
		return t.renderLoading()
	}

	if t.err != nil {
		return t.renderError()
	}

	if len(t.queues) == 0 {
		return t.renderEmpty()
	}

	return t.renderTable()
}

func (t *SQSTable) renderLoading() string {
	style := lipgloss.NewStyle().
		Width(t.width).
		Height(t.height).
		Align(lipgloss.Center, lipgloss.Center)

	loadingStyle := lipgloss.NewStyle().Foreground(theme.Primary)
	return style.Render(loadingStyle.Render(t.spinner.View() + " Loading SQS queues..."))
}

func (t *SQSTable) renderError() string {
	style := lipgloss.NewStyle().
		Width(t.width).
		Height(t.height).
		Align(lipgloss.Center, lipgloss.Center)

	errorStyle := lipgloss.NewStyle().Foreground(theme.Error)
	return style.Render(errorStyle.Render("Error: " + t.err.Error()))
}

func (t *SQSTable) renderEmpty() string {
	style := lipgloss.NewStyle().
		Width(t.width).
		Height(t.height).
		Align(lipgloss.Center, lipgloss.Center)

	emptyStyle := lipgloss.NewStyle().Foreground(theme.TextDim)
	return style.Render(emptyStyle.Render("No SQS queues found"))
}

func (t *SQSTable) renderTable() string {
	var b strings.Builder

	// Add top margin
	b.WriteString("\n")

	// Fixed column widths - compact
	msgWidth := 10
	flightWidth := 12

	// NAME gets remaining space but with reasonable limit
	availableForName := t.width - msgWidth - flightWidth - 8
	nameWidth := availableForName
	if nameWidth > 80 {
		nameWidth = 80
	}
	if nameWidth < 20 {
		nameWidth = 20
	}

	// Total used width
	totalWidth := nameWidth + msgWidth + flightWidth + 4

	// Styles
	headerStyle := lipgloss.NewStyle().
		Foreground(theme.Primary).
		Bold(true)

	dimStyle := lipgloss.NewStyle().Foreground(theme.TextDim)
	selectedStyle := lipgloss.NewStyle().Foreground(theme.Primary).Bold(true)

	// Header
	header := fmt.Sprintf("  %-*s  %*s  %*s",
		nameWidth, "NAME",
		msgWidth, "MESSAGES",
		flightWidth, "IN FLIGHT",
	)
	b.WriteString(headerStyle.Render(header))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render(strings.Repeat("─", totalWidth+2)))
	b.WriteString("\n")

	// Calculate visible rows (accounting for top margin, header, separator)
	maxRows := t.height - 4
	if maxRows < 1 {
		maxRows = 1
	}

	// Scroll offset
	startIdx := 0
	if t.cursor >= maxRows {
		startIdx = t.cursor - maxRows + 1
	}

	endIdx := startIdx + maxRows
	if endIdx > len(t.queues) {
		endIdx = len(t.queues)
	}

	// Render rows
	for i := startIdx; i < endIdx; i++ {
		q := t.queues[i]
		isSelected := i == t.cursor

		// Cursor
		cursor := "  "
		if isSelected {
			cursor = "> "
		}

		// Name (truncate if needed)
		name := q.Name
		if len(name) > nameWidth {
			name = name[:nameWidth-3] + "..."
		}

		// Build row with consistent spacing
		row := fmt.Sprintf("%s%-*s  %*d  %*d",
			cursor,
			nameWidth, name,
			msgWidth, q.ApproximateMessageCount,
			flightWidth, q.ApproximateInFlight,
		)

		// Apply style
		if isSelected {
			b.WriteString(selectedStyle.Render(row))
		} else {
			b.WriteString(row)
		}

		if i < endIdx-1 {
			b.WriteString("\n")
		}
	}

	return b.String()
}
