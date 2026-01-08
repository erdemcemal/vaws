package ui

import (
	"context"
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"golang.design/x/clipboard"

	"vaws/internal/aws"
	"vaws/internal/model"
	"vaws/internal/state"
)

// handleKeyMsg handles key messages when not in special input modes.
func (m *Model) handleKeyMsg(msg tea.KeyMsg) tea.Cmd {
	// Handle filter mode separately
	if m.filtering {
		return m.handleFilterKey(msg)
	}

	// Handle details search mode separately
	if m.detailsSearching {
		return m.handleDetailsSearchKey(msg)
	}

	// Handle port input mode separately
	if m.enteringPort {
		return m.handlePortInputKey(msg)
	}

	// Handle payload input mode separately
	if m.enteringPayload {
		return m.handlePayloadInputKey(msg)
	}

	// Handle DynamoDB query dialog
	if m.dynamodbQueryDialog.IsActive() {
		return m.handleDynamoDBQueryDialogKey(msg)
	}

	// Handle copy mode - allow scroll keys and y/esc to exit
	if m.copyMode {
		switch msg.String() {
		case "y", "esc":
			m.copyMode = false
			m.copyModeScroll = 0
			// Re-enable mouse capture when exiting copy mode
			return tea.EnableMouseCellMotion
		case "ctrl+c":
			m.tunnelManager.StopAllTunnels()
			return tea.Quit
		case "j", "down":
			m.copyModeScroll++
			return nil
		case "k", "up":
			if m.copyModeScroll > 0 {
				m.copyModeScroll--
			}
			return nil
		case "ctrl+d":
			m.copyModeScroll += 10
			return nil
		case "ctrl+u":
			if m.copyModeScroll >= 10 {
				m.copyModeScroll -= 10
			} else {
				m.copyModeScroll = 0
			}
			return nil
		case "g":
			m.copyModeScroll = 0
			return nil
		case "G":
			m.copyModeScroll = 9999 // Will be clamped in view
			return nil
		}
		return nil // Ignore other keys in copy mode
	}

	// Handle DynamoDB query results navigation
	if m.state.View == state.ViewDynamoDBQuery {
		return m.handleDynamoDBQueryResultsKey(msg)
	}

	switch {
	case matchKey(msg, m.keys.Quit):
		m.tunnelManager.StopAllTunnels()
		return tea.Quit

	case msg.String() == "q":
		// Query DynamoDB table
		if m.state.View == state.ViewDynamoDB {
			return m.handleDynamoDBQuery()
		}

	case matchKey(msg, m.keys.Up):
		if m.details.IsFocused() {
			m.details.ScrollUp()
		} else {
			m.moveCursorUp()
		}

	case matchKey(msg, m.keys.Down):
		if m.details.IsFocused() {
			m.details.ScrollDown()
		} else {
			m.moveCursorDown()
		}

	case matchKey(msg, m.keys.Top):
		if m.details.IsFocused() {
			m.details.ScrollToTop()
		} else {
			m.moveCursorTop()
		}

	case matchKey(msg, m.keys.Bottom):
		if m.details.IsFocused() {
			m.details.ScrollToBottom()
		} else {
			m.moveCursorBottom()
		}

	case msg.String() == "ctrl+d":
		// Half page down (vim-like)
		if m.details.IsFocused() {
			m.details.ScrollHalfPageDown()
		}

	case msg.String() == "ctrl+u":
		// Half page up (vim-like)
		if m.details.IsFocused() {
			m.details.ScrollHalfPageUp()
		}

	case msg.String() == "ctrl+f":
		// Full page down (vim-like)
		if m.details.IsFocused() {
			m.details.ScrollPageDown()
		}

	case msg.String() == "ctrl+b":
		// Full page up (vim-like)
		if m.details.IsFocused() {
			m.details.ScrollPageUp()
		}

	case msg.String() == "pgdown":
		// Page down
		if m.details.IsFocused() {
			m.details.ScrollPageDown()
		}

	case msg.String() == "pgup":
		// Page up
		if m.details.IsFocused() {
			m.details.ScrollPageUp()
		}

	case matchKey(msg, m.keys.Enter), matchKey(msg, m.keys.Right):
		return m.handleEnter()

	case matchKey(msg, m.keys.Back), matchKey(msg, m.keys.Left):
		m.handleBack()

	case matchKey(msg, m.keys.Filter):
		if m.state.View != state.ViewTunnels {
			// Start details search when details is focused, otherwise list filter
			if m.details.IsFocused() {
				m.startDetailsSearch()
			} else {
				m.startFiltering()
			}
		}

	case matchKey(msg, m.keys.Logs):
		m.state.ToggleLogs()
		m.updateComponentSizes()

	case matchKey(msg, m.keys.CloudWatchLogs):
		return m.handleCloudWatchLogs()

	case matchKey(msg, m.keys.PortForward):
		return m.handlePortForward()

	case matchKey(msg, m.keys.LambdaInvoke):
		return m.handleLambdaInvoke()

	case msg.String() == "s":
		// Scan DynamoDB table
		if m.state.View == state.ViewDynamoDB {
			return m.handleDynamoDBScan()
		}

	case matchKey(msg, m.keys.Tunnels):
		m.showTunnelsView()

	case matchKey(msg, m.keys.StopTunnel):
		return m.handleStopTunnel()

	case matchKey(msg, m.keys.RestartTunnel):
		// In tunnels view, 'r' restarts a tunnel
		if m.state.View == state.ViewTunnels {
			return m.handleRestartTunnel()
		}
		// In other views, 'r' is refresh (handled by Refresh binding)
		return m.handleRefresh()

	case matchKey(msg, m.keys.ClearTunnels):
		// Clear terminated tunnels when in tunnels view
		if m.state.View == state.ViewTunnels {
			m.tunnelManager.ClearTerminated()
			if m.apiGWManager != nil {
				m.apiGWManager.ClearTerminated()
			}
			m.updateTunnelsPanel()
			m.logger.Info("Cleared terminated tunnels")
		}

	case msg.String() == ":":
		// Open command palette (k9s-style)
		m.commandPalette.SetWidth(m.width)
		return m.commandPalette.Activate()

	case matchKey(msg, m.keys.Help):
		// Show help
		m.showHelp()

	case msg.String() == "a":
		// Toggle auto-refresh
		m.state.ToggleAutoRefresh()
		m.refreshIndicator.SetEnabled(m.state.AutoRefresh)
		if m.state.AutoRefresh {
			m.logger.Info("Auto-refresh enabled")
		} else {
			m.logger.Info("Auto-refresh disabled")
		}

	case matchKey(msg, m.keys.LogScrollUp):
		// Scroll logs up (back in history)
		if m.state.ShowLogs {
			m.logs.ScrollUp()
		}

	case matchKey(msg, m.keys.LogScrollDown):
		// Scroll logs down (toward newest)
		if m.state.ShowLogs {
			m.logs.ScrollDown()
		}

	case matchKey(msg, m.keys.LogScrollEnd):
		// Scroll logs to newest
		if m.state.ShowLogs {
			m.logs.ScrollToBottom()
		}

	case msg.String() == "tab":
		// Switch to next container in CloudWatch logs view
		if m.state.View == state.ViewCloudWatchLogs {
			m.cloudWatchLogsPanel.SelectNextTab()
			m.state.CloudWatchLastFetchTime = 0
			m.state.CloudWatchLogs = nil
			m.cloudWatchLogsPanel.Clear()
			return m.fetchCloudWatchLogs()
		}
		// Toggle focus between list and details in split view
		if m.getLayoutMode() == layoutFull && m.state.View != state.ViewTunnels &&
			m.state.View != state.ViewDynamoDBQuery {
			m.details.SetFocused(!m.details.IsFocused())
		}

	case msg.String() == "shift+tab":
		// Switch to previous container in CloudWatch logs view
		if m.state.View == state.ViewCloudWatchLogs {
			m.cloudWatchLogsPanel.SelectPrevTab()
			m.state.CloudWatchLastFetchTime = 0
			m.state.CloudWatchLogs = nil
			m.cloudWatchLogsPanel.Clear()
			return m.fetchCloudWatchLogs()
		}

	case matchKey(msg, m.keys.CopyMode):
		// Enter copy mode in full layout (split view)
		if m.getLayoutMode() == layoutFull {
			m.copyMode = true
			m.copyModeScroll = 0
			m.logger.Info("Copy mode enabled - select text with mouse, press y or Esc to exit")
			// Disable mouse capture to allow terminal text selection
			return tea.DisableMouse
		}

	case matchKey(msg, m.keys.YankClipboard):
		// Yank details to system clipboard
		if m.getLayoutMode() == layoutFull {
			text := m.details.PlainTextView()
			if text == "" {
				m.logger.Warn("No details to copy")
				return nil
			}
			err := clipboard.Init()
			if err != nil {
				m.logger.Warn("Clipboard not available - use copy mode (y) instead: " + err.Error())
				return nil
			}
			clipboard.Write(clipboard.FmtText, []byte(text))
			m.logger.Info("Details copied to clipboard")
		}

	// Quick resource switching with number keys
	case msg.String() == "0":
		return m.switchToMain()
	case msg.String() == "1":
		return m.switchToECS()
	case msg.String() == "2":
		return m.switchToLambda()
	case msg.String() == "3":
		return m.switchToSQS()
	case msg.String() == "4":
		return m.switchToDynamoDB()
	case msg.String() == "5":
		return m.switchToAPIGateway()
	case msg.String() == "6":
		return m.switchToStacks()

	case msg.String() == "n":
		// Next search match in details (when details focused and has search)
		if m.details.IsFocused() && m.details.MatchCount() > 0 {
			m.details.NextMatch()
		}

	case msg.String() == "N":
		// Previous search match in details (when details focused and has search)
		if m.details.IsFocused() && m.details.MatchCount() > 0 {
			m.details.PrevMatch()
		}
	}

	return nil
}

// handleFilterKey handles key messages when in filter mode.
func (m *Model) handleFilterKey(msg tea.KeyMsg) tea.Cmd {
	switch {
	case matchKey(msg, m.keys.FilterAccept):
		m.state.FilterText = m.filterInput.Value()
		m.filtering = false
		m.filterInput.Blur()
		m.updateCurrentList()
		return nil

	case matchKey(msg, m.keys.FilterClear):
		m.filterInput.SetValue("")
		m.state.FilterText = ""
		m.filtering = false
		m.filterInput.Blur()
		m.updateCurrentList()
		return nil
	}

	return nil
}

// handleDetailsSearchKey handles key messages when in details search mode.
func (m *Model) handleDetailsSearchKey(msg tea.KeyMsg) tea.Cmd {
	switch {
	case matchKey(msg, m.keys.FilterAccept):
		// Accept search and exit search input mode (keep matches highlighted)
		m.detailsSearching = false
		m.detailsSearchInput.Blur()
		return nil

	case matchKey(msg, m.keys.FilterClear):
		// Clear search and exit
		m.detailsSearchInput.SetValue("")
		m.details.ClearSearch()
		m.detailsSearching = false
		m.detailsSearchInput.Blur()
		return nil
	}

	// Handle text input
	var cmd tea.Cmd
	m.detailsSearchInput, cmd = m.detailsSearchInput.Update(msg)
	// Update search query in details component
	m.details.SetSearchQuery(m.detailsSearchInput.Value())
	return cmd
}

// handleProfileSelectKey handles key messages in profile selection view.
func (m *Model) handleProfileSelectKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit

	case "up", "k":
		m.profileSelector.Up()

	case "down", "j":
		m.profileSelector.Down()

	case "enter":
		// Select the profile and create AWS client
		selectedProfile := m.profileSelector.SelectedProfile()
		if selectedProfile == "" {
			return m, nil
		}

		m.logger.Info("Selected profile: %s", selectedProfile)
		m.awaitingClientCreate = true

		// Create AWS client asynchronously
		return m, func() tea.Msg {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			client, err := aws.NewClient(ctx, selectedProfile, m.pendingRegion)
			return clientCreatedMsg{client: client, err: err}
		}
	}

	return m, nil
}

// handleRegionSelectKey handles key messages in region selection view.
func (m *Model) handleRegionSelectKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// Cancel region selection, go back to previous view
		m.state.View = m.viewBeforeRegionSelect
		return m, nil

	case "up", "k":
		m.regionSelector.Up()

	case "down", "j":
		m.regionSelector.Down()

	case "enter":
		// Select the region and create new AWS client
		selectedRegion := m.regionSelector.SelectedRegion()
		if selectedRegion == "" || selectedRegion == m.state.Region {
			// No change or no selection - return to previous view
			m.state.View = m.viewBeforeRegionSelect
			return m, nil
		}

		m.logger.Info("Changing region to: %s", selectedRegion)

		// Create new AWS client with new region
		return m, func() tea.Msg {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			client, err := aws.NewClient(ctx, m.state.Profile, selectedRegion)
			return regionChangedMsg{client: client, region: selectedRegion, err: err}
		}
	}

	return m, nil
}

// handleEnter handles the enter key press based on current view.
func (m *Model) handleEnter() tea.Cmd {
	switch m.state.View {
	case state.ViewMain:
		item := m.mainMenuList.SelectedItem()
		if item == nil {
			return nil
		}
		m.state.FilterText = ""
		m.filterInput.SetValue("")
		switch item.ID {
		case "ecs-clusters":
			return m.switchToECS()
		case "lambda-functions":
			return m.switchToLambda()
		case "sqs-queues":
			return m.switchToSQS()
		case "dynamodb-tables":
			return m.switchToDynamoDB()
		case "api-gateway":
			return m.switchToAPIGateway()
		case "cloudformation-stacks":
			return m.switchToStacks()
		}
		return nil
	case state.ViewClusters:
		item := m.clustersList.SelectedItem()
		if item == nil {
			return nil
		}
		// Find the cluster and navigate to its services
		for i := range m.state.Clusters {
			if m.state.Clusters[i].Name == item.ID {
				m.state.SelectCluster(&m.state.Clusters[i])
				m.state.FilterText = ""
				m.filterInput.SetValue("")
				return m.loadServicesForCluster()
			}
		}
		return nil
	case state.ViewStacks:
		item := m.stacksList.SelectedItem()
		if item == nil {
			return nil
		}
		// Find the stack
		for i := range m.state.Stacks {
			if m.state.Stacks[i].Name == item.ID {
				m.state.SelectStack(&m.state.Stacks[i])
				m.state.FilterText = ""
				m.filterInput.SetValue("")
				m.updateStackResourcesList()
				return nil
			}
		}
	case state.ViewStackResources:
		item := m.stackResourcesList.SelectedItem()
		if item == nil {
			return nil
		}
		m.state.FilterText = ""
		m.filterInput.SetValue("")
		// Navigate to the selected resource type
		switch item.ID {
		case "ecs-services":
			m.state.View = state.ViewServices
			return m.loadServices()
		case "lambda-functions":
			m.state.View = state.ViewLambda
			return m.loadFunctions()
		case "api-gateway":
			m.state.View = state.ViewAPIGateway
			return m.loadAPIs()
		case "sqs-queues":
			m.state.View = state.ViewSQS
			return m.loadQueues()
		}
		return nil
	case state.ViewAPIGateway:
		item := m.apiGatewayList.SelectedItem()
		if item == nil {
			return nil
		}
		// Check if it's a REST or HTTP API based on ID prefix
		if len(item.ID) > 5 && item.ID[:5] == "rest:" {
			apiID := item.ID[5:]
			for i := range m.state.RestAPIs {
				if m.state.RestAPIs[i].ID == apiID {
					m.state.SelectRestAPI(&m.state.RestAPIs[i])
					m.state.FilterText = ""
					m.filterInput.SetValue("")
					return m.loadAPIStages()
				}
			}
		} else if len(item.ID) > 5 && item.ID[:5] == "http:" {
			apiID := item.ID[5:]
			for i := range m.state.HttpAPIs {
				if m.state.HttpAPIs[i].ID == apiID {
					m.state.SelectHttpAPI(&m.state.HttpAPIs[i])
					m.state.FilterText = ""
					m.filterInput.SetValue("")
					return m.loadAPIStages()
				}
			}
		}
	case state.ViewJumpHostSelect:
		// User selected a jump host for private API Gateway tunnel
		item := m.ec2List.SelectedItem()
		if item == nil {
			return nil
		}
		// Find the selected EC2 instance
		for i := range m.state.EC2Instances {
			if m.state.EC2Instances[i].InstanceID == item.ID {
				jumpHost := &m.state.EC2Instances[i]
				m.logger.Info("Selected jump host: %s (%s)", jumpHost.Name, jumpHost.InstanceID)

				// Get the pending tunnel info
				if m.state.PendingTunnelStage == nil || m.state.PendingTunnelAPI == nil {
					m.logger.Error("No pending tunnel info found")
					return nil
				}

				// Start the tunnel with the selected jump host
				return m.startPrivateAPIGWTunnelWithJumpHost(jumpHost)
			}
		}
	case state.ViewContainerSelect:
		// User selected a container for port forwarding
		item := m.containerList.SelectedItem()
		if item == nil {
			return nil
		}
		// Find the selected container
		for i := range m.state.PendingContainers {
			if m.state.PendingContainers[i].Name == item.ID {
				container := m.state.PendingContainers[i]
				service := m.state.PendingContainerService
				task := m.state.PendingContainerTask

				if service == nil || task == nil {
					m.logger.Error("No pending service/task info found")
					m.state.ClearPendingContainer()
					return nil
				}

				remotePort := container.GetBestPort()
				localPort := m.pendingLocalPort
				localPortStr := "random"
				if localPort > 0 {
					localPortStr = fmt.Sprintf("%d", localPort)
				}

				m.logger.Info("Selected container '%s' for tunnel (local: %s, remote: %d)", container.Name, localPortStr, remotePort)

				// Clear pending state and go back to services view
				svc := *service
				tsk := *task
				m.state.ClearPendingContainer()
				m.pendingLocalPort = 0
				m.state.View = state.ViewServices

				return m.startTunnelWithPort(svc, tsk, container, remotePort, localPort)
			}
		}
	}
	return nil
}

// handleBack handles the back/escape key press based on current view.
func (m *Model) handleBack() {
	switch m.state.View {
	case state.ViewStacks:
		// Go back to main menu
		m.state.View = state.ViewMain
		m.state.FilterText = ""
		m.filterInput.SetValue("")
		m.state.ClearStacks()
		m.updateMainMenuList()
	case state.ViewStackResources:
		m.state.View = state.ViewStacks
		m.state.SelectedStack = nil
		m.state.FilterText = ""
		m.filterInput.SetValue("")
		m.updateStacksList()
	case state.ViewServices:
		m.state.FilterText = ""
		m.filterInput.SetValue("")
		m.state.ClearServices()
		// If we came from a cluster, go back to clusters
		if m.state.SelectedCluster != nil {
			m.state.SelectedCluster = nil
			m.state.View = state.ViewClusters
			m.updateClustersList()
		} else {
			// Otherwise go back to stack resources
			m.state.View = state.ViewStackResources
			m.updateStackResourcesList()
		}
	case state.ViewLambda:
		// If we came from stack resources, go back there
		if m.state.SelectedStack != nil {
			m.state.View = state.ViewStackResources
			m.state.FilterText = ""
			m.filterInput.SetValue("")
			m.state.ClearFunctions()
			m.updateStackResourcesList()
		}
	case state.ViewAPIGateway:
		// If we came from stack resources, go back there
		if m.state.SelectedStack != nil {
			m.state.View = state.ViewStackResources
			m.state.FilterText = ""
			m.filterInput.SetValue("")
			m.state.ClearAPIs()
			m.updateStackResourcesList()
		}
	case state.ViewSQS:
		m.state.FilterText = ""
		m.filterInput.SetValue("")
		// If we came from stack resources, go back there and clear stack-specific queues
		if m.state.SelectedStack != nil {
			m.state.ClearQueues() // Clear stack-specific queues
			m.state.View = state.ViewStackResources
			m.updateStackResourcesList()
		} else {
			// Going back to main menu - keep queues cached
			m.state.View = state.ViewMain
			m.updateMainMenuList()
		}
	case state.ViewDynamoDB:
		m.state.FilterText = ""
		m.filterInput.SetValue("")
		// Going back to main menu - keep tables cached
		m.state.View = state.ViewMain
		m.updateMainMenuList()
	case state.ViewAPIStages:
		m.state.GoBack()
		m.state.FilterText = ""
		m.filterInput.SetValue("")
		m.updateAPIGatewayList()
	case state.ViewJumpHostSelect:
		// Go back to API stages, clear pending tunnel info
		m.state.View = state.ViewAPIStages
		m.state.FilterText = ""
		m.filterInput.SetValue("")
		m.state.ClearEC2Instances()
		m.state.ClearPendingTunnel()
		m.updateAPIStagesList()
	case state.ViewContainerSelect:
		// Go back to services, clear pending container info
		m.state.View = state.ViewServices
		m.state.FilterText = ""
		m.filterInput.SetValue("")
		m.state.ClearPendingContainer()
		m.pendingLocalPort = 0
		m.updateServicesList()
	case state.ViewCloudWatchLogs:
		// Go back to the source view (Lambda or Services), stop streaming
		if m.state.CloudWatchLambdaContext != nil {
			m.state.View = state.ViewLambda
			m.updateLambdaList()
		} else {
			m.state.View = state.ViewServices
			m.updateServicesList()
		}
		m.state.CloudWatchLogsStreaming = false
		m.state.ClearCloudWatchLogs()
		m.cloudWatchLogsPanel.SetStreaming(false)
		m.cloudWatchLogsPanel.Clear()
	case state.ViewTunnels:
		// Go back to previous view (stacks or services)
		if m.state.SelectedStack != nil {
			m.state.View = state.ViewServices
		} else {
			m.state.View = state.ViewStacks
		}
	}
}

// handleRefresh handles the refresh key press based on current view.
func (m *Model) handleRefresh() tea.Cmd {
	switch m.state.View {
	case state.ViewStacks:
		return m.loadStacks()
	case state.ViewServices:
		return m.loadServices()
	case state.ViewLambda:
		return m.loadFunctions()
	case state.ViewAPIGateway:
		return m.loadAPIs()
	case state.ViewAPIStages:
		return m.loadAPIStages()
	case state.ViewJumpHostSelect:
		return m.loadEC2Instances()
	case state.ViewTunnels:
		m.updateTunnelsPanel()
	case state.ViewSQS:
		return m.loadQueues()
	case state.ViewDynamoDB:
		return m.loadTables()
	}
	return nil
}

// handleCloudWatchLogs handles the CloudWatch logs key press.
func (m *Model) handleCloudWatchLogs() tea.Cmd {
	// Handle Lambda view
	if m.state.View == state.ViewLambda {
		return m.handleLambdaCloudWatchLogs()
	}

	// Only works in Services view
	if m.state.View != state.ViewServices {
		m.logger.Debug("CloudWatch logs: only available in services view")
		return nil
	}

	item := m.serviceList.SelectedItem()
	if item == nil {
		m.logger.Warn("CloudWatch logs: no service selected")
		return nil
	}

	// Find the service
	var selectedService *model.Service
	for i := range m.state.Services {
		if m.state.Services[i].Name == item.ID {
			selectedService = &m.state.Services[i]
			break
		}
	}

	if selectedService == nil {
		return nil
	}

	if selectedService.ClusterARN == "" {
		m.logger.Error("CloudWatch logs: service has no cluster ARN")
		return nil
	}

	m.logger.Info("Loading CloudWatch logs for service: %s", selectedService.Name)

	// First, fetch tasks to get task ID
	service := *selectedService
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		tasks, err := m.client.ListTasksForService(ctx, service.ClusterARN, service.Name)
		if err != nil || len(tasks) == 0 {
			errMsg := "no running tasks found"
			if err != nil {
				errMsg = err.Error()
			}
			return cloudWatchLogConfigsLoadedMsg{err: fmt.Errorf("failed to get tasks: %s", errMsg)}
		}

		task := tasks[0] // Use first running task

		configs, err := m.client.GetContainerLogConfigs(ctx, task.TaskDefinitionARN, task.TaskID)
		return cloudWatchLogConfigsLoadedMsg{
			configs: configs,
			service: service,
			task:    task,
			err:     err,
		}
	}
}

// handleLambdaCloudWatchLogs handles CloudWatch logs for Lambda functions.
func (m *Model) handleLambdaCloudWatchLogs() tea.Cmd {
	item := m.lambdaList.SelectedItem()
	if item == nil {
		m.logger.Warn("CloudWatch logs: no Lambda function selected")
		return nil
	}

	// Find the function
	var selectedFn *model.Function
	for i := range m.state.Functions {
		if m.state.Functions[i].Name == item.ID {
			selectedFn = &m.state.Functions[i]
			break
		}
	}

	if selectedFn == nil {
		return nil
	}

	m.logger.Info("Loading CloudWatch logs for Lambda: %s", selectedFn.Name)

	// Set up Lambda context and transition to CloudWatch logs view
	fn := *selectedFn
	logGroup := fmt.Sprintf("/aws/lambda/%s", fn.Name)

	// Create a synthetic log config for Lambda
	config := model.ContainerLogConfig{
		ContainerName: fn.Name,
		LogGroup:      logGroup,
		LogStreamName: "", // Lambda logs query across all streams
	}

	m.state.ClearCloudWatchLogs()
	m.state.CloudWatchLogConfigs = []model.ContainerLogConfig{config}
	m.state.CloudWatchLambdaContext = &fn
	m.state.View = state.ViewCloudWatchLogs
	m.state.CloudWatchLogsStreaming = true
	m.state.CloudWatchLastFetchTime = 0

	m.cloudWatchLogsPanel.SetContainers([]model.ContainerLogConfig{config})
	m.cloudWatchLogsPanel.SetContext(fn.Name, "Lambda")
	m.cloudWatchLogsPanel.SetStreaming(true)
	m.cloudWatchLogsPanel.Clear()

	// Start fetching logs
	return tea.Batch(
		m.fetchLambdaCloudWatchLogs(logGroup),
		m.cloudWatchLogsPanel.TickCmd(),
	)
}

// handlePortForward handles the port forward key press.
func (m *Model) handlePortForward() tea.Cmd {
	// Handle API Gateway stages view
	if m.state.View == state.ViewAPIStages {
		return m.handleAPIGatewayPortForward()
	}

	// From tunnels view, if we have services loaded, show port input for selected service
	if m.state.View == state.ViewTunnels {
		if len(m.state.Services) > 0 {
			// Use the currently selected service from the service list
			item := m.serviceList.SelectedItem()
			if item != nil {
				for i := range m.state.Services {
					if m.state.Services[i].Name == item.ID {
						selectedService := &m.state.Services[i]
						if selectedService.ClusterARN != "" {
							m.pendingPortForward = selectedService
							m.enteringPort = true
							m.portInput.SetValue("")
							m.portInput.Focus()
							return textinput.Blink
						}
						break
					}
				}
			}
		}
		// No services loaded, go to services/stacks view
		if m.state.SelectedStack != nil {
			m.state.View = state.ViewServices
		} else {
			m.state.View = state.ViewStacks
		}
		return nil
	}

	// Only works in services view
	if m.state.View != state.ViewServices {
		m.logger.Debug("Port forward ignored: not in services or API stages view")
		return nil
	}

	item := m.serviceList.SelectedItem()
	if item == nil {
		m.logger.Warn("Port forward: no service selected")
		return nil
	}

	// Find the service
	var selectedService *model.Service
	for i := range m.state.Services {
		if m.state.Services[i].Name == item.ID {
			selectedService = &m.state.Services[i]
			break
		}
	}

	if selectedService == nil {
		m.logger.Error("Port forward: service '%s' not found in state", item.ID)
		return nil
	}

	if selectedService.ClusterARN == "" {
		m.logger.Error("Port forward: service '%s' has no ClusterARN", selectedService.Name)
		m.state.ShowLogs = true
		m.updateComponentSizes()
		return nil
	}

	// Start port input mode
	m.pendingPortForward = selectedService
	m.enteringPort = true
	m.portInput.SetValue("")
	m.portInput.Focus()

	return textinput.Blink
}

// handlePortInputKey handles key messages when entering a port number.
func (m *Model) handlePortInputKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "enter":
		// Parse port from input
		portStr := m.portInput.Value()
		localPort := 0 // 0 means random
		if portStr != "" {
			var err error
			_, err = fmt.Sscanf(portStr, "%d", &localPort)
			if err != nil || localPort < 0 || localPort > 65535 {
				m.logger.Error("Invalid port number: %s", portStr)
				m.enteringPort = false
				m.portInput.Blur()
				m.pendingPortForward = nil
				m.pendingAPIGWPortForward = nil
				m.pendingAPIGWAPI = nil
				return nil
			}
		}

		// Handle API Gateway port forward
		if m.pendingAPIGWPortForward != nil {
			stage := m.pendingAPIGWPortForward
			api := m.pendingAPIGWAPI
			m.enteringPort = false
			m.portInput.Blur()
			m.pendingAPIGWPortForward = nil
			m.pendingAPIGWAPI = nil

			return m.startAPIGatewayTunnel(api, *stage, localPort)
		}

		// Store the port and start loading tasks for ECS service
		service := m.pendingPortForward
		m.enteringPort = false
		m.portInput.Blur()
		m.pendingPortForward = nil

		if service == nil {
			return nil
		}

		m.logger.Info("Loading tasks for service: %s (cluster: %s)", service.Name, service.ClusterName)

		// Store the requested local port in context for later use
		requestedPort := localPort

		return func() tea.Msg {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			tasks, err := m.client.ListTasksForService(ctx, service.ClusterARN, service.Name)
			return tasksLoadedMsgWithPort{service: *service, tasks: tasks, err: err, localPort: requestedPort}
		}

	case "esc":
		m.enteringPort = false
		m.portInput.Blur()
		m.pendingPortForward = nil
		m.pendingAPIGWPortForward = nil
		m.pendingAPIGWAPI = nil
		return nil
	}

	// Pass other keys to the input
	var cmd tea.Cmd
	m.portInput, cmd = m.portInput.Update(msg)
	return cmd
}

// handleLambdaInvoke handles the Lambda invoke key press.
func (m *Model) handleLambdaInvoke() tea.Cmd {
	if m.state.View != state.ViewLambda {
		return nil
	}

	item := m.lambdaList.SelectedItem()
	if item == nil {
		return nil
	}

	// Find the selected function
	var selectedFn *model.Function
	for i := range m.state.Functions {
		if m.state.Functions[i].Name == item.ID {
			selectedFn = &m.state.Functions[i]
			break
		}
	}

	if selectedFn == nil {
		return nil
	}

	// Set up payload input dialog
	m.enteringPayload = true
	m.pendingInvokeFunction = selectedFn
	m.payloadInput.Reset()
	m.payloadInput.Focus()

	m.logger.Info("Opening payload dialog for Lambda: %s", selectedFn.Name)

	return textinput.Blink
}

// handlePayloadInputKey handles key messages when entering a Lambda payload.
func (m *Model) handlePayloadInputKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "enter":
		payload := m.payloadInput.Value()
		fn := m.pendingInvokeFunction

		m.enteringPayload = false
		m.payloadInput.Blur()
		m.pendingInvokeFunction = nil

		if fn == nil {
			return nil
		}

		// Clear previous invocation state
		m.state.ClearLambdaInvocation()
		m.state.LambdaInvocationLoading = true
		m.updateLambdaDetails()

		m.logger.Info("Invoking Lambda %s with payload: %s", fn.Name, truncateString(payload, 50))

		functionName := fn.Name
		return func() tea.Msg {
			ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
			defer cancel()

			result, err := m.client.InvokeFunction(ctx, functionName, payload)
			return lambdaInvocationResultMsg{result: result, err: err}
		}

	case "esc":
		m.enteringPayload = false
		m.payloadInput.Blur()
		m.pendingInvokeFunction = nil
		return nil
	}

	// Pass other keys to the input
	var cmd tea.Cmd
	m.payloadInput, cmd = m.payloadInput.Update(msg)
	return cmd
}

// handleStopTunnel handles stopping a tunnel.
func (m *Model) handleStopTunnel() tea.Cmd {
	// Only works in tunnels view
	if m.state.View != state.ViewTunnels {
		return nil
	}

	// Check for ECS tunnel first
	ecsTunnel := m.tunnelsPanel.SelectedTunnel()
	if ecsTunnel != nil {
		m.logger.Info("Stopping ECS tunnel: %s", ecsTunnel.ID)
		if err := m.tunnelManager.StopTunnel(ecsTunnel.ID); err != nil {
			m.logger.Error("Failed to stop tunnel: %v", err)
		}
		m.updateTunnelsPanel()
		return nil
	}

	// Check for API Gateway tunnel
	apiGWTunnel := m.tunnelsPanel.SelectedAPIGatewayTunnel()
	if apiGWTunnel != nil {
		m.logger.Info("Stopping API Gateway tunnel: %s", apiGWTunnel.ID)
		if err := m.apiGWManager.StopTunnel(apiGWTunnel.ID); err != nil {
			m.logger.Error("Failed to stop API Gateway tunnel: %v", err)
		}
		m.updateTunnelsPanel()
		return nil
	}

	return nil
}

// handleRestartTunnel handles restarting a tunnel.
func (m *Model) handleRestartTunnel() tea.Cmd {
	// Only works in tunnels view
	if m.state.View != state.ViewTunnels {
		return nil
	}

	tunnel := m.tunnelsPanel.SelectedTunnel()
	if tunnel == nil {
		return nil
	}

	// Can only restart terminated or errored tunnels
	if tunnel.Status == model.TunnelStatusActive || tunnel.Status == model.TunnelStatusStarting {
		m.logger.Warn("Tunnel '%s' is still active. Stop it first before restarting.", tunnel.ID)
		return nil
	}

	// Check if we have the cluster ARN needed to fetch tasks
	if tunnel.ClusterARN == "" {
		m.logger.Error("Cannot restart tunnel '%s': missing cluster ARN (tunnel was created in an older version)", tunnel.ID)
		return nil
	}

	m.logger.Info("Restarting tunnel '%s' for service '%s'...", tunnel.ID, tunnel.ServiceName)

	// Prepare the tunnel for restart (removes it from the list)
	tunnelInfo, err := m.tunnelManager.PrepareRestart(tunnel.ID)
	if err != nil {
		m.logger.Error("Failed to prepare tunnel restart: %v", err)
		return nil
	}

	// Fetch tasks for the service and start the tunnel
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		tasks, err := m.client.ListTasksForService(ctx, tunnelInfo.ClusterARN, tunnelInfo.ServiceName)
		return tasksLoadedMsgForRestart{
			tunnelInfo: *tunnelInfo,
			tasks:      tasks,
			err:        err,
		}
	}
}

// handleAPIGatewayPortForward starts port forwarding for the selected API Gateway stage.
func (m *Model) handleAPIGatewayPortForward() tea.Cmd {
	item := m.apiStagesList.SelectedItem()
	if item == nil {
		m.logger.Warn("Port forward: no API stage selected")
		return nil
	}

	// Find the stage
	var selectedStage *model.APIStage
	for i := range m.state.APIStages {
		if m.state.APIStages[i].Name == item.ID {
			selectedStage = &m.state.APIStages[i]
			break
		}
	}

	if selectedStage == nil {
		m.logger.Error("Port forward: stage '%s' not found in state", item.ID)
		return nil
	}

	// Get the API
	var api interface{}
	if m.state.SelectedRestAPI != nil {
		api = m.state.SelectedRestAPI
	} else if m.state.SelectedHttpAPI != nil {
		api = m.state.SelectedHttpAPI
	} else {
		m.logger.Error("Port forward: no API selected")
		return nil
	}

	// Start port input mode
	m.pendingAPIGWPortForward = selectedStage
	m.pendingAPIGWAPI = api
	m.enteringPort = true
	m.portInput.SetValue("")
	m.portInput.Focus()

	return textinput.Blink
}

// handleDynamoDBQuery opens the query dialog for the selected table.
func (m *Model) handleDynamoDBQuery() tea.Cmd {
	if m.state.View != state.ViewDynamoDB {
		return nil
	}

	table := m.dynamodbTable.SelectedTable()
	if table == nil {
		m.logger.Warn("Query: no table selected")
		return nil
	}

	m.state.SelectTable(table)
	m.logger.Info("Opening query dialog for table: %s", table.Name)

	// Set size for dialog
	m.dynamodbQueryDialog.SetSize(m.width, m.height)

	return m.dynamodbQueryDialog.Activate(table.Name, table.PartitionKey(), table.SortKey(), true)
}

// handleDynamoDBScan opens the scan dialog for the selected table.
func (m *Model) handleDynamoDBScan() tea.Cmd {
	if m.state.View != state.ViewDynamoDB {
		return nil
	}

	table := m.dynamodbTable.SelectedTable()
	if table == nil {
		m.logger.Warn("Scan: no table selected")
		return nil
	}

	m.state.SelectTable(table)
	m.logger.Info("Opening scan dialog for table: %s", table.Name)

	// Set size for dialog
	m.dynamodbQueryDialog.SetSize(m.width, m.height)

	return m.dynamodbQueryDialog.Activate(table.Name, table.PartitionKey(), table.SortKey(), false)
}

// handleDynamoDBQueryDialogKey handles key presses when the query dialog is active.
func (m *Model) handleDynamoDBQueryDialogKey(msg tea.KeyMsg) tea.Cmd {
	result, cmd := m.dynamodbQueryDialog.Update(msg)
	if result != nil {
		if result.Cancelled {
			m.logger.Debug("Query dialog cancelled")
			return nil
		}

		// Execute the query or scan
		if result.QueryParams != nil {
			m.state.DynamoDBQueryParams = result.QueryParams
			m.state.DynamoDBScanParams = nil
			m.state.DynamoDBIsQuery = true
			m.state.DynamoDBQueryLoading = true
			m.state.DynamoDBLastKey = nil
			m.state.View = state.ViewDynamoDBQuery
			m.dynamodbQueryResults.SetLoading(true)
			m.dynamodbQueryResults.Clear()
			m.logger.Info("Executing query on table: %s (PK: %s)", result.QueryParams.TableName, result.QueryParams.PartitionKeyVal)
			return m.executeDynamoDBQuery(result.QueryParams)
		} else if result.ScanParams != nil {
			m.state.DynamoDBQueryParams = nil
			m.state.DynamoDBScanParams = result.ScanParams
			m.state.DynamoDBIsQuery = false
			m.state.DynamoDBQueryLoading = true
			m.state.DynamoDBLastKey = nil
			m.state.View = state.ViewDynamoDBQuery
			m.dynamodbQueryResults.SetLoading(true)
			m.dynamodbQueryResults.Clear()
			m.logger.Info("Executing scan on table: %s", result.ScanParams.TableName)
			return m.executeDynamoDBScan(result.ScanParams)
		}
	}
	return cmd
}

// handleDynamoDBQueryResultsKey handles key presses in the query results view.
func (m *Model) handleDynamoDBQueryResultsKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "ctrl+c":
		m.tunnelManager.StopAllTunnels()
		return tea.Quit

	case "q":
		// Start a new query on the same table
		if m.state.SelectedTable != nil {
			m.dynamodbQueryDialog.SetSize(m.width, m.height)
			return m.dynamodbQueryDialog.Activate(
				m.state.SelectedTable.Name,
				m.state.SelectedTable.PartitionKey(),
				m.state.SelectedTable.SortKey(),
				true, // isQuery
			)
		}
		return nil

	case "s":
		// Start a new scan on the same table
		if m.state.SelectedTable != nil {
			m.dynamodbQueryDialog.SetSize(m.width, m.height)
			return m.dynamodbQueryDialog.Activate(
				m.state.SelectedTable.Name,
				m.state.SelectedTable.PartitionKey(),
				m.state.SelectedTable.SortKey(),
				false, // isScan
			)
		}
		return nil

	case "esc", "backspace":
		// Go back to table list
		m.state.View = state.ViewDynamoDB
		m.state.ClearDynamoDBQuery()
		m.dynamodbQueryResults.Clear()
		m.updateTablesList()
		return nil

	case "up", "k":
		m.dynamodbQueryResults.Up()
		return nil

	case "down", "j":
		m.dynamodbQueryResults.Down()
		return nil

	case "g":
		m.dynamodbQueryResults.Top()
		return nil

	case "G":
		m.dynamodbQueryResults.Bottom()
		return nil

	case "J":
		// Scroll JSON panel down
		m.dynamodbQueryResults.ScrollJSONDown()
		return nil

	case "K":
		// Scroll JSON panel up
		m.dynamodbQueryResults.ScrollJSONUp()
		return nil

	case "ctrl+d":
		// Half page down in JSON panel
		m.dynamodbQueryResults.ScrollJSONHalfPageDown()
		return nil

	case "ctrl+u":
		// Half page up in JSON panel
		m.dynamodbQueryResults.ScrollJSONHalfPageUp()
		return nil

	case "n":
		// Load next page if available
		if m.dynamodbQueryResults.HasMorePages() && !m.state.DynamoDBQueryLoading {
			m.state.DynamoDBQueryLoading = true
			m.dynamodbQueryResults.SetLoading(true)
			m.logger.Info("Loading next page of results...")
			return m.loadNextDynamoDBPage()
		}
		return nil

	case "r":
		// Re-run the query/scan
		if m.state.DynamoDBIsQuery && m.state.DynamoDBQueryParams != nil {
			m.state.DynamoDBQueryLoading = true
			m.state.DynamoDBLastKey = nil
			m.dynamodbQueryResults.SetLoading(true)
			m.dynamodbQueryResults.Clear()
			return m.executeDynamoDBQuery(m.state.DynamoDBQueryParams)
		} else if !m.state.DynamoDBIsQuery && m.state.DynamoDBScanParams != nil {
			m.state.DynamoDBQueryLoading = true
			m.state.DynamoDBLastKey = nil
			m.dynamodbQueryResults.SetLoading(true)
			m.dynamodbQueryResults.Clear()
			return m.executeDynamoDBScan(m.state.DynamoDBScanParams)
		}
		return nil

	case "y":
		// Copy mode - show JSON content for selection
		m.copyMode = true
		m.copyModeScroll = 0
		m.logger.Info("Copy mode enabled - select text with mouse, press y or Esc to exit")
		// Disable mouse capture to allow terminal text selection
		return tea.DisableMouse

	case "Y":
		// Yank current item's JSON to clipboard
		text := m.dynamodbQueryResults.SelectedJSON()
		if text == "" {
			m.logger.Warn("No item selected to copy")
			return nil
		}
		err := clipboard.Init()
		if err != nil {
			m.logger.Warn("Clipboard not available - use copy mode (y) instead: " + err.Error())
			return nil
		}
		clipboard.Write(clipboard.FmtText, []byte(text))
		m.logger.Info("JSON copied to clipboard")
		return nil

	case ":":
		// Open command palette
		m.commandPalette.SetWidth(m.width)
		return m.commandPalette.Activate()

	case "?":
		// Show help
		m.showHelp()
		return nil

	case "l":
		// Toggle logs
		m.state.ToggleLogs()
		m.updateComponentSizes()
		return nil

	// Quick resource switching with number keys
	case "0":
		return m.switchToMain()
	case "1":
		return m.switchToECS()
	case "2":
		return m.switchToLambda()
	case "3":
		return m.switchToSQS()
	case "4":
		return m.switchToDynamoDB()
	case "5":
		return m.switchToAPIGateway()
	case "6":
		return m.switchToStacks()
	}

	return nil
}

// handleMouseWheelUp handles mouse wheel scroll up events.
func (m *Model) handleMouseWheelUp(x int) {
	// Determine which pane was scrolled based on X coordinate
	layout := m.getLayoutMode()
	if layout != layoutFull {
		// Single pane - scroll the list
		m.moveCursorUp()
		return
	}

	// Split view - determine which pane based on X position
	listWidth := int(float64(m.width) * listPaneRatio)

	if x < listWidth {
		// Left pane (list) - move cursor up
		m.moveCursorUp()
	} else {
		// Right pane (details/JSON) - scroll up
		if m.state.View == state.ViewDynamoDBQuery {
			m.dynamodbQueryResults.ScrollJSONUp()
		} else {
			m.details.ScrollUp()
		}
	}
}

// handleMouseWheelDown handles mouse wheel scroll down events.
func (m *Model) handleMouseWheelDown(x int) {
	// Determine which pane was scrolled based on X coordinate
	layout := m.getLayoutMode()
	if layout != layoutFull {
		// Single pane - scroll the list
		m.moveCursorDown()
		return
	}

	// Split view - determine which pane based on X position
	listWidth := int(float64(m.width) * listPaneRatio)

	if x < listWidth {
		// Left pane (list) - move cursor down
		m.moveCursorDown()
	} else {
		// Right pane (details/JSON) - scroll down
		if m.state.View == state.ViewDynamoDBQuery {
			m.dynamodbQueryResults.ScrollJSONDown()
		} else {
			m.details.ScrollDown()
		}
	}
}
