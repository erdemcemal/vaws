// Package ui implements the terminal user interface using bubbletea.
package ui

import (
	"context"
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"vaws/internal/aws"
	"vaws/internal/config"
	"vaws/internal/log"
	"vaws/internal/model"
	"vaws/internal/state"
	"vaws/internal/tunnel"
	"vaws/internal/ui/components"
	"vaws/internal/ui/theme"
)

// Messages for bubbletea.
type (
	// stacksLoadedMsg is sent when stacks are loaded.
	stacksLoadedMsg struct {
		stacks []model.Stack
		err    error
	}

	// servicesLoadedMsg is sent when services are loaded.
	servicesLoadedMsg struct {
		services []model.Service
		err      error
	}

	// functionsLoadedMsg is sent when Lambda functions are loaded.
	functionsLoadedMsg struct {
		functions []model.Function
		err       error
	}

	// restAPIsLoadedMsg is sent when REST APIs are loaded.
	restAPIsLoadedMsg struct {
		apis []model.RestAPI
		err  error
	}

	// httpAPIsLoadedMsg is sent when HTTP APIs are loaded.
	httpAPIsLoadedMsg struct {
		apis []model.HttpAPI
		err  error
	}

	// apiStagesLoadedMsg is sent when API stages are loaded.
	apiStagesLoadedMsg struct {
		stages []model.APIStage
		err    error
	}

	// tasksLoadedMsg is sent when tasks are loaded for a service.
	tasksLoadedMsg struct {
		service model.Service
		tasks   []model.Task
		err     error
	}

	// tasksLoadedMsgWithPort is sent when tasks are loaded with a custom port.
	tasksLoadedMsgWithPort struct {
		service   model.Service
		tasks     []model.Task
		err       error
		localPort int
	}

	// tasksLoadedMsgForRestart is sent when tasks are loaded for tunnel restart.
	tasksLoadedMsgForRestart struct {
		tunnelInfo model.Tunnel
		tasks      []model.Task
		err        error
	}

	// tunnelStartedMsg is sent when a tunnel is started.
	tunnelStartedMsg struct {
		tunnel *model.Tunnel
		err    error
	}

	// apiGWTunnelStartedMsg is sent when an API Gateway tunnel is started.
	apiGWTunnelStartedMsg struct {
		tunnel *model.APIGatewayTunnel
		err    error
	}

	// jumpHostFoundMsg is sent when a jump host is found for private API Gateway.
	jumpHostFoundMsg struct {
		jumpHost       *model.EC2Instance
		vpcEndpoint    *model.VpcEndpoint
		vpcEndpointErr error
		stage          model.APIStage
		api            interface{}
		localPort      int
		err            error
	}

	// ec2InstancesLoadedMsg is sent when EC2 instances are loaded for jump host selection.
	ec2InstancesLoadedMsg struct {
		instances []model.EC2Instance
		err       error
	}

	// tunnelRefreshMsg triggers a refresh of the tunnel list.
	tunnelRefreshMsg struct{}

	// errMsg is sent when an error occurs.
	errMsg struct {
		err error
	}

	// clientCreatedMsg is sent when AWS client is created after profile selection.
	clientCreatedMsg struct {
		client *aws.Client
		err    error
	}
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
	splash             *components.Splash
	header             *components.Header
	menuBar            *components.MenuBarWithCrumbs
	footer             *components.Footer
	stacksList         *components.List
	stackResourcesList *components.List
	serviceList        *components.List
	lambdaList         *components.List
	apiGatewayList     *components.List
	apiStagesList      *components.List
	ec2List            *components.List // For jump host selection
	details            *components.Details
	logs               *components.Logs
	tunnelsPanel       *components.TunnelsPanel
	profileSelector    *components.ProfileSelector

	// k9s-inspired components
	commandPalette   *components.CommandPalette
	actionBar        *components.ActionBar
	refreshIndicator *components.RefreshIndicator
	xrayTree         *components.XRayTree

	// Filter input
	filterInput textinput.Model
	filtering   bool

	// Port forward input
	portInput          textinput.Model
	enteringPort       bool
	pendingPortForward *model.Service

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
}

// New creates a new Model.
func New(client *aws.Client, logger *log.Logger) *Model {
	ti := textinput.New()
	ti.Placeholder = "Type to filter..."
	ti.CharLimit = 64

	portInput := textinput.New()
	portInput.Placeholder = "Enter port (or press Enter for random)"
	portInput.CharLimit = 5
	portInput.Width = 40

	// Load configuration
	cfg, _ := config.Load()

	m := &Model{
		client:             client,
		logger:             logger,
		tunnelManager:      tunnel.NewManager(client.Profile(), client.Region()),
		apiGWManager:       tunnel.NewAPIGatewayManager(client.Profile(), client.Region()),
		cfg:                cfg,
		state:              state.New(),
		splash:             components.NewSplash("v0.4.2"),
		header:             components.NewHeader(),
		menuBar:            components.NewMenuBarWithCrumbs(),
		footer:             components.NewFooter(),
		stacksList:         components.NewList("CloudFormation Stacks"),
		stackResourcesList: components.NewList("Stack Resources"),
		serviceList:        components.NewList("ECS Services"),
		lambdaList:         components.NewList("Lambda Functions"),
		apiGatewayList:     components.NewList("API Gateway"),
		apiStagesList:      components.NewList("API Stages"),
		ec2List:            components.NewList("Select Jump Host"),
		details:            components.NewDetails(),
		logs:               components.NewLogs(logger),
		tunnelsPanel:       components.NewTunnelsPanel(),
		commandPalette:     components.NewCommandPalette(),
		actionBar:          components.NewActionBar(),
		refreshIndicator:   components.NewRefreshIndicator(),
		xrayTree:           components.NewXRayTree(),
		filterInput:        ti,
		portInput:          portInput,
		keys:               DefaultKeyMap(),
		showSplash:         true,
	}

	m.state.Profile = client.Profile()
	m.state.Region = client.Region()

	return m
}

// NewWithProfileSelection creates a new Model that shows profile selection first.
func NewWithProfileSelection(profiles []string, region string, logger *log.Logger) *Model {
	ti := textinput.New()
	ti.Placeholder = "Type to filter..."
	ti.CharLimit = 64

	portInput := textinput.New()
	portInput.Placeholder = "Enter port (or press Enter for random)"
	portInput.CharLimit = 5
	portInput.Width = 40

	profileSelector := components.NewProfileSelector()
	profileSelector.SetProfiles(profiles)

	// Load configuration
	cfg, _ := config.Load()

	m := &Model{
		client:             nil, // Will be created after profile selection
		logger:             logger,
		tunnelManager:      nil, // Will be created after profile selection
		apiGWManager:       nil, // Will be created after profile selection
		cfg:                cfg,
		state:              state.New(),
		splash:             components.NewSplash("v0.4.2"),
		header:             components.NewHeader(),
		menuBar:            components.NewMenuBarWithCrumbs(),
		footer:             components.NewFooter(),
		stacksList:         components.NewList("CloudFormation Stacks"),
		stackResourcesList: components.NewList("Stack Resources"),
		serviceList:        components.NewList("ECS Services"),
		lambdaList:         components.NewList("Lambda Functions"),
		apiGatewayList:     components.NewList("API Gateway"),
		apiStagesList:      components.NewList("API Stages"),
		ec2List:            components.NewList("Select Jump Host"),
		details:            components.NewDetails(),
		logs:               components.NewLogs(logger),
		tunnelsPanel:       components.NewTunnelsPanel(),
		profileSelector:    profileSelector,
		commandPalette:     components.NewCommandPalette(),
		actionBar:          components.NewActionBar(),
		refreshIndicator:   components.NewRefreshIndicator(),
		xrayTree:           components.NewXRayTree(),
		filterInput:        ti,
		portInput:          portInput,
		keys:               DefaultKeyMap(),
		showSplash:         false, // Skip splash, go straight to profile selection
		pendingRegion:      region,
	}

	m.state.View = state.ViewProfileSelect
	m.state.Profiles = profiles

	return m
}

// Init implements tea.Model.
func (m *Model) Init() tea.Cmd {
	// If in profile selection mode, don't load stacks yet
	if m.state.View == state.ViewProfileSelect {
		return nil
	}
	return tea.Batch(
		m.loadStacks(),
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
		m.state.View = state.ViewStacks
		m.showSplash = true
		m.updateComponentSizes()
		// Start loading stacks
		return m, tea.Batch(m.loadStacks(), m.splash.TickCmd())

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

		// Keep splash animation ticking while shown
		if m.showSplash {
			cmds = append(cmds, m.splash.TickCmd())
		}

	case components.SpinnerTickMsg:
		// Update list spinners for loading states
		m.stacksList.Spinner().Tick()
		m.serviceList.Spinner().Tick()

		// Keep ticking while anything is loading
		if m.state.StacksLoading || m.state.ServicesLoading {
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
			m.splash.SetLoading(fmt.Sprintf("Loaded %d stacks", len(msg.stacks)))
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
		// Use the first task and first container with a runtime ID
		task := msg.tasks[0]
		foundContainer := false
		for _, container := range task.Containers {
			if container.RuntimeID != "" {
				foundContainer = true
				// Default to port 80, but could be made configurable
				remotePort := 80
				if len(container.NetworkBindings) > 0 {
					remotePort = container.NetworkBindings[0].ContainerPort
				}
				m.logger.Info("Found container '%s' with RuntimeID, starting tunnel to port %d...", container.Name, remotePort)
				cmds = append(cmds, m.startTunnel(msg.service, task, container, remotePort))
				break
			}
		}
		if !foundContainer {
			m.logger.Error("No container with RuntimeID found. Is ECS Exec enabled for service '%s'? Task: %s", msg.service.Name, task.TaskID)
			m.state.ShowLogs = true
			m.updateComponentSizes()
		}

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
		// Use the first task and first container with a runtime ID
		task := msg.tasks[0]
		foundContainer := false
		for _, container := range task.Containers {
			if container.RuntimeID != "" {
				foundContainer = true
				remotePort := 80
				if len(container.NetworkBindings) > 0 {
					remotePort = container.NetworkBindings[0].ContainerPort
				}
				localPortStr := "random"
				if msg.localPort > 0 {
					localPortStr = fmt.Sprintf("%d", msg.localPort)
				}
				m.logger.Info("Found container '%s', starting tunnel (local: %s, remote: %d)...", container.Name, localPortStr, remotePort)
				cmds = append(cmds, m.startTunnelWithPort(msg.service, task, container, remotePort, msg.localPort))
				break
			}
		}
		if !foundContainer {
			m.logger.Error("No container with RuntimeID found. Is ECS Exec enabled for service '%s'? Task: %s", msg.service.Name, task.TaskID)
			m.state.ShowLogs = true
			m.updateComponentSizes()
		}

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
		// Use the first task and find a container with runtime ID
		task := msg.tasks[0]
		foundContainer := false
		for _, container := range task.Containers {
			if container.RuntimeID != "" {
				foundContainer = true
				// Create a service-like object for the tunnel start
				service := model.Service{
					Name:        msg.tunnelInfo.ServiceName,
					ClusterARN:  msg.tunnelInfo.ClusterARN,
					ClusterName: msg.tunnelInfo.ClusterName,
				}
				m.logger.Info("Restarting tunnel for '%s' (local: %d, remote: %d)...", msg.tunnelInfo.ServiceName, msg.tunnelInfo.LocalPort, msg.tunnelInfo.RemotePort)
				cmds = append(cmds, m.startTunnelWithPort(service, task, container, msg.tunnelInfo.RemotePort, msg.tunnelInfo.LocalPort))
				break
			}
		}
		if !foundContainer {
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
		m.updateMenuBar()

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
			m.updateMenuBar()
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
		if msg.vpcEndpoint != nil {
			m.logger.Info("Found VPC endpoint: %s (%s)", msg.vpcEndpoint.VpcEndpointID, msg.vpcEndpoint.ServiceName)
			if len(msg.vpcEndpoint.DNSEntries) > 0 {
				m.logger.Info("VPC endpoint DNS: %s", msg.vpcEndpoint.DNSEntries[0])
			}
		} else {
			if msg.vpcEndpointErr != nil {
				m.logger.Warn("VPC endpoint lookup: %v", msg.vpcEndpointErr)
			} else {
				m.logger.Warn("No VPC endpoint found for execute-api in VPC %s", msg.jumpHost.VpcID)
			}
		}
		// Start the private API Gateway tunnel
		return m, m.startPrivateAPIGWTunnel(msg.api, msg.stage, msg.jumpHost, msg.vpcEndpoint, msg.localPort)

	case tunnelRefreshMsg:
		m.updateTunnelsPanel()

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
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) handleKeyMsg(msg tea.KeyMsg) tea.Cmd {
	// Handle filter mode separately
	if m.filtering {
		return m.handleFilterKey(msg)
	}

	// Handle port input mode separately
	if m.enteringPort {
		return m.handlePortInputKey(msg)
	}

	switch {
	case matchKey(msg, m.keys.Quit):
		m.tunnelManager.StopAllTunnels()
		return tea.Quit

	case matchKey(msg, m.keys.Up):
		m.moveCursorUp()

	case matchKey(msg, m.keys.Down):
		m.moveCursorDown()

	case matchKey(msg, m.keys.Top):
		m.moveCursorTop()

	case matchKey(msg, m.keys.Bottom):
		m.moveCursorBottom()

	case matchKey(msg, m.keys.Enter), matchKey(msg, m.keys.Right):
		return m.handleEnter()

	case matchKey(msg, m.keys.Back), matchKey(msg, m.keys.Left):
		m.handleBack()

	case matchKey(msg, m.keys.Filter):
		if m.state.View != state.ViewTunnels {
			m.startFiltering()
		}

	case matchKey(msg, m.keys.Logs):
		m.state.ToggleLogs()
		m.updateComponentSizes()

	case matchKey(msg, m.keys.PortForward):
		return m.handlePortForward()

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
			m.updateTunnelsPanel()
		}

	case msg.String() == ":":
		// Open command palette (k9s-style)
		m.commandPalette.SetWidth(m.width)
		return m.commandPalette.Activate()

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
	}

	return nil
}

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

func (m *Model) startFiltering() {
	m.filtering = true
	m.filterInput.SetValue(m.state.FilterText)
	m.filterInput.Focus()
}

func (m *Model) moveCursorUp() {
	switch m.state.View {
	case state.ViewStacks:
		m.stacksList.Up()
		m.updateStackDetails()
	case state.ViewStackResources:
		m.stackResourcesList.Up()
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
	case state.ViewTunnels:
		m.tunnelsPanel.Up()
	}
}

func (m *Model) moveCursorDown() {
	switch m.state.View {
	case state.ViewStacks:
		m.stacksList.Down()
		m.updateStackDetails()
	case state.ViewStackResources:
		m.stackResourcesList.Down()
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
	case state.ViewTunnels:
		m.tunnelsPanel.Down()
	}
}

func (m *Model) moveCursorTop() {
	switch m.state.View {
	case state.ViewStacks:
		m.stacksList.Top()
		m.updateStackDetails()
	case state.ViewStackResources:
		m.stackResourcesList.Top()
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
	}
}

func (m *Model) moveCursorBottom() {
	switch m.state.View {
	case state.ViewStacks:
		m.stacksList.Bottom()
		m.updateStackDetails()
	case state.ViewStackResources:
		m.stackResourcesList.Bottom()
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
	}
}

func (m *Model) handleEnter() tea.Cmd {
	switch m.state.View {
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
				m.updateMenuBar()
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
			m.updateMenuBar()
			return m.loadServices()
		case "lambda-functions":
			m.state.View = state.ViewLambda
			m.updateMenuBar()
			return m.loadFunctions()
		case "api-gateway":
			m.state.View = state.ViewAPIGateway
			m.updateMenuBar()
			return m.loadAPIs()
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
					m.updateMenuBar()
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
					m.updateMenuBar()
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
	}
	return nil
}

func (m *Model) handleBack() {
	switch m.state.View {
	case state.ViewStackResources:
		m.state.View = state.ViewStacks
		m.state.SelectedStack = nil
		m.state.FilterText = ""
		m.filterInput.SetValue("")
		m.updateMenuBar()
		m.updateStacksList()
	case state.ViewServices:
		m.state.View = state.ViewStackResources
		m.state.FilterText = ""
		m.filterInput.SetValue("")
		m.state.ClearServices()
		m.updateMenuBar()
		m.updateStackResourcesList()
	case state.ViewLambda:
		// If we came from stack resources, go back there
		if m.state.SelectedStack != nil {
			m.state.View = state.ViewStackResources
			m.state.FilterText = ""
			m.filterInput.SetValue("")
			m.state.ClearFunctions()
			m.updateMenuBar()
			m.updateStackResourcesList()
		}
	case state.ViewAPIGateway:
		// If we came from stack resources, go back there
		if m.state.SelectedStack != nil {
			m.state.View = state.ViewStackResources
			m.state.FilterText = ""
			m.filterInput.SetValue("")
			m.state.ClearAPIs()
			m.updateMenuBar()
			m.updateStackResourcesList()
		}
	case state.ViewAPIStages:
		m.state.GoBack()
		m.state.FilterText = ""
		m.filterInput.SetValue("")
		m.updateMenuBar()
		m.updateAPIGatewayList()
	case state.ViewJumpHostSelect:
		// Go back to API stages, clear pending tunnel info
		m.state.View = state.ViewAPIStages
		m.state.FilterText = ""
		m.filterInput.SetValue("")
		m.state.ClearEC2Instances()
		m.state.ClearPendingTunnel()
		m.updateMenuBar()
		m.updateAPIStagesList()
	case state.ViewTunnels:
		// Go back to previous view (stacks or services)
		if m.state.SelectedStack != nil {
			m.state.View = state.ViewServices
		} else {
			m.state.View = state.ViewStacks
		}
		m.updateMenuBar()
	}
}

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
	}
	return nil
}

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
			m.updateMenuBar()
		} else {
			m.state.View = state.ViewStacks
			m.updateMenuBar()
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

func (m *Model) showTunnelsView() {
	m.state.View = state.ViewTunnels
	m.updateTunnelsPanel()
	m.updateMenuBar()
}

// executeCommand executes a command from the command palette
func (m *Model) executeCommand(result *components.CommandResult) tea.Cmd {
	if result == nil {
		return nil
	}

	m.logger.Debug("Executing command: %s", result.Command)

	switch result.Command {
	case "stacks":
		m.state.View = state.ViewStacks
		m.state.FilterText = ""
		m.filterInput.SetValue("")
		m.updateMenuBar()
		return m.loadStacks()

	case "services":
		if m.state.SelectedStack != nil {
			m.state.View = state.ViewServices
			m.updateMenuBar()
			return m.loadServices()
		}
		m.logger.Warn("Select a stack first to view services")
		return nil

	case "clusters":
		m.state.View = state.ViewClusters
		m.updateMenuBar()
		// TODO: Implement cluster loading
		m.logger.Info("Clusters view - coming soon")
		return nil

	case "tasks":
		if m.state.SelectedService != nil {
			m.state.View = state.ViewTasks
			m.updateMenuBar()
			// TODO: Implement tasks view
			m.logger.Info("Tasks view - coming soon")
		} else {
			m.logger.Warn("Select a service first to view tasks")
		}
		return nil

	case "tunnels":
		m.showTunnelsView()
		return nil

	case "lambda":
		m.state.View = state.ViewLambda
		m.state.FilterText = ""
		m.filterInput.SetValue("")
		m.updateMenuBar()
		return m.loadFunctions()

	case "apigateway":
		m.state.View = state.ViewAPIGateway
		m.state.FilterText = ""
		m.filterInput.SetValue("")
		m.updateMenuBar()
		return m.loadAPIs()

	case "logs":
		m.state.ToggleLogs()
		m.updateComponentSizes()
		return nil

	case "refresh":
		return m.handleRefresh()

	case "quit":
		if m.tunnelManager != nil {
			m.tunnelManager.StopAllTunnels()
		}
		return tea.Quit

	case "help":
		m.logger.Info("Commands: :stacks :services :lambda :apigateway :tunnels :logs :refresh :quit")
		m.state.ShowLogs = true
		m.updateComponentSizes()
		return nil

	default:
		m.logger.Warn("Unknown command: %s", result.Command)
		return nil
	}
}

func (m *Model) handleStopTunnel() tea.Cmd {
	// Only works in tunnels view
	if m.state.View != state.ViewTunnels {
		return nil
	}

	tunnel := m.tunnelsPanel.SelectedTunnel()
	if tunnel == nil {
		return nil
	}

	m.logger.Info("Stopping tunnel: %s", tunnel.ID)
	if err := m.tunnelManager.StopTunnel(tunnel.ID); err != nil {
		m.logger.Error("Failed to stop tunnel: %v", err)
	}

	m.updateTunnelsPanel()
	return nil
}

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

func (m *Model) updateTunnelsPanel() {
	tunnels := m.tunnelManager.GetTunnels()
	var apiGWTunnels []model.APIGatewayTunnel
	if m.apiGWManager != nil {
		apiGWTunnels = m.apiGWManager.GetTunnels()
	}
	m.tunnelsPanel.SetTunnels(tunnels)
	m.tunnelsPanel.SetAPIGatewayTunnels(apiGWTunnels)
}

func (m *Model) startTunnel(service model.Service, task model.Task, container model.Container, remotePort int) tea.Cmd {
	return m.startTunnelWithPort(service, task, container, remotePort, 0)
}

func (m *Model) startTunnelWithPort(service model.Service, task model.Task, container model.Container, remotePort, localPort int) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		tunnel, err := m.tunnelManager.StartTunnel(ctx, service, task, container, remotePort, localPort)
		return tunnelStartedMsg{tunnel: tunnel, err: err}
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

// startAPIGatewayTunnel starts a tunnel for the API Gateway based on its type.
func (m *Model) startAPIGatewayTunnel(api interface{}, stage model.APIStage, localPort int) tea.Cmd {
	// Determine if this is a private or public API Gateway
	isPrivate := false
	if restAPI, ok := api.(*model.RestAPI); ok {
		isPrivate = restAPI.EndpointType == "PRIVATE"
	}

	if isPrivate {
		m.logger.Info("Loading EC2 instances for jump host selection...")
		// Store pending tunnel info and show jump host selection
		m.state.PendingTunnelAPI = api
		m.state.PendingTunnelStage = &stage
		m.state.PendingTunnelLocalPort = localPort
		m.state.View = state.ViewJumpHostSelect
		m.state.EC2InstancesLoading = true
		return m.loadEC2Instances()
	}

	// Public API Gateway - start local HTTP proxy
	m.logger.Info("Starting public API Gateway proxy for stage: %s", stage.Name)
	return m.startPublicAPIGWTunnel(api, stage, localPort)
}

// startPublicAPIGWTunnel starts a local HTTP proxy for public API Gateway.
func (m *Model) startPublicAPIGWTunnel(api interface{}, stage model.APIStage, localPort int) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		tunnel, err := m.apiGWManager.StartPublicTunnel(ctx, api, stage, localPort)
		return apiGWTunnelStartedMsg{tunnel: tunnel, err: err}
	}
}

// findJumpHostForAPIGateway finds a jump host for private API Gateway access.
func (m *Model) findJumpHostForAPIGateway(api interface{}, stage model.APIStage, localPort int) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		// Get config for the current profile
		jumpHostConfig := ""
		jumpHostTagConfig := ""
		if m.cfg != nil {
			jumpHostConfig = m.cfg.GetJumpHost(m.state.Profile)
			jumpHostTagConfig = m.cfg.GetJumpHostTag(m.state.Profile)
		}

		defaultTags := []string{
			"vaws:jump-host=true",
			"Name=bastion",
			"Name=jump-host",
		}
		defaultNames := []string{
			"bastion",
			"jump-host",
			"jumphost",
		}

		if m.cfg != nil && len(m.cfg.Defaults.JumpHostTags) > 0 {
			defaultTags = m.cfg.Defaults.JumpHostTags
		}
		if m.cfg != nil && len(m.cfg.Defaults.JumpHostNames) > 0 {
			defaultNames = m.cfg.Defaults.JumpHostNames
		}

		// Find jump host
		jumpHost, err := m.client.FindJumpHost(ctx, "", jumpHostConfig, jumpHostTagConfig, defaultTags, defaultNames)
		if err != nil {
			return jumpHostFoundMsg{err: fmt.Errorf("failed to find jump host: %w", err)}
		}

		// Try to find VPC endpoint for execute-api
		var vpcEndpoint *model.VpcEndpoint
		var vpcEndpointErr error
		if jumpHost.VpcID != "" {
			vpcEndpoint, vpcEndpointErr = m.client.FindAPIGatewayVpcEndpoint(ctx, jumpHost.VpcID)
			// Note: vpcEndpointErr is informational - we'll handle missing endpoint in the tunnel manager
		}

		return jumpHostFoundMsg{
			jumpHost:       jumpHost,
			vpcEndpoint:    vpcEndpoint,
			vpcEndpointErr: vpcEndpointErr,
			stage:          stage,
			api:            api,
			localPort:      localPort,
		}
	}
}

// startPrivateAPIGWTunnel starts an SSM tunnel for private API Gateway.
func (m *Model) startPrivateAPIGWTunnel(api interface{}, stage model.APIStage, jumpHost *model.EC2Instance, vpcEndpoint *model.VpcEndpoint, localPort int) tea.Cmd {
	// Get configured VPC endpoint ID for cross-account access
	var configuredVPCEndpointID string
	if m.cfg != nil {
		configuredVPCEndpointID = m.cfg.GetVPCEndpointID(m.state.Profile)
	}

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		tunnel, err := m.apiGWManager.StartPrivateTunnel(ctx, api, stage, jumpHost, vpcEndpoint, configuredVPCEndpointID, localPort)
		return apiGWTunnelStartedMsg{tunnel: tunnel, err: err}
	}
}

// startPrivateAPIGWTunnelWithJumpHost starts a private API Gateway tunnel using the selected jump host.
func (m *Model) startPrivateAPIGWTunnelWithJumpHost(jumpHost *model.EC2Instance) tea.Cmd {
	// Get pending tunnel info
	api := m.state.PendingTunnelAPI
	stage := m.state.PendingTunnelStage
	localPort := m.state.PendingTunnelLocalPort

	// Clear pending tunnel state and go back to stages view
	m.state.ClearPendingTunnel()
	m.state.ClearEC2Instances()
	m.state.View = state.ViewAPIStages
	m.updateMenuBar()
	m.updateAPIStagesList()

	// Get configured VPC endpoint ID for cross-account access
	var configuredVPCEndpointID string
	if m.cfg != nil {
		configuredVPCEndpointID = m.cfg.GetVPCEndpointID(m.state.Profile)
	}

	m.logger.Info("Starting private API Gateway tunnel via jump host: %s", jumpHost.Name)

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		// Try to find VPC endpoint for execute-api
		var vpcEndpoint *model.VpcEndpoint
		if jumpHost.VpcID != "" {
			vpcEndpoint, _ = m.client.FindAPIGatewayVpcEndpoint(ctx, jumpHost.VpcID)
		}

		tunnel, err := m.apiGWManager.StartPrivateTunnel(ctx, api, *stage, jumpHost, vpcEndpoint, configuredVPCEndpointID, localPort)
		return apiGWTunnelStartedMsg{tunnel: tunnel, err: err}
	}
}

func (m *Model) loadStacks() tea.Cmd {
	m.state.StacksLoading = true
	m.stacksList.SetLoading(true)
	m.splash.SetLoading("Loading CloudFormation stacks...")
	m.logger.Info("Loading CloudFormation stacks...")

	return tea.Batch(
		m.splash.Spinner().TickCmd(), // Ensure spinner keeps ticking
		func() tea.Msg {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			stacks, err := m.client.ListStacks(ctx)
			return stacksLoadedMsg{stacks: stacks, err: err}
		},
	)
}

func (m *Model) loadServices() tea.Cmd {
	if m.state.SelectedStack == nil {
		return nil
	}

	m.state.ServicesLoading = true
	m.serviceList.SetLoading(true)
	stackName := m.state.SelectedStack.Name
	m.logger.Info("Loading ECS services for stack: %s", stackName)

	return tea.Batch(
		m.serviceList.Spinner().TickCmd(), // Ensure spinner keeps ticking
		func() tea.Msg {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			services, err := m.client.GetServicesForStack(ctx, stackName)
			return servicesLoadedMsg{services: services, err: err}
		},
	)
}

func (m *Model) loadFunctions() tea.Cmd {
	m.state.FunctionsLoading = true
	m.lambdaList.SetLoading(true)

	// Check if a stack is selected - if so, only load functions from that stack
	var stackName string
	if m.state.SelectedStack != nil {
		stackName = m.state.SelectedStack.Name
		m.logger.Info("Loading Lambda functions for stack: %s", stackName)
	} else {
		m.logger.Info("Loading all Lambda functions...")
	}

	return tea.Batch(
		m.lambdaList.Spinner().TickCmd(),
		func() tea.Msg {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			if stackName != "" {
				// Get function names from the stack
				functionNames, err := m.client.GetLambdaFunctionsFromStack(ctx, stackName)
				if err != nil {
					return functionsLoadedMsg{functions: nil, err: err}
				}

				// If no functions in stack, return empty list
				if len(functionNames) == 0 {
					return functionsLoadedMsg{functions: []model.Function{}, err: nil}
				}

				// Get details for each function
				var functions []model.Function
				for _, name := range functionNames {
					fn, err := m.client.DescribeFunction(ctx, name)
					if err != nil {
						// Log but continue with other functions
						continue
					}
					functions = append(functions, *fn)
				}
				return functionsLoadedMsg{functions: functions, err: nil}
			}

			// No stack selected - load all functions
			functions, err := m.client.ListFunctions(ctx)
			return functionsLoadedMsg{functions: functions, err: err}
		},
	)
}

func (m *Model) loadAPIs() tea.Cmd {
	m.state.APIsLoading = true
	m.apiGatewayList.SetLoading(true)

	// Check if a stack is selected - if so, only load APIs from that stack
	var stackName string
	if m.state.SelectedStack != nil {
		stackName = m.state.SelectedStack.Name
		m.logger.Info("Loading API Gateway APIs for stack: %s", stackName)
	} else {
		m.logger.Info("Loading all API Gateway APIs...")
	}

	return tea.Batch(
		m.apiGatewayList.Spinner().TickCmd(),
		func() tea.Msg {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			if stackName != "" {
				// Get API IDs from the stack
				restAPIIDs, _, err := m.client.GetAPIGatewaysFromStack(ctx, stackName)
				if err != nil {
					return restAPIsLoadedMsg{apis: nil, err: err}
				}

				// If no REST APIs in stack, return empty list
				if len(restAPIIDs) == 0 {
					return restAPIsLoadedMsg{apis: []model.RestAPI{}, err: nil}
				}

				// Get details for each API
				var apis []model.RestAPI
				for _, id := range restAPIIDs {
					api, err := m.client.GetRestAPI(ctx, id)
					if err != nil {
						continue
					}
					apis = append(apis, *api)
				}
				return restAPIsLoadedMsg{apis: apis, err: nil}
			}

			restAPIs, err := m.client.ListRestAPIs(ctx)
			return restAPIsLoadedMsg{apis: restAPIs, err: err}
		},
		func() tea.Msg {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			if stackName != "" {
				// Get API IDs from the stack
				_, httpAPIIDs, err := m.client.GetAPIGatewaysFromStack(ctx, stackName)
				if err != nil {
					return httpAPIsLoadedMsg{apis: nil, err: err}
				}

				// If no HTTP APIs in stack, return empty list
				if len(httpAPIIDs) == 0 {
					return httpAPIsLoadedMsg{apis: []model.HttpAPI{}, err: nil}
				}

				// Get details for each API
				var apis []model.HttpAPI
				for _, id := range httpAPIIDs {
					api, err := m.client.GetHttpAPI(ctx, id)
					if err != nil {
						continue
					}
					apis = append(apis, *api)
				}
				return httpAPIsLoadedMsg{apis: apis, err: nil}
			}

			httpAPIs, err := m.client.ListHttpAPIs(ctx)
			return httpAPIsLoadedMsg{apis: httpAPIs, err: err}
		},
	)
}

// loadEC2Instances loads SSM-managed EC2 instances for jump host selection.
func (m *Model) loadEC2Instances() tea.Cmd {
	m.state.EC2InstancesLoading = true
	m.ec2List.SetLoading(true)
	m.logger.Info("Loading SSM-managed EC2 instances...")

	return tea.Batch(
		m.ec2List.Spinner().TickCmd(),
		func() tea.Msg {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			instances, err := m.client.ListSSMManagedInstances(ctx)
			return ec2InstancesLoadedMsg{instances: instances, err: err}
		},
	)
}

func (m *Model) loadAPIStages() tea.Cmd {
	m.state.APIStagesLoading = true
	m.apiStagesList.SetLoading(true)

	var apiID string
	var isRest bool

	if m.state.SelectedRestAPI != nil {
		apiID = m.state.SelectedRestAPI.ID
		isRest = true
		m.logger.Info("Loading stages for REST API: %s", m.state.SelectedRestAPI.Name)
	} else if m.state.SelectedHttpAPI != nil {
		apiID = m.state.SelectedHttpAPI.ID
		isRest = false
		m.logger.Info("Loading stages for HTTP API: %s", m.state.SelectedHttpAPI.Name)
	} else {
		return nil
	}

	return tea.Batch(
		m.apiStagesList.Spinner().TickCmd(),
		func() tea.Msg {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			var stages []model.APIStage
			var err error
			if isRest {
				stages, err = m.client.GetRestAPIStages(ctx, apiID)
			} else {
				stages, err = m.client.GetHttpAPIStages(ctx, apiID)
			}
			return apiStagesLoadedMsg{stages: stages, err: err}
		},
	)
}

func (m *Model) updateComponentSizes() {
	if !m.ready {
		return
	}

	// Calculate layout
	// Menu bar is 6 rows (5 info rows + 1 resource bar)
	menuBarHeight := 7
	footerHeight := 1
	logsHeight := 0
	if m.state.ShowLogs {
		logsHeight = min(10, m.height/4)
	}

	contentHeight := m.height - menuBarHeight - footerHeight - logsHeight
	listWidth := m.width / 2
	detailsWidth := m.width - listWidth

	m.header.SetWidth(m.width)
	m.header.SetProfile(m.state.Profile)
	m.header.SetRegion(m.state.Region)

	// Configure menu bar
	m.menuBar.SetWidth(m.width)
	m.menuBar.SetProfile(m.state.Profile)
	m.menuBar.SetRegion(m.state.Region)

	m.footer.SetWidth(m.width)
	m.footer.SetBindings(components.DefaultBindings())

	m.stacksList.SetSize(listWidth, contentHeight)
	m.serviceList.SetSize(listWidth, contentHeight)
	m.details.SetSize(detailsWidth, contentHeight)
	m.logs.SetSize(m.width, logsHeight)

	m.updateMenuBar()
}

func (m *Model) updateHeader() {
	var path []string
	if m.state.SelectedStack != nil {
		path = append(path, m.state.SelectedStack.Name)
	}
	m.header.SetPath(path)
}

// updateMenuBar updates the menu bar based on current state
func (m *Model) updateMenuBar() {
	// Clear and rebuild breadcrumbs based on current navigation
	m.menuBar.ClearCrumbs()

	var resourceType components.ResourceType
	var resourceCount int

	switch m.state.View {
	case state.ViewStacks:
		m.menuBar.AddCrumb("Stacks", components.ResourceStacks)
		resourceType = components.ResourceStacks
		resourceCount = len(m.state.FilteredStacks())
	case state.ViewStackResources:
		m.menuBar.AddCrumb("Stacks", components.ResourceStacks)
		if m.state.SelectedStack != nil {
			m.menuBar.AddCrumb(m.state.SelectedStack.Name, components.ResourceStacks)
		}
		m.menuBar.AddCrumb("Resources", components.ResourceStackResources)
		resourceType = components.ResourceStackResources
		resourceCount = 3 // ECS, Lambda, API Gateway
	case state.ViewClusters:
		m.menuBar.AddCrumb("Clusters", components.ResourceClusters)
		resourceType = components.ResourceClusters
		resourceCount = len(m.state.FilteredClusters())
	case state.ViewServices:
		m.menuBar.AddCrumb("Stacks", components.ResourceStacks)
		if m.state.SelectedStack != nil {
			m.menuBar.AddCrumb(m.state.SelectedStack.Name, components.ResourceStacks)
		}
		m.menuBar.AddCrumb("Services", components.ResourceServices)
		resourceType = components.ResourceServices
		resourceCount = len(m.state.FilteredServices())
	case state.ViewTasks:
		m.menuBar.AddCrumb("Stacks", components.ResourceStacks)
		if m.state.SelectedStack != nil {
			m.menuBar.AddCrumb(m.state.SelectedStack.Name, components.ResourceStacks)
		}
		m.menuBar.AddCrumb("Services", components.ResourceServices)
		if m.state.SelectedService != nil {
			m.menuBar.AddCrumb(m.state.SelectedService.Name, components.ResourceServices)
		}
		m.menuBar.AddCrumb("Tasks", components.ResourceTasks)
		resourceType = components.ResourceTasks
		resourceCount = len(m.state.FilteredTasks())
	case state.ViewTunnels:
		m.menuBar.AddCrumb("Tunnels", components.ResourceTunnels)
		resourceType = components.ResourceTunnels
		if m.tunnelManager != nil {
			resourceCount = len(m.tunnelManager.GetTunnels())
		}
	case state.ViewLambda:
		m.menuBar.AddCrumb("Lambda", components.ResourceLambda)
		resourceType = components.ResourceLambda
		resourceCount = len(m.state.FilteredFunctions())
	case state.ViewAPIGateway:
		m.menuBar.AddCrumb("API Gateway", components.ResourceAPIGateway)
		resourceType = components.ResourceAPIGateway
		resourceCount = len(m.state.FilteredRestAPIs()) + len(m.state.FilteredHttpAPIs())
	case state.ViewAPIStages:
		m.menuBar.AddCrumb("API Gateway", components.ResourceAPIGateway)
		if m.state.SelectedRestAPI != nil {
			m.menuBar.AddCrumb(m.state.SelectedRestAPI.Name, components.ResourceAPIGateway)
		} else if m.state.SelectedHttpAPI != nil {
			m.menuBar.AddCrumb(m.state.SelectedHttpAPI.Name, components.ResourceAPIGateway)
		}
		m.menuBar.AddCrumb("Stages", components.ResourceAPIStages)
		resourceType = components.ResourceAPIStages
		resourceCount = len(m.state.FilteredAPIStages())
	case state.ViewJumpHostSelect:
		m.menuBar.AddCrumb("API Gateway", components.ResourceAPIGateway)
		if m.state.SelectedRestAPI != nil {
			m.menuBar.AddCrumb(m.state.SelectedRestAPI.Name, components.ResourceAPIGateway)
		}
		m.menuBar.AddCrumb("Select Jump Host", components.ResourceServices) // Using Services icon for EC2
		resourceType = components.ResourceServices
		resourceCount = len(m.state.FilteredEC2Instances())
	case state.ViewXRay:
		m.menuBar.AddCrumb("Stacks", components.ResourceStacks)
		if m.state.SelectedStack != nil {
			m.menuBar.AddCrumb(m.state.SelectedStack.Name+" (XRay)", components.ResourceStacks)
		}
		resourceType = components.ResourceStacks
	}

	// Set resource count
	m.menuBar.SetResourceCount(resourceCount)

	// Set context-specific key bindings
	m.menuBar.SetContextBindings(components.GetDefaultBindingsForResource(resourceType))

	// Set active tunnels count
	if m.tunnelManager != nil {
		activeTunnels := 0
		for _, t := range m.tunnelManager.GetTunnels() {
			if t.Status == model.TunnelStatusActive {
				activeTunnels++
			}
		}
		m.menuBar.SetActiveTunnels(activeTunnels)
	}

	// Update refresh indicator state
	m.menuBar.RefreshIndicator().SetEnabled(m.state.AutoRefresh)
}

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

func (m *Model) updateEC2List() {
	instances := m.state.FilteredEC2Instances()
	items := make([]components.ListItem, len(instances))
	for i, inst := range instances {
		statusStyle := lipgloss.NewStyle().Foreground(theme.Success)
		if !inst.SSMManaged {
			statusStyle = lipgloss.NewStyle().Foreground(theme.Warning)
		}
		items[i] = components.ListItem{
			ID:          inst.InstanceID,
			Title:       inst.Name,
			Status:      inst.InstanceID,
			StatusStyle: statusStyle,
			Extra:       inst.PrivateIPAddress,
		}
	}
	m.ec2List.SetItems(items)
	m.ec2List.SetLoading(false)
	m.ec2List.SetError(m.state.EC2InstancesError)
	m.ec2List.SetEmptyMessage("No SSM-managed EC2 instances found")
}

func (m *Model) updateCurrentList() {
	switch m.state.View {
	case state.ViewStacks:
		m.updateStacksList()
	case state.ViewStackResources:
		m.updateStackResourcesList()
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
	}
}

func (m *Model) updateStackDetails() {
	item := m.stacksList.SelectedItem()
	if item == nil {
		m.details.SetRows(nil)
		return
	}

	// Find the stack
	for _, s := range m.state.Stacks {
		if s.Name == item.ID {
			rows := components.StackDetails(
				s.Name,
				string(s.Status),
				s.CreatedAt.Format("2006-01-02 15:04:05"),
				s.UpdatedAt.Format("2006-01-02 15:04:05"),
				s.Description,
				StatusStyle(string(s.Status)),
			)
			m.details.SetTitle("Stack Details")
			m.details.SetRows(rows)
			return
		}
	}
}

func (m *Model) updateServiceDetails() {
	item := m.serviceList.SelectedItem()
	if item == nil {
		m.details.SetRows(nil)
		return
	}

	// Find the service
	for _, s := range m.state.Services {
		if s.Name == item.ID {
			rows := components.ServiceDetails(
				s.Name,
				s.ClusterName,
				string(s.Status),
				s.RunningCount,
				s.DesiredCount,
				s.PendingCount,
				s.TaskDefinition,
				s.LaunchType,
				ServiceStatusStyle(s.RunningCount, s.DesiredCount),
			)
			m.details.SetTitle("Service Details")
			m.details.SetRows(rows)
			return
		}
	}
}

func (m *Model) updateLambdaDetails() {
	item := m.lambdaList.SelectedItem()
	if item == nil {
		m.details.SetRows(nil)
		return
	}

	// Find the function
	for _, fn := range m.state.Functions {
		if fn.Name == item.ID {
			rows := []components.DetailRow{
				{Label: "Name", Value: fn.Name},
				{Label: "Runtime", Value: fn.Runtime},
				{Label: "Handler", Value: fn.Handler},
				{Label: "Memory", Value: fmt.Sprintf("%d MB", fn.MemorySize)},
				{Label: "Timeout", Value: fmt.Sprintf("%d seconds", fn.Timeout)},
				{Label: "Code Size", Value: formatBytes(fn.CodeSize)},
				{Label: "State", Value: string(fn.State), Style: FunctionStatusStyle(fn.State)},
				{Label: "Package Type", Value: fn.PackageType},
				{Label: "Last Modified", Value: fn.LastModified.Format("2006-01-02 15:04:05")},
				{Label: "Description", Value: fn.Description},
			}
			m.details.SetTitle("Lambda Function Details")
			m.details.SetRows(rows)
			return
		}
	}
}

func (m *Model) updateAPIGatewayDetails() {
	item := m.apiGatewayList.SelectedItem()
	if item == nil {
		m.details.SetRows(nil)
		return
	}

	// Check if it's a REST or HTTP API based on ID prefix
	if len(item.ID) > 5 && item.ID[:5] == "rest:" {
		apiID := item.ID[5:]
		for _, api := range m.state.RestAPIs {
			if api.ID == apiID {
				// Build the endpoint URL
				var endpointURL string
				if api.EndpointType == "PRIVATE" {
					endpointURL = "(Private - requires VPC endpoint)"
				} else {
					endpointURL = fmt.Sprintf("https://%s.execute-api.%s.amazonaws.com/", api.ID, m.state.Region)
				}

				rows := []components.DetailRow{
					{Label: "Name", Value: api.Name},
					{Label: "ID", Value: api.ID},
					{Label: "Type", Value: "REST API (v1)"},
					{Label: "Endpoint Type", Value: api.EndpointType},
					{Label: "Endpoint URL", Value: endpointURL},
					{Label: "Version", Value: api.Version},
					{Label: "Created", Value: api.CreatedDate.Format("2006-01-02 15:04:05")},
					{Label: "Description", Value: api.Description},
				}
				m.details.SetTitle("REST API Details")
				m.details.SetRows(rows)
				return
			}
		}
	} else if len(item.ID) > 5 && item.ID[:5] == "http:" {
		apiID := item.ID[5:]
		for _, api := range m.state.HttpAPIs {
			if api.ID == apiID {
				rows := []components.DetailRow{
					{Label: "Name", Value: api.Name},
					{Label: "ID", Value: api.ID},
					{Label: "Type", Value: "HTTP API (v2)"},
					{Label: "Protocol", Value: api.ProtocolType},
					{Label: "Endpoint", Value: api.ApiEndpoint},
					{Label: "Version", Value: api.Version},
					{Label: "Created", Value: api.CreatedDate.Format("2006-01-02 15:04:05")},
					{Label: "Description", Value: api.Description},
				}
				m.details.SetTitle("HTTP API Details")
				m.details.SetRows(rows)
				return
			}
		}
	}
}

func (m *Model) updateAPIStageDetails() {
	item := m.apiStagesList.SelectedItem()
	if item == nil {
		m.details.SetRows(nil)
		return
	}

	// Find the stage
	for _, stage := range m.state.APIStages {
		if stage.Name == item.ID {
			rows := []components.DetailRow{
				{Label: "Stage Name", Value: stage.Name},
				{Label: "Deployment ID", Value: stage.DeploymentID},
				{Label: "Invoke URL", Value: stage.InvokeURL},
				{Label: "Created", Value: stage.CreatedDate.Format("2006-01-02 15:04:05")},
				{Label: "Last Updated", Value: stage.LastUpdated.Format("2006-01-02 15:04:05")},
				{Label: "Description", Value: stage.Description},
			}
			m.details.SetTitle("API Stage Details")
			m.details.SetRows(rows)
			return
		}
	}
}

// formatBytes formats bytes into a human-readable string
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

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

// truncateString truncates a string to fit within maxWidth.
func truncateString(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	if len(s) <= maxWidth {
		return s
	}
	if maxWidth <= 3 {
		return s[:maxWidth]
	}
	return s[:maxWidth-3] + "..."
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
	// Menu bar is 6 rows (5 info rows + 1 resource bar)
	menuBarHeight := 7
	footerHeight := 1
	currentLogsHeight := 0
	if m.shouldShowLogs() {
		currentLogsHeight = logsHeight
	}
	contentHeight := m.height - menuBarHeight - footerHeight - currentLogsHeight
	if contentHeight < 1 {
		contentHeight = 1
	}

	// Menu bar (k9s-style top bar)
	m.menuBar.SetWidth(m.width)
	m.menuBar.SetProfile(m.state.Profile)
	m.menuBar.SetRegion(m.state.Region)
	m.updateMenuBar()
	header := m.menuBar.View()

	// Main content area
	var contentView string

	if m.state.View == state.ViewTunnels {
		// Tunnels view takes full width
		m.tunnelsPanel.SetSize(m.width, contentHeight)
		contentView = lipgloss.NewStyle().
			Width(m.width).
			Height(contentHeight).
			MaxWidth(m.width).
			MaxHeight(contentHeight).
			Render(m.tunnelsPanel.View())
	} else {
		contentView = m.renderMainContent(layout, contentHeight)
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

	// Footer
	m.updateFooterBindings()
	m.footer.SetWidth(m.width)
	footer := m.footer.View()

	// Combine all sections
	var sections []string
	sections = append(sections, header)

	if m.commandPalette.IsActive() {
		// Show command palette overlay
		cmdPalette := m.commandPalette.View()
		centeredPalette := lipgloss.Place(m.width, contentHeight, lipgloss.Center, lipgloss.Center, cmdPalette)
		sections = append(sections, centeredPalette)
	} else if m.enteringPort {
		// Center the port input dialog
		centeredDialog := lipgloss.Place(m.width, contentHeight, lipgloss.Center, lipgloss.Center, portInputView)
		sections = append(sections, centeredDialog)
	} else {
		sections = append(sections, contentView)
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
	var listView string
	switch m.state.View {
	case state.ViewStacks:
		listView = m.stacksList.View()
	case state.ViewStackResources:
		listView = m.stackResourcesList.View()
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
		m.stacksList.SetSize(m.width, contentHeight)
		m.stackResourcesList.SetSize(m.width, contentHeight)
		m.serviceList.SetSize(m.width, contentHeight)
		m.lambdaList.SetSize(m.width, contentHeight)
		m.apiGatewayList.SetSize(m.width, contentHeight)
		m.apiStagesList.SetSize(m.width, contentHeight)
		return lipgloss.NewStyle().
			Width(m.width).
			Height(contentHeight).
			MaxWidth(m.width).
			MaxHeight(contentHeight).
			Render(listView)
	}

	// Full two-pane layout
	listWidth := int(float64(m.width) * listPaneRatio)
	detailsWidth := m.width - listWidth

	m.stacksList.SetSize(listWidth, contentHeight)
	m.stackResourcesList.SetSize(listWidth, contentHeight)
	m.serviceList.SetSize(listWidth, contentHeight)
	m.lambdaList.SetSize(listWidth, contentHeight)
	m.apiGatewayList.SetSize(listWidth, contentHeight)
	m.apiStagesList.SetSize(listWidth, contentHeight)
	m.details.SetSize(detailsWidth, contentHeight)

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

// updateFooterBindings sets appropriate footer bindings based on current state.
func (m *Model) updateFooterBindings() {
	if m.enteringPort {
		m.footer.SetBindings(components.PortInputBindings())
	} else if m.filtering {
		m.footer.SetBindings(components.FilterBindings())
	} else {
		switch m.state.View {
		case state.ViewStacks:
			m.footer.SetBindings(components.DefaultBindings())
		case state.ViewServices:
			m.footer.SetBindings(components.ServiceBindings())
		case state.ViewTunnels:
			m.footer.SetBindings(components.TunnelBindings())
		}
	}
}

// matchKey checks if a key message matches a key binding.
func matchKey(msg tea.KeyMsg, binding key.Binding) bool {
	for _, k := range binding.Keys() {
		if msg.String() == k {
			return true
		}
	}
	return false
}
