package tunnel

import (
	"encoding/json"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"vaws/internal/log"
	"vaws/internal/model"
)

// persistedTunnel represents tunnel data saved to disk.
type persistedTunnel struct {
	ID            string             `json:"id"`
	PID           int                `json:"pid"`
	LocalPort     int                `json:"local_port"`
	RemotePort    int                `json:"remote_port"`
	ServiceName   string             `json:"service_name"`
	ClusterARN    string             `json:"cluster_arn"`
	ClusterName   string             `json:"cluster_name"`
	TaskID        string             `json:"task_id"`
	ContainerName string             `json:"container_name"`
	StartedAt     time.Time          `json:"started_at"`
	Status        model.TunnelStatus `json:"status"`
	Error         string             `json:"error,omitempty"`
}

// persistenceFile returns the path to the tunnels persistence file.
func persistenceFile() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	dir := filepath.Join(homeDir, ".vaws")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}

	return filepath.Join(dir, "tunnels.json"), nil
}

// saveTunnels saves the current tunnels to disk.
// Saves both active and terminated tunnels (for restart capability).
func (m *Manager) saveTunnels() error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var tunnels []persistedTunnel
	for _, t := range m.tunnels {
		pt := persistedTunnel{
			ID:            t.ID,
			LocalPort:     t.LocalPort,
			RemotePort:    t.RemotePort,
			ServiceName:   t.ServiceName,
			ClusterARN:    t.ClusterARN,
			ClusterName:   t.ClusterName,
			TaskID:        t.TaskID,
			ContainerName: t.ContainerName,
			StartedAt:     t.StartedAt,
			Status:        t.Status,
			Error:         t.Error,
		}

		// Include PID for active tunnels
		if (t.Status == model.TunnelStatusActive || t.Status == model.TunnelStatusStarting) &&
			t.cmd != nil && t.cmd.Process != nil {
			pt.PID = t.cmd.Process.Pid
		}

		tunnels = append(tunnels, pt)
	}

	file, err := persistenceFile()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(tunnels, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(file, data, 0644); err != nil {
		return err
	}

	log.Debug("Saved %d tunnels to %s", len(tunnels), file)
	return nil
}

// loadTunnels loads persisted tunnels and re-adopts running processes.
// Also loads terminated tunnels for restart capability.
func (m *Manager) loadTunnels() error {
	file, err := persistenceFile()
	if err != nil {
		return err
	}

	data, err := os.ReadFile(file)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No saved tunnels
		}
		return err
	}

	var tunnels []persistedTunnel
	if err := json.Unmarshal(data, &tunnels); err != nil {
		log.Warn("Failed to parse tunnels file: %v", err)
		return nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	adopted := 0
	loaded := 0
	for _, pt := range tunnels {
		// Create the tunnel model
		tunnel := model.Tunnel{
			ID:            pt.ID,
			LocalPort:     pt.LocalPort,
			RemotePort:    pt.RemotePort,
			ServiceName:   pt.ServiceName,
			ClusterARN:    pt.ClusterARN,
			ClusterName:   pt.ClusterName,
			TaskID:        pt.TaskID,
			ContainerName: pt.ContainerName,
			StartedAt:     pt.StartedAt,
			Error:         pt.Error,
		}

		// For tunnels that were active, check if process is still running
		if pt.Status == model.TunnelStatusActive || pt.Status == model.TunnelStatusStarting {
			if pt.PID > 0 && isProcessRunning(pt.PID) {
				// Re-adopt the running tunnel
				tunnel.Status = model.TunnelStatusActive

				process, err := os.FindProcess(pt.PID)
				if err != nil {
					log.Debug("Could not find process %d: %v", pt.PID, err)
					tunnel.Status = model.TunnelStatusTerminated
				} else {
					m.tunnels[pt.ID] = &activeTunnel{
						Tunnel:    tunnel,
						cmd:       nil,
						cancel:    func() {},
						stderrBuf: nil,
						process:   process,
					}
					adopted++
					log.Info("Re-adopted tunnel: %s on localhost:%d (PID %d)", pt.ID, pt.LocalPort, pt.PID)
					continue
				}
			} else {
				// Process not running, mark as terminated
				tunnel.Status = model.TunnelStatusTerminated
			}
		} else {
			// Keep the saved status for terminated/error tunnels
			tunnel.Status = pt.Status
		}

		// Load terminated/error tunnels for restart capability
		m.tunnels[pt.ID] = &activeTunnel{
			Tunnel:    tunnel,
			cmd:       nil,
			cancel:    func() {},
			stderrBuf: nil,
			process:   nil,
		}
		loaded++
	}

	if adopted > 0 {
		log.Info("Re-adopted %d active tunnels from previous session", adopted)
	}
	if loaded > 0 {
		log.Debug("Loaded %d terminated tunnels from previous session", loaded)
	}

	return nil
}

// isProcessRunning checks if a process with the given PID is still running.
func isProcessRunning(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// On Unix, FindProcess always succeeds. We need to send signal 0 to check.
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

// clearPersistence removes the persistence file.
func clearPersistence() error {
	file, err := persistenceFile()
	if err != nil {
		return err
	}
	return os.Remove(file)
}
