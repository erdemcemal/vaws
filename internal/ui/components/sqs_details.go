package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"vaws/internal/model"
	"vaws/internal/ui/theme"
)

// SQSDetails displays detailed information about an SQS queue in a bordered container.
type SQSDetails struct {
	width  int
	height int
	queue  *model.Queue
}

// NewSQSDetails creates a new SQSDetails component.
func NewSQSDetails() *SQSDetails {
	return &SQSDetails{}
}

// SetSize sets the component dimensions.
func (d *SQSDetails) SetSize(width, height int) {
	d.width = width
	d.height = height
}

// SetQueue sets the queue to display.
func (d *SQSDetails) SetQueue(queue *model.Queue) {
	d.queue = queue
}

// View renders the SQS details view.
func (d *SQSDetails) View() string {
	if d.queue == nil {
		return d.renderEmpty()
	}

	return d.renderDetails()
}

func (d *SQSDetails) renderEmpty() string {
	style := lipgloss.NewStyle().
		Width(d.width).
		Height(d.height).
		Align(lipgloss.Center, lipgloss.Center)

	emptyStyle := lipgloss.NewStyle().Foreground(theme.TextDim)
	return style.Render(emptyStyle.Render("No queue selected"))
}

func (d *SQSDetails) renderDetails() string {
	q := d.queue

	// Container width - centered box
	containerWidth := 70
	if d.width < 80 {
		containerWidth = d.width - 10
	}
	if containerWidth < 50 {
		containerWidth = 50
	}

	// Styles
	titleStyle := lipgloss.NewStyle().
		Foreground(theme.Primary).
		Bold(true)

	labelStyle := lipgloss.NewStyle().
		Foreground(theme.TextDim).
		Width(18)

	valueStyle := lipgloss.NewStyle().
		Foreground(theme.Text)

	sectionStyle := lipgloss.NewStyle().
		Foreground(theme.Primary).
		Bold(true)

	warningStyle := lipgloss.NewStyle().
		Foreground(theme.Warning).
		Bold(true)

	// Build content
	var content strings.Builder

	// Title
	content.WriteString(titleStyle.Render(q.Name))
	content.WriteString("\n\n")

	// Type badge
	queueType := "Standard"
	if q.Type == model.QueueTypeFIFO {
		queueType = "FIFO"
	}
	content.WriteString(labelStyle.Render("Type:"))
	content.WriteString(valueStyle.Render(queueType))
	content.WriteString("\n\n")

	// Message Statistics
	content.WriteString(sectionStyle.Render("Message Statistics"))
	content.WriteString("\n")
	content.WriteString(labelStyle.Render("Messages:"))
	content.WriteString(valueStyle.Render(fmt.Sprintf("%d", q.ApproximateMessageCount)))
	content.WriteString("\n")
	content.WriteString(labelStyle.Render("In Flight:"))
	content.WriteString(valueStyle.Render(fmt.Sprintf("%d", q.ApproximateInFlight)))
	content.WriteString("\n\n")

	// Configuration
	content.WriteString(sectionStyle.Render("Configuration"))
	content.WriteString("\n")
	content.WriteString(labelStyle.Render("Visibility:"))
	content.WriteString(valueStyle.Render(fmt.Sprintf("%ds", q.VisibilityTimeout)))
	content.WriteString("\n")
	content.WriteString(labelStyle.Render("Retention:"))
	content.WriteString(valueStyle.Render(formatRetention(q.MessageRetentionPeriod)))
	content.WriteString("\n")
	content.WriteString(labelStyle.Render("Delay:"))
	content.WriteString(valueStyle.Render(fmt.Sprintf("%ds", q.DelaySeconds)))
	content.WriteString("\n")
	if !q.CreatedAt.IsZero() {
		content.WriteString(labelStyle.Render("Created:"))
		content.WriteString(valueStyle.Render(q.CreatedAt.Format("2006-01-02")))
		content.WriteString("\n")
	}
	content.WriteString("\n")

	// DLQ Info
	if q.HasDLQ {
		content.WriteString(sectionStyle.Render("Dead Letter Queue"))
		content.WriteString("\n")
		content.WriteString(labelStyle.Render("DLQ Name:"))
		content.WriteString(valueStyle.Render(q.DLQName))
		content.WriteString("\n")
		content.WriteString(labelStyle.Render("DLQ Messages:"))
		if q.DLQMessageCount > 0 {
			content.WriteString(warningStyle.Render(fmt.Sprintf("%d", q.DLQMessageCount)))
		} else {
			content.WriteString(valueStyle.Render("0"))
		}
		content.WriteString("\n")
		content.WriteString(labelStyle.Render("Max Receives:"))
		content.WriteString(valueStyle.Render(fmt.Sprintf("%d", q.MaxReceiveCount)))
		content.WriteString("\n\n")
	}

	// URLs
	content.WriteString(sectionStyle.Render("Identifiers"))
	content.WriteString("\n")

	// Truncate URL if too long
	url := q.URL
	maxURLLen := containerWidth - 22
	if len(url) > maxURLLen {
		url = url[:maxURLLen-3] + "..."
	}
	content.WriteString(labelStyle.Render("URL:"))
	content.WriteString(valueStyle.Render(url))
	content.WriteString("\n")

	// Truncate ARN if too long
	arn := q.ARN
	if len(arn) > maxURLLen {
		arn = arn[:maxURLLen-3] + "..."
	}
	content.WriteString(labelStyle.Render("ARN:"))
	content.WriteString(valueStyle.Render(arn))

	// Create bordered container
	containerStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Border).
		Padding(1, 2).
		Width(containerWidth)

	box := containerStyle.Render(content.String())

	// Center the box in the available space
	centered := lipgloss.Place(
		d.width,
		d.height,
		lipgloss.Center,
		lipgloss.Center,
		box,
	)

	return centered
}

func formatRetention(seconds int) string {
	if seconds >= 86400 {
		days := seconds / 86400
		return fmt.Sprintf("%d days", days)
	} else if seconds >= 3600 {
		hours := seconds / 3600
		return fmt.Sprintf("%d hours", hours)
	} else if seconds >= 60 {
		mins := seconds / 60
		return fmt.Sprintf("%d minutes", mins)
	}
	return fmt.Sprintf("%d seconds", seconds)
}
