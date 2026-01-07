package components

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"vaws/internal/model"
	"vaws/internal/ui/theme"
)

// TunnelsPanel displays active port forwarding tunnels.
type TunnelsPanel struct {
	width        int
	height       int
	tunnels      []model.Tunnel
	apiGWTunnels []model.APIGatewayTunnel
	cursor       int
}

// NewTunnelsPanel creates a new TunnelsPanel.
func NewTunnelsPanel() *TunnelsPanel {
	return &TunnelsPanel{}
}

// SetSize sets the panel dimensions.
func (t *TunnelsPanel) SetSize(width, height int) {
	t.width = width
	t.height = height
}

// SetTunnels sets the ECS tunnel list.
func (t *TunnelsPanel) SetTunnels(tunnels []model.Tunnel) {
	t.tunnels = tunnels
	totalCount := len(tunnels) + len(t.apiGWTunnels)
	if t.cursor >= totalCount {
		t.cursor = max(0, totalCount-1)
	}
}

// SetAPIGatewayTunnels sets the API Gateway tunnel list.
func (t *TunnelsPanel) SetAPIGatewayTunnels(tunnels []model.APIGatewayTunnel) {
	t.apiGWTunnels = tunnels
	totalCount := len(t.tunnels) + len(tunnels)
	if t.cursor >= totalCount {
		t.cursor = max(0, totalCount-1)
	}
}

// Cursor returns the current cursor position.
func (t *TunnelsPanel) Cursor() int {
	return t.cursor
}

// SelectedTunnel returns the currently selected tunnel.
func (t *TunnelsPanel) SelectedTunnel() *model.Tunnel {
	if t.cursor >= 0 && t.cursor < len(t.tunnels) {
		return &t.tunnels[t.cursor]
	}
	return nil
}

// Up moves the cursor up.
func (t *TunnelsPanel) Up() {
	if t.cursor > 0 {
		t.cursor--
	}
}

// Down moves the cursor down.
func (t *TunnelsPanel) Down() {
	totalCount := len(t.tunnels) + len(t.apiGWTunnels)
	if t.cursor < totalCount-1 {
		t.cursor++
	}
}

// View renders the tunnels panel.
func (t *TunnelsPanel) View() string {
	s := theme.DefaultStyles()

	// Adaptive styles for tunnels
	tunnelActiveStyle := lipgloss.NewStyle().Foreground(theme.Success).Bold(true)
	tunnelStartingStyle := lipgloss.NewStyle().Foreground(theme.Warning)
	tunnelErrorStyle := lipgloss.NewStyle().Foreground(theme.Error)
	tunnelTerminatedStyle := lipgloss.NewStyle().Foreground(theme.TextDim)
	tunnelPortStyle := lipgloss.NewStyle().Foreground(theme.Info).Bold(true)
	tunnelServiceStyle := lipgloss.NewStyle().Foreground(theme.Text)
	tunnelHeaderStyle := lipgloss.NewStyle().Bold(true).Foreground(theme.Primary).PaddingBottom(1)
	tunnelContainerStyle := lipgloss.NewStyle().PaddingLeft(1).PaddingRight(1)
	tunnelSelectedStyle := lipgloss.NewStyle().Background(theme.BgHighlight).Bold(true)
	tunnelTypeStyle := lipgloss.NewStyle().Foreground(theme.Info)

	var b strings.Builder

	// Header with count
	activeCount := 0
	for _, tun := range t.tunnels {
		if tun.Status == model.TunnelStatusActive || tun.Status == model.TunnelStatusStarting {
			activeCount++
		}
	}
	for _, tun := range t.apiGWTunnels {
		if tun.Status == model.TunnelStatusActive || tun.Status == model.TunnelStatusStarting {
			activeCount++
		}
	}

	title := "Tunnels"
	if activeCount > 0 {
		title = fmt.Sprintf("Tunnels (%d active)", activeCount)
	}
	b.WriteString(tunnelHeaderStyle.Render(title))
	b.WriteString("\n")

	totalCount := len(t.tunnels) + len(t.apiGWTunnels)
	if totalCount == 0 {
		emptyMsg := s.Muted.Render("No active tunnels. Press 'p' on a service or API stage to start port forwarding.")
		b.WriteString(emptyMsg)
		return tunnelContainerStyle.Render(b.String())
	}

	itemIndex := 0

	// Render ECS tunnels
	for i, tun := range t.tunnels {
		isSelected := itemIndex == t.cursor
		itemIndex++

		// Status indicator
		var statusIcon string
		var statusStyle lipgloss.Style
		switch tun.Status {
		case model.TunnelStatusActive:
			statusIcon = "●"
			statusStyle = tunnelActiveStyle
		case model.TunnelStatusStarting:
			statusIcon = "◐"
			statusStyle = tunnelStartingStyle
		case model.TunnelStatusError:
			statusIcon = "✗"
			statusStyle = tunnelErrorStyle
		case model.TunnelStatusTerminated:
			statusIcon = "○"
			statusStyle = tunnelTerminatedStyle
		}

		// Build line
		var line strings.Builder

		// Cursor
		if isSelected {
			line.WriteString(s.SidebarCursor.Render("▸ "))
		} else {
			line.WriteString("  ")
		}

		// Status icon
		line.WriteString(statusStyle.Render(statusIcon))
		line.WriteString(" ")

		// Type indicator
		line.WriteString(tunnelTypeStyle.Render("[ECS] "))

		// Port info
		portInfo := tunnelPortStyle.Render(fmt.Sprintf("localhost:%d", tun.LocalPort))
		line.WriteString(portInfo)
		line.WriteString(" → ")
		line.WriteString(fmt.Sprintf(":%d", tun.RemotePort))
		line.WriteString("  ")

		// Service name
		line.WriteString(tunnelServiceStyle.Render(tun.ServiceName))

		// Duration
		if tun.Status == model.TunnelStatusActive {
			duration := time.Since(tun.StartedAt).Truncate(time.Second)
			line.WriteString(s.Muted.Render(fmt.Sprintf("  (%s)", duration)))
		}

		// Error message
		if tun.Status == model.TunnelStatusError && tun.Error != "" {
			errText := tun.Error
			if len(errText) > 30 {
				errText = errText[:27] + "..."
			}
			line.WriteString("\n    ")
			line.WriteString(tunnelErrorStyle.Render(errText))
		}

		lineStr := line.String()
		if isSelected {
			lineStr = tunnelSelectedStyle.Render(lineStr)
		}

		b.WriteString(lineStr)
		if i < len(t.tunnels)-1 || len(t.apiGWTunnels) > 0 {
			b.WriteString("\n")
		}
	}

	// Render API Gateway tunnels
	for i, tun := range t.apiGWTunnels {
		isSelected := itemIndex == t.cursor
		itemIndex++

		// Status indicator
		var statusIcon string
		var statusStyle lipgloss.Style
		switch tun.Status {
		case model.TunnelStatusActive:
			statusIcon = "●"
			statusStyle = tunnelActiveStyle
		case model.TunnelStatusStarting:
			statusIcon = "◐"
			statusStyle = tunnelStartingStyle
		case model.TunnelStatusError:
			statusIcon = "✗"
			statusStyle = tunnelErrorStyle
		case model.TunnelStatusTerminated:
			statusIcon = "○"
			statusStyle = tunnelTerminatedStyle
		}

		// Build line
		var line strings.Builder

		// Cursor
		if isSelected {
			line.WriteString(s.SidebarCursor.Render("▸ "))
		} else {
			line.WriteString("  ")
		}

		// Status icon
		line.WriteString(statusStyle.Render(statusIcon))
		line.WriteString(" ")

		// Type indicator
		tunnelTypeLabel := "[APIGW]"
		if tun.TunnelType == model.APIGatewayTunnelPrivate {
			tunnelTypeLabel = "[APIGW-VPC]"
		}
		line.WriteString(tunnelTypeStyle.Render(tunnelTypeLabel + " "))

		// Port info
		portInfo := tunnelPortStyle.Render(fmt.Sprintf("localhost:%d", tun.LocalPort))
		line.WriteString(portInfo)
		line.WriteString(" → ")
		line.WriteString(tunnelServiceStyle.Render(fmt.Sprintf("%s/%s", tun.APIName, tun.StageName)))

		// Duration
		if tun.Status == model.TunnelStatusActive {
			duration := time.Since(tun.StartedAt).Truncate(time.Second)
			line.WriteString(s.Muted.Render(fmt.Sprintf("  (%s)", duration)))
		}

		// Error message
		if tun.Status == model.TunnelStatusError && tun.Error != "" {
			errText := tun.Error
			if len(errText) > 30 {
				errText = errText[:27] + "..."
			}
			line.WriteString("\n    ")
			line.WriteString(tunnelErrorStyle.Render(errText))
		}

		lineStr := line.String()
		if isSelected {
			lineStr = tunnelSelectedStyle.Render(lineStr)
		}

		b.WriteString(lineStr)
		if i < len(t.apiGWTunnels)-1 {
			b.WriteString("\n")
		}
	}

	return tunnelContainerStyle.Render(b.String())
}
