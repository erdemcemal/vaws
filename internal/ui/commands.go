package ui

import (
	tea "github.com/charmbracelet/bubbletea"

	"vaws/internal/state"
	"vaws/internal/ui/components"
)

// executeCommand executes a command from the command palette.
func (m *Model) executeCommand(result *components.CommandResult) tea.Cmd {
	if result == nil {
		return nil
	}

	m.logger.Debug("Executing command: %s", result.Command)

	switch result.Command {
	// Resource views (matching quick keys 0-5)
	case "main", "home", "menu":
		return m.switchToMain()

	case "ecs", "clusters":
		return m.switchToECS()

	case "lambda":
		return m.switchToLambda()

	case "sqs":
		return m.switchToSQS()

	case "apigateway":
		return m.switchToAPIGateway()

	case "stacks":
		return m.switchToStacks()

	case "dynamodb", "ddb", "tables":
		return m.switchToDynamoDB()

	// Other views
	case "tunnels":
		m.showTunnelsView()
		return nil

	// Settings
	case "region":
		// Show region picker - save current view to return to it
		m.viewBeforeRegionSelect = m.state.View
		m.regionSelector.SetCurrentRegion(m.state.Region)
		m.state.View = state.ViewRegionSelect
		return nil

	// Actions
	case "refresh":
		return m.handleRefresh()

	case "logs":
		m.state.ToggleLogs()
		m.updateComponentSizes()
		if m.state.ShowLogs {
			m.logger.Info("Logs panel enabled")
		} else {
			m.logger.Info("Logs panel disabled")
		}
		return nil

	case "help":
		m.showHelp()
		return nil

	case "quit":
		if m.tunnelManager != nil {
			m.tunnelManager.StopAllTunnels()
		}
		return tea.Quit

	default:
		m.logger.Warn("Unknown command: %s", result.Command)
		return nil
	}
}

// switchToMain switches to the main menu view.
func (m *Model) switchToMain() tea.Cmd {
	m.state.SelectedStack = nil
	m.state.View = state.ViewMain
	m.state.FilterText = ""
	m.filterInput.SetValue("")
	m.quickBar.SetActiveResource("0")
	m.updateMainMenuList()
	return nil
}

// switchToECS switches to the ECS clusters view.
func (m *Model) switchToECS() tea.Cmd {
	m.state.SelectedStack = nil
	m.state.View = state.ViewClusters
	m.state.FilterText = ""
	m.filterInput.SetValue("")
	m.quickBar.SetActiveResource("1")
	return m.loadClusters()
}

// switchToLambda switches to the Lambda functions view.
func (m *Model) switchToLambda() tea.Cmd {
	m.state.SelectedStack = nil
	m.state.View = state.ViewLambda
	m.state.FilterText = ""
	m.filterInput.SetValue("")
	m.quickBar.SetActiveResource("2")
	// Only load if not already loaded
	if len(m.state.Functions) == 0 && !m.state.FunctionsLoading {
		return m.loadFunctions()
	}
	m.updateLambdaList()
	return nil
}

// switchToSQS switches to the SQS queues view.
func (m *Model) switchToSQS() tea.Cmd {
	m.state.SelectedStack = nil
	m.state.View = state.ViewSQS
	m.state.FilterText = ""
	m.filterInput.SetValue("")
	m.quickBar.SetActiveResource("3")
	// Only load if not already loaded
	if len(m.state.Queues) == 0 && !m.state.QueuesLoading {
		return m.loadQueues()
	}
	m.updateQueuesList()
	return nil
}

// switchToAPIGateway switches to the API Gateway view.
func (m *Model) switchToAPIGateway() tea.Cmd {
	m.state.SelectedStack = nil
	m.state.View = state.ViewAPIGateway
	m.state.FilterText = ""
	m.filterInput.SetValue("")
	m.quickBar.SetActiveResource("4")
	// Only load if not already loaded
	if len(m.state.RestAPIs) == 0 && len(m.state.HttpAPIs) == 0 && !m.state.APIsLoading {
		return m.loadAPIs()
	}
	m.updateAPIGatewayList()
	return nil
}

// switchToStacks switches to the CloudFormation stacks view.
func (m *Model) switchToStacks() tea.Cmd {
	m.state.SelectedStack = nil
	m.state.View = state.ViewStacks
	m.state.FilterText = ""
	m.filterInput.SetValue("")
	m.quickBar.SetActiveResource("5")
	// Only load if not already loaded
	if len(m.state.Stacks) == 0 && !m.state.StacksLoading {
		return m.loadStacks()
	}
	m.updateStacksList()
	return nil
}
