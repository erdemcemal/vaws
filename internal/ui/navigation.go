package ui

import (
	tea "github.com/charmbracelet/bubbletea"

	"vaws/internal/state"
)

// startFiltering enters filter mode.
func (m *Model) startFiltering() {
	m.filtering = true
	m.filterInput.SetValue(m.state.FilterText)
	m.filterInput.Focus()
}

// startDetailsSearch enters details search mode.
func (m *Model) startDetailsSearch() {
	m.detailsSearching = true
	m.detailsSearchInput.SetValue(m.details.SearchQuery())
	m.detailsSearchInput.Focus()
}

// moveCursorUp moves the cursor up in the current list.
func (m *Model) moveCursorUp() {
	switch m.state.View {
	case state.ViewMain:
		m.mainMenuList.Up()
	case state.ViewStacks:
		m.stacksList.Up()
		m.updateStackDetails()
	case state.ViewStackResources:
		m.stackResourcesList.Up()
	case state.ViewClusters:
		m.clustersList.Up()
	case state.ViewServices:
		m.serviceList.Up()
		m.updateServiceDetails()
	case state.ViewLambda:
		m.lambdaList.Up()
		m.updateLambdaDetails()
	case state.ViewAPIGateway:
		m.apiGatewayList.Up()
		m.updateAPIGatewayDetails()
	case state.ViewAPIStages:
		m.apiStagesList.Up()
		m.updateAPIStageDetails()
	case state.ViewJumpHostSelect:
		m.ec2List.Up()
	case state.ViewContainerSelect:
		m.containerList.Up()
	case state.ViewSQS:
		m.sqsTable.Up()
		m.updateQueueDetails()
	case state.ViewDynamoDB:
		m.dynamodbTable.Up()
		m.updateTableDetails()
	case state.ViewTunnels:
		m.tunnelsPanel.Up()
	}
}

// moveCursorDown moves the cursor down in the current list.
func (m *Model) moveCursorDown() {
	switch m.state.View {
	case state.ViewMain:
		m.mainMenuList.Down()
	case state.ViewStacks:
		m.stacksList.Down()
		m.updateStackDetails()
	case state.ViewStackResources:
		m.stackResourcesList.Down()
	case state.ViewClusters:
		m.clustersList.Down()
	case state.ViewServices:
		m.serviceList.Down()
		m.updateServiceDetails()
	case state.ViewLambda:
		m.lambdaList.Down()
		m.updateLambdaDetails()
	case state.ViewAPIGateway:
		m.apiGatewayList.Down()
		m.updateAPIGatewayDetails()
	case state.ViewAPIStages:
		m.apiStagesList.Down()
		m.updateAPIStageDetails()
	case state.ViewJumpHostSelect:
		m.ec2List.Down()
	case state.ViewContainerSelect:
		m.containerList.Down()
	case state.ViewSQS:
		m.sqsTable.Down()
		m.updateQueueDetails()
	case state.ViewDynamoDB:
		m.dynamodbTable.Down()
		m.updateTableDetails()
	case state.ViewTunnels:
		m.tunnelsPanel.Down()
	}
}

// moveCursorTop moves the cursor to the top of the current list.
func (m *Model) moveCursorTop() {
	switch m.state.View {
	case state.ViewMain:
		m.mainMenuList.Top()
	case state.ViewStacks:
		m.stacksList.Top()
		m.updateStackDetails()
	case state.ViewStackResources:
		m.stackResourcesList.Top()
	case state.ViewClusters:
		m.clustersList.Top()
	case state.ViewServices:
		m.serviceList.Top()
		m.updateServiceDetails()
	case state.ViewLambda:
		m.lambdaList.Top()
		m.updateLambdaDetails()
	case state.ViewAPIGateway:
		m.apiGatewayList.Top()
		m.updateAPIGatewayDetails()
	case state.ViewAPIStages:
		m.apiStagesList.Top()
		m.updateAPIStageDetails()
	case state.ViewJumpHostSelect:
		m.ec2List.Top()
	case state.ViewContainerSelect:
		m.containerList.Top()
	case state.ViewSQS:
		m.sqsTable.Top()
		m.updateQueueDetails()
	case state.ViewDynamoDB:
		m.dynamodbTable.Top()
		m.updateTableDetails()
	}
}

// moveCursorBottom moves the cursor to the bottom of the current list.
func (m *Model) moveCursorBottom() {
	switch m.state.View {
	case state.ViewMain:
		m.mainMenuList.Bottom()
	case state.ViewStacks:
		m.stacksList.Bottom()
		m.updateStackDetails()
	case state.ViewStackResources:
		m.stackResourcesList.Bottom()
	case state.ViewClusters:
		m.clustersList.Bottom()
	case state.ViewServices:
		m.serviceList.Bottom()
		m.updateServiceDetails()
	case state.ViewLambda:
		m.lambdaList.Bottom()
		m.updateLambdaDetails()
	case state.ViewAPIGateway:
		m.apiGatewayList.Bottom()
		m.updateAPIGatewayDetails()
	case state.ViewAPIStages:
		m.apiStagesList.Bottom()
		m.updateAPIStageDetails()
	case state.ViewJumpHostSelect:
		m.ec2List.Bottom()
	case state.ViewContainerSelect:
		m.containerList.Bottom()
	case state.ViewSQS:
		m.sqsTable.Bottom()
		m.updateQueueDetails()
	case state.ViewDynamoDB:
		m.dynamodbTable.Bottom()
		m.updateTableDetails()
	}
}

// switchToDynamoDB switches to the DynamoDB tables view.
func (m *Model) switchToDynamoDB() tea.Cmd {
	m.state.View = state.ViewDynamoDB
	m.state.SelectedStack = nil
	m.state.FilterText = ""
	m.filterInput.SetValue("")
	m.quickBar.SetActiveResource("6")
	// Only load if not already loaded
	if len(m.state.Tables) == 0 && !m.state.TablesLoading {
		return m.loadTables()
	}
	m.updateTablesList()
	return nil
}

// showTunnelsView switches to the tunnels view.
func (m *Model) showTunnelsView() {
	m.state.View = state.ViewTunnels
	m.updateTunnelsPanel()
}

// showHelp displays the help information in the logs panel.
func (m *Model) showHelp() {
	m.logger.Info("═══════════════════════════════════════════════════════════════")
	m.logger.Info("                        VAWS HELP")
	m.logger.Info("═══════════════════════════════════════════════════════════════")
	m.logger.Info("")
	m.logger.Info("NAVIGATION:")
	m.logger.Info("  ↑/k, ↓/j     Navigate up/down")
	m.logger.Info("  Enter/→      Select item")
	m.logger.Info("  Esc/←        Go back")
	m.logger.Info("  g/G          Jump to top/bottom")
	m.logger.Info("")
	m.logger.Info("QUICK KEYS:")
	m.logger.Info("  0            Main menu")
	m.logger.Info("  1            ECS Clusters")
	m.logger.Info("  2            Lambda Functions")
	m.logger.Info("  3            SQS Queues")
	m.logger.Info("  4            API Gateway")
	m.logger.Info("  5            CloudFormation Stacks")
	m.logger.Info("  6            DynamoDB Tables")
	m.logger.Info("")
	m.logger.Info("ACTIONS:")
	m.logger.Info("  :            Open command palette")
	m.logger.Info("  /            Filter current list")
	m.logger.Info("  r            Refresh current view")
	m.logger.Info("  l            Toggle logs panel")
	m.logger.Info("  L            View CloudWatch logs (on service/Lambda)")
	m.logger.Info("  i            Invoke Lambda function")
	m.logger.Info("  p            Port forward (on service)")
	m.logger.Info("  t            View tunnels")
	m.logger.Info("  a            Toggle auto-refresh")
	m.logger.Info("  ?            Show this help")
	m.logger.Info("  q            Quit")
	m.logger.Info("")
	m.logger.Info("COMMANDS (type : then command):")
	m.logger.Info("  :main        Main menu")
	m.logger.Info("  :ecs         ECS clusters")
	m.logger.Info("  :lambda      Lambda functions")
	m.logger.Info("  :sqs         SQS queues")
	m.logger.Info("  :apigateway  API Gateway")
	m.logger.Info("  :stacks      CloudFormation stacks")
	m.logger.Info("  :dynamodb    DynamoDB tables")
	m.logger.Info("  :region      Change AWS region")
	m.logger.Info("  :tunnels     Port forward tunnels")
	m.logger.Info("  :logs        Toggle logs panel")
	m.logger.Info("  :refresh     Refresh current view")
	m.logger.Info("  :quit        Quit application")
	m.logger.Info("═══════════════════════════════════════════════════════════════")

	// Ensure logs are visible
	m.state.ShowLogs = true
	m.updateComponentSizes()
}
