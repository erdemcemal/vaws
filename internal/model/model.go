// Package model defines domain types used throughout the application.
// These types are decoupled from AWS SDK types to keep the UI and business logic clean.
package model

import (
	"sort"
	"strings"
	"time"
)

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

// ContainerPort represents a container and its exposed ports.
type ContainerPort struct {
	ContainerName string
	Ports         []int
}

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
	ContainerPorts       []ContainerPort // Container name -> ports mapping
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
	PortMappings    []PortMapping // Ports from task definition
}

// PortMapping represents a port mapping from the task definition.
type PortMapping struct {
	ContainerPort int
	HostPort      int
	Protocol      string
	Name          string // Optional port name
}

// GetExposedPorts returns all ports this container exposes.
// It combines NetworkBindings (for EC2/bridge) and PortMappings (for Fargate/awsvpc).
// Ports are returned in ascending order for consistency.
func (c *Container) GetExposedPorts() []int {
	portSet := make(map[int]bool)

	// Add ports from NetworkBindings (EC2 launch type)
	for _, nb := range c.NetworkBindings {
		if nb.ContainerPort > 0 {
			portSet[nb.ContainerPort] = true
		}
	}

	// Add ports from PortMappings (Fargate or task definition)
	for _, pm := range c.PortMappings {
		if pm.ContainerPort > 0 {
			portSet[pm.ContainerPort] = true
		}
	}

	// Convert to slice and sort
	ports := make([]int, 0, len(portSet))
	for port := range portSet {
		ports = append(ports, port)
	}
	sort.Ints(ports)

	return ports
}

// IsSidecar returns true if this container looks like a sidecar/helper container.
func (c *Container) IsSidecar() bool {
	name := strings.ToLower(c.Name)
	sidecars := []string{
		"otel", "opentelemetry", "collector",
		"datadog", "dd-agent",
		"xray", "x-ray", "aws-xray",
		"envoy", "proxy",
		"fluentbit", "fluent-bit", "fluentd",
		"cloudwatch-agent", "cwagent",
		"newrelic", "nr-",
		"splunk",
		"jaeger",
		"zipkin",
		"prometheus",
		"grafana-agent",
	}
	for _, s := range sidecars {
		if strings.Contains(name, s) {
			return true
		}
	}
	return false
}

// HasAppPort returns true if this container exposes common application ports.
func (c *Container) HasAppPort() bool {
	appPorts := map[int]bool{80: true, 443: true, 8080: true, 8000: true, 3000: true, 5000: true, 9000: true}
	for _, port := range c.GetExposedPorts() {
		if appPorts[port] {
			return true
		}
	}
	return false
}

// GetBestPort returns the best port for port forwarding (prefers common app ports).
func (c *Container) GetBestPort() int {
	ports := c.GetExposedPorts()
	if len(ports) == 0 {
		return 80 // Default fallback
	}

	// Prefer common app ports in order of preference
	preferredPorts := []int{80, 8080, 443, 8000, 3000, 5000, 9000}
	for _, preferred := range preferredPorts {
		for _, port := range ports {
			if port == preferred {
				return port
			}
		}
	}

	// Return first port if no preferred port found
	return ports[0]
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

// InvocationResult represents the result of a Lambda function invocation.
type InvocationResult struct {
	FunctionName    string
	StatusCode      int
	ExecutedVersion string
	Payload         string        // Response payload as JSON string
	FunctionError   string        // Error type if function errored (e.g., "Handled", "Unhandled")
	LogResult       string        // Base64 encoded last 4KB of execution log
	Duration        time.Duration // Client-side measured duration
	InvokedAt       time.Time
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

// CloudWatchLogEntry represents a single CloudWatch log event.
type CloudWatchLogEntry struct {
	Timestamp     time.Time
	Message       string
	IngestionTime time.Time
	LogStreamName string
}

// ContainerLogConfig holds CloudWatch log configuration for a container.
type ContainerLogConfig struct {
	ContainerName   string
	LogGroup        string
	LogStreamPrefix string
	LogRegion       string // May differ from current region
	LogStreamName   string // Computed: prefix/container/task-id
}

// QueueType represents the type of SQS queue.
type QueueType string

const (
	QueueTypeStandard QueueType = "Standard"
	QueueTypeFIFO     QueueType = "FIFO"
)

// Queue represents an SQS queue.
type Queue struct {
	Name                    string
	URL                     string
	ARN                     string
	Type                    QueueType
	ApproximateMessageCount int
	ApproximateInFlight     int // Messages currently being processed
	VisibilityTimeout       int // In seconds
	MessageRetentionPeriod  int // In seconds
	DelaySeconds            int
	CreatedAt               time.Time
	// DLQ info
	HasDLQ          bool
	DLQArn          string
	DLQURL          string
	DLQName         string
	DLQMessageCount int
	MaxReceiveCount int // Number of receives before message goes to DLQ
}

// HasDLQMessages returns true if the queue has messages in its DLQ.
func (q *Queue) HasDLQMessages() bool {
	return q.HasDLQ && q.DLQMessageCount > 0
}

// TableStatus represents the status of a DynamoDB table.
type TableStatus string

const (
	TableStatusCreating    TableStatus = "CREATING"
	TableStatusActive      TableStatus = "ACTIVE"
	TableStatusDeleting    TableStatus = "DELETING"
	TableStatusUpdating    TableStatus = "UPDATING"
	TableStatusArchiving   TableStatus = "ARCHIVING"
	TableStatusArchived    TableStatus = "ARCHIVED"
	TableStatusInaccessible TableStatus = "INACCESSIBLE_ENCRYPTION_CREDENTIALS"
)

// IsHealthy returns true if the table is active.
func (s TableStatus) IsHealthy() bool {
	return s == TableStatusActive
}

// IsInProgress returns true if the table is being modified.
func (s TableStatus) IsInProgress() bool {
	switch s {
	case TableStatusCreating, TableStatusDeleting, TableStatusUpdating, TableStatusArchiving:
		return true
	default:
		return false
	}
}

// BillingMode represents the billing mode of a DynamoDB table.
type BillingMode string

const (
	BillingModeProvisioned   BillingMode = "PROVISIONED"
	BillingModePayPerRequest BillingMode = "PAY_PER_REQUEST"
)

// KeySchemaElement represents a key attribute in DynamoDB.
type KeySchemaElement struct {
	AttributeName string
	KeyType       string // HASH or RANGE
}

// GlobalSecondaryIndex represents a GSI on a DynamoDB table.
type GlobalSecondaryIndex struct {
	IndexName  string
	KeySchema  []KeySchemaElement
	Status     string
	ItemCount  int64
	SizeBytes  int64
}

// LocalSecondaryIndex represents an LSI on a DynamoDB table.
type LocalSecondaryIndex struct {
	IndexName  string
	KeySchema  []KeySchemaElement
	ItemCount  int64
	SizeBytes  int64
}

// Table represents a DynamoDB table.
type Table struct {
	Name                   string
	ARN                    string
	Status                 TableStatus
	KeySchema              []KeySchemaElement
	ItemCount              int64
	SizeBytes              int64
	CreatedAt              time.Time
	BillingMode            BillingMode
	ReadCapacityUnits      int64
	WriteCapacityUnits     int64
	GlobalSecondaryIndexes []GlobalSecondaryIndex
	LocalSecondaryIndexes  []LocalSecondaryIndex
	TTLEnabled             bool
	TTLAttribute           string
	StreamEnabled          bool
	StreamViewType         string
	DeletionProtection     bool
}

// IsHealthy returns true if the table is active.
func (t *Table) IsHealthy() bool {
	return t.Status == TableStatusActive
}

// PartitionKey returns the partition key attribute name.
func (t *Table) PartitionKey() string {
	for _, k := range t.KeySchema {
		if k.KeyType == "HASH" {
			return k.AttributeName
		}
	}
	return ""
}

// SortKey returns the sort key attribute name (if any).
func (t *Table) SortKey() string {
	for _, k := range t.KeySchema {
		if k.KeyType == "RANGE" {
			return k.AttributeName
		}
	}
	return ""
}

// SortKeyCondition represents the condition type for sort key in a query.
type SortKeyCondition string

const (
	SortKeyConditionEquals     SortKeyCondition = "="
	SortKeyConditionLessThan   SortKeyCondition = "<"
	SortKeyConditionLessEqual  SortKeyCondition = "<="
	SortKeyConditionGreater    SortKeyCondition = ">"
	SortKeyConditionGreaterEq  SortKeyCondition = ">="
	SortKeyConditionBetween    SortKeyCondition = "BETWEEN"
	SortKeyConditionBeginsWith SortKeyCondition = "begins_with"
)

// QueryParams holds parameters for a DynamoDB query.
type QueryParams struct {
	TableName         string
	PartitionKeyName  string
	PartitionKeyVal   string
	SortKeyName       string
	SortKeyVal        string
	SortKeyVal2       string // For BETWEEN condition
	SortKeyCondition  SortKeyCondition
	FilterExpression  string
	FilterAttrName    string
	FilterAttrValue   string
	Limit             int32
	ScanIndexForward  bool // true = ascending, false = descending
	IndexName         string
}

// ScanParams holds parameters for a DynamoDB scan.
type ScanParams struct {
	TableName        string
	PartitionKeyName string
	SortKeyName      string
	FilterExpression string
	FilterAttrName   string
	FilterAttrValue  string
	Limit            int32
	IndexName        string
}

// DynamoDBItem represents a single item from DynamoDB.
type DynamoDBItem struct {
	// Raw holds the item as a map of attribute name to value
	// Values are stored as their string representation for display
	Raw map[string]interface{}
	// JSON is the formatted JSON string of the item
	JSON string
	// PartitionKeyValue is the PK value for quick display
	PartitionKeyValue string
	// SortKeyValue is the SK value for quick display (may be empty)
	SortKeyValue string
}

// Preview returns a truncated preview of the item for list display.
func (d *DynamoDBItem) Preview(maxLen int) string {
	if len(d.JSON) <= maxLen {
		return d.JSON
	}
	return d.JSON[:maxLen-3] + "..."
}

// QueryResult holds the result of a DynamoDB query or scan.
type QueryResult struct {
	Items             []DynamoDBItem
	Count             int
	ScannedCount      int
	LastEvaluatedKey  map[string]interface{}
	ConsumedCapacity  float64
	HasMorePages      bool
}
