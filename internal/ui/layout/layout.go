// Package layout provides responsive layout calculations for the TUI.
package layout

// Dimensions holds the calculated dimensions for each UI region.
type Dimensions struct {
	// Total terminal size
	TermWidth  int
	TermHeight int

	// Header
	HeaderHeight int
	HeaderWidth  int

	// Status bar
	StatusHeight int
	StatusWidth  int

	// Content area (between header and status)
	ContentHeight int
	ContentWidth  int

	// Sidebar (left pane)
	SidebarWidth  int
	SidebarHeight int

	// Main pane (right pane)
	MainWidth  int
	MainHeight int

	// Layout mode
	Mode LayoutMode
}

// LayoutMode determines how content is arranged.
type LayoutMode int

const (
	// ModeTooSmall - terminal too small to display anything useful
	ModeTooSmall LayoutMode = iota
	// ModeSinglePane - only show one pane (sidebar or main)
	ModeSinglePane
	// ModeSplitPane - show both sidebar and main pane
	ModeSplitPane
)

// Constraints define minimum sizes and ratios.
type Constraints struct {
	MinWidth        int     // Minimum usable terminal width
	MinHeight       int     // Minimum usable terminal height
	MinSplitWidth   int     // Minimum width to show split pane
	SidebarRatio    float64 // Sidebar width as ratio of total (0.0-1.0)
	MinSidebarWidth int     // Minimum sidebar width in split mode
	MaxSidebarWidth int     // Maximum sidebar width in split mode
	HeaderHeight    int     // Fixed header height
	StatusHeight    int     // Fixed status bar height
}

// DefaultConstraints returns sensible default constraints.
func DefaultConstraints() Constraints {
	return Constraints{
		MinWidth:        30,
		MinHeight:       10,
		MinSplitWidth:   80,
		SidebarRatio:    0.35,
		MinSidebarWidth: 25,
		MaxSidebarWidth: 50,
		HeaderHeight:    1,
		StatusHeight:    1,
	}
}

// Calculate computes layout dimensions based on terminal size and constraints.
func Calculate(width, height int, c Constraints) Dimensions {
	d := Dimensions{
		TermWidth:    width,
		TermHeight:   height,
		HeaderHeight: c.HeaderHeight,
		HeaderWidth:  width,
		StatusHeight: c.StatusHeight,
		StatusWidth:  width,
	}

	// Check if terminal is too small
	if width < c.MinWidth || height < c.MinHeight {
		d.Mode = ModeTooSmall
		return d
	}

	// Calculate content area (between header and status)
	d.ContentHeight = height - c.HeaderHeight - c.StatusHeight
	if d.ContentHeight < 1 {
		d.ContentHeight = 1
	}
	d.ContentWidth = width

	// Determine layout mode based on width
	if width < c.MinSplitWidth {
		d.Mode = ModeSinglePane
		d.SidebarWidth = width - 2 // Account for borders
		d.SidebarHeight = d.ContentHeight
		d.MainWidth = 0
		d.MainHeight = 0
	} else {
		d.Mode = ModeSplitPane

		// Calculate sidebar width
		sidebarWidth := int(float64(width) * c.SidebarRatio)
		if sidebarWidth < c.MinSidebarWidth {
			sidebarWidth = c.MinSidebarWidth
		}
		if sidebarWidth > c.MaxSidebarWidth {
			sidebarWidth = c.MaxSidebarWidth
		}

		d.SidebarWidth = sidebarWidth
		d.SidebarHeight = d.ContentHeight
		d.MainWidth = width - sidebarWidth
		d.MainHeight = d.ContentHeight
	}

	return d
}

// CalculateWithDefaults is a convenience function using default constraints.
func CalculateWithDefaults(width, height int) Dimensions {
	return Calculate(width, height, DefaultConstraints())
}
