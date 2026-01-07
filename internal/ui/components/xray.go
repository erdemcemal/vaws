package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"vaws/internal/ui/theme"
)

// XRayNodeType represents the type of node in the tree
type XRayNodeType int

const (
	NodeTypeStack XRayNodeType = iota
	NodeTypeCluster
	NodeTypeService
	NodeTypeTask
	NodeTypeContainer
)

// XRayNode represents a node in the XRay tree
type XRayNode struct {
	ID       string
	Name     string
	Type     XRayNodeType
	Status   string
	Details  string
	Children []*XRayNode
	Expanded bool
	Parent   *XRayNode
}

// XRayTree displays a hierarchical view of AWS resources
type XRayTree struct {
	root     *XRayNode
	cursor   int
	width    int
	height   int
	visible  []*XRayNode // Flattened visible nodes for navigation
	selected *XRayNode
}

// NewXRayTree creates a new XRay tree view
func NewXRayTree() *XRayTree {
	return &XRayTree{
		root:    nil,
		cursor:  0,
		visible: []*XRayNode{},
	}
}

// SetSize sets the tree dimensions
func (x *XRayTree) SetSize(width, height int) {
	x.width = width
	x.height = height
}

// SetRoot sets the root node
func (x *XRayTree) SetRoot(root *XRayNode) {
	x.root = root
	x.rebuildVisible()
}

// rebuildVisible rebuilds the flattened list of visible nodes
func (x *XRayTree) rebuildVisible() {
	x.visible = []*XRayNode{}
	if x.root != nil {
		x.addVisibleNodes(x.root, 0)
	}
	if x.cursor >= len(x.visible) {
		x.cursor = max(0, len(x.visible)-1)
	}
	if len(x.visible) > 0 {
		x.selected = x.visible[x.cursor]
	}
}

func (x *XRayTree) addVisibleNodes(node *XRayNode, depth int) {
	x.visible = append(x.visible, node)
	if node.Expanded {
		for _, child := range node.Children {
			x.addVisibleNodes(child, depth+1)
		}
	}
}

// Up moves cursor up
func (x *XRayTree) Up() {
	if x.cursor > 0 {
		x.cursor--
		x.selected = x.visible[x.cursor]
	}
}

// Down moves cursor down
func (x *XRayTree) Down() {
	if x.cursor < len(x.visible)-1 {
		x.cursor++
		x.selected = x.visible[x.cursor]
	}
}

// Toggle expands or collapses the current node
func (x *XRayTree) Toggle() {
	if x.selected != nil && len(x.selected.Children) > 0 {
		x.selected.Expanded = !x.selected.Expanded
		x.rebuildVisible()
	}
}

// Expand expands the current node
func (x *XRayTree) Expand() {
	if x.selected != nil && len(x.selected.Children) > 0 {
		x.selected.Expanded = true
		x.rebuildVisible()
	}
}

// Collapse collapses the current node
func (x *XRayTree) Collapse() {
	if x.selected != nil {
		if x.selected.Expanded && len(x.selected.Children) > 0 {
			x.selected.Expanded = false
			x.rebuildVisible()
		} else if x.selected.Parent != nil {
			// Go to parent
			for i, node := range x.visible {
				if node == x.selected.Parent {
					x.cursor = i
					x.selected = node
					break
				}
			}
		}
	}
}

// SelectedNode returns the currently selected node
func (x *XRayTree) SelectedNode() *XRayNode {
	return x.selected
}

// getDepth returns the depth of a node
func (x *XRayTree) getDepth(node *XRayNode) int {
	depth := 0
	current := node
	for current.Parent != nil {
		depth++
		current = current.Parent
	}
	return depth
}

// View renders the XRay tree
func (x *XRayTree) View() string {
	if x.root == nil || len(x.visible) == 0 {
		s := theme.DefaultStyles()
		return s.Muted.Render("No resources loaded. Press 'r' to refresh.")
	}

	titleStyle := lipgloss.NewStyle().
		Foreground(theme.Primary).
		Bold(true).
		MarginBottom(1)

	var b strings.Builder
	b.WriteString(titleStyle.Render("⎈ XRay View"))
	b.WriteString("\n\n")

	// Calculate visible range
	maxVisible := x.height - 6
	if maxVisible < 5 {
		maxVisible = 5
	}

	startIdx := 0
	if x.cursor >= maxVisible {
		startIdx = x.cursor - maxVisible + 1
	}

	endIdx := startIdx + maxVisible
	if endIdx > len(x.visible) {
		endIdx = len(x.visible)
	}

	for i := startIdx; i < endIdx; i++ {
		node := x.visible[i]
		isSelected := i == x.cursor
		depth := x.getDepth(node)

		line := x.renderNode(node, depth, isSelected)
		b.WriteString(line)
		if i < endIdx-1 {
			b.WriteString("\n")
		}
	}

	// Scroll indicator
	if len(x.visible) > maxVisible {
		b.WriteString("\n")
		scrollStyle := lipgloss.NewStyle().Foreground(theme.TextDim)
		b.WriteString(scrollStyle.Render(fmt.Sprintf("  %d/%d", x.cursor+1, len(x.visible))))
	}

	containerStyle := lipgloss.NewStyle().
		Padding(1, 2).
		Width(x.width)

	return containerStyle.Render(b.String())
}

func (x *XRayTree) renderNode(node *XRayNode, depth int, isSelected bool) string {
	s := theme.DefaultStyles()

	// Indentation
	indent := strings.Repeat("  ", depth)

	// Tree connector
	var connector string
	if len(node.Children) > 0 {
		if node.Expanded {
			connector = "▼ "
		} else {
			connector = "▶ "
		}
	} else {
		connector = "  "
	}

	// Icon based on type
	icon := x.getIcon(node.Type)

	// Status style
	statusStyle := x.getStatusStyle(node.Status)

	// Build the line
	var line strings.Builder

	// Cursor indicator
	if isSelected {
		line.WriteString(s.SidebarCursor.Render("▸ "))
	} else {
		line.WriteString("  ")
	}

	line.WriteString(indent)

	// Connector with appropriate style
	connectorStyle := lipgloss.NewStyle().Foreground(theme.TextDim)
	line.WriteString(connectorStyle.Render(connector))

	// Icon
	iconStyle := x.getIconStyle(node.Type)
	line.WriteString(iconStyle.Render(icon))
	line.WriteString(" ")

	// Name
	nameStyle := lipgloss.NewStyle().Foreground(theme.Text)
	if isSelected {
		nameStyle = s.SidebarSelected
	}
	line.WriteString(nameStyle.Render(node.Name))

	// Status badge
	if node.Status != "" {
		line.WriteString(" ")
		line.WriteString(statusStyle.Render("[" + node.Status + "]"))
	}

	// Details
	if node.Details != "" {
		line.WriteString(" ")
		detailStyle := lipgloss.NewStyle().Foreground(theme.TextDim)
		line.WriteString(detailStyle.Render(node.Details))
	}

	return line.String()
}

func (x *XRayTree) getIcon(nodeType XRayNodeType) string {
	switch nodeType {
	case NodeTypeStack:
		return "📦"
	case NodeTypeCluster:
		return "🔷"
	case NodeTypeService:
		return "⚙"
	case NodeTypeTask:
		return "📋"
	case NodeTypeContainer:
		return "🐳"
	default:
		return "○"
	}
}

func (x *XRayTree) getIconStyle(nodeType XRayNodeType) lipgloss.Style {
	switch nodeType {
	case NodeTypeStack:
		return lipgloss.NewStyle().Foreground(theme.Warning)
	case NodeTypeCluster:
		return lipgloss.NewStyle().Foreground(theme.Info)
	case NodeTypeService:
		return lipgloss.NewStyle().Foreground(theme.Primary)
	case NodeTypeTask:
		return lipgloss.NewStyle().Foreground(theme.Success)
	case NodeTypeContainer:
		return lipgloss.NewStyle().Foreground(theme.Info)
	default:
		return lipgloss.NewStyle().Foreground(theme.TextMuted)
	}
}

func (x *XRayTree) getStatusStyle(status string) lipgloss.Style {
	statusLower := strings.ToLower(status)
	switch {
	case strings.Contains(statusLower, "running") ||
		strings.Contains(statusLower, "active") ||
		strings.Contains(statusLower, "complete"):
		return lipgloss.NewStyle().Foreground(theme.Success)
	case strings.Contains(statusLower, "pending") ||
		strings.Contains(statusLower, "progress") ||
		strings.Contains(statusLower, "starting"):
		return lipgloss.NewStyle().Foreground(theme.Warning)
	case strings.Contains(statusLower, "failed") ||
		strings.Contains(statusLower, "error") ||
		strings.Contains(statusLower, "stopped"):
		return lipgloss.NewStyle().Foreground(theme.Error)
	default:
		return lipgloss.NewStyle().Foreground(theme.TextMuted)
	}
}

// BuildTreeFromStack creates an XRay tree from a stack and its services
func BuildTreeFromStack(stackName string, stackStatus string, services []ServiceInfo) *XRayNode {
	root := &XRayNode{
		ID:       stackName,
		Name:     stackName,
		Type:     NodeTypeStack,
		Status:   stackStatus,
		Expanded: true,
		Children: []*XRayNode{},
	}

	// Group services by cluster
	clusterMap := make(map[string]*XRayNode)

	for _, svc := range services {
		// Get or create cluster node
		clusterNode, exists := clusterMap[svc.ClusterName]
		if !exists {
			clusterNode = &XRayNode{
				ID:       svc.ClusterARN,
				Name:     svc.ClusterName,
				Type:     NodeTypeCluster,
				Status:   "ACTIVE",
				Expanded: true,
				Parent:   root,
				Children: []*XRayNode{},
			}
			clusterMap[svc.ClusterName] = clusterNode
			root.Children = append(root.Children, clusterNode)
		}

		// Create service node
		serviceNode := &XRayNode{
			ID:       svc.ServiceARN,
			Name:     svc.ServiceName,
			Type:     NodeTypeService,
			Status:   svc.Status,
			Details:  fmt.Sprintf("%d/%d", svc.RunningCount, svc.DesiredCount),
			Expanded: false,
			Parent:   clusterNode,
			Children: []*XRayNode{},
		}

		// Add task nodes
		for _, task := range svc.Tasks {
			taskNode := &XRayNode{
				ID:       task.TaskID,
				Name:     task.TaskID[:min(12, len(task.TaskID))],
				Type:     NodeTypeTask,
				Status:   task.Status,
				Expanded: false,
				Parent:   serviceNode,
				Children: []*XRayNode{},
			}

			// Add container nodes
			for _, container := range task.Containers {
				containerNode := &XRayNode{
					ID:       container.Name,
					Name:     container.Name,
					Type:     NodeTypeContainer,
					Status:   container.Status,
					Details:  container.Image,
					Parent:   taskNode,
					Children: []*XRayNode{},
				}
				taskNode.Children = append(taskNode.Children, containerNode)
			}

			serviceNode.Children = append(serviceNode.Children, taskNode)
		}

		clusterNode.Children = append(clusterNode.Children, serviceNode)
	}

	return root
}

// ServiceInfo holds service information for building the tree
type ServiceInfo struct {
	ServiceName  string
	ServiceARN   string
	ClusterName  string
	ClusterARN   string
	Status       string
	RunningCount int
	DesiredCount int
	Tasks        []TaskInfo
}

// TaskInfo holds task information for building the tree
type TaskInfo struct {
	TaskID     string
	Status     string
	Containers []ContainerInfo
}

// ContainerInfo holds container information for building the tree
type ContainerInfo struct {
	Name   string
	Status string
	Image  string
}
