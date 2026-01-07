// Package config manages application configuration from ~/.vaws/config.yaml
package config

import (
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	// Profiles contains profile-specific settings
	Profiles map[string]ProfileConfig `yaml:"profiles"`

	// Defaults contains default settings applied to all profiles
	Defaults DefaultConfig `yaml:"defaults"`
}

// ProfileConfig contains settings for a specific AWS profile
type ProfileConfig struct {
	// JumpHost is the EC2 instance name or ID to use for private API Gateway access
	JumpHost string `yaml:"jump_host,omitempty"`

	// JumpHostTag is a tag filter to find jump host (e.g., "Name=bastion")
	JumpHostTag string `yaml:"jump_host_tag,omitempty"`

	// Region overrides the default region for this profile
	Region string `yaml:"region,omitempty"`

	// VPCEndpointID is the VPC endpoint ID for cross-account private API Gateway access
	// When set, uses URL format: https://<api-id>-<vpce-id>.execute-api.<region>.amazonaws.com
	VPCEndpointID string `yaml:"vpc_endpoint_id,omitempty"`
}

// DefaultConfig contains default settings
type DefaultConfig struct {
	// JumpHostTags are tags to search for when auto-discovering jump hosts
	// Priority order: first match wins
	JumpHostTags []string `yaml:"jump_host_tags,omitempty"`

	// JumpHostNames are instance names to search for when auto-discovering
	// Priority order: first match wins
	JumpHostNames []string `yaml:"jump_host_names,omitempty"`
}

var (
	globalConfig *Config
	configOnce   sync.Once
	configPath   string
)

// DefaultConfigPath returns the default config file path
func DefaultConfigPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(homeDir, ".vaws", "config.yaml")
}

// Load loads the configuration from the default path
func Load() (*Config, error) {
	return LoadFrom(DefaultConfigPath())
}

// LoadFrom loads the configuration from a specific path
func LoadFrom(path string) (*Config, error) {
	configPath = path

	// Start with defaults
	cfg := &Config{
		Profiles: make(map[string]ProfileConfig),
		Defaults: DefaultConfig{
			JumpHostTags: []string{
				"vaws:jump-host=true",
				"Name=bastion",
				"Name=jump-host",
			},
			JumpHostNames: []string{
				"bastion",
				"jump-host",
				"jumphost",
			},
		},
	}

	// Try to read the config file
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Config file doesn't exist, use defaults
			globalConfig = cfg
			return cfg, nil
		}
		return nil, err
	}

	// Parse YAML
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	globalConfig = cfg
	return cfg, nil
}

// Get returns the global config, loading it if necessary
func Get() *Config {
	configOnce.Do(func() {
		cfg, err := Load()
		if err != nil {
			// Use defaults on error
			cfg = &Config{
				Profiles: make(map[string]ProfileConfig),
				Defaults: DefaultConfig{
					JumpHostTags: []string{
						"vaws:jump-host=true",
						"Name=bastion",
						"Name=jump-host",
					},
					JumpHostNames: []string{
						"bastion",
						"jump-host",
						"jumphost",
					},
				},
			}
		}
		globalConfig = cfg
	})
	return globalConfig
}

// GetProfileConfig returns the configuration for a specific profile
func (c *Config) GetProfileConfig(profile string) ProfileConfig {
	if pc, ok := c.Profiles[profile]; ok {
		return pc
	}
	return ProfileConfig{}
}

// GetJumpHost returns the configured jump host for a profile
// Returns empty string if not configured
func (c *Config) GetJumpHost(profile string) string {
	if pc, ok := c.Profiles[profile]; ok {
		if pc.JumpHost != "" {
			return pc.JumpHost
		}
	}
	return ""
}

// GetJumpHostTag returns the configured jump host tag for a profile
func (c *Config) GetJumpHostTag(profile string) string {
	if pc, ok := c.Profiles[profile]; ok {
		if pc.JumpHostTag != "" {
			return pc.JumpHostTag
		}
	}
	return ""
}

// GetVPCEndpointID returns the configured VPC endpoint ID for cross-account private API Gateway access
func (c *Config) GetVPCEndpointID(profile string) string {
	if pc, ok := c.Profiles[profile]; ok {
		if pc.VPCEndpointID != "" {
			return pc.VPCEndpointID
		}
	}
	return ""
}

// Save saves the configuration to disk
func (c *Config) Save() error {
	return c.SaveTo(configPath)
}

// SaveTo saves the configuration to a specific path
func (c *Config) SaveTo(path string) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// SetJumpHost sets the jump host for a profile
func (c *Config) SetJumpHost(profile, jumpHost string) {
	if c.Profiles == nil {
		c.Profiles = make(map[string]ProfileConfig)
	}
	pc := c.Profiles[profile]
	pc.JumpHost = jumpHost
	c.Profiles[profile] = pc
}

// CreateDefaultConfig creates a default config file with example settings
func CreateDefaultConfig() error {
	cfg := &Config{
		Profiles: map[string]ProfileConfig{
			"example-profile": {
				JumpHost: "bastion",
				Region:   "us-east-1",
			},
		},
		Defaults: DefaultConfig{
			JumpHostTags: []string{
				"vaws:jump-host=true",
				"Name=bastion",
				"Name=jump-host",
			},
			JumpHostNames: []string{
				"bastion",
				"jump-host",
				"jumphost",
			},
		},
	}
	return cfg.SaveTo(DefaultConfigPath())
}
