// Package model defines domain types used throughout the application.
// These types are decoupled from AWS SDK types to keep the UI and business logic clean.
package model

import "time"

// StackStatus represents the status of a CloudFormation stack.
type StackStatus string

const (
	StackStatusCreateComplete         StackStatus = "CREATE_COMPLETE"
	StackStatusCreateInProgress       StackStatus = "CREATE_IN_PROGRESS"
	StackStatusCreateFailed           StackStatus = "CREATE_FAILED"
	StackStatusDeleteComplete         StackStatus = "DELETE_COMPLETE"
	StackStatusDeleteInProgress       StackStatus = "DELETE_IN_PROGRESS"
	StackStatusDeleteFailed           StackStatus = "DELETE_FAILED"
	StackStatusRollbackComplete       StackStatus = "ROLLBACK_COMPLETE"
	StackStatusRollbackInProgress     StackStatus = "ROLLBACK_IN_PROGRESS"
	StackStatusRollbackFailed         StackStatus = "ROLLBACK_FAILED"
	StackStatusUpdateComplete         StackStatus = "UPDATE_COMPLETE"
	StackStatusUpdateInProgress       StackStatus = "UPDATE_IN_PROGRESS"
	StackStatusUpdateRollbackComplete StackStatus = "UPDATE_ROLLBACK_COMPLETE"
)

// IsHealthy returns true if the stack is in a healthy state.
func (s StackStatus) IsHealthy() bool {
	switch s {
	case StackStatusCreateComplete, StackStatusUpdateComplete:
		return true
	default:
		return false
	}
}

// IsInProgress returns true if the stack is currently being modified.
func (s StackStatus) IsInProgress() bool {
	switch s {
	case StackStatusCreateInProgress, StackStatusDeleteInProgress,
		StackStatusRollbackInProgress, StackStatusUpdateInProgress:
		return true
	default:
		return false
	}
}

// IsFailed returns true if the stack is in a failed state.
func (s StackStatus) IsFailed() bool {
	switch s {
	case StackStatusCreateFailed, StackStatusDeleteFailed,
		StackStatusRollbackFailed, StackStatusRollbackComplete,
		StackStatusUpdateRollbackComplete:
		return true
	default:
		return false
	}
}

// Stack represents a CloudFormation stack.
type Stack struct {
	Name         string
	ID           string
	Status       StackStatus
	StatusReason string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	Description  string
	Tags         map[string]string
	Outputs      []StackOutput
	Parameters   []StackParameter
}

// StackOutput represents a CloudFormation stack output.
type StackOutput struct {
	Key         string
	Value       string
	Description string
	ExportName  string
}

// StackParameter represents a CloudFormation stack parameter.
type StackParameter struct {
	Key   string
	Value string
}

// ServiceStatus represents the status of an ECS service.
type ServiceStatus string

const (
	ServiceStatusActive   ServiceStatus = "ACTIVE"
	ServiceStatusDraining ServiceStatus = "DRAINING"
	ServiceStatusInactive ServiceStatus = "INACTIVE"
)

// Service represents an ECS service.
type Service struct {
	Name                 string
	ARN                  string
	ClusterARN           string
	ClusterName          string
	Status               ServiceStatus
	DesiredCount         int
	RunningCount         int
	PendingCount         int
	TaskDefinition       string
	LaunchType           string
	CreatedAt            time.Time
	Deployments          []Deployment
	EnableExecuteCommand bool
}

// Task represents an ECS task.
type Task struct {
	TaskARN           string
	TaskID            string
	ClusterARN        string
	TaskDefinitionARN string
	LastStatus        string
	DesiredStatus     string
	LaunchType        string
	Containers        []Container
	StartedAt         time.Time
}

// Container represents a container in an ECS task.
type Container struct {
	Name            string
	ContainerARN    string
	RuntimeID       string
	LastStatus      string
	Image           string
	NetworkBindings []NetworkBinding
}

// NetworkBinding represents a port binding in a container.
type NetworkBinding struct {
	ContainerPort int
	HostPort      int
	Protocol      string
}

// Tunnel represents an active port forwarding tunnel.
type Tunnel struct {
	ID            string
	LocalPort     int
	RemotePort    int
	ServiceName   string
	ClusterARN    string
	ClusterName   string
	TaskID        string
	ContainerName string
	Status        TunnelStatus
	StartedAt     time.Time
	Error         string
}

// TunnelStatus represents the status of a tunnel.
type TunnelStatus string

const (
	TunnelStatusStarting   TunnelStatus = "STARTING"
	TunnelStatusActive     TunnelStatus = "ACTIVE"
	TunnelStatusError      TunnelStatus = "ERROR"
	TunnelStatusTerminated TunnelStatus = "TERMINATED"
)

// IsHealthy returns true if the service has all desired tasks running.
func (s *Service) IsHealthy() bool {
	return s.Status == ServiceStatusActive && s.RunningCount == s.DesiredCount
}

// Deployment represents an ECS service deployment.
type Deployment struct {
	ID             string
	Status         string
	DesiredCount   int
	RunningCount   int
	PendingCount   int
	TaskDefinition string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// Cluster represents an ECS cluster.
type Cluster struct {
	Name                              string
	ARN                               string
	Status                            string
	ActiveServicesCount               int
	RunningTasksCount                 int
	PendingTasksCount                 int
	RegisteredContainerInstancesCount int
}

// AWSProfile represents an AWS SSO profile.
type AWSProfile struct {
	Name      string
	AccountID string
	Region    string
}

// FunctionState represents the state of a Lambda function.
type FunctionState string

const (
	FunctionStateActive   FunctionState = "Active"
	FunctionStatePending  FunctionState = "Pending"
	FunctionStateInactive FunctionState = "Inactive"
	FunctionStateFailed   FunctionState = "Failed"
)

// IsHealthy returns true if the function is in an active state.
func (s FunctionState) IsHealthy() bool {
	return s == FunctionStateActive
}

// Function represents a Lambda function.
type Function struct {
	Name         string
	ARN          string
	Runtime      string
	Handler      string
	MemorySize   int
	Timeout      int
	CodeSize     int64
	LastModified time.Time
	Description  string
	State        FunctionState
	Role         string
	PackageType  string // Zip or Image
}

// RestAPI represents an API Gateway REST API (v1).
type RestAPI struct {
	ID             string
	Name           string
	Description    string
	CreatedDate    time.Time
	EndpointType   string // REGIONAL, EDGE, PRIVATE
	Version        string
	VpcEndpointIds []string // VPC endpoint IDs for PRIVATE APIs
}

// HttpAPI represents an API Gateway HTTP API (v2).
type HttpAPI struct {
	ID           string
	Name         string
	Description  string
	ProtocolType string // HTTP, WEBSOCKET
	CreatedDate  time.Time
	ApiEndpoint  string
	Version      string
}

// APIStage represents a stage in API Gateway.
type APIStage struct {
	Name         string
	Description  string
	DeploymentID string
	CreatedDate  time.Time
	LastUpdated  time.Time
	InvokeURL    string
}

// APIRoute represents a route in API Gateway HTTP API.
type APIRoute struct {
	RouteKey string // e.g., "GET /users", "POST /orders"
	RouteID  string
	Target   string // Integration target
	AuthType string
}

// EC2Instance represents an EC2 instance.
type EC2Instance struct {
	InstanceID       string
	Name             string
	InstanceType     string
	State            string
	PrivateIPAddress string
	PublicIPAddress  string
	VpcID            string
	SubnetID         string
	LaunchTime       time.Time
	Tags             map[string]string
	SSMManaged       bool
}

// VpcEndpoint represents a VPC endpoint.
type VpcEndpoint struct {
	VpcEndpointID   string
	VpcID           string
	ServiceName     string
	State           string
	VpcEndpointType string // Interface or Gateway
	DNSEntries      []string
	SubnetIDs       []string
}

// APIGatewayTunnelType represents the type of API Gateway tunnel.
type APIGatewayTunnelType string

const (
	APIGatewayTunnelPublic  APIGatewayTunnelType = "PUBLIC"
	APIGatewayTunnelPrivate APIGatewayTunnelType = "PRIVATE"
)

// APIGatewayTunnel represents an active API Gateway port forwarding tunnel.
type APIGatewayTunnel struct {
	ID          string
	LocalPort   int
	APIName     string
	APIID       string
	APIType     string // REST or HTTP
	StageName   string
	InvokeURL   string
	TunnelType  APIGatewayTunnelType
	JumpHost    *EC2Instance // For private API Gateway
	VpcEndpoint *VpcEndpoint // For private API Gateway
	Status      TunnelStatus
	StartedAt   time.Time
	Error       string
}
