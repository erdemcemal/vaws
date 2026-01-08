package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"vaws/internal/model"
	"vaws/internal/ui/theme"
)

// DynamoDBTable displays DynamoDB tables in a simple table format.
type DynamoDBTable struct {
	width   int
	height  int
	tables  []model.Table
	cursor  int
	loading bool
	err     error
	spinner *Spinner
}

// NewDynamoDBTable creates a new DynamoDBTable.
func NewDynamoDBTable() *DynamoDBTable {
	return &DynamoDBTable{
		spinner: NewSpinner(),
	}
}

// SetSize sets the table dimensions.
func (t *DynamoDBTable) SetSize(width, height int) {
	t.width = width
	t.height = height
}

// SetTables sets the table list.
func (t *DynamoDBTable) SetTables(tables []model.Table) {
	t.tables = tables
	if t.cursor >= len(tables) {
		t.cursor = max(0, len(tables)-1)
	}
}

// SetLoading sets the loading state.
func (t *DynamoDBTable) SetLoading(loading bool) {
	t.loading = loading
}

// SetError sets the error state.
func (t *DynamoDBTable) SetError(err error) {
	t.err = err
}

// Spinner returns the spinner for loading animation.
func (t *DynamoDBTable) Spinner() *Spinner {
	return t.spinner
}

// Cursor returns the current cursor position.
func (t *DynamoDBTable) Cursor() int {
	return t.cursor
}

// SelectedTable returns the currently selected table.
func (t *DynamoDBTable) SelectedTable() *model.Table {
	if t.cursor >= 0 && t.cursor < len(t.tables) {
		return &t.tables[t.cursor]
	}
	return nil
}

// Up moves the cursor up.
func (t *DynamoDBTable) Up() {
	if t.cursor > 0 {
		t.cursor--
	}
}

// Down moves the cursor down.
func (t *DynamoDBTable) Down() {
	if t.cursor < len(t.tables)-1 {
		t.cursor++
	}
}

// Top moves the cursor to the top.
func (t *DynamoDBTable) Top() {
	t.cursor = 0
}

// Bottom moves the cursor to the bottom.
func (t *DynamoDBTable) Bottom() {
	if len(t.tables) > 0 {
		t.cursor = len(t.tables) - 1
	}
}

// TableCount returns the number of tables.
func (t *DynamoDBTable) TableCount() int {
	return len(t.tables)
}

// View renders the DynamoDB table.
func (t *DynamoDBTable) View() string {
	if t.loading {
		return t.renderLoading()
	}

	if t.err != nil {
		return t.renderError()
	}

	if len(t.tables) == 0 {
		return t.renderEmpty()
	}

	return t.renderTable()
}

func (t *DynamoDBTable) renderLoading() string {
	style := lipgloss.NewStyle().
		Width(t.width).
		Height(t.height).
		Align(lipgloss.Center, lipgloss.Center)

	loadingStyle := lipgloss.NewStyle().Foreground(theme.Primary)
	return style.Render(loadingStyle.Render(t.spinner.View() + " Loading DynamoDB tables..."))
}

func (t *DynamoDBTable) renderError() string {
	style := lipgloss.NewStyle().
		Width(t.width).
		Height(t.height).
		Align(lipgloss.Center, lipgloss.Center)

	errorStyle := lipgloss.NewStyle().Foreground(theme.Error)
	return style.Render(errorStyle.Render("Error: " + t.err.Error()))
}

func (t *DynamoDBTable) renderEmpty() string {
	style := lipgloss.NewStyle().
		Width(t.width).
		Height(t.height).
		Align(lipgloss.Center, lipgloss.Center)

	emptyStyle := lipgloss.NewStyle().Foreground(theme.TextDim)
	return style.Render(emptyStyle.Render("No DynamoDB tables found"))
}

func (t *DynamoDBTable) renderTable() string {
	var b strings.Builder

	// Add top margin
	b.WriteString("\n")

	// Fixed column widths
	statusWidth := 8
	itemsWidth := 12
	sizeWidth := 10
	pkWidth := 15

	// NAME gets remaining space but with reasonable limit
	availableForName := t.width - statusWidth - itemsWidth - sizeWidth - pkWidth - 12
	nameWidth := availableForName
	if nameWidth > 60 {
		nameWidth = 60
	}
	if nameWidth < 20 {
		nameWidth = 20
	}

	// Total used width
	totalWidth := nameWidth + statusWidth + itemsWidth + sizeWidth + pkWidth + 8

	// Styles
	headerStyle := lipgloss.NewStyle().
		Foreground(theme.Primary).
		Bold(true)

	dimStyle := lipgloss.NewStyle().Foreground(theme.TextDim)
	selectedStyle := lipgloss.NewStyle().Foreground(theme.Primary).Bold(true)
	activeStyle := lipgloss.NewStyle().Foreground(theme.Success)
	inProgressStyle := lipgloss.NewStyle().Foreground(theme.Warning)

	// Header
	header := fmt.Sprintf("  %-*s  %-*s  %*s  %*s  %-*s",
		nameWidth, "NAME",
		statusWidth, "STATUS",
		itemsWidth, "ITEMS",
		sizeWidth, "SIZE",
		pkWidth, "PK",
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
	if endIdx > len(t.tables) {
		endIdx = len(t.tables)
	}

	// Render rows
	for i := startIdx; i < endIdx; i++ {
		tbl := t.tables[i]
		isSelected := i == t.cursor

		// Cursor
		cursor := "  "
		if isSelected {
			cursor = "> "
		}

		// Name (truncate if needed)
		name := tbl.Name
		if len(name) > nameWidth {
			name = name[:nameWidth-3] + "..."
		}

		// Status with color
		status := string(tbl.Status)
		if len(status) > statusWidth {
			status = status[:statusWidth]
		}
		var statusStr string
		if tbl.Status.IsHealthy() {
			statusStr = activeStyle.Render(fmt.Sprintf("%-*s", statusWidth, status))
		} else if tbl.Status.IsInProgress() {
			statusStr = inProgressStyle.Render(fmt.Sprintf("%-*s", statusWidth, status))
		} else {
			statusStr = fmt.Sprintf("%-*s", statusWidth, status)
		}

		// Items count
		itemsStr := formatCount(tbl.ItemCount)

		// Size
		sizeStr := formatSize(tbl.SizeBytes)

		// Partition key
		pk := tbl.PartitionKey()
		if len(pk) > pkWidth {
			pk = pk[:pkWidth-3] + "..."
		}

		// Build row with consistent spacing
		// Pad name to exact width
		paddedName := fmt.Sprintf("%-*s", nameWidth, name)

		if isSelected {
			b.WriteString(selectedStyle.Render(cursor + paddedName))
			// Render remaining columns without selection styling
			rest := fmt.Sprintf("  %s  %*s  %*s  %-*s",
				statusStr,
				itemsWidth, itemsStr,
				sizeWidth, sizeStr,
				pkWidth, pk,
			)
			b.WriteString(rest)
		} else {
			row := fmt.Sprintf("%s%s  %s  %*s  %*s  %-*s",
				cursor,
				paddedName,
				statusStr,
				itemsWidth, itemsStr,
				sizeWidth, sizeStr,
				pkWidth, pk,
			)
			b.WriteString(row)
		}

		if i < endIdx-1 {
			b.WriteString("\n")
		}
	}

	return b.String()
}

// formatCount formats large numbers with K/M suffixes.
func formatCount(n int64) string {
	if n >= 1_000_000 {
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	}
	if n >= 1_000 {
		return fmt.Sprintf("%.1fK", float64(n)/1_000)
	}
	return fmt.Sprintf("%d", n)
}

// formatSize formats bytes into human-readable sizes.
func formatSize(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.1fGB", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.1fMB", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.1fKB", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%dB", bytes)
	}
}
