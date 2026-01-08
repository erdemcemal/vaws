package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"vaws/internal/model"
	"vaws/internal/ui/theme"
)

// DynamoDBQueryResults displays query/scan results.
type DynamoDBQueryResults struct {
	width        int
	height       int
	items        []model.DynamoDBItem
	cursor       int
	scrollOffset int
	hasMorePages bool
	count        int
	scannedCount int
	capacity     float64
	tableName    string
	pkName       string
	skName       string
	loading      bool
	err          error
	jsonScroll   int // Scroll offset for JSON panel
}

// NewDynamoDBQueryResults creates a new results panel.
func NewDynamoDBQueryResults() *DynamoDBQueryResults {
	return &DynamoDBQueryResults{}
}

// SetSize sets the panel size.
func (r *DynamoDBQueryResults) SetSize(width, height int) {
	r.width = width
	r.height = height
}

// SetResult sets the query result.
func (r *DynamoDBQueryResults) SetResult(result *model.QueryResult, tableName, pkName, skName string) {
	r.items = result.Items
	r.hasMorePages = result.HasMorePages
	r.count = result.Count
	r.scannedCount = result.ScannedCount
	r.capacity = result.ConsumedCapacity
	r.tableName = tableName
	r.pkName = pkName
	r.skName = skName
	r.cursor = 0
	r.scrollOffset = 0
	r.jsonScroll = 0
	r.loading = false
	r.err = nil
}

// SetLoading sets the loading state.
func (r *DynamoDBQueryResults) SetLoading(loading bool) {
	r.loading = loading
}

// SetError sets the error state.
func (r *DynamoDBQueryResults) SetError(err error) {
	r.err = err
	r.loading = false
}

// Clear clears the results.
func (r *DynamoDBQueryResults) Clear() {
	r.items = nil
	r.cursor = 0
	r.scrollOffset = 0
	r.jsonScroll = 0
	r.hasMorePages = false
	r.count = 0
	r.scannedCount = 0
	r.capacity = 0
}

// Up moves the cursor up.
func (r *DynamoDBQueryResults) Up() {
	if r.cursor > 0 {
		r.cursor--
		r.jsonScroll = 0 // Reset JSON scroll when changing item
		// Adjust scroll offset
		if r.cursor < r.scrollOffset {
			r.scrollOffset = r.cursor
		}
	}
}

// Down moves the cursor down.
func (r *DynamoDBQueryResults) Down() {
	if r.cursor < len(r.items)-1 {
		r.cursor++
		r.jsonScroll = 0 // Reset JSON scroll when changing item
		// Adjust scroll offset
		visibleRows := r.height - 4 // Account for header and status
		if r.cursor >= r.scrollOffset+visibleRows {
			r.scrollOffset = r.cursor - visibleRows + 1
		}
	}
}

// Top moves the cursor to the top.
func (r *DynamoDBQueryResults) Top() {
	r.cursor = 0
	r.scrollOffset = 0
	r.jsonScroll = 0
}

// Bottom moves the cursor to the bottom.
func (r *DynamoDBQueryResults) Bottom() {
	if len(r.items) > 0 {
		r.cursor = len(r.items) - 1
		r.jsonScroll = 0
		visibleRows := r.height - 4
		if r.cursor >= visibleRows {
			r.scrollOffset = r.cursor - visibleRows + 1
		}
	}
}

// ScrollJSONUp scrolls the JSON panel up.
func (r *DynamoDBQueryResults) ScrollJSONUp() {
	if r.jsonScroll > 0 {
		r.jsonScroll--
	}
}

// ScrollJSONDown scrolls the JSON panel down.
func (r *DynamoDBQueryResults) ScrollJSONDown() {
	r.jsonScroll++
}

// ScrollJSONHalfPageUp scrolls the JSON panel up by half a page.
func (r *DynamoDBQueryResults) ScrollJSONHalfPageUp() {
	halfPage := (r.height - 10) / 2
	if halfPage < 1 {
		halfPage = 1
	}
	r.jsonScroll = max(0, r.jsonScroll-halfPage)
}

// ScrollJSONHalfPageDown scrolls the JSON panel down by half a page.
func (r *DynamoDBQueryResults) ScrollJSONHalfPageDown() {
	halfPage := (r.height - 10) / 2
	if halfPage < 1 {
		halfPage = 1
	}
	r.jsonScroll += halfPage
}

// SelectedJSON returns the JSON representation of the selected item.
func (r *DynamoDBQueryResults) SelectedJSON() string {
	item := r.SelectedItem()
	if item == nil {
		return ""
	}
	return item.JSON
}

// SelectedItem returns the currently selected item.
func (r *DynamoDBQueryResults) SelectedItem() *model.DynamoDBItem {
	if r.cursor >= 0 && r.cursor < len(r.items) {
		return &r.items[r.cursor]
	}
	return nil
}

// HasMorePages returns whether there are more pages.
func (r *DynamoDBQueryResults) HasMorePages() bool {
	return r.hasMorePages
}

// ItemCount returns the number of items.
func (r *DynamoDBQueryResults) ItemCount() int {
	return len(r.items)
}

// View renders the results panel.
func (r *DynamoDBQueryResults) View() string {
	if r.loading {
		return r.renderLoading()
	}

	if r.err != nil {
		return r.renderError()
	}

	if len(r.items) == 0 {
		return r.renderEmpty()
	}

	return r.renderResults()
}

func (r *DynamoDBQueryResults) renderLoading() string {
	style := lipgloss.NewStyle().
		Width(r.width).
		Height(r.height).
		Align(lipgloss.Center, lipgloss.Center)

	loadingStyle := lipgloss.NewStyle().Foreground(theme.Primary)
	return style.Render(loadingStyle.Render("Querying..."))
}

func (r *DynamoDBQueryResults) renderError() string {
	style := lipgloss.NewStyle().
		Width(r.width).
		Height(r.height).
		Align(lipgloss.Center, lipgloss.Center)

	errorStyle := lipgloss.NewStyle().Foreground(theme.Error)
	return style.Render(errorStyle.Render("Error: " + r.err.Error()))
}

func (r *DynamoDBQueryResults) renderEmpty() string {
	style := lipgloss.NewStyle().
		Width(r.width).
		Height(r.height).
		Align(lipgloss.Center, lipgloss.Center)

	emptyStyle := lipgloss.NewStyle().Foreground(theme.TextDim)
	return style.Render(emptyStyle.Render("No items found"))
}

func (r *DynamoDBQueryResults) renderResults() string {
	// Split into list (left) and JSON detail (right)
	listWidth := r.width / 2
	jsonWidth := r.width - listWidth - 1 // -1 for separator

	listView := r.renderList(listWidth)
	jsonView := r.renderJSON(jsonWidth)

	// Separator
	separatorStyle := lipgloss.NewStyle().Foreground(theme.Border)
	separator := separatorStyle.Render(strings.Repeat("│\n", r.height))

	return lipgloss.JoinHorizontal(lipgloss.Top, listView, separator, jsonView)
}

func (r *DynamoDBQueryResults) renderList(width int) string {
	var b strings.Builder

	// Header
	headerStyle := lipgloss.NewStyle().
		Foreground(theme.Primary).
		Bold(true)

	statusStyle := lipgloss.NewStyle().
		Foreground(theme.TextDim)

	// Status line
	status := fmt.Sprintf("Items: %d", r.count)
	if r.hasMorePages {
		status += " (more available)"
	}
	if r.capacity > 0 {
		status += fmt.Sprintf(" | RCU: %.2f", r.capacity)
	}
	b.WriteString(statusStyle.Render(status))
	b.WriteString("\n")

	// Column header
	pkWidth := width / 2
	skWidth := width - pkWidth - 4
	if r.skName == "" {
		pkWidth = width - 4
		skWidth = 0
	}

	if r.skName != "" {
		header := fmt.Sprintf("  %-*s  %-*s", pkWidth, r.pkName, skWidth, r.skName)
		b.WriteString(headerStyle.Render(header))
	} else {
		header := fmt.Sprintf("  %-*s", pkWidth, r.pkName)
		b.WriteString(headerStyle.Render(header))
	}
	b.WriteString("\n")

	dimStyle := lipgloss.NewStyle().Foreground(theme.TextDim)
	b.WriteString(dimStyle.Render(strings.Repeat("─", width)))
	b.WriteString("\n")

	// Items
	visibleRows := r.height - 4
	if visibleRows < 1 {
		visibleRows = 1
	}

	selectedStyle := lipgloss.NewStyle().
		Foreground(theme.Primary).
		Bold(true)

	normalStyle := lipgloss.NewStyle().
		Foreground(theme.Text)

	endIdx := r.scrollOffset + visibleRows
	if endIdx > len(r.items) {
		endIdx = len(r.items)
	}

	for i := r.scrollOffset; i < endIdx; i++ {
		item := r.items[i]
		isSelected := i == r.cursor

		// Cursor
		cursor := "  "
		if isSelected {
			cursor = "> "
		}

		// Truncate values
		pk := item.PartitionKeyValue
		if len(pk) > pkWidth {
			pk = pk[:pkWidth-3] + "..."
		}

		var line string
		if r.skName != "" {
			sk := item.SortKeyValue
			if len(sk) > skWidth {
				sk = sk[:skWidth-3] + "..."
			}
			line = fmt.Sprintf("%s%-*s  %-*s", cursor, pkWidth, pk, skWidth, sk)
		} else {
			line = fmt.Sprintf("%s%-*s", cursor, pkWidth, pk)
		}

		if isSelected {
			b.WriteString(selectedStyle.Render(line))
		} else {
			b.WriteString(normalStyle.Render(line))
		}

		if i < endIdx-1 {
			b.WriteString("\n")
		}
	}

	// Pad remaining space
	renderedLines := endIdx - r.scrollOffset + 3 // +3 for header lines
	for i := renderedLines; i < r.height; i++ {
		b.WriteString("\n")
	}

	return lipgloss.NewStyle().Width(width).Render(b.String())
}

func (r *DynamoDBQueryResults) renderJSON(width int) string {
	item := r.SelectedItem()
	if item == nil {
		emptyStyle := lipgloss.NewStyle().
			Width(width).
			Height(r.height).
			Foreground(theme.TextDim).
			Align(lipgloss.Center, lipgloss.Center)
		return emptyStyle.Render("No item selected")
	}

	var b strings.Builder

	// Header
	headerStyle := lipgloss.NewStyle().
		Foreground(theme.Primary).
		Bold(true)

	b.WriteString(headerStyle.Render("Item Details"))
	b.WriteString("\n")

	dimStyle := lipgloss.NewStyle().Foreground(theme.TextDim)
	b.WriteString(dimStyle.Render(strings.Repeat("─", width)))
	b.WriteString("\n")

	// JSON content with syntax highlighting
	jsonLines := strings.Split(item.JSON, "\n")

	// Apply scroll offset
	startLine := r.jsonScroll
	if startLine >= len(jsonLines) {
		startLine = max(0, len(jsonLines)-1)
	}

	visibleLines := r.height - 3 // Account for header
	endLine := startLine + visibleLines
	if endLine > len(jsonLines) {
		endLine = len(jsonLines)
	}

	keyStyle := lipgloss.NewStyle().Foreground(theme.Primary)
	stringStyle := lipgloss.NewStyle().Foreground(theme.Success)
	numberStyle := lipgloss.NewStyle().Foreground(theme.Warning)
	boolStyle := lipgloss.NewStyle().Foreground(theme.Info)
	nullStyle := lipgloss.NewStyle().Foreground(theme.TextDim)

	for i := startLine; i < endLine; i++ {
		line := jsonLines[i]
		// Truncate line if too long
		if len(line) > width-1 {
			line = line[:width-4] + "..."
		}

		// Simple syntax highlighting
		highlighted := highlightJSONLine(line, keyStyle, stringStyle, numberStyle, boolStyle, nullStyle)
		b.WriteString(highlighted)
		if i < endLine-1 {
			b.WriteString("\n")
		}
	}

	// Scroll indicator
	if len(jsonLines) > visibleLines {
		b.WriteString("\n")
		scrollInfo := fmt.Sprintf("Lines %d-%d of %d (j/k to scroll JSON)", startLine+1, endLine, len(jsonLines))
		b.WriteString(dimStyle.Render(scrollInfo))
	}

	return lipgloss.NewStyle().Width(width).Render(b.String())
}

// highlightJSONLine applies simple syntax highlighting to a JSON line.
func highlightJSONLine(line string, keyStyle, stringStyle, numberStyle, boolStyle, nullStyle lipgloss.Style) string {
	// Very simple highlighting - just color strings, numbers, bools, null
	var result strings.Builder
	inString := false
	isKey := false
	i := 0

	for i < len(line) {
		c := line[i]

		if c == '"' {
			if !inString {
				inString = true
				// Check if this is a key (followed by ":")
				endQuote := strings.Index(line[i+1:], "\"")
				if endQuote >= 0 {
					afterQuote := i + 1 + endQuote + 1
					if afterQuote < len(line) && strings.HasPrefix(strings.TrimSpace(line[afterQuote:]), ":") {
						isKey = true
					}
				}
				result.WriteByte(c)
			} else {
				result.WriteByte(c)
				inString = false
				isKey = false
			}
			i++
			continue
		}

		if inString {
			// Inside a string - collect until end
			start := i
			for i < len(line) && line[i] != '"' {
				if line[i] == '\\' && i+1 < len(line) {
					i += 2 // Skip escaped char
				} else {
					i++
				}
			}
			str := line[start:i]
			if isKey {
				result.WriteString(keyStyle.Render(str))
			} else {
				result.WriteString(stringStyle.Render(str))
			}
			continue
		}

		// Check for keywords
		remaining := line[i:]
		if strings.HasPrefix(remaining, "true") {
			result.WriteString(boolStyle.Render("true"))
			i += 4
			continue
		}
		if strings.HasPrefix(remaining, "false") {
			result.WriteString(boolStyle.Render("false"))
			i += 5
			continue
		}
		if strings.HasPrefix(remaining, "null") {
			result.WriteString(nullStyle.Render("null"))
			i += 4
			continue
		}

		// Check for numbers
		if (c >= '0' && c <= '9') || c == '-' {
			start := i
			for i < len(line) && ((line[i] >= '0' && line[i] <= '9') || line[i] == '.' || line[i] == '-' || line[i] == 'e' || line[i] == 'E' || line[i] == '+') {
				i++
			}
			result.WriteString(numberStyle.Render(line[start:i]))
			continue
		}

		result.WriteByte(c)
		i++
	}

	return result.String()
}
