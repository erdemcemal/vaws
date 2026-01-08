package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"vaws/internal/state"
	"vaws/internal/ui/components"
	"vaws/internal/ui/theme"
)

// updateComponentSizes updates the sizes of all UI components based on terminal dimensions.
func (m *Model) updateComponentSizes() {
	if !m.ready {
		return
	}

	// Calculate logs height
	logsHeight := 0
	if m.state.ShowLogs {
		logsHeight = min(10, m.height/4)
	}

	// Note: List and details sizes are set in renderMainContent() before View() calls
	// to ensure consistent sizing with the actual rendered layout
	m.logs.SetSize(m.width, logsHeight)
}

// updateQuickBarActions sets context-specific action keys based on current view.
func (m *Model) updateQuickBarActions() {
	var actions []components.QuickKey

	switch m.state.View {
	case state.ViewServices:
		actions = []components.QuickKey{
			{Key: "p", Label: "port-forward"},
			{Key: "l", Label: "logs"},
		}
	case state.ViewAPIStages:
		actions = []components.QuickKey{
			{Key: "p", Label: "port-forward"},
		}
	case state.ViewLambda:
		actions = []components.QuickKey{
			{Key: "i", Label: "invoke"},
			{Key: "l", Label: "logs"},
		}
	case state.ViewTunnels:
		actions = []components.QuickKey{
			{Key: "p", Label: "new tunnel"},
			{Key: "s", Label: "stop"},
			{Key: "r", Label: "restart"},
		}
	case state.ViewSQS:
		// No special actions for SQS list
	case state.ViewCloudWatchLogs:
		actions = []components.QuickKey{
			{Key: "Tab", Label: "switch container"},
		}
	}

	if len(actions) > 0 {
		m.quickBar.SetContextActions(actions)
	} else {
		m.quickBar.ClearContextActions()
	}
}

// updateMainMenuList updates the main menu list items.
func (m *Model) updateMainMenuList() {
	// Show all supported AWS resource types with shortcuts
	items := []components.ListItem{
		{
			ID:          "ecs-clusters",
			Title:       "[1] ECS Clusters",
			Description: "View ECS clusters and services",
			Status:      "🚀",
			StatusStyle: lipgloss.NewStyle().Foreground(theme.Success),
		},
		{
			ID:          "lambda-functions",
			Title:       "[2] Lambda Functions",
			Description: "Browse Lambda functions",
			Status:      "λ",
			StatusStyle: lipgloss.NewStyle().Foreground(theme.Warning),
		},
		{
			ID:          "sqs-queues",
			Title:       "[3] SQS Queues",
			Description: "View SQS queues with DLQ visibility",
			Status:      "📨",
			StatusStyle: lipgloss.NewStyle().Foreground(theme.Info),
		},
		{
			ID:          "api-gateway",
			Title:       "[4] API Gateway",
			Description: "Browse REST and HTTP APIs",
			Status:      "🌐",
			StatusStyle: lipgloss.NewStyle().Foreground(theme.Primary),
		},
		{
			ID:          "cloudformation-stacks",
			Title:       "[5] CloudFormation Stacks",
			Description: "Browse resources organized by stacks",
			Status:      "📦",
			StatusStyle: lipgloss.NewStyle().Foreground(theme.TextMuted),
		},
	}
	m.mainMenuList.SetItems(items)
	m.mainMenuList.SetLoading(false)
	m.mainMenuList.SetError(nil)
	m.mainMenuList.SetEmptyMessage("No resources available")

	// Clear details pane
	m.details.SetTitle("AWS Resources")
	m.details.SetRows([]components.DetailRow{
		{Label: "Profile", Value: m.state.Profile},
		{Label: "Region", Value: m.state.Region},
		{Label: "", Value: ""},
		{Label: "Hint", Value: "Select a resource or press 1-5"},
	})
}

// updateStacksList updates the stacks list with current data.
func (m *Model) updateStacksList() {
	stacks := m.state.FilteredStacks()
	items := make([]components.ListItem, len(stacks))
	for i, s := range stacks {
		items[i] = components.ListItem{
			ID:          s.Name,
			Title:       s.Name,
			Status:      string(s.Status),
			StatusStyle: StatusStyle(string(s.Status)),
		}
	}
	m.stacksList.SetItems(items)
	m.stacksList.SetLoading(false)
	m.stacksList.SetError(m.state.StacksError)
	m.updateStackDetails()
}

// updateClustersList updates the clusters list with current data.
func (m *Model) updateClustersList() {
	clusters := m.state.FilteredClusters()
	items := make([]components.ListItem, len(clusters))
	for i, c := range clusters {
		status := "ACTIVE"
		if c.Status != "" {
			status = c.Status
		}
		items[i] = components.ListItem{
			ID:          c.Name,
			Title:       c.Name,
			Status:      fmt.Sprintf("%d services", c.ActiveServicesCount),
			StatusStyle: lipgloss.NewStyle().Foreground(theme.Success),
		}
		_ = status // Status available if needed
	}
	m.clustersList.SetItems(items)
	m.clustersList.SetLoading(m.state.ClustersLoading)
	m.clustersList.SetError(m.state.ClustersError)
}

// updateStackResourcesList updates the stack resources list.
func (m *Model) updateStackResourcesList() {
	// Show available resource types for the selected stack
	items := []components.ListItem{
		{
			ID:          "ecs-services",
			Title:       "ECS Services",
			Description: "View ECS services deployed in this stack",
			Status:      "⚙️",
			StatusStyle: lipgloss.NewStyle().Foreground(theme.Primary),
		},
		{
			ID:          "lambda-functions",
			Title:       "Lambda Functions",
			Description: "View Lambda functions deployed in this stack",
			Status:      "λ",
			StatusStyle: lipgloss.NewStyle().Foreground(theme.Warning),
		},
		{
			ID:          "api-gateway",
			Title:       "API Gateway",
			Description: "View API Gateway REST and HTTP APIs",
			Status:      "🌐",
			StatusStyle: lipgloss.NewStyle().Foreground(theme.Success),
		},
		{
			ID:          "sqs-queues",
			Title:       "SQS Queues",
			Description: "View SQS queues with DLQ visibility",
			Status:      "📨",
			StatusStyle: lipgloss.NewStyle().Foreground(theme.Info),
		},
	}
	m.stackResourcesList.SetItems(items)
	m.stackResourcesList.SetLoading(false)
	m.stackResourcesList.SetError(nil)
	m.stackResourcesList.SetEmptyMessage("No resources available")

	// Update details to show stack info
	if m.state.SelectedStack != nil {
		rows := []components.DetailRow{
			{Label: "Stack Name", Value: m.state.SelectedStack.Name},
			{Label: "Status", Value: string(m.state.SelectedStack.Status)},
			{Label: "Description", Value: m.state.SelectedStack.Description},
			{Label: "Created", Value: m.state.SelectedStack.CreatedAt.Format("2006-01-02 15:04:05")},
		}
		m.details.SetTitle("Stack Info")
		m.details.SetRows(rows)
	}
}

// updateServicesList updates the services list with current data.
func (m *Model) updateServicesList() {
	services := m.state.FilteredServices()
	items := make([]components.ListItem, len(services))
	for i, s := range services {
		items[i] = components.ListItem{
			ID:          s.Name,
			Title:       s.Name,
			Status:      fmt.Sprintf("%d/%d", s.RunningCount, s.DesiredCount),
			StatusStyle: ServiceStatusStyle(s.RunningCount, s.DesiredCount),
			Extra:       s.ClusterName,
		}
	}
	m.serviceList.SetItems(items)
	m.serviceList.SetLoading(false)
	m.serviceList.SetError(m.state.ServicesError)
	m.serviceList.SetEmptyMessage("No ECS services found in this stack")
	m.updateServiceDetails()
}

// updateLambdaList updates the Lambda functions list with current data.
func (m *Model) updateLambdaList() {
	functions := m.state.FilteredFunctions()
	items := make([]components.ListItem, len(functions))
	for i, fn := range functions {
		items[i] = components.ListItem{
			ID:          fn.Name,
			Title:       fn.Name,
			Status:      string(fn.State),
			StatusStyle: FunctionStatusStyle(fn.State),
			Extra:       fn.Runtime,
		}
	}
	m.lambdaList.SetItems(items)
	m.lambdaList.SetLoading(false)
	m.lambdaList.SetError(m.state.FunctionsError)
	m.lambdaList.SetEmptyMessage("No Lambda functions found")
	m.updateLambdaDetails()
}

// updateAPIGatewayList updates the API Gateway list with current data.
func (m *Model) updateAPIGatewayList() {
	// Combine REST and HTTP APIs into a single list
	restAPIs := m.state.FilteredRestAPIs()
	httpAPIs := m.state.FilteredHttpAPIs()

	items := make([]components.ListItem, 0, len(restAPIs)+len(httpAPIs))

	for _, api := range restAPIs {
		// Determine visibility based on endpoint type
		visibility := "Public"
		visibilityStyle := lipgloss.NewStyle().Foreground(theme.Success)
		if api.EndpointType == "PRIVATE" {
			visibility = "Private"
			visibilityStyle = lipgloss.NewStyle().Foreground(theme.Warning)
		}

		items = append(items, components.ListItem{
			ID:          "rest:" + api.ID,
			Title:       api.Name,
			Status:      fmt.Sprintf("REST %s", visibility),
			StatusStyle: visibilityStyle,
			Extra:       api.EndpointType,
		})
	}

	for _, api := range httpAPIs {
		// HTTP APIs are public by default
		visibility := "Public"
		visibilityStyle := lipgloss.NewStyle().Foreground(theme.Success)

		items = append(items, components.ListItem{
			ID:          "http:" + api.ID,
			Title:       api.Name,
			Status:      fmt.Sprintf("%s %s", api.ProtocolType, visibility),
			StatusStyle: visibilityStyle,
			Extra:       api.ApiEndpoint,
		})
	}

	m.apiGatewayList.SetItems(items)
	m.apiGatewayList.SetLoading(false)
	m.apiGatewayList.SetError(m.state.APIsError)
	m.apiGatewayList.SetEmptyMessage("No API Gateway APIs found")
	m.updateAPIGatewayDetails()
}

// updateAPIStagesList updates the API stages list with current data.
func (m *Model) updateAPIStagesList() {
	stages := m.state.FilteredAPIStages()
	items := make([]components.ListItem, len(stages))
	for i, stage := range stages {
		items[i] = components.ListItem{
			ID:          stage.Name,
			Title:       stage.Name,
			Status:      stage.DeploymentID,
			StatusStyle: lipgloss.NewStyle().Foreground(theme.Success),
			Extra:       stage.InvokeURL,
		}
	}
	m.apiStagesList.SetItems(items)
	m.apiStagesList.SetLoading(false)
	m.apiStagesList.SetError(m.state.APIStagesError)
	m.apiStagesList.SetEmptyMessage("No stages found for this API")
	m.updateAPIStageDetails()
}

// updateEC2List updates the EC2 instances list for jump host selection.
func (m *Model) updateEC2List() {
	instances := m.state.FilteredEC2Instances()
	items := make([]components.ListItem, len(instances))
	for i, inst := range instances {
		statusStyle := lipgloss.NewStyle().Foreground(theme.Success)
		if !inst.SSMManaged {
			statusStyle = lipgloss.NewStyle().Foreground(theme.Warning)
		}

		// Show VPC ID (truncated) for easier debugging
		vpcShort := inst.VpcID
		if len(vpcShort) > 12 {
			vpcShort = vpcShort[len(vpcShort)-12:] // Show last 12 chars like "vpc-0abc1234"
		}

		items[i] = components.ListItem{
			ID:          inst.InstanceID,
			Title:       inst.Name,
			Status:      vpcShort,
			StatusStyle: statusStyle,
			Extra:       inst.PrivateIPAddress,
		}
	}
	m.ec2List.SetItems(items)
	m.ec2List.SetLoading(false)
	m.ec2List.SetError(m.state.EC2InstancesError)
	m.ec2List.SetEmptyMessage("No SSM-managed EC2 instances found")
}

// updateContainerList updates the container list for container selection.
func (m *Model) updateContainerList() {
	containers := m.state.FilteredContainers()
	items := make([]components.ListItem, len(containers))
	for i, c := range containers {
		// Format ports as string
		ports := c.GetExposedPorts()
		portStr := ""
		if len(ports) > 0 {
			portStrs := make([]string, len(ports))
			for j, p := range ports {
				portStrs[j] = fmt.Sprintf("%d", p)
			}
			portStr = "[" + strings.Join(portStrs, ", ") + "]"
		}

		// Style: sidecars get dimmed, app containers get highlighted
		statusStyle := lipgloss.NewStyle().Foreground(theme.Success)
		if c.IsSidecar() {
			statusStyle = lipgloss.NewStyle().Foreground(theme.TextDim)
		}

		items[i] = components.ListItem{
			ID:          c.Name,
			Title:       c.Name,
			Status:      portStr,
			StatusStyle: statusStyle,
			Extra:       c.LastStatus,
		}
	}
	m.containerList.SetItems(items)
	m.containerList.SetLoading(false)
	m.containerList.SetEmptyMessage("No containers found")
}

// updateQueuesList updates the SQS queues list with current data.
func (m *Model) updateQueuesList() {
	queues := m.state.FilteredQueues()
	m.sqsTable.SetQueues(queues)
	m.sqsTable.SetLoading(false)
	m.sqsTable.SetError(m.state.QueuesError)
	m.updateQueueDetails()
}

// updateCurrentList updates the current list based on the active view.
func (m *Model) updateCurrentList() {
	switch m.state.View {
	case state.ViewMain:
		m.updateMainMenuList()
	case state.ViewStacks:
		m.updateStacksList()
	case state.ViewStackResources:
		m.updateStackResourcesList()
	case state.ViewClusters:
		m.updateClustersList()
	case state.ViewServices:
		m.updateServicesList()
	case state.ViewLambda:
		m.updateLambdaList()
	case state.ViewAPIGateway:
		m.updateAPIGatewayList()
	case state.ViewAPIStages:
		m.updateAPIStagesList()
	case state.ViewJumpHostSelect:
		m.updateEC2List()
	case state.ViewContainerSelect:
		m.updateContainerList()
	case state.ViewSQS:
		m.updateQueuesList()
	}
}

// updateContainerContext sets the container's title and context based on current view.
func (m *Model) updateContainerContext() {
	region := m.state.Region
	m.container.SetContext(region)
	// Don't use Container's loading/error - Lists handle their own states
	m.container.SetLoading(false)
	m.container.SetError(nil)

	switch m.state.View {
	case state.ViewMain:
		m.container.SetTitle("Main Menu")
		m.container.SetItemCount(0)
	case state.ViewStacks:
		m.container.SetTitle("CloudFormation Stacks")
		if m.state.StacksLoading {
			m.container.SetItemCount(0)
		} else {
			m.container.SetItemCount(len(m.state.FilteredStacks()))
		}
	case state.ViewStackResources:
		title := "Stack Resources"
		if m.state.SelectedStack != nil {
			title = m.state.SelectedStack.Name
		}
		m.container.SetTitle(title)
		m.container.SetItemCount(0)
	case state.ViewClusters:
		m.container.SetTitle("ECS Clusters")
		if m.state.ClustersLoading {
			m.container.SetItemCount(0)
		} else {
			m.container.SetItemCount(len(m.state.FilteredClusters()))
		}
	case state.ViewServices:
		title := "ECS Services"
		if m.state.SelectedCluster != nil {
			title = "ECS: " + m.state.SelectedCluster.Name
		}
		m.container.SetTitle(title)
		if m.state.ServicesLoading {
			m.container.SetItemCount(0)
		} else {
			m.container.SetItemCount(len(m.state.FilteredServices()))
		}
	case state.ViewLambda:
		m.container.SetTitle("Lambda Functions")
		if m.state.FunctionsLoading {
			m.container.SetItemCount(0)
		} else {
			m.container.SetItemCount(len(m.state.FilteredFunctions()))
		}
	case state.ViewAPIGateway:
		m.container.SetTitle("API Gateway")
		if m.state.APIsLoading {
			m.container.SetItemCount(0)
		} else {
			total := len(m.state.FilteredRestAPIs()) + len(m.state.FilteredHttpAPIs())
			m.container.SetItemCount(total)
		}
	case state.ViewAPIStages:
		title := "API Stages"
		if m.state.SelectedRestAPI != nil {
			title = "REST API: " + m.state.SelectedRestAPI.Name
		} else if m.state.SelectedHttpAPI != nil {
			title = "HTTP API: " + m.state.SelectedHttpAPI.Name
		}
		m.container.SetTitle(title)
		m.container.SetItemCount(len(m.state.APIStages))
	case state.ViewSQS:
		m.container.SetTitle("SQS Queues")
		if m.state.QueuesLoading {
			m.container.SetItemCount(0)
		} else {
			m.container.SetItemCount(len(m.state.FilteredQueues()))
		}
	case state.ViewJumpHostSelect:
		m.container.SetTitle("Select Jump Host")
		m.container.SetItemCount(len(m.state.EC2Instances))
	case state.ViewContainerSelect:
		m.container.SetTitle("Select Container")
		m.container.SetItemCount(len(m.state.PendingContainers))
	case state.ViewTunnels:
		m.container.SetTitle("Active Tunnels")
		m.container.SetItemCount(len(m.tunnelManager.GetTunnels()))
	case state.ViewCloudWatchLogs:
		m.container.SetTitle("CloudWatch Logs")
		m.container.SetItemCount(0)
	default:
		m.container.SetTitle("vaws")
		m.container.SetItemCount(0)
	}
}
