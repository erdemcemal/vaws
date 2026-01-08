package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"vaws/internal/ui/theme"
)

// ListItem represents an item in the list.
type ListItem struct {
	ID          string
	Title       string
	Description string
	Status      string
	StatusStyle lipgloss.Style
	Extra       string
	IsHeader    bool // Non-selectable category header
}

// List is a scrollable, selectable list component.
type List struct {
	title     string
	showTitle bool
	items     []ListItem
	cursor    int
	offset    int
	width     int
	height    int
	loading   bool
	errMsg    string
	emptyMsg  string
	spinner   *Spinner
}

// NewList creates a new List component.
func NewList(title string) *List {
	return &List{
		title:     title,
		showTitle: false, // Title is shown in Container border now
		emptyMsg:  "No items found",
		spinner:   NewSpinner(),
	}
}

// SetShowTitle sets whether to show the title header.
func (l *List) SetShowTitle(show bool) {
	l.showTitle = show
}

// Spinner returns the list's spinner for external tick updates.
func (l *List) Spinner() *Spinner {
	return l.spinner
}

// SetTitle sets the list title.
func (l *List) SetTitle(title string) {
	l.title = title
}

// SetItems sets the list items.
func (l *List) SetItems(items []ListItem) {
	l.items = items
	if l.cursor >= len(items) {
		l.cursor = max(0, len(items)-1)
	}
	l.clampOffset()
}

// SetSize sets the list dimensions.
func (l *List) SetSize(width, height int) {
	l.width = width
	l.height = height
	l.clampOffset()
}

// SetLoading sets the loading state.
func (l *List) SetLoading(loading bool) {
	l.loading = loading
}

// SetError sets the error message.
func (l *List) SetError(err error) {
	if err != nil {
		l.errMsg = err.Error()
	} else {
		l.errMsg = ""
	}
}

// SetEmptyMessage sets the message to display when list is empty.
func (l *List) SetEmptyMessage(msg string) {
	l.emptyMsg = msg
}

// Cursor returns the current cursor position.
func (l *List) Cursor() int {
	return l.cursor
}

// SelectedItem returns the currently selected item, or nil if none.
func (l *List) SelectedItem() *ListItem {
	if l.cursor >= 0 && l.cursor < len(l.items) {
		return &l.items[l.cursor]
	}
	return nil
}

// Up moves the cursor up, skipping headers.
func (l *List) Up() {
	if l.cursor > 0 {
		l.cursor--
		l.skipHeadersUp()
		l.clampOffset()
	}
}

// Down moves the cursor down, skipping headers.
func (l *List) Down() {
	if l.cursor < len(l.items)-1 {
		l.cursor++
		l.skipHeadersDown()
		l.clampOffset()
	}
}

// Top moves the cursor to the first selectable item.
func (l *List) Top() {
	l.cursor = 0
	l.skipHeadersDown()
	l.offset = 0
}

// Bottom moves the cursor to the last selectable item.
func (l *List) Bottom() {
	l.cursor = max(0, len(l.items)-1)
	l.skipHeadersUp()
	l.clampOffset()
}

// skipHeadersDown moves cursor down to skip any headers.
func (l *List) skipHeadersDown() {
	for l.cursor < len(l.items) && l.items[l.cursor].IsHeader {
		l.cursor++
	}
	if l.cursor >= len(l.items) {
		l.cursor = max(0, len(l.items)-1)
		l.skipHeadersUp()
	}
}

// skipHeadersUp moves cursor up to skip any headers.
func (l *List) skipHeadersUp() {
	for l.cursor >= 0 && l.cursor < len(l.items) && l.items[l.cursor].IsHeader {
		l.cursor--
	}
	if l.cursor < 0 {
		l.cursor = 0
		l.skipHeadersDown()
	}
}

func (l *List) clampOffset() {
	visibleItems := l.visibleItemCount()
	if visibleItems <= 0 {
		return
	}

	if l.cursor < l.offset {
		l.offset = l.cursor
	} else if l.cursor >= l.offset+visibleItems {
		l.offset = l.cursor - visibleItems + 1
	}

	maxOffset := max(0, len(l.items)-visibleItems)
	l.offset = min(l.offset, maxOffset)
	l.offset = max(0, l.offset)
}

func (l *List) visibleItemCount() int {
	// Account for title line if shown, plus some padding
	if l.showTitle {
		return max(1, l.height-4)
	}
	return max(1, l.height-2)
}

// View renders the list.
func (l *List) View() string {
	s := theme.DefaultStyles()
	var b strings.Builder

	containerStyle := lipgloss.NewStyle().
		PaddingLeft(1).
		PaddingRight(1)

	// Title with count (only if showTitle is true)
	if l.showTitle {
		titleText := l.title
		if len(l.items) > 0 {
			titleText = fmt.Sprintf("%s (%d)", l.title, len(l.items))
		}
		b.WriteString(s.SidebarTitle.Render(titleText))
		b.WriteString("\n")
	}

	// Loading state
	if l.loading {
		loadingText := l.spinner.View() + " " + s.Muted.Render("Loading...")
		b.WriteString(loadingText)
		return containerStyle.Render(b.String())
	}

	// Error state
	if l.errMsg != "" {
		errStyle := s.StatusError.Copy().Width(l.width - 6)
		b.WriteString(errStyle.Render("✗ " + l.errMsg))
		return containerStyle.Render(b.String())
	}

	// Empty state
	if len(l.items) == 0 {
		b.WriteString(s.Muted.Render("  " + l.emptyMsg))
		return containerStyle.Render(b.String())
	}

	// Calculate column widths
	nameWidth := l.width - 30
	if nameWidth < 20 {
		nameWidth = 20
	}

	// Render visible items
	visibleCount := l.visibleItemCount()
	end := min(l.offset+visibleCount, len(l.items))

	// Header style
	headerStyle := lipgloss.NewStyle().
		Foreground(theme.TextMuted).
		Bold(true)

	for i := l.offset; i < end; i++ {
		item := l.items[i]
		isSelected := i == l.cursor

		var line strings.Builder

		// Handle header items (non-selectable category separators)
		if item.IsHeader {
			line.WriteString(headerStyle.Render(item.Title))
			b.WriteString(line.String())
			if i < end-1 {
				b.WriteString("\n")
			}
			continue
		}

		// Cursor indicator
		if isSelected {
			line.WriteString(s.SidebarCursor.Render("▸ "))
		} else {
			line.WriteString("  ")
		}

		// Item name (truncated if needed)
		name := item.Title
		if len(name) > nameWidth {
			name = name[:nameWidth-3] + "..."
		}
		namePadded := fmt.Sprintf("%-*s", nameWidth, name)

		if isSelected {
			line.WriteString(s.SidebarSelected.Render(namePadded))
		} else {
			line.WriteString(s.SidebarItem.Render(namePadded))
		}

		// Status with styling
		if item.Status != "" {
			line.WriteString(" ")
			line.WriteString(item.StatusStyle.Render(item.Status))
		}

		b.WriteString(line.String())
		if i < end-1 {
			b.WriteString("\n")
		}
	}

	// Scroll indicator
	if len(l.items) > visibleCount {
		b.WriteString("\n")
		scrollText := fmt.Sprintf("↑↓ %d-%d of %d", l.offset+1, end, len(l.items))
		b.WriteString(s.Muted.Render(scrollText))
	}

	return containerStyle.Render(b.String())
}
