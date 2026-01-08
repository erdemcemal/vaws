package ui

import (
	"context"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"vaws/internal/model"
	"vaws/internal/state"
)

// updateTunnelsPanel updates the tunnels panel with current tunnel data.
func (m *Model) updateTunnelsPanel() {
	tunnels := m.tunnelManager.GetTunnels()
	var apiGWTunnels []model.APIGatewayTunnel
	if m.apiGWManager != nil {
		apiGWTunnels = m.apiGWManager.GetTunnels()
	}
	m.tunnelsPanel.SetTunnels(tunnels)
	m.tunnelsPanel.SetAPIGatewayTunnels(apiGWTunnels)
}

// startTunnel starts a tunnel with a random local port.
func (m *Model) startTunnel(service model.Service, task model.Task, container model.Container, remotePort int) tea.Cmd {
	return m.startTunnelWithPort(service, task, container, remotePort, 0)
}

// startTunnelWithPort starts a tunnel with a specific local port.
func (m *Model) startTunnelWithPort(service model.Service, task model.Task, container model.Container, remotePort, localPort int) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		tunnel, err := m.tunnelManager.StartTunnel(ctx, service, task, container, remotePort, localPort)
		return tunnelStartedMsg{tunnel: tunnel, err: err}
	}
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
// It prioritizes jump hosts in VPCs that have execute-api VPC endpoints.
func (m *Model) findJumpHostForAPIGateway(api interface{}, stage model.APIStage, localPort int) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		// First, find all VPCs that have execute-api endpoints
		vpcEndpoints, err := m.client.ListAPIGatewayVpcEndpoints(ctx)
		if err != nil {
			m.logger.Warn("Failed to list API Gateway VPC endpoints: %v", err)
			// Continue anyway - we'll try with whatever jump host we find
		}

		// Log which VPCs have execute-api endpoints
		if len(vpcEndpoints) > 0 {
			vpcIDs := make([]string, 0, len(vpcEndpoints))
			for vpcID := range vpcEndpoints {
				vpcIDs = append(vpcIDs, vpcID)
			}
			m.logger.Info("Found execute-api VPC endpoints in VPCs: %v", vpcIDs)
		} else {
			m.logger.Warn("No execute-api VPC endpoints found in account")
		}

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

		// Build list of preferred VPCs (those with execute-api endpoints)
		preferredVPCs := make([]string, 0, len(vpcEndpoints))
		for vpcID := range vpcEndpoints {
			preferredVPCs = append(preferredVPCs, vpcID)
		}

		// Find jump host, preferring VPCs with execute-api endpoints
		jumpHost, err := m.client.FindJumpHost(ctx, "", jumpHostConfig, jumpHostTagConfig, defaultTags, defaultNames, preferredVPCs...)
		if err != nil {
			return jumpHostFoundMsg{err: fmt.Errorf("failed to find jump host: %w", err)}
		}

		// Get VPC endpoint for the jump host's VPC
		var vpcEndpoint *model.VpcEndpoint
		var vpcEndpointErr error
		if jumpHost.VpcID != "" {
			if ep, ok := vpcEndpoints[jumpHost.VpcID]; ok {
				vpcEndpoint = ep
				m.logger.Info("Jump host %s is in VPC %s which has execute-api endpoint", jumpHost.Name, jumpHost.VpcID)
			} else {
				vpcEndpointErr = fmt.Errorf("no execute-api VPC endpoint in jump host's VPC %s", jumpHost.VpcID)
				m.logger.Warn("Jump host %s is in VPC %s which does NOT have execute-api endpoint", jumpHost.Name, jumpHost.VpcID)
			}
		}

		return jumpHostFoundMsg{
			jumpHost:          jumpHost,
			vpcEndpoint:       vpcEndpoint,
			vpcEndpointErr:    vpcEndpointErr,
			stage:             stage,
			api:               api,
			localPort:         localPort,
			vpcsWithEndpoints: preferredVPCs,
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
	m.updateAPIStagesList()

	// Get configured VPC endpoint ID for cross-account access
	var configuredVPCEndpointID string
	if m.cfg != nil {
		configuredVPCEndpointID = m.cfg.GetVPCEndpointID(m.state.Profile)
	}

	m.logger.Info("Starting private API Gateway tunnel via jump host: %s (VPC: %s)", jumpHost.Name, jumpHost.VpcID)

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		// First, discover all VPCs with execute-api endpoints for diagnostics
		vpcEndpoints, err := m.client.ListAPIGatewayVpcEndpoints(ctx)
		if err != nil {
			m.logger.Warn("Failed to list API Gateway VPC endpoints: %v", err)
		}

		// Log which VPCs have execute-api endpoints
		var vpcsWithEndpoints []string
		for vpcID := range vpcEndpoints {
			vpcsWithEndpoints = append(vpcsWithEndpoints, vpcID)
		}
		if len(vpcsWithEndpoints) > 0 {
			m.logger.Info("VPCs with execute-api endpoints: %v", vpcsWithEndpoints)
		} else {
			m.logger.Warn("No execute-api VPC endpoints found in this account!")
		}

		// Try to find VPC endpoint in jump host's VPC
		var vpcEndpoint *model.VpcEndpoint
		if jumpHost.VpcID != "" {
			if ep, ok := vpcEndpoints[jumpHost.VpcID]; ok {
				vpcEndpoint = ep
				m.logger.Info("Jump host VPC has execute-api endpoint: %s", ep.VpcEndpointID)
			} else {
				m.logger.Error("Jump host VPC (%s) does NOT have execute-api endpoint!", jumpHost.VpcID)
				if len(vpcsWithEndpoints) > 0 {
					m.logger.Error("Execute-api endpoints exist in: %v", vpcsWithEndpoints)
					m.logger.Error("Select a jump host in one of those VPCs, or configure vpc_endpoint_id")
				}
			}
		}

		tunnel, err := m.apiGWManager.StartPrivateTunnel(ctx, api, *stage, jumpHost, vpcEndpoint, configuredVPCEndpointID, localPort)
		return apiGWTunnelStartedMsg{tunnel: tunnel, err: err}
	}
}
