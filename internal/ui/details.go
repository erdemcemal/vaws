package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"vaws/internal/ui/components"
	"vaws/internal/ui/theme"
)

// updateStackDetails updates the details panel with stack information.
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

// updateServiceDetails updates the details panel with service information.
func (m *Model) updateServiceDetails() {
	item := m.serviceList.SelectedItem()
	if item == nil {
		m.details.SetRows(nil)
		return
	}

	// Find the service
	for _, s := range m.state.Services {
		if s.Name == item.ID {
			// Format container ports as "container:port1,port2; ..."
			var containerPortsStr string
			if len(s.ContainerPorts) > 0 {
				var parts []string
				for _, cp := range s.ContainerPorts {
					var portStrs []string
					for _, p := range cp.Ports {
						portStrs = append(portStrs, fmt.Sprintf("%d", p))
					}
					parts = append(parts, fmt.Sprintf("%s:%s", cp.ContainerName, strings.Join(portStrs, ",")))
				}
				containerPortsStr = strings.Join(parts, "; ")
			}

			rows := components.ServiceDetails(
				s.Name,
				s.ClusterName,
				string(s.Status),
				s.RunningCount,
				s.DesiredCount,
				s.PendingCount,
				s.TaskDefinition,
				s.LaunchType,
				containerPortsStr,
				ServiceStatusStyle(s.RunningCount, s.DesiredCount),
			)
			m.details.SetTitle("Service Details")
			m.details.SetRows(rows)
			return
		}
	}
}

// updateLambdaDetails updates the details panel with Lambda function information.
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

			// Add invocation state if available
			if m.state.LambdaInvocationLoading {
				rows = append(rows, components.DetailRow{Label: "", Value: ""}) // Spacer
				rows = append(rows, components.DetailRow{
					Label: "Invocation",
					Value: "Invoking...",
					Style: lipgloss.NewStyle().Foreground(theme.Warning),
				})
			} else if m.state.LambdaInvocationError != nil {
				rows = append(rows, components.DetailRow{Label: "", Value: ""}) // Spacer
				rows = append(rows, components.DetailRow{
					Label: "Invoke Error",
					Value: m.state.LambdaInvocationError.Error(),
					Style: lipgloss.NewStyle().Foreground(theme.Error),
				})
			} else if m.state.LambdaInvocationResult != nil {
				result := m.state.LambdaInvocationResult
				rows = append(rows, components.DetailRow{Label: "", Value: ""}) // Spacer

				// Status with color based on success/error
				statusStyle := lipgloss.NewStyle().Foreground(theme.Success)
				if result.FunctionError != "" {
					statusStyle = lipgloss.NewStyle().Foreground(theme.Error)
				}
				rows = append(rows, components.DetailRow{
					Label: "Last Invoke",
					Value: fmt.Sprintf("Status %d (%v)", result.StatusCode, result.Duration.Round(time.Millisecond)),
					Style: statusStyle,
				})

				if result.FunctionError != "" {
					rows = append(rows, components.DetailRow{
						Label: "Error Type",
						Value: result.FunctionError,
						Style: lipgloss.NewStyle().Foreground(theme.Error),
					})
				}

				// Show truncated response
				response := result.Payload
				if len(response) > 100 {
					response = response[:100] + "..."
				}
				rows = append(rows, components.DetailRow{
					Label: "Response",
					Value: response,
				})
			}

			m.details.SetTitle("Lambda Function Details")
			m.details.SetRows(rows)
			return
		}
	}
}

// updateAPIGatewayDetails updates the details panel with API Gateway information.
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

// updateAPIStageDetails updates the details panel with API stage information.
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

// updateQueueDetails updates the details panel with SQS queue information.
func (m *Model) updateQueueDetails() {
	q := m.sqsTable.SelectedQueue()
	if q == nil {
		m.details.SetTitle("SQS Queue Details")
		m.details.SetRows(nil)
		return
	}

	rows := []components.DetailRow{
		{Label: "Name", Value: q.Name},
		{Label: "Type", Value: string(q.Type)},
		{Label: "", Value: ""}, // Spacer
		{Label: "Messages", Value: fmt.Sprintf("%d", q.ApproximateMessageCount)},
		{Label: "In Flight", Value: fmt.Sprintf("%d", q.ApproximateInFlight)},
		{Label: "", Value: ""}, // Spacer
		{Label: "Visibility", Value: fmt.Sprintf("%ds", q.VisibilityTimeout)},
		{Label: "Retention", Value: formatDuration(q.MessageRetentionPeriod)},
		{Label: "Delay", Value: fmt.Sprintf("%ds", q.DelaySeconds)},
	}

	if !q.CreatedAt.IsZero() {
		rows = append(rows, components.DetailRow{Label: "Created", Value: q.CreatedAt.Format("2006-01-02")})
	}

	// Add DLQ info if present
	if q.HasDLQ {
		rows = append(rows, components.DetailRow{Label: "", Value: ""}) // Spacer
		// Extract DLQ name from ARN (format: arn:aws:sqs:region:account:queue-name)
		dlqName := q.DLQArn
		if parts := strings.Split(q.DLQArn, ":"); len(parts) > 0 {
			dlqName = parts[len(parts)-1]
		}
		rows = append(rows, components.DetailRow{Label: "DLQ", Value: dlqName})
		rows = append(rows, components.DetailRow{Label: "Max Receives", Value: fmt.Sprintf("%d", q.MaxReceiveCount)})
	}

	rows = append(rows, components.DetailRow{Label: "", Value: ""}) // Spacer
	rows = append(rows, components.DetailRow{Label: "URL", Value: q.URL})
	rows = append(rows, components.DetailRow{Label: "ARN", Value: q.ARN})

	m.details.SetTitle("SQS Queue Details")
	m.details.SetRows(rows)
}

// updateTableDetails updates the details panel with DynamoDB table information.
func (m *Model) updateTableDetails() {
	t := m.dynamodbTable.SelectedTable()
	if t == nil {
		m.details.SetTitle("DynamoDB Table Details")
		m.details.SetRows(nil)
		return
	}

	rows := []components.DetailRow{
		{Label: "Name", Value: t.Name},
		{Label: "Status", Value: string(t.Status), Style: TableStatusStyle(t.Status)},
		{Label: "", Value: ""}, // Spacer
	}

	// Key schema
	pk := t.PartitionKey()
	sk := t.SortKey()
	if pk != "" {
		rows = append(rows, components.DetailRow{Label: "Partition Key", Value: pk})
	}
	if sk != "" {
		rows = append(rows, components.DetailRow{Label: "Sort Key", Value: sk})
	}

	rows = append(rows, components.DetailRow{Label: "", Value: ""}) // Spacer

	// Capacity
	rows = append(rows, components.DetailRow{Label: "Billing Mode", Value: string(t.BillingMode)})
	if t.BillingMode == "PROVISIONED" {
		rows = append(rows, components.DetailRow{Label: "Read Capacity", Value: fmt.Sprintf("%d", t.ReadCapacityUnits)})
		rows = append(rows, components.DetailRow{Label: "Write Capacity", Value: fmt.Sprintf("%d", t.WriteCapacityUnits)})
	}

	rows = append(rows, components.DetailRow{Label: "", Value: ""}) // Spacer

	// Stats
	rows = append(rows, components.DetailRow{Label: "Items", Value: fmt.Sprintf("%d", t.ItemCount)})
	rows = append(rows, components.DetailRow{Label: "Size", Value: formatBytes(t.SizeBytes)})

	// Indexes
	if len(t.GlobalSecondaryIndexes) > 0 {
		rows = append(rows, components.DetailRow{Label: "", Value: ""}) // Spacer
		rows = append(rows, components.DetailRow{Label: "GSIs", Value: fmt.Sprintf("%d", len(t.GlobalSecondaryIndexes))})
		for _, gsi := range t.GlobalSecondaryIndexes {
			rows = append(rows, components.DetailRow{Label: "  " + gsi.IndexName, Value: gsi.Status})
		}
	}
	if len(t.LocalSecondaryIndexes) > 0 {
		rows = append(rows, components.DetailRow{Label: "LSIs", Value: fmt.Sprintf("%d", len(t.LocalSecondaryIndexes))})
	}

	// Features
	rows = append(rows, components.DetailRow{Label: "", Value: ""}) // Spacer
	ttlStatus := "Disabled"
	if t.TTLEnabled {
		ttlStatus = fmt.Sprintf("Enabled (%s)", t.TTLAttribute)
	}
	rows = append(rows, components.DetailRow{Label: "TTL", Value: ttlStatus})

	streamStatus := "Disabled"
	if t.StreamEnabled {
		streamStatus = fmt.Sprintf("Enabled (%s)", t.StreamViewType)
	}
	rows = append(rows, components.DetailRow{Label: "Streams", Value: streamStatus})

	if t.DeletionProtection {
		rows = append(rows, components.DetailRow{Label: "Delete Protection", Value: "Enabled"})
	}

	if !t.CreatedAt.IsZero() {
		rows = append(rows, components.DetailRow{Label: "", Value: ""}) // Spacer
		rows = append(rows, components.DetailRow{Label: "Created", Value: t.CreatedAt.Format("2006-01-02 15:04:05")})
	}

	m.details.SetTitle("DynamoDB Table Details")
	m.details.SetRows(rows)
}
