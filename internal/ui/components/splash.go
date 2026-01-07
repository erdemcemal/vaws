package components

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"vaws/internal/ui/theme"
)

// Logo lines - clean ASCII art
var logoLines = []string{
	" ██╗   ██╗ █████╗ ██╗    ██╗███████╗",
	" ██║   ██║██╔══██╗██║    ██║██╔════╝",
	" ██║   ██║███████║██║ █╗ ██║███████╗",
	" ╚██╗ ██╔╝██╔══██║██║███╗██║╚════██║",
	"  ╚████╔╝ ██║  ██║╚███╔███╔╝███████║",
	"   ╚═══╝  ╚═╝  ╚═╝ ╚══╝╚══╝ ╚══════╝",
}

const (
	animationFPS  = 30
	frameInterval = time.Second / animationFPS
	revealSpeed   = 3
	slideSpeed    = 2
	initialSlide  = 6
)

// SplashTickMsg is sent on each animation frame
type SplashTickMsg time.Time

// Splash renders the startup splash screen with animations.
type Splash struct {
	width   int
	height  int
	version string
	loading string
	spinner *Spinner

	frame       int
	revealed    int
	slideOffset int
	glowIndex   int
	ready       bool
}

// NewSplash creates a new Splash component.
func NewSplash(version string) *Splash {
	return &Splash{
		version:     version,
		loading:     "Initializing...",
		spinner:     NewSpinner(),
		frame:       0,
		revealed:    0,
		slideOffset: initialSlide,
		glowIndex:   0,
		ready:       false,
	}
}

// SetSize sets the splash dimensions.
func (s *Splash) SetSize(width, height int) {
	s.width = width
	s.height = height
}

// SetLoading sets the loading message.
func (s *Splash) SetLoading(msg string) {
	s.loading = msg
}

// Spinner returns the splash spinner.
func (s *Splash) Spinner() *Spinner {
	return s.spinner
}

// Tick advances the animation state.
func (s *Splash) Tick() {
	s.frame++

	if s.frame%4 == 0 {
		s.glowIndex = (s.glowIndex + 1) % 8
	}

	if s.revealed < len(logoLines) && s.frame%revealSpeed == 0 {
		s.revealed++
	}

	if s.slideOffset > 0 && s.frame%slideSpeed == 0 {
		s.slideOffset--
	}

	if s.revealed >= len(logoLines) && s.slideOffset == 0 {
		s.ready = true
	}
}

// TickCmd returns a command for animation timing.
func (s *Splash) TickCmd() tea.Cmd {
	return tea.Tick(frameInterval, func(t time.Time) tea.Msg {
		return SplashTickMsg(t)
	})
}

// IsReady returns true if the animation has completed.
func (s *Splash) IsReady() bool {
	return s.ready
}

// View renders the splash screen.
func (s *Splash) View() string {
	if s.width == 0 || s.height == 0 {
		return ""
	}

	st := theme.DefaultStyles()

	// Logo style using adaptive primary color
	logoStyle := lipgloss.NewStyle().
		Foreground(theme.Primary).
		Bold(true)

	taglineStyle := st.Muted.Copy().Italic(true)
	versionStyle := st.Muted
	loadingStyle := lipgloss.NewStyle().Foreground(theme.Text)
	hintStyle := st.Muted.Copy().Italic(true)

	// Decorative line
	decorLine := lipgloss.NewStyle().
		Foreground(theme.PrimaryMuted).
		Render("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	var lines []string

	if s.revealed > 0 {
		lines = append(lines, decorLine)
		lines = append(lines, "")
	}

	for i := 0; i < s.revealed && i < len(logoLines); i++ {
		lines = append(lines, logoStyle.Render(logoLines[i]))
	}

	for i := s.revealed; i < len(logoLines); i++ {
		lines = append(lines, strings.Repeat(" ", len(logoLines[0])))
	}

	if s.revealed > 0 {
		lines = append(lines, "")
		lines = append(lines, decorLine)
	}

	lines = append(lines, "")
	lines = append(lines, "")

	if s.revealed >= len(logoLines)-1 {
		lines = append(lines, taglineStyle.Render("AWS CloudFormation & ECS Explorer"))
	} else {
		lines = append(lines, "")
	}

	lines = append(lines, "")

	if s.version != "" && s.ready {
		lines = append(lines, versionStyle.Render(s.version))
	} else {
		lines = append(lines, "")
	}

	lines = append(lines, "")

	if s.loading != "" {
		lines = append(lines, s.spinner.View()+" "+loadingStyle.Render(s.loading))
	}

	lines = append(lines, "")

	if s.ready {
		lines = append(lines, hintStyle.Render("Press any key to continue"))
	} else {
		lines = append(lines, "")
	}

	// Center each line
	var centeredLines []string
	for _, line := range lines {
		lineWidth := lipgloss.Width(line)
		padding := (s.width - lineWidth) / 2
		if padding < 0 {
			padding = 0
		}
		centeredLines = append(centeredLines, strings.Repeat(" ", padding)+line)
	}

	content := strings.Join(centeredLines, "\n")

	// Vertical positioning with slide
	contentHeight := len(centeredLines)
	targetY := (s.height - contentHeight) / 2
	if targetY < 0 {
		targetY = 0
	}

	offsetPixels := s.slideOffset * 2
	currentY := targetY - offsetPixels
	if currentY < 0 {
		currentY = 0
	}

	var output strings.Builder
	for i := 0; i < currentY; i++ {
		output.WriteString("\n")
	}
	output.WriteString(content)

	outputLines := strings.Count(output.String(), "\n") + 1
	for i := outputLines; i < s.height; i++ {
		output.WriteString("\n")
	}

	return output.String()
}
