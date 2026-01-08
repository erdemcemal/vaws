// Package ui implements the terminal user interface using bubbletea.
package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"vaws/internal/aws"
	"vaws/internal/config"
	"vaws/internal/log"
	"vaws/internal/model"
	"vaws/internal/state"
	"vaws/internal/tunnel"
	"vaws/internal/ui/components"
)

// Layout constants for responsive design
const (
	// Minimum dimensions
	minWidth      = 40
	minHeight     = 10
	minWidthFull  = 80 // Minimum width for two-pane layout
	minHeightLogs = 20 // Minimum height to show logs panel

	// Panel proportions
	listPaneRatio    = 0.4 // List takes 40% of width in two-pane mode
	detailsPaneRatio = 0.6 // Details takes 60%
	logsHeight       = 8   // Fixed height for logs panel
)

// layoutMode represents the current layout mode based on terminal size
type layoutMode int

const (
	layoutTooSmall layoutMode = iota // Window too small to display
	layoutSingle                     // Single pane (list only)
	layoutFull                       // Full two-pane layout
)

// Model is the main bubbletea model.
type Model struct {
	// Dependencies
	client        *aws.Client
	logger        *log.Logger
	tunnelManager *tunnel.Manager
	apiGWManager  *tunnel.APIGatewayManager
	cfg           *config.Config

	// State
	state *state.State

	// UI components
	splash              *components.Splash
	mainMenuList        *components.List // Main menu with resource type selection
	stacksList          *components.List
	stackResourcesList  *components.List
	clustersList        *components.List // ECS clusters list
	serviceList         *components.List
	lambdaList          *components.List
	apiGatewayList      *components.List
	apiStagesList       *components.List
	ec2List             *components.List       // For jump host selection
	containerList       *components.List       // For container selection in port forwarding
	sqsTable            *components.SQSTable   // For SQS queues table view
	sqsDetails          *components.SQSDetails // For SQS queue details view
	details             *components.Details
	logs                *components.Logs
	tunnelsPanel        *components.TunnelsPanel
	cloudWatchLogsPanel *components.CloudWatchLogsPanel
	profileSelector     *components.ProfileSelector
	commandPalette      *components.CommandPalette
	refreshIndicator    *components.RefreshIndicator

	// Phase 1 UI components
	statusBar      *components.StatusBar
	container      *components.Container
	quickBar       *components.QuickBar
	regionSelector *components.RegionSelector

	// Filter input
	filterInput textinput.Model
	filtering   bool

	// Port forward input
	portInput          textinput.Model
	enteringPort       bool
	pendingPortForward *model.Service
	pendingLocalPort   int // Stores local port while selecting container

	// Lambda invocation input
	payloadInput          textinput.Model
	enteringPayload       bool
	pendingInvokeFunction *model.Function

	// API Gateway port forward
	pendingAPIGWPortForward *model.APIStage
	pendingAPIGWAPI         interface{} // *model.RestAPI or *model.HttpAPI

	// Key bindings
	keys KeyMap

	// Dimensions
	width  int
	height int

	// Status
	ready      bool
	showSplash bool

	// Profile selection mode (when no profile specified on command line)
	pendingRegion        string
	awaitingClientCreate bool

	// Track view before region selection to return to it
	viewBeforeRegionSelect state.View
}

// New creates a new Model.
func New(client *aws.Client, logger *log.Logger, version string) *Model {
	ti := textinput.New()
	ti.Placeholder = "Type to filter..."
	ti.CharLimit = 64

	portInput := textinput.New()
	portInput.Placeholder = "Enter port (or press Enter for random)"
	portInput.CharLimit = 5
	portInput.Width = 40

	payloadInput := textinput.New()
	payloadInput.Placeholder = "{} or press Enter for empty payload"
	payloadInput.CharLimit = 10000
	payloadInput.Width = 60

	// Load configuration
	cfg, _ := config.Load()

	statusBar := components.NewStatusBar()
	statusBar.SetVersion(version)
	quickBar := components.NewQuickBar()

	m := &Model{
		client:              client,
		logger:              logger,
		tunnelManager:       tunnel.NewManager(client.Profile(), client.Region()),
		apiGWManager:        tunnel.NewAPIGatewayManager(client.Profile(), client.Region()),
		cfg:                 cfg,
		state:               state.New(),
		splash:              components.NewSplash(version),
		mainMenuList:        components.NewList("AWS Resources"),
		stacksList:          components.NewList("CloudFormation Stacks"),
		stackResourcesList:  components.NewList("Stack Resources"),
		clustersList:        components.NewList("ECS Clusters"),
		serviceList:         components.NewList("ECS Services"),
		lambdaList:          components.NewList("Lambda Functions"),
		apiGatewayList:      components.NewList("API Gateway"),
		apiStagesList:       components.NewList("API Stages"),
		ec2List:             components.NewList("Select Jump Host"),
		containerList:       components.NewList("Select Container"),
		sqsTable:            components.NewSQSTable(),
		sqsDetails:          components.NewSQSDetails(),
		details:             components.NewDetails(),
		logs:                components.NewLogs(logger),
		tunnelsPanel:        components.NewTunnelsPanel(),
		cloudWatchLogsPanel: components.NewCloudWatchLogsPanel(),
		commandPalette:      components.NewCommandPalette(),
		refreshIndicator:    components.NewRefreshIndicator(),
		statusBar:           statusBar,
		container:           components.NewContainer(),
		quickBar:            quickBar,
		regionSelector:      components.NewRegionSelector(),
		filterInput:         ti,
		portInput:           portInput,
		payloadInput:        payloadInput,
		keys:                DefaultKeyMap(),
		showSplash:          true,
	}

	m.state.Profile = client.Profile()
	m.state.Region = client.Region()

	return m
}

// NewWithProfileSelection creates a new Model that shows profile selection first.
func NewWithProfileSelection(profiles []string, region string, logger *log.Logger, version string) *Model {
	ti := textinput.New()
	ti.Placeholder = "Type to filter..."
	ti.CharLimit = 64

	portInput := textinput.New()
	portInput.Placeholder = "Enter port (or press Enter for random)"
	portInput.CharLimit = 5
	portInput.Width = 40

	payloadInput := textinput.New()
	payloadInput.Placeholder = "{} or press Enter for empty payload"
	payloadInput.CharLimit = 10000
	payloadInput.Width = 60

	profileSelector := components.NewProfileSelector()
	profileSelector.SetProfiles(profiles)

	// Load configuration
	cfg, _ := config.Load()

	statusBar := components.NewStatusBar()
	statusBar.SetVersion(version)
	quickBar := components.NewQuickBar()

	m := &Model{
		client:              nil, // Will be created after profile selection
		logger:              logger,
		tunnelManager:       nil, // Will be created after profile selection
		apiGWManager:        nil, // Will be created after profile selection
		cfg:                 cfg,
		state:               state.New(),
		splash:              components.NewSplash(version),
		mainMenuList:        components.NewList("AWS Resources"),
		stacksList:          components.NewList("CloudFormation Stacks"),
		stackResourcesList:  components.NewList("Stack Resources"),
		clustersList:        components.NewList("ECS Clusters"),
		serviceList:         components.NewList("ECS Services"),
		lambdaList:          components.NewList("Lambda Functions"),
		apiGatewayList:      components.NewList("API Gateway"),
		apiStagesList:       components.NewList("API Stages"),
		ec2List:             components.NewList("Select Jump Host"),
		containerList:       components.NewList("Select Container"),
		sqsTable:            components.NewSQSTable(),
		sqsDetails:          components.NewSQSDetails(),
		details:             components.NewDetails(),
		logs:                components.NewLogs(logger),
		tunnelsPanel:        components.NewTunnelsPanel(),
		cloudWatchLogsPanel: components.NewCloudWatchLogsPanel(),
		profileSelector:     profileSelector,
		commandPalette:      components.NewCommandPalette(),
		refreshIndicator:    components.NewRefreshIndicator(),
		statusBar:           statusBar,
		container:           components.NewContainer(),
		quickBar:            quickBar,
		regionSelector:      components.NewRegionSelector(),
		filterInput:         ti,
		portInput:           portInput,
		payloadInput:        payloadInput,
		keys:                DefaultKeyMap(),
		showSplash:          false, // Skip splash, go straight to profile selection
		pendingRegion:       region,
	}

	m.state.View = state.ViewProfileSelect
	m.state.Profiles = profiles

	return m
}

// Init implements tea.Model.
func (m *Model) Init() tea.Cmd {
	// If in profile selection mode, don't load anything yet
	if m.state.View == state.ViewProfileSelect {
		return nil
	}
	// Start at main menu - don't load stacks automatically
	// User will select what to load from the main menu
	m.updateMainMenuList()
	return tea.Batch(
		m.splash.TickCmd(),           // Start splash animation
		m.refreshIndicator.TickCmd(), // Start auto-refresh timer
	)
}

// Update implements tea.Model.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle profile selection view
		if m.state.View == state.ViewProfileSelect {
			return m.handleProfileSelectKey(msg)
		}

		// Handle region selection view
		if m.state.View == state.ViewRegionSelect {
			return m.handleRegionSelectKey(msg)
		}

		// Any key dismisses splash (except during splash, q quits)
		if m.showSplash {
			if msg.String() == "q" || msg.String() == "ctrl+c" {
				return m, tea.Quit
			}
			m.showSplash = false
			return m, nil
		}

		// Handle command palette if active
		if m.commandPalette.IsActive() {
			result, cmd := m.commandPalette.Update(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			if result != nil {
				// Execute the command
				execCmd := m.executeCommand(result)
				if execCmd != nil {
					cmds = append(cmds, execCmd)
				}
			}
			return m, tea.Batch(cmds...)
		}

		// Track if we were already filtering before handling the key
		wasFiltering := m.filtering

		cmd := m.handleKeyMsg(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

		// Only pass keys to filter input if we were already filtering
		// (not if we just started filtering with this key)
		if wasFiltering && m.filtering {
			var inputCmd tea.Cmd
			m.filterInput, inputCmd = m.filterInput.Update(msg)
			if inputCmd != nil {
				cmds = append(cmds, inputCmd)
			}
		}

	case clientCreatedMsg:
		m.awaitingClientCreate = false
		if msg.err != nil {
			m.logger.Error("Failed to create AWS client: %v", msg.err)
			// Show error and go back to profile selection
			m.state.View = state.ViewProfileSelect
			return m, nil
		}
		// AWS client created successfully
		m.client = msg.client
		m.tunnelManager = tunnel.NewManager(msg.client.Profile(), msg.client.Region())
		m.apiGWManager = tunnel.NewAPIGatewayManager(msg.client.Profile(), msg.client.Region())
		m.state.Profile = msg.client.Profile()
		m.state.Region = msg.client.Region()
		m.state.View = state.ViewMain
		m.showSplash = true
		m.splash.SetLoading("Connected to " + msg.client.Region())
		m.updateComponentSizes()
		m.updateMainMenuList()
		// Show main menu - don't load stacks automatically
		return m, m.splash.TickCmd()

	case regionChangedMsg:
		if msg.err != nil {
			m.logger.Error("Failed to switch region: %v", msg.err)
			m.state.View = m.viewBeforeRegionSelect
			return m, nil
		}
		// Region changed successfully - update client and clear all cached data
		m.client = msg.client
		m.state.Region = msg.region
		m.tunnelManager = tunnel.NewManager(m.state.Profile, msg.region)
		m.apiGWManager = tunnel.NewAPIGatewayManager(m.state.Profile, msg.region)

		// Clear all cached data
		m.state.ClearStacks()
		m.state.ClearServices()
		m.state.ClearQueues()
		m.state.ClearFunctions()
		m.state.ClearAPIs()
		m.state.Clusters = nil
		m.state.ClustersError = nil

		m.logger.Info("Switched to region: %s", msg.region)

		// Go back to previous view and refresh its data
		m.state.View = m.viewBeforeRegionSelect
		return m, m.handleRefresh()

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		m.splash.SetSize(msg.Width, msg.Height)
		m.updateComponentSizes()

	case components.SplashTickMsg:
		// Update splash animation
		m.splash.Tick()
		m.splash.Spinner().Tick()

		// Auto-dismiss splash when animation completes
		if m.showSplash && m.splash.IsReady() {
			m.showSplash = false
		}

		// Keep splash animation ticking while shown
		if m.showSplash {
			cmds = append(cmds, m.splash.TickCmd())
		}

	case components.SpinnerTickMsg:
		// Update list spinners for loading states
		m.stacksList.Spinner().Tick()
		m.clustersList.Spinner().Tick()
		m.serviceList.Spinner().Tick()
		m.sqsTable.Spinner().Tick()
		m.lambdaList.Spinner().Tick()
		m.apiGatewayList.Spinner().Tick()
		m.ec2List.Spinner().Tick()

		// Keep ticking while anything is loading
		if m.state.StacksLoading || m.state.ClustersLoading || m.state.ServicesLoading || m.state.QueuesLoading ||
			m.state.FunctionsLoading || m.state.APIsLoading || m.state.EC2InstancesLoading {
			cmds = append(cmds, m.stacksList.Spinner().TickCmd())
		}

	case components.AutoRefreshTickMsg:
		// Auto-refresh current view data
		if m.state.AutoRefresh && !m.showSplash && m.client != nil {
			m.refreshIndicator.Tick()
			m.refreshIndicator.SetRefreshing(true)

			// Refresh based on current view
			var refreshCmd tea.Cmd
			switch m.state.View {
			case state.ViewStacks:
				refreshCmd = m.loadStacks()
			case state.ViewServices:
				refreshCmd = m.loadServices()
			}

			if refreshCmd != nil {
				cmds = append(cmds, refreshCmd)
			}

			// Schedule next refresh
			cmds = append(cmds, m.refreshIndicator.TickCmd())
		}

	case stacksLoadedMsg:
		m.state.StacksLoading = false
		m.refreshIndicator.SetRefreshing(false)
		if msg.err != nil {
			m.state.StacksError = msg.err
			m.logger.Error("Failed to load stacks: %v", msg.err)
			m.splash.SetLoading("Error loading stacks")
		} else {
			m.state.Stacks = msg.stacks
			m.state.StacksError = nil
			m.logger.Info("Loaded %d CloudFormation stacks", len(msg.stacks))
			m.splash.SetLoading(fmt.Sprintf("Loaded %d stacks", len(msg.stacks)))
			// Auto-dismiss splash when stacks loaded successfully
			if m.showSplash {
				m.showSplash = false
			}
		}
		m.updateStacksList()

	case servicesLoadedMsg:
		m.state.ServicesLoading = false
		m.refreshIndicator.SetRefreshing(false)
		if msg.err != nil {
			m.state.ServicesError = msg.err
			m.logger.Error("Failed to load services: %v", msg.err)
		} else {
			m.state.Services = msg.services
			m.state.ServicesError = nil
		}
		m.updateServicesList()

	case functionsLoadedMsg:
		m.state.FunctionsLoading = false
		m.refreshIndicator.SetRefreshing(false)
		if msg.err != nil {
			m.state.FunctionsError = msg.err
			m.logger.Error("Failed to load Lambda functions: %v", msg.err)
		} else {
			m.state.Functions = msg.functions
			m.state.FunctionsError = nil
		}
		m.updateLambdaList()

	case restAPIsLoadedMsg:
		m.state.APIsLoading = false
		m.refreshIndicator.SetRefreshing(false)
		if msg.err != nil {
			m.state.APIsError = msg.err
			m.logger.Error("Failed to load REST APIs: %v", msg.err)
		} else {
			m.state.RestAPIs = msg.apis
			m.state.APIsError = nil
		}
		m.updateAPIGatewayList()

	case httpAPIsLoadedMsg:
		// This is loaded together with REST APIs
		if msg.err != nil {
			m.logger.Error("Failed to load HTTP APIs: %v", msg.err)
		} else {
			m.state.HttpAPIs = msg.apis
		}
		m.updateAPIGatewayList()

	case ec2InstancesLoadedMsg:
		m.state.EC2InstancesLoading = false
		m.ec2List.SetLoading(false)
		if msg.err != nil {
			m.state.EC2InstancesError = msg.err
			m.logger.Error("Failed to load EC2 instances: %v", msg.err)
		} else {
			m.state.EC2Instances = msg.instances
			m.state.EC2InstancesError = nil
			m.logger.Info("Loaded %d SSM-managed EC2 instances", len(msg.instances))
		}
		m.updateEC2List()

	case apiStagesLoadedMsg:
		m.state.APIStagesLoading = false
		if msg.err != nil {
			m.state.APIStagesError = msg.err
			m.logger.Error("Failed to load API stages: %v", msg.err)
		} else {
			m.state.APIStages = msg.stages
			m.state.APIStagesError = nil
		}
		m.updateAPIStagesList()

	case tasksLoadedMsg:
		if msg.err != nil {
			m.logger.Error("Failed to load tasks: %v", msg.err)
			// Show logs panel so user can see the error
			m.state.ShowLogs = true
			m.updateComponentSizes()
			return m, nil
		}
		if len(msg.tasks) == 0 {
			m.logger.Error("No running tasks found for service '%s'. Make sure the service has running tasks.", msg.service.Name)
			m.state.ShowLogs = true
			m.updateComponentSizes()
			return m, nil
		}

		task := msg.tasks[0]

		// Get containers with RuntimeID
		var containersWithRuntime []model.Container
		for _, c := range task.Containers {
			if c.RuntimeID != "" {
				containersWithRuntime = append(containersWithRuntime, c)
			}
		}

		if len(containersWithRuntime) == 0 {
			m.logger.Error("No container with RuntimeID found. Is ECS Exec enabled for service '%s'? Task: %s", msg.service.Name, task.TaskID)
			m.state.ShowLogs = true
			m.updateComponentSizes()
			return m, nil
		}

		// If only one container, use it directly
		if len(containersWithRuntime) == 1 {
			container := &containersWithRuntime[0]
			remotePort := container.GetBestPort()
			m.logger.Info("Selected container '%s' for tunnel, port %d", container.Name, remotePort)
			cmds = append(cmds, m.startTunnel(msg.service, task, *container, remotePort))
			return m, tea.Batch(cmds...)
		}

		// Multiple containers - show container picker
		m.logger.Info("Found %d containers - select one for port forwarding", len(containersWithRuntime))
		m.state.PendingContainerService = &msg.service
		m.state.PendingContainerTask = &task
		m.state.PendingContainers = containersWithRuntime
		m.pendingLocalPort = 0 // Use random port
		m.state.View = state.ViewContainerSelect
		m.updateContainerList()

	case tasksLoadedMsgWithPort:
		if msg.err != nil {
			m.logger.Error("Failed to load tasks: %v", msg.err)
			m.state.ShowLogs = true
			m.updateComponentSizes()
			return m, nil
		}
		if len(msg.tasks) == 0 {
			m.logger.Error("No running tasks found for service '%s'. Make sure the service has running tasks.", msg.service.Name)
			m.state.ShowLogs = true
			m.updateComponentSizes()
			return m, nil
		}

		task := msg.tasks[0]

		// Get containers with RuntimeID
		var containersWithRuntime []model.Container
		for _, c := range task.Containers {
			if c.RuntimeID != "" {
				containersWithRuntime = append(containersWithRuntime, c)
			}
		}

		if len(containersWithRuntime) == 0 {
			m.logger.Error("No container with RuntimeID found. Is ECS Exec enabled for service '%s'? Task: %s", msg.service.Name, task.TaskID)
			m.state.ShowLogs = true
			m.updateComponentSizes()
			return m, nil
		}

		// If only one container, use it directly
		if len(containersWithRuntime) == 1 {
			container := &containersWithRuntime[0]
			remotePort := container.GetBestPort()
			localPortStr := "random"
			if msg.localPort > 0 {
				localPortStr = fmt.Sprintf("%d", msg.localPort)
			}
			m.logger.Info("Selected container '%s' for tunnel (local: %s, remote: %d)", container.Name, localPortStr, remotePort)
			cmds = append(cmds, m.startTunnelWithPort(msg.service, task, *container, remotePort, msg.localPort))
			return m, tea.Batch(cmds...)
		}

		// Multiple containers - show container picker
		m.logger.Info("Found %d containers - select one for port forwarding", len(containersWithRuntime))
		m.state.PendingContainerService = &msg.service
		m.state.PendingContainerTask = &task
		m.state.PendingContainers = containersWithRuntime
		m.pendingLocalPort = msg.localPort
		m.state.View = state.ViewContainerSelect
		m.updateContainerList()

	case tasksLoadedMsgForRestart:
		if msg.err != nil {
			m.logger.Error("Failed to load tasks for restart: %v", msg.err)
			m.state.ShowLogs = true
			m.updateComponentSizes()
			return m, nil
		}
		if len(msg.tasks) == 0 {
			m.logger.Error("No running tasks found for service '%s'. Cannot restart tunnel.", msg.tunnelInfo.ServiceName)
			m.state.ShowLogs = true
			m.updateComponentSizes()
			return m, nil
		}
		// Use the first task and find the original container by name
		task := msg.tasks[0]
		var container *model.Container
		// Try to find the original container by name
		for i := range task.Containers {
			if task.Containers[i].Name == msg.tunnelInfo.ContainerName && task.Containers[i].RuntimeID != "" {
				container = &task.Containers[i]
				break
			}
		}
		// Fall back to best container if original not found
		if container == nil {
			container = findBestContainer(task.Containers)
		}
		if container != nil {
			// Create a service-like object for the tunnel start
			service := model.Service{
				Name:        msg.tunnelInfo.ServiceName,
				ClusterARN:  msg.tunnelInfo.ClusterARN,
				ClusterName: msg.tunnelInfo.ClusterName,
			}
			m.logger.Info("Restarting tunnel for '%s' (local: %d, remote: %d)...", msg.tunnelInfo.ServiceName, msg.tunnelInfo.LocalPort, msg.tunnelInfo.RemotePort)
			cmds = append(cmds, m.startTunnelWithPort(service, task, *container, msg.tunnelInfo.RemotePort, msg.tunnelInfo.LocalPort))
		} else {
			m.logger.Error("No container with RuntimeID found for restart. Task: %s", task.TaskID)
			m.state.ShowLogs = true
			m.updateComponentSizes()
		}

	case tunnelStartedMsg:
		if msg.err != nil {
			m.logger.Error("Failed to start tunnel: %v", msg.err)
		} else if msg.tunnel != nil {
			m.logger.Info("Tunnel started: localhost:%d -> %s:%d",
				msg.tunnel.LocalPort, msg.tunnel.ServiceName, msg.tunnel.RemotePort)
		}
		m.updateTunnelsPanel()
		// Switch to tunnels view to show the new tunnel
		m.state.View = state.ViewTunnels

	case apiGWTunnelStartedMsg:
		if msg.err != nil {
			m.logger.Error("Failed to start API Gateway tunnel: %v", msg.err)
			m.state.ShowLogs = true
			m.updateComponentSizes()
		} else if msg.tunnel != nil {
			m.logger.Info("API Gateway tunnel started: localhost:%d -> %s (%s)",
				msg.tunnel.LocalPort, msg.tunnel.APIName, msg.tunnel.StageName)
			// Switch to tunnels view to show the new tunnel
			m.state.View = state.ViewTunnels
		}
		m.updateTunnelsPanel()

	case jumpHostFoundMsg:
		if msg.err != nil {
			m.logger.Error("Failed to find jump host: %v", msg.err)
			m.state.ShowLogs = true
			m.updateComponentSizes()
			return m, nil
		}
		m.logger.Info("Found jump host: %s (%s) in VPC %s", msg.jumpHost.Name, msg.jumpHost.InstanceID, msg.jumpHost.VpcID)

		// Log VPCs that have execute-api endpoints
		if len(msg.vpcsWithEndpoints) > 0 {
			m.logger.Info("VPCs with execute-api endpoints: %v", msg.vpcsWithEndpoints)
		} else {
			m.logger.Warn("No execute-api VPC endpoints found in this account!")
		}

		if msg.vpcEndpoint != nil {
			m.logger.Info("Found VPC endpoint: %s (%s)", msg.vpcEndpoint.VpcEndpointID, msg.vpcEndpoint.ServiceName)
			if len(msg.vpcEndpoint.DNSEntries) > 0 {
				m.logger.Info("VPC endpoint DNS: %s", msg.vpcEndpoint.DNSEntries[0])
			}
		} else {
			if len(msg.vpcsWithEndpoints) > 0 {
				m.logger.Error("Jump host VPC (%s) does NOT have execute-api endpoint!", msg.jumpHost.VpcID)
				m.logger.Error("Execute-api endpoints exist in: %v", msg.vpcsWithEndpoints)
				m.logger.Error("You need a jump host in one of those VPCs, or configure vpc_endpoint_id in ~/.vaws/config.yaml")
			} else {
				m.logger.Error("No execute-api VPC endpoint exists in this account!")
				m.logger.Error("Create one to access private API Gateways, or configure vpc_endpoint_id for cross-account access")
			}
		}
		// Start the private API Gateway tunnel
		return m, m.startPrivateAPIGWTunnel(msg.api, msg.stage, msg.jumpHost, msg.vpcEndpoint, msg.localPort)

	case tunnelRefreshMsg:
		m.updateTunnelsPanel()

	case cloudWatchLogConfigsLoadedMsg:
		if msg.err != nil {
			m.logger.Error("Failed to load log configs: %v", msg.err)
			m.state.ShowLogs = true
			m.updateComponentSizes()
			return m, nil
		}

		if len(msg.configs) == 0 {
			m.logger.Error("No CloudWatch log configurations found. Is the service using awslogs driver?")
			m.state.ShowLogs = true
			m.updateComponentSizes()
			return m, nil
		}

		m.state.CloudWatchLogConfigs = msg.configs
		m.state.CloudWatchServiceContext = &msg.service
		m.state.CloudWatchTaskContext = &msg.task
		m.state.View = state.ViewCloudWatchLogs
		m.state.CloudWatchLogsStreaming = true
		m.state.CloudWatchLastFetchTime = 0

		m.cloudWatchLogsPanel.SetContainers(msg.configs)
		m.cloudWatchLogsPanel.SetContext(msg.service.Name, msg.task.TaskID)
		m.cloudWatchLogsPanel.SetStreaming(true)
		m.cloudWatchLogsPanel.Clear()

		// Start fetching logs
		return m, tea.Batch(
			m.fetchCloudWatchLogs(),
			m.cloudWatchLogsPanel.TickCmd(),
		)

	case cloudWatchLogsLoadedMsg:
		if msg.err != nil {
			m.logger.Error("Failed to fetch CloudWatch logs: %v", msg.err)
			return m, nil
		}

		m.state.CloudWatchLastFetchTime = msg.lastTimestamp

		if len(m.state.CloudWatchLogs) == 0 {
			m.state.CloudWatchLogs = msg.entries
			m.cloudWatchLogsPanel.SetEntries(msg.entries)
		} else {
			m.state.CloudWatchLogs = append(m.state.CloudWatchLogs, msg.entries...)
			m.cloudWatchLogsPanel.AppendEntries(msg.entries)
		}

	case components.CloudWatchLogsTickMsg:
		// Continue polling if still in CloudWatch logs view and streaming
		if m.state.View == state.ViewCloudWatchLogs && m.state.CloudWatchLogsStreaming {
			var fetchCmd tea.Cmd
			if m.state.CloudWatchLambdaContext != nil {
				// Lambda logs - query across all streams
				logGroup := fmt.Sprintf("/aws/lambda/%s", m.state.CloudWatchLambdaContext.Name)
				fetchCmd = m.fetchLambdaCloudWatchLogs(logGroup)
			} else {
				// ECS container logs - query specific stream
				fetchCmd = m.fetchCloudWatchLogs()
			}
			return m, tea.Batch(
				fetchCmd,
				m.cloudWatchLogsPanel.TickCmd(),
			)
		}

	case queuesLoadedMsg:
		m.state.QueuesLoading = false
		m.sqsTable.SetLoading(false)
		m.refreshIndicator.SetRefreshing(false)
		if msg.err != nil {
			m.state.QueuesError = msg.err
			m.logger.Error("Failed to load SQS queues: %v", msg.err)
		} else {
			m.state.Queues = msg.queues
			m.state.QueuesError = nil
			m.logger.Info("Loaded %d SQS queues", len(msg.queues))
		}
		m.updateQueuesList()

	case clustersLoadedMsg:
		m.state.ClustersLoading = false
		m.refreshIndicator.SetRefreshing(false)
		if msg.err != nil {
			m.state.ClustersError = msg.err
			m.logger.Error("Failed to load ECS clusters: %v", msg.err)
		} else {
			m.state.Clusters = msg.clusters
			m.state.ClustersError = nil
			m.logger.Info("Loaded %d ECS clusters", len(msg.clusters))
		}
		m.updateClustersList()

	case lambdaInvocationResultMsg:
		m.state.LambdaInvocationLoading = false
		if msg.err != nil {
			m.state.LambdaInvocationError = msg.err
			m.logger.Error("Lambda invocation failed: %v", msg.err)
		} else {
			m.state.LambdaInvocationResult = msg.result
			if msg.result.FunctionError != "" {
				m.logger.Warn("Lambda %s returned error: %s (Status: %d, Duration: %v)",
					msg.result.FunctionName, msg.result.FunctionError, msg.result.StatusCode, msg.result.Duration)
			} else {
				m.logger.Info("Lambda %s invoked successfully (Status: %d, Duration: %v)",
					msg.result.FunctionName, msg.result.StatusCode, msg.result.Duration)
			}
		}
		m.updateLambdaDetails()

	default:
		// Pass other messages to filter input if filtering
		if m.filtering {
			var cmd tea.Cmd
			m.filterInput, cmd = m.filterInput.Update(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
		// Pass other messages to port input if entering port
		if m.enteringPort {
			var cmd tea.Cmd
			m.portInput, cmd = m.portInput.Update(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
		// Pass other messages to payload input if entering payload
		if m.enteringPayload {
			var cmd tea.Cmd
			m.payloadInput, cmd = m.payloadInput.Update(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
	}

	return m, tea.Batch(cmds...)
}
