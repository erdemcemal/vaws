// Package state manages the application state.
package state

import (
	"vaws/internal/model"
)

// View represents the current view/screen.
type View int

const (
	ViewProfileSelect View = iota
	ViewMain               // Main menu with resource type selection
	ViewStacks
	ViewStackResources // Shows resource types available in a stack
	ViewClusters
	ViewServices
	ViewTasks
	ViewTunnels
	ViewXRay
	ViewLambda
	ViewAPIGateway
	ViewAPIStages
	ViewAPIRoutes
	ViewJumpHostSelect    // Select jump host for private API Gateway tunnel
	ViewContainerSelect   // Select container for port forwarding
	ViewCloudWatchLogs    // CloudWatch logs streaming view
	ViewSQS               // SQS queues view
	ViewSQSDetails        // SQS queue details view
)

// State holds all application state.
type State struct {
	// Current view
	View View

	// AWS profile and region
	Profile  string
	Region   string
	Profiles []string // Available AWS profiles

	// Stacks data
	Stacks        []model.Stack
	StacksLoading bool
	StacksError   error

	// Selected stack
	SelectedStack *model.Stack

	// Clusters data
	Clusters        []model.Cluster
	ClustersLoading bool
	ClustersError   error
	SelectedCluster *model.Cluster

	// Services data (for selected stack or cluster)
	Services        []model.Service
	ServicesLoading bool
	ServicesError   error

	// Selected service
	SelectedService *model.Service

	// Tasks data
	Tasks        []model.Task
	TasksLoading bool
	TasksError   error
	SelectedTask *model.Task

	// Lambda Functions data
	Functions        []model.Function
	FunctionsLoading bool
	FunctionsError   error
	SelectedFunction *model.Function

	// API Gateway data
	RestAPIs         []model.RestAPI
	HttpAPIs         []model.HttpAPI
	APIsLoading      bool
	APIsError        error
	SelectedRestAPI  *model.RestAPI
	SelectedHttpAPI  *model.HttpAPI
	APIStages        []model.APIStage
	APIStagesLoading bool
	APIStagesError   error
	SelectedAPIStage *model.APIStage
	APIRoutes        []model.APIRoute
	APIRoutesLoading bool
	APIRoutesError   error

	// EC2 instances for jump host selection
	EC2Instances        []model.EC2Instance
	EC2InstancesLoading bool
	EC2InstancesError   error

	// Pending tunnel info (while selecting jump host)
	PendingTunnelAPI       interface{}
	PendingTunnelStage     *model.APIStage
	PendingTunnelLocalPort int

	// Pending container selection for port forwarding
	PendingContainerService *model.Service
	PendingContainerTask    *model.Task
	PendingContainers       []model.Container

	// CloudWatch Logs state
	CloudWatchLogs              []model.CloudWatchLogEntry
	CloudWatchLogsLoading       bool
	CloudWatchLogsError         error
	CloudWatchLogsStreaming     bool
	CloudWatchLastFetchTime     int64 // Unix ms for incremental fetch
	CloudWatchLogConfigs        []model.ContainerLogConfig
	CloudWatchSelectedContainer int
	CloudWatchServiceContext    *model.Service
	CloudWatchTaskContext       *model.Task

	// SQS Queues state
	Queues        []model.Queue
	QueuesLoading bool
	QueuesError   error
	SelectedQueue *model.Queue

	// UI state
	ShowLogs      bool
	FilterText    string
	AutoRefresh   bool
	CommandMode   bool
	LastRefreshAt int64 // Unix timestamp
}

// New creates a new State with defaults.
func New() *State {
	return &State{
		View:        ViewMain,
		ShowLogs:    true, // Show logs by default
		AutoRefresh: true,
	}
}

// ClearClusters clears cluster data.
func (s *State) ClearClusters() {
	s.Clusters = nil
	s.ClustersLoading = false
	s.ClustersError = nil
	s.SelectedCluster = nil
}

// ClearTasks clears task data.
func (s *State) ClearTasks() {
	s.Tasks = nil
	s.TasksLoading = false
	s.TasksError = nil
	s.SelectedTask = nil
}

// SelectCluster sets the selected cluster and changes view to services.
func (s *State) SelectCluster(cluster *model.Cluster) {
	s.SelectedCluster = cluster
	s.View = ViewServices
	s.ClearServices()
}

// SelectTask sets the selected task.
func (s *State) SelectTask(task *model.Task) {
	s.SelectedTask = task
}

// FilteredClusters returns clusters filtered by the current filter text.
func (s *State) FilteredClusters() []model.Cluster {
	if s.FilterText == "" {
		return s.Clusters
	}

	var filtered []model.Cluster
	for _, cluster := range s.Clusters {
		if containsIgnoreCase(cluster.Name, s.FilterText) {
			filtered = append(filtered, cluster)
		}
	}
	return filtered
}

// FilteredTasks returns tasks filtered by the current filter text.
func (s *State) FilteredTasks() []model.Task {
	if s.FilterText == "" {
		return s.Tasks
	}

	var filtered []model.Task
	for _, task := range s.Tasks {
		if containsIgnoreCase(task.TaskID, s.FilterText) {
			filtered = append(filtered, task)
		}
	}
	return filtered
}

// ToggleAutoRefresh toggles auto-refresh.
func (s *State) ToggleAutoRefresh() {
	s.AutoRefresh = !s.AutoRefresh
}

// SetProfile sets the AWS profile and resets dependent state.
func (s *State) SetProfile(profile string) {
	s.Profile = profile
	s.ClearStacks()
}

// SetRegion sets the AWS region and resets dependent state.
func (s *State) SetRegion(region string) {
	s.Region = region
	s.ClearStacks()
}

// ClearStacks clears all stack and service data.
func (s *State) ClearStacks() {
	s.Stacks = nil
	s.StacksLoading = false
	s.StacksError = nil
	s.SelectedStack = nil
	s.ClearServices()
}

// ClearServices clears service data.
func (s *State) ClearServices() {
	s.Services = nil
	s.ServicesLoading = false
	s.ServicesError = nil
	s.SelectedService = nil
}

// ClearFunctions clears Lambda function data.
func (s *State) ClearFunctions() {
	s.Functions = nil
	s.FunctionsLoading = false
	s.FunctionsError = nil
	s.SelectedFunction = nil
}

// ClearAPIs clears API Gateway data.
func (s *State) ClearAPIs() {
	s.RestAPIs = nil
	s.HttpAPIs = nil
	s.APIsLoading = false
	s.APIsError = nil
	s.SelectedRestAPI = nil
	s.SelectedHttpAPI = nil
	s.ClearAPIStages()
}

// ClearAPIStages clears API stages data.
func (s *State) ClearAPIStages() {
	s.APIStages = nil
	s.APIStagesLoading = false
	s.APIStagesError = nil
	s.SelectedAPIStage = nil
	s.ClearAPIRoutes()
}

// ClearAPIRoutes clears API routes data.
func (s *State) ClearAPIRoutes() {
	s.APIRoutes = nil
	s.APIRoutesLoading = false
	s.APIRoutesError = nil
}

// ClearEC2Instances clears EC2 instance data.
func (s *State) ClearEC2Instances() {
	s.EC2Instances = nil
	s.EC2InstancesLoading = false
	s.EC2InstancesError = nil
}

// ClearPendingTunnel clears pending tunnel info.
func (s *State) ClearPendingTunnel() {
	s.PendingTunnelAPI = nil
	s.PendingTunnelStage = nil
	s.PendingTunnelLocalPort = 0
}

// ClearPendingContainer clears pending container selection.
func (s *State) ClearPendingContainer() {
	s.PendingContainerService = nil
	s.PendingContainerTask = nil
	s.PendingContainers = nil
}

// ClearCloudWatchLogs clears CloudWatch logs state.
func (s *State) ClearCloudWatchLogs() {
	s.CloudWatchLogs = nil
	s.CloudWatchLogsLoading = false
	s.CloudWatchLogsError = nil
	s.CloudWatchLogsStreaming = false
	s.CloudWatchLastFetchTime = 0
	s.CloudWatchLogConfigs = nil
	s.CloudWatchSelectedContainer = 0
	s.CloudWatchServiceContext = nil
	s.CloudWatchTaskContext = nil
}

// ClearQueues clears SQS queue data.
func (s *State) ClearQueues() {
	s.Queues = nil
	s.QueuesLoading = false
	s.QueuesError = nil
	s.SelectedQueue = nil
}

// SelectQueue sets the selected SQS queue.
func (s *State) SelectQueue(queue *model.Queue) {
	s.SelectedQueue = queue
}

// SelectStack sets the selected stack and changes view to services.
func (s *State) SelectStack(stack *model.Stack) {
	s.SelectedStack = stack
	s.View = ViewStackResources
}

// SelectService sets the selected service.
func (s *State) SelectService(service *model.Service) {
	s.SelectedService = service
}

// SelectFunction sets the selected Lambda function.
func (s *State) SelectFunction(fn *model.Function) {
	s.SelectedFunction = fn
}

// SelectRestAPI sets the selected REST API and changes view to stages.
func (s *State) SelectRestAPI(api *model.RestAPI) {
	s.SelectedRestAPI = api
	s.SelectedHttpAPI = nil
	s.View = ViewAPIStages
	s.ClearAPIStages()
}

// SelectHttpAPI sets the selected HTTP API and changes view to stages.
func (s *State) SelectHttpAPI(api *model.HttpAPI) {
	s.SelectedHttpAPI = api
	s.SelectedRestAPI = nil
	s.View = ViewAPIStages
	s.ClearAPIStages()
}

// SelectAPIStage sets the selected API stage.
func (s *State) SelectAPIStage(stage *model.APIStage) {
	s.SelectedAPIStage = stage
}

// GoBack navigates back to the previous view.
func (s *State) GoBack() {
	switch s.View {
	case ViewServices:
		s.View = ViewStacks
		s.SelectedStack = nil
		s.ClearServices()
	case ViewAPIStages:
		s.View = ViewAPIGateway
		s.SelectedRestAPI = nil
		s.SelectedHttpAPI = nil
		s.ClearAPIStages()
	case ViewAPIRoutes:
		s.View = ViewAPIStages
		s.ClearAPIRoutes()
	case ViewJumpHostSelect:
		s.View = ViewAPIStages
		s.ClearEC2Instances()
		s.ClearPendingTunnel()
	}
}

// ToggleLogs toggles the logs panel visibility.
func (s *State) ToggleLogs() {
	s.ShowLogs = !s.ShowLogs
}

// FilteredStacks returns stacks filtered by the current filter text.
func (s *State) FilteredStacks() []model.Stack {
	if s.FilterText == "" {
		return s.Stacks
	}

	var filtered []model.Stack
	for _, stack := range s.Stacks {
		if containsIgnoreCase(stack.Name, s.FilterText) {
			filtered = append(filtered, stack)
		}
	}
	return filtered
}

// FilteredServices returns services filtered by the current filter text.
func (s *State) FilteredServices() []model.Service {
	if s.FilterText == "" {
		return s.Services
	}

	var filtered []model.Service
	for _, svc := range s.Services {
		if containsIgnoreCase(svc.Name, s.FilterText) {
			filtered = append(filtered, svc)
		}
	}
	return filtered
}

// FilteredFunctions returns Lambda functions filtered by the current filter text.
func (s *State) FilteredFunctions() []model.Function {
	if s.FilterText == "" {
		return s.Functions
	}

	var filtered []model.Function
	for _, fn := range s.Functions {
		if containsIgnoreCase(fn.Name, s.FilterText) {
			filtered = append(filtered, fn)
		}
	}
	return filtered
}

// FilteredRestAPIs returns REST APIs filtered by the current filter text.
func (s *State) FilteredRestAPIs() []model.RestAPI {
	if s.FilterText == "" {
		return s.RestAPIs
	}

	var filtered []model.RestAPI
	for _, api := range s.RestAPIs {
		if containsIgnoreCase(api.Name, s.FilterText) {
			filtered = append(filtered, api)
		}
	}
	return filtered
}

// FilteredHttpAPIs returns HTTP APIs filtered by the current filter text.
func (s *State) FilteredHttpAPIs() []model.HttpAPI {
	if s.FilterText == "" {
		return s.HttpAPIs
	}

	var filtered []model.HttpAPI
	for _, api := range s.HttpAPIs {
		if containsIgnoreCase(api.Name, s.FilterText) {
			filtered = append(filtered, api)
		}
	}
	return filtered
}

// FilteredAPIStages returns API stages filtered by the current filter text.
func (s *State) FilteredAPIStages() []model.APIStage {
	if s.FilterText == "" {
		return s.APIStages
	}

	var filtered []model.APIStage
	for _, stage := range s.APIStages {
		if containsIgnoreCase(stage.Name, s.FilterText) {
			filtered = append(filtered, stage)
		}
	}
	return filtered
}

// FilteredAPIRoutes returns API routes filtered by the current filter text.
func (s *State) FilteredAPIRoutes() []model.APIRoute {
	if s.FilterText == "" {
		return s.APIRoutes
	}

	var filtered []model.APIRoute
	for _, route := range s.APIRoutes {
		if containsIgnoreCase(route.RouteKey, s.FilterText) {
			filtered = append(filtered, route)
		}
	}
	return filtered
}

// FilteredEC2Instances returns EC2 instances filtered by the current filter text.
func (s *State) FilteredEC2Instances() []model.EC2Instance {
	if s.FilterText == "" {
		return s.EC2Instances
	}

	var filtered []model.EC2Instance
	for _, inst := range s.EC2Instances {
		if containsIgnoreCase(inst.Name, s.FilterText) || containsIgnoreCase(inst.InstanceID, s.FilterText) {
			filtered = append(filtered, inst)
		}
	}
	return filtered
}

// FilteredContainers returns containers filtered by the current filter text.
func (s *State) FilteredContainers() []model.Container {
	if s.FilterText == "" {
		return s.PendingContainers
	}

	var filtered []model.Container
	for _, c := range s.PendingContainers {
		if containsIgnoreCase(c.Name, s.FilterText) {
			filtered = append(filtered, c)
		}
	}
	return filtered
}

// FilteredQueues returns SQS queues filtered by the current filter text.
func (s *State) FilteredQueues() []model.Queue {
	if s.FilterText == "" {
		return s.Queues
	}

	var filtered []model.Queue
	for _, q := range s.Queues {
		if containsIgnoreCase(q.Name, s.FilterText) {
			filtered = append(filtered, q)
		}
	}
	return filtered
}

func containsIgnoreCase(s, substr string) bool {
	return len(s) >= len(substr) && (substr == "" ||
		findIgnoreCase(s, substr) >= 0)
}

func findIgnoreCase(s, substr string) int {
	if len(substr) == 0 {
		return 0
	}
	if len(substr) > len(s) {
		return -1
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if equalFoldAt(s, i, substr) {
			return i
		}
	}
	return -1
}

func equalFoldAt(s string, offset int, substr string) bool {
	for i := 0; i < len(substr); i++ {
		c1 := s[offset+i]
		c2 := substr[i]
		if c1 == c2 {
			continue
		}
		// Simple ASCII case-insensitive comparison
		if c1 >= 'A' && c1 <= 'Z' {
			c1 += 32
		}
		if c2 >= 'A' && c2 <= 'Z' {
			c2 += 32
		}
		if c1 != c2 {
			return false
		}
	}
	return true
}
