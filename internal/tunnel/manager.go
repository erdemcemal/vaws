// Package tunnel manages SSM port forwarding sessions.
package tunnel

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	"vaws/internal/log"
	"vaws/internal/model"
)

// Manager handles port forwarding tunnels.
type Manager struct {
	mu      sync.RWMutex
	tunnels map[string]*activeTunnel
	region  string
	profile string
}

type activeTunnel struct {
	model.Tunnel
	cmd       *exec.Cmd
	cancel    context.CancelFunc
	stderrBuf *bytes.Buffer
	process   *os.Process // For re-adopted tunnels where we only have the process
}

// NewManager creates a new tunnel manager.
func NewManager(profile, region string) *Manager {
	m := &Manager{
		tunnels: make(map[string]*activeTunnel),
		region:  region,
		profile: profile,
	}

	// Load tunnels from previous session
	if err := m.loadTunnels(); err != nil {
		log.Warn("Failed to load persisted tunnels: %v", err)
	}

	return m
}

// StartTunnel starts a new port forwarding tunnel.
func (m *Manager) StartTunnel(ctx context.Context, service model.Service, task model.Task, container model.Container, remotePort, localPort int) (*model.Tunnel, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if requested port is already in use by an active tunnel
	if localPort != 0 {
		for _, t := range m.tunnels {
			if t.LocalPort == localPort && (t.Status == model.TunnelStatusActive || t.Status == model.TunnelStatusStarting) {
				return nil, fmt.Errorf("port %d is already in use by tunnel '%s'. Stop it first or use a different port", localPort, t.ID)
			}
		}
	}

	// Find a free local port if not specified
	if localPort == 0 {
		var err error
		localPort, err = m.findFreePortExcludingActive()
		if err != nil {
			return nil, fmt.Errorf("failed to find free port: %w", err)
		}
	}

	// Create tunnel ID
	tunnelID := fmt.Sprintf("%s-%s-%d", service.Name, task.TaskID[:8], localPort)

	// Check if tunnel already exists
	if _, exists := m.tunnels[tunnelID]; exists {
		return nil, fmt.Errorf("tunnel %s already exists", tunnelID)
	}

	// Build SSM target
	// Format: ecs:<cluster-name>_<task-id>_<runtime-id>
	target := fmt.Sprintf("ecs:%s_%s_%s", service.ClusterName, task.TaskID, container.RuntimeID)

	// Create the tunnel model
	tunnel := model.Tunnel{
		ID:            tunnelID,
		LocalPort:     localPort,
		RemotePort:    remotePort,
		ServiceName:   service.Name,
		ClusterARN:    service.ClusterARN,
		ClusterName:   service.ClusterName,
		TaskID:        task.TaskID,
		ContainerName: container.Name,
		Status:        model.TunnelStatusStarting,
		StartedAt:     time.Now(),
	}

	// Build AWS SSM command
	args := []string{
		"ssm", "start-session",
		"--target", target,
		"--document-name", "AWS-StartPortForwardingSession",
		"--parameters", fmt.Sprintf(`{"portNumber":["%d"],"localPortNumber":["%d"]}`, remotePort, localPort),
	}

	if m.region != "" {
		args = append(args, "--region", m.region)
	}
	if m.profile != "" {
		args = append(args, "--profile", m.profile)
	}

	// Create cancellable context for the process
	// Use Background context so the tunnel isn't killed when the caller's context times out
	cmdCtx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(cmdCtx, "aws", args...)

	// Set process group so we can kill all child processes (session-manager-plugin)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	// Capture stderr for error messages
	var stderrBuf bytes.Buffer
	cmd.Stderr = &stderrBuf

	// Start the tunnel process
	log.Info("Starting tunnel: %s -> localhost:%d (remote port %d)", target, localPort, remotePort)
	log.Debug("Running: aws %v", args)
	if err := cmd.Start(); err != nil {
		cancel()
		tunnel.Status = model.TunnelStatusError
		tunnel.Error = err.Error()
		return &tunnel, fmt.Errorf("failed to start tunnel: %w", err)
	}

	tunnel.Status = model.TunnelStatusActive

	// Store the active tunnel
	at := &activeTunnel{
		Tunnel:    tunnel,
		cmd:       cmd,
		cancel:    cancel,
		stderrBuf: &stderrBuf,
	}
	m.tunnels[tunnelID] = at

	// Monitor the process in background
	go m.monitorTunnel(tunnelID, at)

	log.Info("Tunnel started: %s on localhost:%d", tunnelID, localPort)

	// Save tunnels to disk for persistence
	go func() {
		if err := m.saveTunnels(); err != nil {
			log.Debug("Failed to save tunnels: %v", err)
		}
	}()

	return &tunnel, nil
}

// monitorTunnel watches a tunnel process and updates status when it exits.
func (m *Manager) monitorTunnel(id string, at *activeTunnel) {
	err := at.cmd.Wait()

	m.mu.Lock()

	if t, exists := m.tunnels[id]; exists {
		if err != nil {
			t.Status = model.TunnelStatusError
			// Include stderr output in error message for better debugging
			errMsg := err.Error()
			if at.stderrBuf != nil && at.stderrBuf.Len() > 0 {
				stderr := strings.TrimSpace(at.stderrBuf.String())
				if stderr != "" {
					errMsg = stderr
				}
			}
			t.Error = errMsg
			log.Error("Tunnel %s exited with error: %s", id, errMsg)
		} else {
			t.Status = model.TunnelStatusTerminated
			log.Info("Tunnel %s terminated normally", id)
		}
	}

	m.mu.Unlock()

	// Update persistence
	if err := m.saveTunnels(); err != nil {
		log.Debug("Failed to save tunnels: %v", err)
	}
}

// StopTunnel stops an active tunnel.
func (m *Manager) StopTunnel(id string) error {
	m.mu.Lock()
	tunnel, exists := m.tunnels[id]
	if !exists {
		m.mu.Unlock()
		return fmt.Errorf("tunnel %s not found", id)
	}

	// Cancel the context (if we created it)
	if tunnel.cancel != nil {
		tunnel.cancel()
	}

	// Kill the entire process group to ensure child processes (session-manager-plugin) are killed
	if tunnel.cmd != nil && tunnel.cmd.Process != nil {
		pid := tunnel.cmd.Process.Pid
		// Kill entire process group with negative PID
		if err := syscall.Kill(-pid, syscall.SIGKILL); err != nil {
			log.Debug("Kill process group error (trying single process): %v", err)
			// Fallback to killing just the process
			if err := tunnel.cmd.Process.Kill(); err != nil {
				log.Debug("Kill process error (may already be dead): %v", err)
			}
		}
	} else if tunnel.process != nil {
		// For re-adopted tunnels where we only have the process
		pid := tunnel.process.Pid
		// Try to kill process group first
		if err := syscall.Kill(-pid, syscall.SIGKILL); err != nil {
			log.Debug("Kill re-adopted process group error (trying single process): %v", err)
			if err := tunnel.process.Kill(); err != nil {
				log.Debug("Kill re-adopted process error (may already be dead): %v", err)
			}
		}
	}

	// Update status
	tunnel.Status = model.TunnelStatusTerminated
	localPort := tunnel.LocalPort
	m.mu.Unlock()

	log.Info("Stopped tunnel: %s (localhost:%d)", id, localPort)

	// Save updated state to disk
	if err := m.saveTunnels(); err != nil {
		log.Debug("Failed to save tunnels: %v", err)
	}

	return nil
}

// StopAllTunnels stops all active tunnels.
func (m *Manager) StopAllTunnels() {
	m.mu.Lock()

	for id, tunnel := range m.tunnels {
		if tunnel.Status != model.TunnelStatusActive && tunnel.Status != model.TunnelStatusStarting {
			continue
		}
		if tunnel.cancel != nil {
			tunnel.cancel()
		}
		// Kill entire process group to ensure child processes are killed
		if tunnel.cmd != nil && tunnel.cmd.Process != nil {
			pid := tunnel.cmd.Process.Pid
			if err := syscall.Kill(-pid, syscall.SIGKILL); err != nil {
				tunnel.cmd.Process.Kill()
			}
		} else if tunnel.process != nil {
			pid := tunnel.process.Pid
			if err := syscall.Kill(-pid, syscall.SIGKILL); err != nil {
				tunnel.process.Kill()
			}
		}
		tunnel.Status = model.TunnelStatusTerminated
		log.Info("Stopped tunnel: %s", id)
	}

	m.mu.Unlock()

	// Save updated state (tunnels remain for restart capability)
	if err := m.saveTunnels(); err != nil {
		log.Debug("Failed to save tunnels: %v", err)
	}
}

// GetTunnels returns all tunnels (active and terminated).
func (m *Manager) GetTunnels() []model.Tunnel {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tunnels := make([]model.Tunnel, 0, len(m.tunnels))
	for _, t := range m.tunnels {
		tunnels = append(tunnels, t.Tunnel)
	}
	return tunnels
}

// GetActiveTunnels returns only active tunnels.
func (m *Manager) GetActiveTunnels() []model.Tunnel {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var tunnels []model.Tunnel
	for _, t := range m.tunnels {
		if t.Status == model.TunnelStatusActive || t.Status == model.TunnelStatusStarting {
			tunnels = append(tunnels, t.Tunnel)
		}
	}
	return tunnels
}

// GetTunnel returns a specific tunnel by ID.
func (m *Manager) GetTunnel(id string) (*model.Tunnel, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if t, exists := m.tunnels[id]; exists {
		return &t.Tunnel, true
	}
	return nil, false
}

// RemoveTunnel removes a terminated tunnel from the list.
func (m *Manager) RemoveTunnel(id string) {
	m.mu.Lock()

	if t, exists := m.tunnels[id]; exists {
		if t.Status == model.TunnelStatusTerminated || t.Status == model.TunnelStatusError {
			delete(m.tunnels, id)
		}
	}

	m.mu.Unlock()

	// Update persistence
	if err := m.saveTunnels(); err != nil {
		log.Debug("Failed to save tunnels: %v", err)
	}
}

// ClearTerminated removes all terminated/errored tunnels.
func (m *Manager) ClearTerminated() {
	m.mu.Lock()

	for id, t := range m.tunnels {
		if t.Status == model.TunnelStatusTerminated || t.Status == model.TunnelStatusError {
			delete(m.tunnels, id)
		}
	}

	remaining := len(m.tunnels)
	m.mu.Unlock()

	// Update or clear persistence
	if remaining == 0 {
		if err := clearPersistence(); err != nil {
			log.Debug("Failed to clear tunnel persistence: %v", err)
		}
	} else {
		if err := m.saveTunnels(); err != nil {
			log.Debug("Failed to save tunnels: %v", err)
		}
	}
}

// ActiveCount returns the number of active tunnels.
func (m *Manager) ActiveCount() int {
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

// PrepareRestart removes a terminated tunnel and returns its info for restart.
// Returns the tunnel info needed to restart (service name, cluster ARN, ports).
func (m *Manager) PrepareRestart(id string) (*model.Tunnel, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	t, exists := m.tunnels[id]
	if !exists {
		return nil, fmt.Errorf("tunnel %s not found", id)
	}

	if t.Status == model.TunnelStatusActive || t.Status == model.TunnelStatusStarting {
		return nil, fmt.Errorf("tunnel %s is still active, stop it first", id)
	}

	// Copy tunnel info before removing
	tunnelCopy := t.Tunnel

	// Remove the old tunnel entry so we can create a new one
	delete(m.tunnels, id)

	return &tunnelCopy, nil
}

// findFreePort finds an available port on localhost.
func findFreePort() (int, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port, nil
}

// findFreePortExcludingActive finds a free port that's not used by any active tunnel.
// Must be called with m.mu held.
func (m *Manager) findFreePortExcludingActive() (int, error) {
	// Collect ports used by active tunnels
	usedPorts := make(map[int]bool)
	for _, t := range m.tunnels {
		if t.Status == model.TunnelStatusActive || t.Status == model.TunnelStatusStarting {
			usedPorts[t.LocalPort] = true
		}
	}

	// Try up to 10 times to find a port not used by active tunnels
	for i := 0; i < 10; i++ {
		port, err := findFreePort()
		if err != nil {
			return 0, err
		}
		if !usedPorts[port] {
			return port, nil
		}
	}

	// Fallback to just returning a free port
	return findFreePort()
}
