package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"vaws/internal/ui/theme"
)

// DetailRow represents a row in the details pane.
type DetailRow struct {
	Label string
	Value string
	Style lipgloss.Style
}

// Details is a component that displays key-value details.
type Details struct {
	title        string
	rows         []DetailRow
	width        int
	height       int
	scrollOffset int  // Current scroll position
	focused      bool // Whether details pane has focus

	// Search state
	searchQuery   string
	searchMatches []int // indices of matching rows
	searchIndex   int   // current match index
}

// NewDetails creates a new Details component.
func NewDetails() *Details {
	return &Details{}
}

// SetTitle sets the details title.
func (d *Details) SetTitle(title string) {
	d.title = title
}

// SetRows sets the detail rows and resets scroll position.
func (d *Details) SetRows(rows []DetailRow) {
	d.rows = rows
	d.scrollOffset = 0 // Reset scroll when content changes
}

// SetSize sets the component dimensions.
func (d *Details) SetSize(width, height int) {
	d.width = width
	d.height = height
}

// visibleRows returns the number of rows that can be displayed.
func (d *Details) visibleRows() int {
	rows := d.height - 6
	if rows < 1 {
		return 1
	}
	return rows
}

// ScrollUp scrolls up by one row.
func (d *Details) ScrollUp() {
	if d.scrollOffset > 0 {
		d.scrollOffset--
	}
}

// ScrollDown scrolls down by one row.
func (d *Details) ScrollDown() {
	maxOffset := max(0, len(d.rows)-d.visibleRows())
	if d.scrollOffset < maxOffset {
		d.scrollOffset++
	}
}

// ScrollHalfPageDown scrolls down by half a page.
func (d *Details) ScrollHalfPageDown() {
	halfPage := d.visibleRows() / 2
	if halfPage < 1 {
		halfPage = 1
	}
	maxOffset := max(0, len(d.rows)-d.visibleRows())
	d.scrollOffset = min(d.scrollOffset+halfPage, maxOffset)
}

// ScrollHalfPageUp scrolls up by half a page.
func (d *Details) ScrollHalfPageUp() {
	halfPage := d.visibleRows() / 2
	if halfPage < 1 {
		halfPage = 1
	}
	d.scrollOffset = max(0, d.scrollOffset-halfPage)
}

// ScrollPageDown scrolls down by a full page.
func (d *Details) ScrollPageDown() {
	page := d.visibleRows()
	maxOffset := max(0, len(d.rows)-d.visibleRows())
	d.scrollOffset = min(d.scrollOffset+page, maxOffset)
}

// ScrollPageUp scrolls up by a full page.
func (d *Details) ScrollPageUp() {
	page := d.visibleRows()
	d.scrollOffset = max(0, d.scrollOffset-page)
}

// ScrollToTop scrolls to the top.
func (d *Details) ScrollToTop() {
	d.scrollOffset = 0
}

// ScrollToBottom scrolls to the bottom.
func (d *Details) ScrollToBottom() {
	d.scrollOffset = max(0, len(d.rows)-d.visibleRows())
}

// SetFocused sets the focus state.
func (d *Details) SetFocused(focused bool) {
	d.focused = focused
}

// IsFocused returns whether the details pane has focus.
func (d *Details) IsFocused() bool {
	return d.focused
}

// ResetScroll resets the scroll position to the top.
func (d *Details) ResetScroll() {
	d.scrollOffset = 0
}

// SetSearchQuery sets the search query and updates matches.
func (d *Details) SetSearchQuery(query string) {
	d.searchQuery = query
	d.updateSearchMatches()
}

// ClearSearch clears the search query and matches.
func (d *Details) ClearSearch() {
	d.searchQuery = ""
	d.searchMatches = nil
	d.searchIndex = 0
}

// updateSearchMatches finds all rows matching the search query.
func (d *Details) updateSearchMatches() {
	d.searchMatches = nil
	d.searchIndex = 0
	if d.searchQuery == "" {
		return
	}
	query := strings.ToLower(d.searchQuery)
	for i, row := range d.rows {
		if strings.Contains(strings.ToLower(row.Label), query) ||
			strings.Contains(strings.ToLower(row.Value), query) {
			d.searchMatches = append(d.searchMatches, i)
		}
	}
	// Jump to first match
	if len(d.searchMatches) > 0 {
		d.scrollToMatch(d.searchMatches[0])
	}
}

// scrollToMatch scrolls to make a row index visible.
func (d *Details) scrollToMatch(idx int) {
	maxRows := d.visibleRows()
	if idx < d.scrollOffset || idx >= d.scrollOffset+maxRows {
		d.scrollOffset = max(0, idx-maxRows/2)
	}
}

// NextMatch moves to the next search match.
func (d *Details) NextMatch() {
	if len(d.searchMatches) == 0 {
		return
	}
	d.searchIndex = (d.searchIndex + 1) % len(d.searchMatches)
	d.scrollToMatch(d.searchMatches[d.searchIndex])
}

// PrevMatch moves to the previous search match.
func (d *Details) PrevMatch() {
	if len(d.searchMatches) == 0 {
		return
	}
	d.searchIndex--
	if d.searchIndex < 0 {
		d.searchIndex = len(d.searchMatches) - 1
	}
	d.scrollToMatch(d.searchMatches[d.searchIndex])
}

// SearchQuery returns the current search query.
func (d *Details) SearchQuery() string {
	return d.searchQuery
}

// MatchCount returns the number of search matches.
func (d *Details) MatchCount() int {
	return len(d.searchMatches)
}

// CurrentMatchIndex returns the current match index (1-based for display).
func (d *Details) CurrentMatchIndex() int {
	if len(d.searchMatches) == 0 {
		return 0
	}
	return d.searchIndex + 1
}

// isMatchRow returns true if the row at index idx is a search match.
func (d *Details) isMatchRow(idx int) bool {
	for _, matchIdx := range d.searchMatches {
		if matchIdx == idx {
			return true
		}
	}
	return false
}

// isCurrentMatch returns true if the row at index is the current match.
func (d *Details) isCurrentMatch(idx int) bool {
	if len(d.searchMatches) == 0 {
		return false
	}
	return d.searchMatches[d.searchIndex] == idx
}

// View renders the details pane.
func (d *Details) View() string {
	s := theme.DefaultStyles()

	if len(d.rows) == 0 {
		return s.Content.
			Width(d.width - 4).
			Height(d.height - 4).
			Render(s.Muted.Render("Select an item to view details"))
	}

	var b strings.Builder

	if d.title != "" {
		b.WriteString(s.ContentTitle.Render(d.title))
		b.WriteString("\n")
	}

	maxRows := d.visibleRows()

	// Clamp scroll offset
	maxOffset := max(0, len(d.rows)-maxRows)
	if d.scrollOffset > maxOffset {
		d.scrollOffset = maxOffset
	}

	// Render visible rows from scrollOffset
	endIdx := min(d.scrollOffset+maxRows, len(d.rows))
	for i := d.scrollOffset; i < endIdx; i++ {
		row := d.rows[i]

		// Skip rows with empty label and value (spacers render as blank lines)
		if row.Label == "" && row.Value == "" {
			b.WriteString("\n")
			continue
		}

		// Check if this row is a search match
		isMatch := d.isMatchRow(i)
		isCurrent := d.isCurrentMatch(i)

		label := s.DetailLabel.Render(row.Label + ":")
		value := row.Value

		if row.Style.String() != "" {
			value = row.Style.Render(value)
		} else {
			value = s.DetailValue.Render(value)
		}

		// Use more of the available width for values
		maxValueWidth := d.width - 18
		if lipgloss.Width(value) > maxValueWidth && maxValueWidth > 0 {
			value = truncate(value, maxValueWidth)
		}

		// Highlight search matches
		line := label + " " + value
		if isCurrent {
			// Current match - more visible highlight
			line = lipgloss.NewStyle().
				Background(theme.Primary).
				Foreground(lipgloss.Color("#FFFFFF")).
				Render("> " + row.Label + ": " + row.Value)
		} else if isMatch {
			// Other matches - subtle highlight
			line = lipgloss.NewStyle().
				Background(theme.PrimaryMuted).
				Render("  " + row.Label + ": " + row.Value)
		}

		b.WriteString(line)
		if i < endIdx-1 {
			b.WriteString("\n")
		}
	}

	// Search indicator (if searching)
	if d.searchQuery != "" {
		searchInfo := fmt.Sprintf("\n\nSearch: \"%s\" (%d/%d)", d.searchQuery, d.CurrentMatchIndex(), d.MatchCount())
		b.WriteString(s.Muted.Render(searchInfo))
	} else if len(d.rows) > maxRows {
		// Scroll indicator (only show if content overflows and not searching)
		indicator := fmt.Sprintf("\n\n↑↓ %d-%d of %d", d.scrollOffset+1, endIdx, len(d.rows))
		b.WriteString(s.Muted.Render(indicator))
	}

	return s.Content.
		Width(d.width - 4).
		Render(b.String())
}

// PlainTextView returns the details content as plain text for copy mode.
func (d *Details) PlainTextView() string {
	var b strings.Builder
	if d.title != "" {
		b.WriteString(d.title + "\n")
		b.WriteString(strings.Repeat("-", len(d.title)) + "\n\n")
	}
	for _, row := range d.rows {
		if row.Label == "" && row.Value == "" {
			b.WriteString("\n")
			continue
		}
		b.WriteString(row.Label + ": " + row.Value + "\n")
	}
	return b.String()
}

// StackDetails returns detail rows for a stack.
func StackDetails(name, status, createdAt, updatedAt, description string, statusStyle lipgloss.Style) []DetailRow {
	rows := []DetailRow{
		{Label: "Name", Value: name},
		{Label: "Status", Value: status, Style: statusStyle},
		{Label: "Created", Value: createdAt},
	}

	if updatedAt != "" && updatedAt != createdAt {
		rows = append(rows, DetailRow{Label: "Updated", Value: updatedAt})
	}

	if description != "" {
		rows = append(rows, DetailRow{Label: "Description", Value: description})
	}

	return rows
}

// ServiceDetails returns detail rows for an ECS service.
func ServiceDetails(name, cluster, status string, running, desired, pending int, taskDef, launchType string, containerPorts string, statusStyle lipgloss.Style) []DetailRow {
	s := theme.DefaultStyles()

	rows := []DetailRow{
		{Label: "Service", Value: name},
		{Label: "Cluster", Value: cluster},
		{Label: "Status", Value: status, Style: statusStyle},
		{Label: "Tasks", Value: fmt.Sprintf("%d/%d running", running, desired), Style: statusStyle},
	}

	if pending > 0 {
		rows = append(rows, DetailRow{
			Label: "Pending",
			Value: fmt.Sprintf("%d", pending),
			Style: s.StatusWarning,
		})
	}

	if taskDef != "" {
		if idx := strings.LastIndex(taskDef, "/"); idx >= 0 {
			taskDef = taskDef[idx+1:]
		}
		rows = append(rows, DetailRow{Label: "Task Def", Value: taskDef})
	}

	if launchType != "" {
		rows = append(rows, DetailRow{Label: "Launch Type", Value: launchType})
	}

	if containerPorts != "" {
		rows = append(rows, DetailRow{Label: "Containers", Value: containerPorts})
	}

	return rows
}
