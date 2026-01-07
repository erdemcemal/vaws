package tunnel

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	"vaws/internal/log"
	"vaws/internal/model"
)

// APIGatewayManager handles API Gateway port forwarding tunnels.
type APIGatewayManager struct {
	mu      sync.RWMutex
	tunnels map[string]*activeAPIGWTunnel
	region  string
	profile string
}

type activeAPIGWTunnel struct {
	model.APIGatewayTunnel
	cmd       *exec.Cmd    // For SSM tunnels (private API GW)
	server    *http.Server // For HTTP proxy (public API GW)
	cancel    context.CancelFunc
	stderrBuf *bytes.Buffer
	stdoutBuf *bytes.Buffer
}

// NewAPIGatewayManager creates a new API Gateway tunnel manager.
func NewAPIGatewayManager(profile, region string) *APIGatewayManager {
	return &APIGatewayManager{
		tunnels: make(map[string]*activeAPIGWTunnel),
		region:  region,
		profile: profile,
	}
}

// StartPublicTunnel starts a local HTTP proxy for public API Gateway.
func (m *APIGatewayManager) StartPublicTunnel(ctx context.Context, api interface{}, stage model.APIStage, localPort int) (*model.APIGatewayTunnel, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Extract API info
	var apiName, apiID, apiType string
	switch a := api.(type) {
	case model.RestAPI:
		apiName = a.Name
		apiID = a.ID
		apiType = "REST"
	case *model.RestAPI:
		apiName = a.Name
		apiID = a.ID
		apiType = "REST"
	case model.HttpAPI:
		apiName = a.Name
		apiID = a.ID
		apiType = "HTTP"
	case *model.HttpAPI:
		apiName = a.Name
		apiID = a.ID
		apiType = "HTTP"
	default:
		return nil, fmt.Errorf("unsupported API type: %T", api)
	}

	// Find a free local port if not specified
	if localPort == 0 {
		var err error
		localPort, err = m.findFreePort()
		if err != nil {
			return nil, fmt.Errorf("failed to find free port: %w", err)
		}
	}

	// Check if port is already in use
	for _, t := range m.tunnels {
		if t.LocalPort == localPort && (t.Status == model.TunnelStatusActive || t.Status == model.TunnelStatusStarting) {
			return nil, fmt.Errorf("port %d is already in use by tunnel '%s'", localPort, t.ID)
		}
	}

	tunnelID := fmt.Sprintf("apigw-%s-%s-%d", apiID, stage.Name, localPort)

	if _, exists := m.tunnels[tunnelID]; exists {
		return nil, fmt.Errorf("tunnel %s already exists", tunnelID)
	}

	tunnel := model.APIGatewayTunnel{
		ID:         tunnelID,
		LocalPort:  localPort,
		APIName:    apiName,
		APIID:      apiID,
		APIType:    apiType,
		StageName:  stage.Name,
		InvokeURL:  stage.InvokeURL,
		TunnelType: model.APIGatewayTunnelPublic,
		Status:     model.TunnelStatusStarting,
		StartedAt:  time.Now(),
	}

	// Parse the target URL
	targetURL, err := url.Parse(stage.InvokeURL)
	if err != nil {
		return nil, fmt.Errorf("invalid invoke URL: %w", err)
	}

	// Create reverse proxy
	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	// Customize the director to properly forward requests
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.Host = targetURL.Host
		req.URL.Host = targetURL.Host
		req.URL.Scheme = targetURL.Scheme
		// Preserve the original path after the stage
		if !strings.HasPrefix(req.URL.Path, targetURL.Path) {
			req.URL.Path = targetURL.Path + req.URL.Path
		}
	}

	// Error handler for proxy
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Error("Proxy error for %s: %v", r.URL.Path, err)
		w.WriteHeader(http.StatusBadGateway)
		fmt.Fprintf(w, "Proxy error: %v", err)
	}

	// Create HTTP server
	server := &http.Server{
		Addr:    fmt.Sprintf("127.0.0.1:%d", localPort),
		Handler: proxy,
	}

	// Create cancellable context
	serverCtx, cancel := context.WithCancel(context.Background())

	// Store the tunnel
	at := &activeAPIGWTunnel{
		APIGatewayTunnel: tunnel,
		server:           server,
		cancel:           cancel,
	}
	m.tunnels[tunnelID] = at

	// Start server in background
	go func() {
		ln, err := net.Listen("tcp", server.Addr)
		if err != nil {
			m.mu.Lock()
			at.Status = model.TunnelStatusError
			at.Error = fmt.Sprintf("failed to listen: %v", err)
			m.mu.Unlock()
			log.Error("Failed to start proxy server: %v", err)
			return
		}

		m.mu.Lock()
		at.Status = model.TunnelStatusActive
		m.mu.Unlock()
		log.Info("API Gateway proxy started:")
		log.Info("  Target URL: %s", stage.InvokeURL)
		log.Info("  Local Port: localhost:%d", localPort)
		log.Info("  Usage: curl http://localhost:%d/your-path", localPort)

		go func() {
			<-serverCtx.Done()
			server.Close()
		}()

		if err := server.Serve(ln); err != nil && err != http.ErrServerClosed {
			m.mu.Lock()
			at.Status = model.TunnelStatusError
			at.Error = err.Error()
			m.mu.Unlock()
			log.Error("Proxy server error: %v", err)
		} else {
			m.mu.Lock()
			at.Status = model.TunnelStatusTerminated
			m.mu.Unlock()
		}
	}()

	// Wait a bit for server to start
	time.Sleep(100 * time.Millisecond)

	return &at.APIGatewayTunnel, nil
}

// StartPrivateTunnel starts an SSM tunnel through a jump host for private API Gateway.
// configuredVPCEndpointID is used for cross-account access (URL format: <api-id>-<vpce-id>.execute-api.<region>.amazonaws.com)
func (m *APIGatewayManager) StartPrivateTunnel(ctx context.Context, api interface{}, stage model.APIStage, jumpHost *model.EC2Instance, vpcEndpoint *model.VpcEndpoint, configuredVPCEndpointID string, localPort int) (*model.APIGatewayTunnel, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Extract API info
	var apiName, apiID, apiType, endpointType string
	var apiVpcEndpointIds []string
	switch a := api.(type) {
	case model.RestAPI:
		apiName = a.Name
		apiID = a.ID
		apiType = "REST"
		endpointType = a.EndpointType
		apiVpcEndpointIds = a.VpcEndpointIds
	case *model.RestAPI:
		apiName = a.Name
		apiID = a.ID
		apiType = "REST"
		endpointType = a.EndpointType
		apiVpcEndpointIds = a.VpcEndpointIds
	case model.HttpAPI:
		apiName = a.Name
		apiID = a.ID
		apiType = "HTTP"
		endpointType = "REGIONAL" // HTTP APIs don't have private endpoint type
	case *model.HttpAPI:
		apiName = a.Name
		apiID = a.ID
		apiType = "HTTP"
		endpointType = "REGIONAL" // HTTP APIs don't have private endpoint type
	default:
		return nil, fmt.Errorf("unsupported API type: %T", api)
	}

	// Find a free local port if not specified
	if localPort == 0 {
		var err error
		localPort, err = m.findFreePort()
		if err != nil {
			return nil, fmt.Errorf("failed to find free port: %w", err)
		}
	}

	// Check if port is already in use
	for _, t := range m.tunnels {
		if t.LocalPort == localPort && (t.Status == model.TunnelStatusActive || t.Status == model.TunnelStatusStarting) {
			return nil, fmt.Errorf("port %d is already in use by tunnel '%s'", localPort, t.ID)
		}
	}

	tunnelID := fmt.Sprintf("apigw-private-%s-%s-%d", apiID, stage.Name, localPort)

	if _, exists := m.tunnels[tunnelID]; exists {
		return nil, fmt.Errorf("tunnel %s already exists", tunnelID)
	}

	// Determine remote host and port
	// For private API Gateway, we need a VPC endpoint ID. Priority:
	// 1. API's own VpcEndpointIds (from API Gateway configuration - best source)
	// 2. Configured vpc_endpoint_id in config file (manual override)
	// 3. Discovered VPC endpoint from the VPC
	// 4. For non-private APIs, standard DNS works
	var remoteHost string
	isPrivateAPI := strings.ToUpper(endpointType) == "PRIVATE"

	if len(apiVpcEndpointIds) > 0 {
		// Use the API's configured VPC endpoint ID (auto-detected)
		vpcEndpointID := apiVpcEndpointIds[0]
		remoteHost = fmt.Sprintf("%s-%s.execute-api.%s.amazonaws.com", apiID, vpcEndpointID, m.region)
		log.Info("Using API's VPC endpoint: %s", vpcEndpointID)
		log.Info("Remote host: %s", remoteHost)
	} else if configuredVPCEndpointID != "" {
		// Cross-account access: use the configured VPC endpoint ID in URL
		// Format: <api-id>-<vpce-id>.execute-api.<region>.amazonaws.com
		remoteHost = fmt.Sprintf("%s-%s.execute-api.%s.amazonaws.com", apiID, configuredVPCEndpointID, m.region)
		log.Info("Using configured VPC endpoint: %s", configuredVPCEndpointID)
	} else if vpcEndpoint != nil && len(vpcEndpoint.DNSEntries) > 0 {
		// Use discovered VPC endpoint DNS (same account)
		remoteHost = vpcEndpoint.DNSEntries[0]
		log.Info("Using discovered VPC endpoint DNS: %s", remoteHost)
	} else if isPrivateAPI {
		// Private API without VPC endpoint - provide helpful error
		return nil, fmt.Errorf("private API Gateway has no VPC endpoint configured.\n" +
			"The API should have vpcEndpointIds in its endpoint configuration.\n" +
			"Check the API Gateway settings or add vpc_endpoint_id to ~/.vaws/config.yaml")
	} else {
		// Regional/Edge API - use standard DNS (jump host should be able to reach public internet)
		remoteHost = fmt.Sprintf("%s.execute-api.%s.amazonaws.com", apiID, m.region)
		log.Info("Using standard API Gateway DNS: %s (public endpoint via jump host)", remoteHost)
	}
	remotePort := 443

	// Find an internal port for the SSM tunnel (user won't interact with this directly)
	ssmPort, err := m.findFreePort()
	if err != nil {
		return nil, fmt.Errorf("failed to find free port for SSM tunnel: %w", err)
	}

	tunnel := model.APIGatewayTunnel{
		ID:          tunnelID,
		LocalPort:   localPort,
		APIName:     apiName,
		APIID:       apiID,
		APIType:     apiType,
		StageName:   stage.Name,
		InvokeURL:   fmt.Sprintf("http://localhost:%d", localPort), // Stage name is auto-prepended
		TunnelType:  model.APIGatewayTunnelPrivate,
		JumpHost:    jumpHost,
		VpcEndpoint: vpcEndpoint,
		Status:      model.TunnelStatusStarting,
		StartedAt:   time.Now(),
	}

	// Build SSM port forwarding command to remote host
	// Using AWS-StartPortForwardingSessionToRemoteHost document
	// SSM tunnel listens on ssmPort, forwarding to remoteHost:443
	args := []string{
		"ssm", "start-session",
		"--target", jumpHost.InstanceID,
		"--document-name", "AWS-StartPortForwardingSessionToRemoteHost",
		"--parameters", fmt.Sprintf(`{"host":["%s"],"portNumber":["%d"],"localPortNumber":["%d"]}`, remoteHost, remotePort, ssmPort),
	}

	if m.region != "" {
		args = append(args, "--region", m.region)
	}
	if m.profile != "" {
		args = append(args, "--profile", m.profile)
	}

	// Create cancellable context for both SSM and proxy
	cmdCtx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(cmdCtx, "aws", args...)

	// Set process group
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	// Capture stdout and stderr
	var stderrBuf bytes.Buffer
	var stdoutBuf bytes.Buffer
	cmd.Stderr = &stderrBuf
	cmd.Stdout = &stdoutBuf

	log.Info("Starting private API Gateway tunnel:")
	log.Info("  Jump Host: %s (%s)", jumpHost.Name, jumpHost.InstanceID)
	log.Info("  Remote Host: %s:%d", remoteHost, remotePort)
	log.Info("  SSM Tunnel Port: %d (internal)", ssmPort)
	log.Info("  HTTP Proxy Port: %d (use this)", localPort)
	log.Debug("Running: aws %v", args)

	if err := cmd.Start(); err != nil {
		cancel()
		tunnel.Status = model.TunnelStatusError
		tunnel.Error = err.Error()
		return &tunnel, fmt.Errorf("failed to start SSM tunnel: %w", err)
	}

	// Wait for SSM tunnel to establish (SSM needs time to set up the port forwarding)
	time.Sleep(2 * time.Second)

	// Create HTTP reverse proxy that forwards to the SSM tunnel with proper TLS
	proxy, err := m.createPrivateAPIProxy(remoteHost, ssmPort, stage.Name)
	if err != nil {
		cancel()
		cmd.Process.Kill()
		return nil, fmt.Errorf("failed to create HTTP proxy: %w", err)
	}

	// Create HTTP server for the proxy
	server := &http.Server{
		Addr:    fmt.Sprintf("127.0.0.1:%d", localPort),
		Handler: proxy,
	}

	// Start the HTTP proxy server
	proxyListener, err := net.Listen("tcp", server.Addr)
	if err != nil {
		cancel()
		cmd.Process.Kill()
		return nil, fmt.Errorf("failed to start HTTP proxy: %w", err)
	}

	tunnel.Status = model.TunnelStatusActive

	at := &activeAPIGWTunnel{
		APIGatewayTunnel: tunnel,
		cmd:              cmd,
		server:           server,
		cancel:           cancel,
		stderrBuf:        &stderrBuf,
		stdoutBuf:        &stdoutBuf,
	}
	m.tunnels[tunnelID] = at

	// Start HTTP proxy server in background
	go func() {
		if err := server.Serve(proxyListener); err != nil && err != http.ErrServerClosed {
			log.Error("HTTP proxy error: %v", err)
		}
	}()

	// Shutdown proxy when context is cancelled
	go func() {
		<-cmdCtx.Done()
		server.Close()
	}()

	// Monitor SSM tunnel in background
	go m.monitorSSMTunnel(tunnelID, at)

	log.Info("Private API Gateway tunnel started!")
	log.Info("  Stage: %s (automatically prepended to requests)", stage.Name)
	log.Info("  Usage: curl http://localhost:%d/your-endpoint", localPort)

	return &at.APIGatewayTunnel, nil
}

// createPrivateAPIProxy creates a reverse proxy that forwards HTTP requests to the SSM tunnel as HTTPS.
func (m *APIGatewayManager) createPrivateAPIProxy(remoteHost string, ssmPort int, stageName string) (http.Handler, error) {
	// Create a custom transport that:
	// 1. Connects to the local SSM tunnel port
	// 2. Uses TLS with the correct ServerName (SNI)
	// 3. Skips certificate verification (tunnel is local, cert is for API Gateway domain)
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			ServerName:         remoteHost,
			InsecureSkipVerify: true, // Safe: we're connecting to localhost tunnel
		},
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			// Always connect to the local SSM tunnel instead of the remote address
			return net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", ssmPort))
		},
	}

	targetURL := &url.URL{
		Scheme: "https",
		Host:   remoteHost,
	}

	proxy := httputil.NewSingleHostReverseProxy(targetURL)
	proxy.Transport = transport

	// Stage path prefix - API Gateway requires stage name in URL
	stagePrefix := "/" + stageName

	// Customize director to set proper headers and prepend stage name
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.Host = remoteHost
		req.URL.Host = remoteHost
		req.URL.Scheme = "https"
		// Prepend stage name to path if not already present
		if !strings.HasPrefix(req.URL.Path, stagePrefix) {
			req.URL.Path = stagePrefix + req.URL.Path
		}
	}

	// Error handler
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Error("Proxy error for %s: %v", r.URL.Path, err)
		w.WriteHeader(http.StatusBadGateway)
		fmt.Fprintf(w, "Proxy error: %v\n\nMake sure the SSM tunnel is still active.", err)
	}

	return proxy, nil
}

// monitorSSMTunnel watches an SSM tunnel process.
func (m *APIGatewayManager) monitorSSMTunnel(id string, at *activeAPIGWTunnel) {
	err := at.cmd.Wait()

	m.mu.Lock()
	defer m.mu.Unlock()

	if t, exists := m.tunnels[id]; exists {
		// Check both stdout and stderr for useful info
		var outputs []string
		if at.stderrBuf != nil && at.stderrBuf.Len() > 0 {
			stderr := strings.TrimSpace(at.stderrBuf.String())
			if stderr != "" {
				outputs = append(outputs, "stderr: "+stderr)
			}
		}
		if at.stdoutBuf != nil && at.stdoutBuf.Len() > 0 {
			stdout := strings.TrimSpace(at.stdoutBuf.String())
			if stdout != "" {
				outputs = append(outputs, "stdout: "+stdout)
			}
		}
		combinedOutput := strings.Join(outputs, "; ")

		if err != nil {
			t.Status = model.TunnelStatusError
			errMsg := err.Error()
			if combinedOutput != "" {
				errMsg = combinedOutput
			}
			t.Error = errMsg
			log.Error("API Gateway tunnel %s exited with error: %s", id, errMsg)
		} else {
			t.Status = model.TunnelStatusTerminated
			if combinedOutput != "" {
				log.Warn("API Gateway tunnel %s terminated. Output: %s", id, combinedOutput)
				t.Error = combinedOutput
			} else {
				log.Info("API Gateway tunnel %s terminated (no output from SSM)", id)
			}
		}
	}
}

// StopTunnel stops an active API Gateway tunnel.
func (m *APIGatewayManager) StopTunnel(id string) error {
	m.mu.Lock()
	tunnel, exists := m.tunnels[id]
	if !exists {
		m.mu.Unlock()
		return fmt.Errorf("tunnel %s not found", id)
	}

	// Cancel context
	if tunnel.cancel != nil {
		tunnel.cancel()
	}

	// Stop HTTP server (for public tunnels)
	if tunnel.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		tunnel.server.Shutdown(ctx)
	}

	// Kill SSM process (for private tunnels)
	if tunnel.cmd != nil && tunnel.cmd.Process != nil {
		pid := tunnel.cmd.Process.Pid
		if err := syscall.Kill(-pid, syscall.SIGKILL); err != nil {
			tunnel.cmd.Process.Kill()
		}
	}

	tunnel.Status = model.TunnelStatusTerminated
	localPort := tunnel.LocalPort
	m.mu.Unlock()

	log.Info("Stopped API Gateway tunnel: %s (localhost:%d)", id, localPort)
	return nil
}

// StopAllTunnels stops all active API Gateway tunnels.
func (m *APIGatewayManager) StopAllTunnels() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for id, tunnel := range m.tunnels {
		if tunnel.Status != model.TunnelStatusActive && tunnel.Status != model.TunnelStatusStarting {
			continue
		}

		if tunnel.cancel != nil {
			tunnel.cancel()
		}

		if tunnel.server != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			tunnel.server.Shutdown(ctx)
			cancel()
		}

		if tunnel.cmd != nil && tunnel.cmd.Process != nil {
			pid := tunnel.cmd.Process.Pid
			if err := syscall.Kill(-pid, syscall.SIGKILL); err != nil {
				tunnel.cmd.Process.Kill()
			}
		}

		tunnel.Status = model.TunnelStatusTerminated
		log.Info("Stopped API Gateway tunnel: %s", id)
	}
}

// GetTunnels returns all API Gateway tunnels.
func (m *APIGatewayManager) GetTunnels() []model.APIGatewayTunnel {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tunnels := make([]model.APIGatewayTunnel, 0, len(m.tunnels))
	for _, t := range m.tunnels {
		tunnels = append(tunnels, t.APIGatewayTunnel)
	}
	return tunnels
}

// GetActiveTunnels returns only active API Gateway tunnels.
func (m *APIGatewayManager) GetActiveTunnels() []model.APIGatewayTunnel {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var tunnels []model.APIGatewayTunnel
	for _, t := range m.tunnels {
		if t.Status == model.TunnelStatusActive || t.Status == model.TunnelStatusStarting {
			tunnels = append(tunnels, t.APIGatewayTunnel)
		}
	}
	return tunnels
}

// GetTunnel returns a specific tunnel by ID.
func (m *APIGatewayManager) GetTunnel(id string) (*model.APIGatewayTunnel, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if t, exists := m.tunnels[id]; exists {
		return &t.APIGatewayTunnel, true
	}
	return nil, false
}

// RemoveTunnel removes a terminated tunnel.
func (m *APIGatewayManager) RemoveTunnel(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if t, exists := m.tunnels[id]; exists {
		if t.Status == model.TunnelStatusTerminated || t.Status == model.TunnelStatusError {
			delete(m.tunnels, id)
		}
	}
}

// ClearTerminated removes all terminated/errored tunnels.
func (m *APIGatewayManager) ClearTerminated() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for id, t := range m.tunnels {
		if t.Status == model.TunnelStatusTerminated || t.Status == model.TunnelStatusError {
			delete(m.tunnels, id)
		}
	}
}

// ActiveCount returns the number of active tunnels.
func (m *APIGatewayManager) ActiveCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	count := 0
	for _, t := range m.tunnels {
		if t.Status == model.TunnelStatusActive || t.Status == model.TunnelStatusStarting {
			count++
		}
	}
	return count
}

// findFreePort finds an available port on localhost.
func (m *APIGatewayManager) findFreePort() (int, error) {
	// Collect ports used by active tunnels
	usedPorts := make(map[int]bool)
	for _, t := range m.tunnels {
		if t.Status == model.TunnelStatusActive || t.Status == model.TunnelStatusStarting {
			usedPorts[t.LocalPort] = true
		}
	}

	for i := 0; i < 10; i++ {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			return 0, err
		}
		port := listener.Addr().(*net.TCPAddr).Port
		listener.Close()
		if !usedPorts[port] {
			return port, nil
		}
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port, nil
}

// SetRegion updates the region for the manager.
func (m *APIGatewayManager) SetRegion(region string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.region = region
}

// SetProfile updates the profile for the manager.
func (m *APIGatewayManager) SetProfile(profile string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.profile = profile
}
