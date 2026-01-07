// Package theme provides adaptive theming for light and dark terminals.
package theme

import "github.com/charmbracelet/lipgloss"

// Adaptive colors that work on both light and dark backgrounds.
// Format: AdaptiveColor{Light: "color for light bg", Dark: "color for dark bg"}
var (
	// Primary brand colors
	Primary      = lipgloss.AdaptiveColor{Light: "#5B21B6", Dark: "#A78BFA"}
	PrimaryBold  = lipgloss.AdaptiveColor{Light: "#4C1D95", Dark: "#7C3AED"}
	PrimaryMuted = lipgloss.AdaptiveColor{Light: "#7C3AED", Dark: "#6D28D9"}

	// Text colors
	Text        = lipgloss.AdaptiveColor{Light: "#1F2937", Dark: "#F3F4F6"}
	TextMuted   = lipgloss.AdaptiveColor{Light: "#4B5563", Dark: "#9CA3AF"}
	TextDim     = lipgloss.AdaptiveColor{Light: "#6B7280", Dark: "#6B7280"}
	TextInverse = lipgloss.AdaptiveColor{Light: "#F9FAFB", Dark: "#111827"}

	// Background colors
	BgSubtle    = lipgloss.AdaptiveColor{Light: "#F3F4F6", Dark: "#1F2937"}
	BgMuted     = lipgloss.AdaptiveColor{Light: "#E5E7EB", Dark: "#374151"}
	BgHighlight = lipgloss.AdaptiveColor{Light: "#DDD6FE", Dark: "#4C1D95"}

	// Status colors
	Success = lipgloss.AdaptiveColor{Light: "#059669", Dark: "#10B981"}
	Warning = lipgloss.AdaptiveColor{Light: "#D97706", Dark: "#F59E0B"}
	Error   = lipgloss.AdaptiveColor{Light: "#DC2626", Dark: "#EF4444"}
	Info    = lipgloss.AdaptiveColor{Light: "#2563EB", Dark: "#3B82F6"}

	// Border colors
	Border      = lipgloss.AdaptiveColor{Light: "#D1D5DB", Dark: "#374151"}
	BorderFocus = lipgloss.AdaptiveColor{Light: "#7C3AED", Dark: "#A78BFA"}
)

// Styles provides all application styles using adaptive colors.
type Styles struct {
	// App layout
	App lipgloss.Style

	// Header
	Header      lipgloss.Style
	HeaderTitle lipgloss.Style
	HeaderInfo  lipgloss.Style

	// Sidebar
	Sidebar         lipgloss.Style
	SidebarTitle    lipgloss.Style
	SidebarItem     lipgloss.Style
	SidebarSelected lipgloss.Style
	SidebarCursor   lipgloss.Style

	// Main content
	Content      lipgloss.Style
	ContentTitle lipgloss.Style
	ContentBody  lipgloss.Style

	// Details
	DetailLabel lipgloss.Style
	DetailValue lipgloss.Style

	// Status bar
	StatusBar     lipgloss.Style
	StatusKey     lipgloss.Style
	StatusValue   lipgloss.Style
	StatusDivider lipgloss.Style

	// Status indicators
	StatusSuccess lipgloss.Style
	StatusWarning lipgloss.Style
	StatusError   lipgloss.Style
	StatusInfo    lipgloss.Style

	// Input
	Input      lipgloss.Style
	InputLabel lipgloss.Style

	// Misc
	Spinner lipgloss.Style
	Muted   lipgloss.Style
	Bold    lipgloss.Style
}

// DefaultStyles returns the default adaptive styles.
func DefaultStyles() Styles {
	return Styles{
		App: lipgloss.NewStyle(),

		// Header - primary color background
		Header: lipgloss.NewStyle().
			Background(PrimaryBold).
			Foreground(TextInverse).
			Padding(0, 2).
			Bold(true),
		HeaderTitle: lipgloss.NewStyle().
			Foreground(TextInverse).
			Bold(true),
		HeaderInfo: lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#DDD6FE", Dark: "#C4B5FD"}),

		// Sidebar
		Sidebar: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Border).
			Padding(1, 1),
		SidebarTitle: lipgloss.NewStyle().
			Foreground(Primary).
			Bold(true).
			MarginBottom(1),
		SidebarItem: lipgloss.NewStyle().
			Foreground(Text).
			PaddingLeft(2),
		SidebarSelected: lipgloss.NewStyle().
			Foreground(TextInverse).
			Background(Primary).
			Bold(true).
			PaddingLeft(1).
			PaddingRight(1),
		SidebarCursor: lipgloss.NewStyle().
			Foreground(Primary).
			Bold(true),

		// Content area
		Content: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Border).
			Padding(1, 2),
		ContentTitle: lipgloss.NewStyle().
			Foreground(Primary).
			Bold(true).
			MarginBottom(1),
		ContentBody: lipgloss.NewStyle().
			Foreground(Text),

		// Details
		DetailLabel: lipgloss.NewStyle().
			Foreground(TextMuted).
			Width(16),
		DetailValue: lipgloss.NewStyle().
			Foreground(Text),

		// Status bar
		StatusBar: lipgloss.NewStyle().
			Background(BgSubtle).
			Foreground(TextMuted).
			Padding(0, 2),
		StatusKey: lipgloss.NewStyle().
			Foreground(Primary).
			Bold(true),
		StatusValue: lipgloss.NewStyle().
			Foreground(TextDim),
		StatusDivider: lipgloss.NewStyle().
			Foreground(Border),

		// Status indicators
		StatusSuccess: lipgloss.NewStyle().Foreground(Success),
		StatusWarning: lipgloss.NewStyle().Foreground(Warning),
		StatusError:   lipgloss.NewStyle().Foreground(Error),
		StatusInfo:    lipgloss.NewStyle().Foreground(Info),

		// Input
		Input: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(BorderFocus).
			Padding(0, 1),
		InputLabel: lipgloss.NewStyle().
			Foreground(Primary).
			Bold(true),

		// Misc
		Spinner: lipgloss.NewStyle().Foreground(Primary),
		Muted:   lipgloss.NewStyle().Foreground(TextMuted),
		Bold:    lipgloss.NewStyle().Bold(true).Foreground(Text),
	}
}
