package ui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"

	"vaws/internal/state"
	"vaws/internal/ui/theme"
)

// getLayoutMode determines the appropriate layout based on terminal size.
func (m *Model) getLayoutMode() layoutMode {
	if m.width < minWidth || m.height < minHeight {
		return layoutTooSmall
	}
	if m.width < minWidthFull {
		return layoutSingle
	}
	return layoutFull
}

// shouldShowLogs returns whether logs can be shown at current height.
func (m *Model) shouldShowLogs() bool {
	return m.state.ShowLogs && m.height >= minHeightLogs
}

// View implements tea.Model.
func (m *Model) View() string {
	if !m.ready {
		return "Initializing..."
	}

	// Show profile selection screen
	if m.state.View == state.ViewProfileSelect {
		m.profileSelector.SetSize(m.width, m.height)
		if m.awaitingClientCreate {
			// Show loading indicator while creating client
			loadingStyle := lipgloss.NewStyle().
				Foreground(theme.Primary).
				Bold(true)
			return lipgloss.Place(
				m.width,
				m.height,
				lipgloss.Center,
				lipgloss.Center,
				loadingStyle.Render("Connecting to AWS..."),
			)
		}
		return m.profileSelector.View()
	}

	// Show region selection screen
	if m.state.View == state.ViewRegionSelect {
		m.regionSelector.SetSize(m.width, m.height)
		return m.regionSelector.View()
	}

	// Show splash screen
	if m.showSplash {
		return m.splash.View()
	}

	// Check layout mode
	layout := m.getLayoutMode()

	// Window too small - show message
	if layout == layoutTooSmall {
		return m.renderTooSmallScreen()
	}

	// Calculate dimensions
	// Status bar is 1 row, quick bar is 1 row
	statusBarHeight := 1
	quickBarHeight := 1
	currentLogsHeight := 0
	if m.shouldShowLogs() {
		currentLogsHeight = logsHeight
	}
	contentHeight := m.height - statusBarHeight - quickBarHeight - currentLogsHeight
	if contentHeight < 1 {
		contentHeight = 1
	}

	// Status bar (single row header)
	m.statusBar.SetWidth(m.width)
	m.statusBar.SetProfile(m.state.Profile)
	m.statusBar.SetRegion(m.state.Region)
	m.statusBar.SetActiveTunnels(len(m.tunnelManager.GetTunnels()))
	header := m.statusBar.View()

	// Update container with current context and size FIRST
	m.updateContainerContext()
	m.container.SetSize(m.width, contentHeight)

	// Use Container's content dimensions for inner components
	innerWidth := m.container.ContentWidth()
	innerHeight := m.container.ContentHeight()

	// Main content area
	var contentView string

	if m.state.View == state.ViewTunnels {
		// Tunnels view takes full width
		m.tunnelsPanel.SetSize(innerWidth, innerHeight)
		contentView = m.tunnelsPanel.View()
	} else {
		contentView = m.renderMainContent(layout, innerHeight)
	}

	// Logs panel (if visible and fits)
	var logsView string
	if m.shouldShowLogs() {
		m.logs.SetSize(m.width, currentLogsHeight)
		logsView = lipgloss.NewStyle().
			Width(m.width).
			Height(currentLogsHeight).
			MaxWidth(m.width).
			MaxHeight(currentLogsHeight).
			Render(m.logs.View())
	}

	// Port input dialog (if entering port)
	var portInputView string
	if m.enteringPort {
		portInputView = m.renderPortDialog()
	}

	// Payload input dialog (if entering payload for Lambda invoke)
	var payloadInputView string
	if m.enteringPayload {
		payloadInputView = m.renderPayloadDialog()
	}

	// QuickBar (footer with quick keys)
	m.quickBar.SetWidth(m.width)

	// Set context-specific actions based on current view
	m.updateQuickBarActions()

	if m.filtering {
		m.quickBar.SetMode("filter")
		m.quickBar.SetFilterText(m.filterInput.Value())
	} else if m.commandPalette.IsActive() {
		m.quickBar.SetMode("command")
	} else {
		m.quickBar.SetMode("")
	}
	footer := m.quickBar.View()

	// Combine all sections
	var sections []string
	sections = append(sections, header)

	if m.commandPalette.IsActive() {
		// Show command palette overlay inside container
		cmdPalette := m.commandPalette.View()
		m.container.SetContent(lipgloss.Place(m.container.ContentWidth(), m.container.ContentHeight(), lipgloss.Center, lipgloss.Center, cmdPalette))
		sections = append(sections, m.container.View())
	} else if m.enteringPort {
		// Center the port input dialog inside container
		m.container.SetContent(lipgloss.Place(m.container.ContentWidth(), m.container.ContentHeight(), lipgloss.Center, lipgloss.Center, portInputView))
		sections = append(sections, m.container.View())
	} else if m.enteringPayload {
		// Center the payload input dialog inside container
		m.container.SetContent(lipgloss.Place(m.container.ContentWidth(), m.container.ContentHeight(), lipgloss.Center, lipgloss.Center, payloadInputView))
		sections = append(sections, m.container.View())
	} else if m.dynamodbQueryDialog.IsActive() {
		// Center the DynamoDB query dialog inside container
		m.dynamodbQueryDialog.SetSize(m.container.ContentWidth(), m.container.ContentHeight())
		queryDialogView := m.dynamodbQueryDialog.View()
		m.container.SetContent(lipgloss.Place(m.container.ContentWidth(), m.container.ContentHeight(), lipgloss.Center, lipgloss.Center, queryDialogView))
		sections = append(sections, m.container.View())
	} else {
		// Set content inside container
		m.container.SetContent(contentView)
		sections = append(sections, m.container.View())
	}

	if m.shouldShowLogs() {
		sections = append(sections, logsView)
	}
	sections = append(sections, footer)

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// renderTooSmallScreen shows a message when the window is too small.
func (m *Model) renderTooSmallScreen() string {
	style := lipgloss.NewStyle().
		Foreground(theme.Warning).
		Bold(true)

	msg := style.Render("Window too small")
	hint := lipgloss.NewStyle().
		Foreground(theme.TextDim).
		Render(fmt.Sprintf("\nMin: %dx%d", minWidth, minHeight))

	content := msg + hint

	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		content,
	)
}

// renderMainContent renders the main content area based on layout mode.
func (m *Model) renderMainContent(layout layoutMode, contentHeight int) string {
	// Use container's content width for inner components
	containerWidth := m.container.ContentWidth()

	// CloudWatch logs view takes full screen
	if m.state.View == state.ViewCloudWatchLogs {
		m.cloudWatchLogsPanel.SetSize(containerWidth, contentHeight)
		return m.cloudWatchLogsPanel.View()
	}

	// DynamoDB query results view takes full screen
	if m.state.View == state.ViewDynamoDBQuery {
		m.dynamodbQueryResults.SetSize(containerWidth, contentHeight)
		return m.dynamodbQueryResults.View()
	}

	// Calculate sizes first
	var listWidth, detailsWidth int
	if layout == layoutSingle {
		listWidth = containerWidth
	} else {
		listWidth = int(float64(containerWidth) * listPaneRatio)
		detailsWidth = containerWidth - listWidth
	}

	// Set sizes on all lists BEFORE calling View()
	m.mainMenuList.SetSize(listWidth, contentHeight)
	m.stacksList.SetSize(listWidth, contentHeight)
	m.stackResourcesList.SetSize(listWidth, contentHeight)
	m.clustersList.SetSize(listWidth, contentHeight)
	m.serviceList.SetSize(listWidth, contentHeight)
	m.lambdaList.SetSize(listWidth, contentHeight)
	m.apiGatewayList.SetSize(listWidth, contentHeight)
	m.apiStagesList.SetSize(listWidth, contentHeight)
	m.ec2List.SetSize(listWidth, contentHeight)
	m.containerList.SetSize(listWidth, contentHeight)
	m.sqsTable.SetSize(listWidth, contentHeight)
	m.dynamodbTable.SetSize(listWidth, contentHeight)
	if layout != layoutSingle {
		m.details.SetSize(detailsWidth, contentHeight)
	}

	// Now render the list view with correct size
	var listView string
	switch m.state.View {
	case state.ViewMain:
		listView = m.mainMenuList.View()
	case state.ViewStacks:
		listView = m.stacksList.View()
	case state.ViewStackResources:
		listView = m.stackResourcesList.View()
	case state.ViewClusters:
		listView = m.clustersList.View()
	case state.ViewServices:
		listView = m.serviceList.View()
	case state.ViewLambda:
		listView = m.lambdaList.View()
	case state.ViewAPIGateway:
		listView = m.apiGatewayList.View()
	case state.ViewAPIStages:
		listView = m.apiStagesList.View()
	case state.ViewJumpHostSelect:
		listView = m.ec2List.View()
	case state.ViewContainerSelect:
		listView = m.containerList.View()
	case state.ViewSQS:
		listView = m.sqsTable.View()
	case state.ViewDynamoDB:
		listView = m.dynamodbTable.View()
	}

	// Filter input (shown above list when filtering)
	if m.filtering {
		filterStyle := lipgloss.NewStyle().
			Foreground(theme.Primary).
			Bold(true)
		filterLabel := filterStyle.Render("Filter: ")
		listView = filterLabel + m.filterInput.View() + "\n\n" + listView
	} else if m.state.FilterText != "" {
		filterStyle := lipgloss.NewStyle().
			Foreground(theme.TextDim)
		filterLabel := filterStyle.Render(fmt.Sprintf("Filtered: \"%s\"", m.state.FilterText))
		listView = filterLabel + "\n\n" + listView
	}

	// Single pane layout - list only, full width
	if layout == layoutSingle {
		return listView
	}

	// Full two-pane layout
	listPane := lipgloss.NewStyle().
		Width(listWidth).
		Height(contentHeight).
		MaxWidth(listWidth).
		MaxHeight(contentHeight).
		Render(listView)

	detailsPane := lipgloss.NewStyle().
		Width(detailsWidth).
		Height(contentHeight).
		MaxWidth(detailsWidth).
		MaxHeight(contentHeight).
		Render(m.details.View())

	return lipgloss.JoinHorizontal(lipgloss.Top, listPane, detailsPane)
}

// renderPortDialog renders the port input dialog.
func (m *Model) renderPortDialog() string {
	dialogWidth := 50
	if m.width < 60 {
		dialogWidth = m.width - 10
		if dialogWidth < 30 {
			dialogWidth = 30
		}
	}

	dialogStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.BorderFocus).
		Padding(1, 2).
		Width(dialogWidth)

	labelStyle := lipgloss.NewStyle().
		Foreground(theme.Primary).
		Bold(true)

	hintStyle := lipgloss.NewStyle().
		Foreground(theme.TextDim).
		Italic(true)

	serviceName := ""
	if m.pendingPortForward != nil {
		serviceName = truncateString(m.pendingPortForward.Name, dialogWidth-20)
	}

	dialogContent := labelStyle.Render("Port Forward: "+serviceName) + "\n\n" +
		"Local port: " + m.portInput.View() + "\n\n" +
		hintStyle.Render("Enter port or press Enter for random")

	return dialogStyle.Render(dialogContent)
}

// renderPayloadDialog renders the Lambda payload input dialog.
func (m *Model) renderPayloadDialog() string {
	dialogWidth := 70
	if m.width < 80 {
		dialogWidth = m.width - 10
		if dialogWidth < 40 {
			dialogWidth = 40
		}
	}

	dialogStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.BorderFocus).
		Padding(1, 2).
		Width(dialogWidth)

	labelStyle := lipgloss.NewStyle().
		Foreground(theme.Primary).
		Bold(true)

	hintStyle := lipgloss.NewStyle().
		Foreground(theme.TextDim).
		Italic(true)

	fnName := ""
	if m.pendingInvokeFunction != nil {
		fnName = truncateString(m.pendingInvokeFunction.Name, dialogWidth-20)
	}

	dialogContent := labelStyle.Render("Invoke Lambda: "+fnName) + "\n\n" +
		"Payload (JSON): " + m.payloadInput.View() + "\n\n" +
		hintStyle.Render("Enter JSON payload or press Enter for empty")

	return dialogStyle.Render(dialogContent)
}
