package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"vaws/internal/ui/theme"
)

// Header renders the application header.
type Header struct {
	width   int
	title   string
	profile string
	region  string
	path    []string
}

// NewHeader creates a new Header component.
func NewHeader() *Header {
	return &Header{
		title: "vaws",
	}
}

// SetWidth sets the header width.
func (h *Header) SetWidth(width int) {
	h.width = width
}

// SetProfile sets the AWS profile display.
func (h *Header) SetProfile(profile string) {
	h.profile = profile
}

// SetRegion sets the AWS region display.
func (h *Header) SetRegion(region string) {
	h.region = region
}

// SetPath sets the breadcrumb path.
func (h *Header) SetPath(path []string) {
	h.path = path
}

// View renders the header.
func (h *Header) View() string {
	s := theme.DefaultStyles()

	breadcrumbSep := s.HeaderInfo.Render(" > ")

	// Left side: title + breadcrumbs
	var left strings.Builder
	left.WriteString(s.HeaderTitle.Render(h.title))

	if len(h.path) > 0 {
		for _, p := range h.path {
			left.WriteString(breadcrumbSep)
			left.WriteString(s.HeaderInfo.Render(p))
		}
	}

	// Right side: profile and region
	var right strings.Builder
	if h.profile != "" {
		profileStyle := lipgloss.NewStyle().
			Background(theme.PrimaryMuted).
			Foreground(theme.TextInverse).
			Padding(0, 1).
			Bold(true)
		right.WriteString(profileStyle.Render(h.profile))
	}
	if h.region != "" {
		if right.Len() > 0 {
			right.WriteString(" ")
		}
		right.WriteString(s.HeaderInfo.Render(h.region))
	}

	leftStr := left.String()
	rightStr := right.String()

	// Calculate padding for right alignment
	leftWidth := lipgloss.Width(leftStr)
	rightWidth := lipgloss.Width(rightStr)
	padding := h.width - leftWidth - rightWidth - 4

	if padding < 2 {
		padding = 2
	}

	content := leftStr + fmt.Sprintf("%*s", padding, "") + rightStr

	return s.Header.Width(h.width).Render(content)
}
