package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"vaws/internal/ui/theme"
)

// ProfileSelector allows users to select an AWS profile.
type ProfileSelector struct {
	profiles []string
	cursor   int
	width    int
	height   int
}

// NewProfileSelector creates a new ProfileSelector.
func NewProfileSelector() *ProfileSelector {
	return &ProfileSelector{
		profiles: []string{},
		cursor:   0,
	}
}

// SetProfiles sets the available profiles.
func (p *ProfileSelector) SetProfiles(profiles []string) {
	p.profiles = profiles
	if p.cursor >= len(profiles) {
		p.cursor = 0
	}
}

// SetSize sets the component dimensions.
func (p *ProfileSelector) SetSize(width, height int) {
	p.width = width
	p.height = height
}

// Up moves cursor up.
func (p *ProfileSelector) Up() {
	if p.cursor > 0 {
		p.cursor--
	}
}

// Down moves cursor down.
func (p *ProfileSelector) Down() {
	if p.cursor < len(p.profiles)-1 {
		p.cursor++
	}
}

// SelectedProfile returns the currently selected profile.
func (p *ProfileSelector) SelectedProfile() string {
	if p.cursor >= 0 && p.cursor < len(p.profiles) {
		return p.profiles[p.cursor]
	}
	return ""
}

// View renders the profile selector.
func (p *ProfileSelector) View() string {
	s := theme.DefaultStyles()

	if len(p.profiles) == 0 {
		return s.Muted.Render("No AWS profiles found. Configure AWS CLI first.")
	}

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Primary).
		Padding(1, 2)

	var b strings.Builder

	// Title
	titleStyle := lipgloss.NewStyle().
		Foreground(theme.Primary).
		Bold(true).
		MarginBottom(1)
	b.WriteString(titleStyle.Render("Select AWS Profile"))
	b.WriteString("\n\n")

	// Calculate visible items
	maxVisible := p.height - 10
	if maxVisible < 5 {
		maxVisible = 5
	}
	if maxVisible > len(p.profiles) {
		maxVisible = len(p.profiles)
	}

	// Calculate scroll offset
	offset := 0
	if p.cursor >= maxVisible {
		offset = p.cursor - maxVisible + 1
	}

	end := offset + maxVisible
	if end > len(p.profiles) {
		end = len(p.profiles)
	}

	// Render profile list
	for i := offset; i < end; i++ {
		profile := p.profiles[i]
		isSelected := i == p.cursor

		var line string
		if isSelected {
			line = s.SidebarCursor.Render("▸ ") + s.SidebarSelected.Render(profile)
		} else {
			line = "  " + s.SidebarItem.Render(profile)
		}

		b.WriteString(line)
		if i < end-1 {
			b.WriteString("\n")
		}
	}

	// Scroll indicator
	if len(p.profiles) > maxVisible {
		b.WriteString("\n")
		scrollText := fmt.Sprintf("  ↑↓ %d of %d profiles", p.cursor+1, len(p.profiles))
		b.WriteString(s.Muted.Render(scrollText))
	}

	// Hint
	b.WriteString("\n\n")
	b.WriteString(s.Muted.Render("Press Enter to select, q to quit"))

	content := boxStyle.Render(b.String())

	return lipgloss.Place(
		p.width,
		p.height,
		lipgloss.Center,
		lipgloss.Center,
		content,
	)
}
