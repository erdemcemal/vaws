package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"vaws/internal/ui/theme"
)

// ResourceType represents the type of AWS resource being viewed
type ResourceType string

const (
	ResourceStacks         ResourceType = "Stacks"
	ResourceStackResources ResourceType = "Resources"
	ResourceClusters       ResourceType = "Clusters"
	ResourceServices       ResourceType = "Services"
	ResourceTasks          ResourceType = "Tasks"
	ResourceTunnels        ResourceType = "Tunnels"
	ResourceEC2            ResourceType = "EC2"
	ResourceS3             ResourceType = "S3"
	ResourceLambda         ResourceType = "Lambda"
	ResourceRDS            ResourceType = "RDS"
	ResourceAPIGateway     ResourceType = "API GW"
	ResourceAPIStages      ResourceType = "Stages"
)

// RegionShortcut represents a quick region switch shortcut
type RegionShortcut struct {
	Key    string
	Region string
}

// DefaultRegionShortcuts returns the default region shortcuts
func DefaultRegionShortcuts() []RegionShortcut {
	return []RegionShortcut{
		{Key: "0", Region: "us-east-1"},
		{Key: "1", Region: "us-west-2"},
		{Key: "2", Region: "eu-west-1"},
		{Key: "3", Region: "eu-central-1"},
		{Key: "4", Region: "ap-northeast-1"},
	}
}

// MenuBar renders the k9s-style top menu bar with multiple rows
type MenuBar struct {
	width            int
	height           int
	profile          string
	region           string
	resource         ResourceType
	resourceContext  string // e.g., cluster name when viewing services
	resourceCount    int    // number of items in current view
	refreshIndicator *RefreshIndicator
	version          string
	activeTunnels    int
	keyBindings      []KeyBinding
	contextBindings  []KeyBinding     // context-specific bindings
	regionShortcuts  []RegionShortcut // configurable region shortcuts
}

// NewMenuBar creates a new menu bar
func NewMenuBar() *MenuBar {
	return &MenuBar{
		resource:         ResourceStacks,
		refreshIndicator: NewRefreshIndicator(),
		version:          "dev",
		height:           6, // Default height for multi-row layout
		regionShortcuts:  DefaultRegionShortcuts(),
		keyBindings: []KeyBinding{
			{Key: "<?>", Desc: "Help"},
			{Key: "</>", Desc: "Filter"},
			{Key: "<:>", Desc: "Command"},
			{Key: "<q>", Desc: "Quit"},
		},
	}
}

// SetWidth sets the menu bar width
func (m *MenuBar) SetWidth(width int) {
	m.width = width
}

// SetHeight sets the menu bar height
func (m *MenuBar) SetHeight(height int) {
	m.height = height
}

// SetProfile sets the AWS profile
func (m *MenuBar) SetProfile(profile string) {
	m.profile = profile
}

// SetRegion sets the AWS region
func (m *MenuBar) SetRegion(region string) {
	m.region = region
}

// SetResource sets the current resource type
func (m *MenuBar) SetResource(resource ResourceType) {
	m.resource = resource
}

// SetResourceContext sets additional context (e.g., stack name, cluster name)
func (m *MenuBar) SetResourceContext(context string) {
	m.resourceContext = context
}

// SetResourceCount sets the number of items in current view
func (m *MenuBar) SetResourceCount(count int) {
	m.resourceCount = count
}

// SetVersion sets the application version
func (m *MenuBar) SetVersion(version string) {
	m.version = version
}

// SetActiveTunnels sets the number of active tunnels
func (m *MenuBar) SetActiveTunnels(count int) {
	m.activeTunnels = count
}

// SetKeyBindings sets the available key bindings for display
func (m *MenuBar) SetKeyBindings(bindings []KeyBinding) {
	m.keyBindings = bindings
}

// SetContextBindings sets context-specific key bindings
func (m *MenuBar) SetContextBindings(bindings []KeyBinding) {
	m.contextBindings = bindings
}

// SetRegionShortcuts sets the region shortcuts for quick switching
func (m *MenuBar) SetRegionShortcuts(shortcuts []RegionShortcut) {
	m.regionShortcuts = shortcuts
}

// GetRegionShortcuts returns the current region shortcuts
func (m *MenuBar) GetRegionShortcuts() []RegionShortcut {
	return m.regionShortcuts
}

// RefreshIndicator returns the refresh indicator for external updates
func (m *MenuBar) RefreshIndicator() *RefreshIndicator {
	return m.refreshIndicator
}

// View renders the menu bar
func (m *MenuBar) View() string {
	if m.width < 60 {
		return m.renderCompact()
	}
	return m.renderFull()
}

func (m *MenuBar) renderFull() string {
	// Header takes up 6 rows like k9s/taws
	// Row 1-5: Context info | Key bindings | Logo
	// Row 6: Current resource indicator

	// Calculate column widths (5 columns like taws)
	contextWidth := m.width * 22 / 100
	shortcutsWidth := m.width * 18 / 100
	bindings1Width := m.width * 22 / 100
	bindings2Width := m.width * 22 / 100
	logoWidth := m.width - contextWidth - shortcutsWidth - bindings1Width - bindings2Width

	// Render each column
	contextCol := m.renderContextColumn(contextWidth)
	shortcutsCol := m.renderShortcutsColumn(shortcutsWidth)
	bindings1Col := m.renderBindingsColumn1(bindings1Width)
	bindings2Col := m.renderBindingsColumn2(bindings2Width)
	logoCol := m.renderLogoColumn(logoWidth)

	// Combine columns horizontally
	header := lipgloss.JoinHorizontal(lipgloss.Top,
		contextCol,
		shortcutsCol,
		bindings1Col,
		bindings2Col,
		logoCol,
	)

	// Add resource indicator bar at bottom
	resourceBar := m.renderResourceBar()

	// Combine header and resource bar
	return lipgloss.JoinVertical(lipgloss.Left, header, resourceBar)
}

func (m *MenuBar) renderContextColumn(width int) string {
	style := lipgloss.NewStyle().Width(width).Padding(0, 1)

	labelStyle := lipgloss.NewStyle().Foreground(theme.TextDim)
	valueStyle := lipgloss.NewStyle().Foreground(theme.Primary).Bold(true)
	regionStyle := lipgloss.NewStyle().Foreground(theme.Info).Bold(true)
	resourceStyle := lipgloss.NewStyle().Foreground(theme.Warning).Bold(true)

	var lines []string

	// Profile
	profileLine := labelStyle.Render("Profile:") + " " + valueStyle.Render(m.profile)
	lines = append(lines, profileLine)

	// Region
	regionLine := labelStyle.Render("Region:") + "  " + regionStyle.Render(m.region)
	lines = append(lines, regionLine)

	// Resource
	resourceName := string(m.resource)
	if m.resourceCount > 0 {
		resourceName = fmt.Sprintf("%s (%d)", m.resource, m.resourceCount)
	}
	resourceLine := labelStyle.Render("Resource:") + " " + resourceStyle.Render(resourceName)
	lines = append(lines, resourceLine)

	// Context (parent navigation)
	if m.resourceContext != "" {
		contextStyle := lipgloss.NewStyle().Foreground(theme.Warning)
		contextLine := labelStyle.Render("Context:") + " " + contextStyle.Render(m.resourceContext)
		lines = append(lines, contextLine)
	}

	// Active tunnels indicator
	if m.activeTunnels > 0 {
		tunnelStyle := lipgloss.NewStyle().Foreground(theme.Success).Bold(true)
		tunnelLine := labelStyle.Render("Tunnels:") + " " + tunnelStyle.Render(fmt.Sprintf("%d active", m.activeTunnels))
		lines = append(lines, tunnelLine)
	}

	// Pad to consistent height
	for len(lines) < 5 {
		lines = append(lines, "")
	}

	return style.Render(strings.Join(lines, "\n"))
}

func (m *MenuBar) renderShortcutsColumn(width int) string {
	style := lipgloss.NewStyle().Width(width).Padding(0, 1)

	keyStyle := lipgloss.NewStyle().Foreground(theme.Warning)
	descStyle := lipgloss.NewStyle().Foreground(theme.Text)
	currentStyle := lipgloss.NewStyle().Foreground(theme.Success).Bold(true)

	var lines []string
	for _, r := range m.regionShortcuts {
		ds := descStyle
		if r.Region == m.region {
			ds = currentStyle
		}
		line := keyStyle.Render(fmt.Sprintf("<%s>", r.Key)) + " " + ds.Render(r.Region)
		lines = append(lines, line)
	}

	// Pad to consistent height
	for len(lines) < 5 {
		lines = append(lines, "")
	}

	return style.Render(strings.Join(lines, "\n"))
}

func (m *MenuBar) renderBindingsColumn1(width int) string {
	style := lipgloss.NewStyle().Width(width).Padding(0, 1)

	keyStyle := lipgloss.NewStyle().Foreground(theme.Warning).Width(9)
	descStyle := lipgloss.NewStyle().Foreground(theme.TextDim)

	// Context-specific bindings or default bindings
	bindings := m.contextBindings
	if len(bindings) == 0 {
		bindings = []KeyBinding{
			{Key: "<enter>", Desc: "Select"},
			{Key: "<d>", Desc: "Details"},
			{Key: "<p>", Desc: "Port Fwd"},
			{Key: "<t>", Desc: "Tunnels"},
			{Key: "<r>", Desc: "Refresh"},
		}
	}

	var lines []string
	for i, b := range bindings {
		if i >= 5 {
			break
		}
		line := keyStyle.Render(b.Key) + descStyle.Render(b.Desc)
		lines = append(lines, line)
	}

	// Pad to consistent height
	for len(lines) < 5 {
		lines = append(lines, "")
	}

	return style.Render(strings.Join(lines, "\n"))
}

func (m *MenuBar) renderBindingsColumn2(width int) string {
	style := lipgloss.NewStyle().Width(width).Padding(0, 1)

	keyStyle := lipgloss.NewStyle().Foreground(theme.Warning).Width(9)
	descStyle := lipgloss.NewStyle().Foreground(theme.TextDim)

	bindings := []KeyBinding{
		{Key: "</>", Desc: "Filter"},
		{Key: "<:>", Desc: "Command"},
		{Key: "<esc>", Desc: "Back"},
		{Key: "<ctrl-c>", Desc: "Quit"},
	}

	var lines []string
	for _, b := range bindings {
		line := keyStyle.Render(b.Key) + descStyle.Render(b.Desc)
		lines = append(lines, line)
	}

	return style.Render(strings.Join(lines, "\n"))
}

func (m *MenuBar) renderLogoColumn(width int) string {
	style := lipgloss.NewStyle().Width(width).Padding(0, 1).Align(lipgloss.Right)

	logoStyle := lipgloss.NewStyle().Foreground(theme.Primary).Bold(true)
	dimStyle := lipgloss.NewStyle().Foreground(theme.TextDim)

	// ASCII art logo
	logo := []string{
		logoStyle.Render("█ █ ▄▀█ █ █ █ █▀"),
		logoStyle.Render("▀▄▀ █▀█ ▀▄▀▄▀ ▄█"),
		"",
		dimStyle.Render("AWS TUI"),
		dimStyle.Render(m.version),
	}

	return style.Render(strings.Join(logo, "\n"))
}

func (m *MenuBar) renderResourceBar() string {
	barStyle := lipgloss.NewStyle().
		Background(theme.BgSubtle).
		Width(m.width).
		Padding(0, 1)

	// Resource indicator with icon
	icon := GetResourceIcon(m.resource)
	resourceStyle := lipgloss.NewStyle().
		Foreground(GetResourceColor(m.resource)).
		Bold(true)

	resourceText := icon + " " + resourceStyle.Render(string(m.resource))

	// Add context in brackets if available
	if m.resourceContext != "" {
		contextStyle := lipgloss.NewStyle().Foreground(theme.TextDim)
		resourceText += contextStyle.Render(" [" + m.resourceContext + "]")
	}

	// Add count
	if m.resourceCount > 0 {
		countStyle := lipgloss.NewStyle().Foreground(theme.TextMuted)
		resourceText += countStyle.Render(fmt.Sprintf(" (%d items)", m.resourceCount))
	}

	// Refresh status on the right
	refreshStatus := m.refreshIndicator.StatusView()
	refreshStyle := lipgloss.NewStyle().Foreground(theme.TextMuted)

	// Calculate padding
	leftWidth := lipgloss.Width(resourceText)
	rightWidth := lipgloss.Width(refreshStatus)
	padding := m.width - leftWidth - rightWidth - 4
	if padding < 1 {
		padding = 1
	}

	content := resourceText + strings.Repeat(" ", padding) + refreshStyle.Render(refreshStatus)

	return barStyle.Render(content)
}

func (m *MenuBar) renderCompact() string {
	// Compact view for narrow terminals (single row)
	barStyle := lipgloss.NewStyle().
		Background(theme.BgSubtle).
		Width(m.width).
		Padding(0, 1)

	logoStyle := lipgloss.NewStyle().
		Foreground(theme.Primary).
		Bold(true)

	profileStyle := lipgloss.NewStyle().
		Background(theme.Primary).
		Foreground(theme.TextInverse).
		Padding(0, 1)

	resourceStyle := lipgloss.NewStyle().
		Foreground(theme.Warning).
		Bold(true)

	content := logoStyle.Render("⎈") + " "
	if m.profile != "" {
		content += profileStyle.Render(m.profile) + " "
	}
	content += resourceStyle.Render(string(m.resource))

	if m.resourceContext != "" {
		dimStyle := lipgloss.NewStyle().Foreground(theme.TextDim)
		content += dimStyle.Render(" › " + m.resourceContext)
	}

	return barStyle.Render(content)
}

// Crumb represents a breadcrumb item
type Crumb struct {
	Label    string
	Resource ResourceType
}

// MenuBarWithCrumbs extends MenuBar with breadcrumb navigation
type MenuBarWithCrumbs struct {
	*MenuBar
	crumbs []Crumb
}

// NewMenuBarWithCrumbs creates a menu bar with breadcrumb support
func NewMenuBarWithCrumbs() *MenuBarWithCrumbs {
	return &MenuBarWithCrumbs{
		MenuBar: NewMenuBar(),
		crumbs:  []Crumb{},
	}
}

// SetCrumbs sets the breadcrumb trail
func (m *MenuBarWithCrumbs) SetCrumbs(crumbs []Crumb) {
	m.crumbs = crumbs
	if len(crumbs) > 0 {
		// Set resource from last crumb
		m.resource = crumbs[len(crumbs)-1].Resource
	}
}

// AddCrumb adds a breadcrumb
func (m *MenuBarWithCrumbs) AddCrumb(label string, resource ResourceType) {
	m.crumbs = append(m.crumbs, Crumb{Label: label, Resource: resource})
	m.resource = resource
}

// PopCrumb removes the last breadcrumb
func (m *MenuBarWithCrumbs) PopCrumb() {
	if len(m.crumbs) > 0 {
		m.crumbs = m.crumbs[:len(m.crumbs)-1]
		if len(m.crumbs) > 0 {
			m.resource = m.crumbs[len(m.crumbs)-1].Resource
		}
	}
}

// ClearCrumbs clears all breadcrumbs
func (m *MenuBarWithCrumbs) ClearCrumbs() {
	m.crumbs = []Crumb{}
}

// View renders the menu bar with breadcrumbs
func (m *MenuBarWithCrumbs) View() string {
	// Build context from breadcrumbs
	if len(m.crumbs) > 1 {
		// Show parent as context
		m.resourceContext = m.crumbs[len(m.crumbs)-2].Label
	}
	return m.MenuBar.View()
}

// GetResourceIcon returns an icon for the resource type
func GetResourceIcon(resource ResourceType) string {
	switch resource {
	case ResourceStacks:
		return "📦"
	case ResourceStackResources:
		return "📂"
	case ResourceClusters:
		return "🔷"
	case ResourceServices:
		return "⚙️"
	case ResourceTasks:
		return "📋"
	case ResourceTunnels:
		return "🔗"
	case ResourceEC2:
		return "🖥️"
	case ResourceS3:
		return "🪣"
	case ResourceLambda:
		return "λ"
	case ResourceRDS:
		return "🗄️"
	case ResourceAPIGateway:
		return "🌐"
	case ResourceAPIStages:
		return "🚀"
	default:
		return "○"
	}
}

// GetResourceColor returns the theme color for a resource type
func GetResourceColor(resource ResourceType) lipgloss.AdaptiveColor {
	switch resource {
	case ResourceStacks:
		return theme.Warning
	case ResourceStackResources:
		return theme.Info
	case ResourceClusters:
		return theme.Info
	case ResourceServices:
		return theme.Primary
	case ResourceTasks:
		return theme.Success
	case ResourceTunnels:
		return theme.Info
	case ResourceEC2:
		return theme.Warning
	case ResourceS3:
		return theme.Success
	case ResourceLambda:
		return theme.Warning
	case ResourceRDS:
		return theme.Info
	case ResourceAPIGateway:
		return theme.Primary
	case ResourceAPIStages:
		return theme.Success
	default:
		return theme.TextMuted
	}
}

// ResourceInfo holds information about a resource type
type ResourceInfo struct {
	Type        ResourceType
	Icon        string
	Command     string
	Aliases     []string
	Description string
}

// AvailableResources lists all supported resource types
var AvailableResources = []ResourceInfo{
	{Type: ResourceStacks, Icon: "📦", Command: "stacks", Aliases: []string{"st", "cfn"}, Description: "CloudFormation Stacks"},
	{Type: ResourceClusters, Icon: "🔷", Command: "clusters", Aliases: []string{"cl"}, Description: "ECS Clusters"},
	{Type: ResourceServices, Icon: "⚙️", Command: "services", Aliases: []string{"svc"}, Description: "ECS Services"},
	{Type: ResourceTasks, Icon: "📋", Command: "tasks", Aliases: []string{"task"}, Description: "ECS Tasks"},
	{Type: ResourceTunnels, Icon: "🔗", Command: "tunnels", Aliases: []string{"tun", "pf"}, Description: "Port Forward Tunnels"},
	{Type: ResourceLambda, Icon: "λ", Command: "lambda", Aliases: []string{"fn", "functions"}, Description: "Lambda Functions"},
	{Type: ResourceAPIGateway, Icon: "🌐", Command: "apigateway", Aliases: []string{"apigw", "api"}, Description: "API Gateway"},
	{Type: ResourceEC2, Icon: "🖥️", Command: "ec2", Aliases: []string{"instances"}, Description: "EC2 Instances"},
	{Type: ResourceS3, Icon: "🪣", Command: "s3", Aliases: []string{"buckets"}, Description: "S3 Buckets"},
	{Type: ResourceRDS, Icon: "🗄️", Command: "rds", Aliases: []string{"databases", "db"}, Description: "RDS Databases"},
}

// FormatResourceCount formats a count for display in the menu
func FormatResourceCount(count int) string {
	if count == 0 {
		return ""
	}
	return fmt.Sprintf("(%d)", count)
}

// GetDefaultBindingsForResource returns the default key bindings for a resource type
func GetDefaultBindingsForResource(resource ResourceType) []KeyBinding {
	switch resource {
	case ResourceStacks:
		return []KeyBinding{
			{Key: "<enter>", Desc: "Resources"},
			{Key: "<d>", Desc: "Details"},
			{Key: "<r>", Desc: "Refresh"},
			{Key: "<x>", Desc: "X-Ray"},
		}
	case ResourceStackResources:
		return []KeyBinding{
			{Key: "<enter>", Desc: "Select"},
			{Key: "<esc>", Desc: "Back"},
			{Key: "<d>", Desc: "Details"},
		}
	case ResourceServices:
		return []KeyBinding{
			{Key: "<enter>", Desc: "Tasks"},
			{Key: "<p>", Desc: "Port Fwd"},
			{Key: "<d>", Desc: "Details"},
			{Key: "<r>", Desc: "Refresh"},
		}
	case ResourceTunnels:
		return []KeyBinding{
			{Key: "<x>", Desc: "Stop"},
			{Key: "<r>", Desc: "Restart"},
			{Key: "<c>", Desc: "Clear"},
			{Key: "<d>", Desc: "Details"},
		}
	case ResourceLambda:
		return []KeyBinding{
			{Key: "<d>", Desc: "Details"},
			{Key: "<r>", Desc: "Refresh"},
			{Key: "<l>", Desc: "Logs"},
		}
	case ResourceAPIGateway:
		return []KeyBinding{
			{Key: "<enter>", Desc: "Stages"},
			{Key: "<d>", Desc: "Details"},
			{Key: "<r>", Desc: "Refresh"},
		}
	case ResourceAPIStages:
		return []KeyBinding{
			{Key: "<d>", Desc: "Details"},
			{Key: "<r>", Desc: "Refresh"},
			{Key: "<esc>", Desc: "Back"},
		}
	default:
		return []KeyBinding{
			{Key: "<enter>", Desc: "Select"},
			{Key: "<d>", Desc: "Details"},
			{Key: "<r>", Desc: "Refresh"},
		}
	}
}
