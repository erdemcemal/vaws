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
	title  string
	rows   []DetailRow
	width  int
	height int
}

// NewDetails creates a new Details component.
func NewDetails() *Details {
	return &Details{}
}

// SetTitle sets the details title.
func (d *Details) SetTitle(title string) {
	d.title = title
}

// SetRows sets the detail rows.
func (d *Details) SetRows(rows []DetailRow) {
	d.rows = rows
}

// SetSize sets the component dimensions.
func (d *Details) SetSize(width, height int) {
	d.width = width
	d.height = height
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

	maxRows := d.height - 6
	if maxRows < 1 {
		maxRows = 1
	}

	for i, row := range d.rows {
		if i >= maxRows {
			remaining := len(d.rows) - i
			b.WriteString(fmt.Sprintf("\n... and %d more", remaining))
			break
		}

		label := s.DetailLabel.Render(row.Label + ":")
		value := row.Value

		if row.Style.String() != "" {
			value = row.Style.Render(value)
		} else {
			value = s.DetailValue.Render(value)
		}

		maxValueWidth := d.width - 26
		if lipgloss.Width(value) > maxValueWidth && maxValueWidth > 0 {
			value = truncate(value, maxValueWidth)
		}

		b.WriteString(label + " " + value)
		if i < len(d.rows)-1 && i < maxRows-1 {
			b.WriteString("\n")
		}
	}

	return s.Content.
		Width(d.width - 4).
		Render(b.String())
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
func ServiceDetails(name, cluster, status string, running, desired, pending int, taskDef, launchType string, statusStyle lipgloss.Style) []DetailRow {
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

	return rows
}
