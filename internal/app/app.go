// Package app handles application lifecycle and dependency wiring.
package app

import (
	"context"
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"vaws/internal/aws"
	"vaws/internal/log"
	"vaws/internal/ui"
	"vaws/internal/ui/theme"
)

// Config holds application configuration.
type Config struct {
	Profile     string
	Region      string
	Debug       bool
	NoAltScreen bool     // Disable alternate screen for easier copy/paste
	Profiles    []string // Available AWS profiles (populated if no profile specified)
	Theme       string   // Theme override: "auto", "dark", or "light"
}

// Run starts the application with the given configuration.
func Run(cfg Config) error {
	// Initialize theme
	switch cfg.Theme {
	case "dark":
		theme.SetByName(theme.ThemeDark)
	case "light":
		theme.SetByName(theme.ThemeLight)
	default:
		// Auto-detect theme
		theme.SetByName(theme.ThemeAuto)
	}

	// If no profile specified, load available profiles for selection
	if cfg.Profile == "" {
		profiles, err := aws.ListProfiles()
		if err != nil {
			profiles = []string{"default"}
		}
		cfg.Profiles = profiles

		// Create TUI model without AWS client (will be created after profile selection)
		model := ui.NewWithProfileSelection(cfg.Profiles, cfg.Region, log.Default())

		// Create and run the program
		opts := []tea.ProgramOption{
			tea.WithMouseCellMotion(),
		}
		if !cfg.NoAltScreen {
			opts = append(opts, tea.WithAltScreen())
		}
		p := tea.NewProgram(model, opts...)

		if _, err := p.Run(); err != nil {
			return fmt.Errorf("failed to run TUI: %w", err)
		}
		return nil
	}

	// Create AWS client with specified profile
	ctx := context.Background()
	client, err := aws.NewClient(ctx, cfg.Profile, cfg.Region)
	if err != nil {
		return fmt.Errorf("failed to create AWS client: %w", err)
	}

	// Create TUI model
	model := ui.New(client, log.Default())

	// Create and run the program
	opts := []tea.ProgramOption{
		tea.WithMouseCellMotion(),
	}
	if !cfg.NoAltScreen {
		opts = append(opts, tea.WithAltScreen())
	}
	p := tea.NewProgram(model, opts...)

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("failed to run TUI: %w", err)
	}

	return nil
}

// ListProfiles returns all available AWS profiles.
func ListProfiles() ([]string, error) {
	return aws.ListProfiles()
}

// PrintProfiles prints all available AWS profiles to stdout.
func PrintProfiles() error {
	profiles, err := ListProfiles()
	if err != nil {
		return err
	}

	fmt.Println("Available AWS profiles:")
	for _, p := range profiles {
		fmt.Printf("  - %s\n", p)
	}
	return nil
}

// Version information (set by build flags).
var (
	Version   = "dev"
	Commit    = "unknown"
	BuildDate = "unknown"
)

// PrintVersion prints version information.
func PrintVersion() {
	fmt.Printf("vaws %s\n", Version)
	fmt.Printf("  Commit:     %s\n", Commit)
	fmt.Printf("  Build date: %s\n", BuildDate)
}

// MustRun runs the application and exits on error.
func MustRun(cfg Config) {
	if err := Run(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// TestConnection tests AWS connectivity by attempting to list stacks.
func TestConnection(cfg Config) error {
	fmt.Printf("Testing AWS connection...\n")
	fmt.Printf("  Profile: %s\n", cfg.Profile)
	fmt.Printf("  Region:  %s\n", cfg.Region)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	client, err := aws.NewClient(ctx, cfg.Profile, cfg.Region)
	if err != nil {
		return fmt.Errorf("failed to create AWS client: %w", err)
	}

	fmt.Printf("  Resolved region: %s\n", client.Region())
	fmt.Printf("\nListing CloudFormation stacks...\n")

	stacks, err := client.ListStacks(ctx)
	if err != nil {
		return fmt.Errorf("failed to list stacks: %w", err)
	}

	fmt.Printf("Success! Found %d stacks.\n", len(stacks))
	if len(stacks) > 0 {
		fmt.Printf("\nFirst 5 stacks:\n")
		for i, s := range stacks {
			if i >= 5 {
				break
			}
			fmt.Printf("  - %s (%s)\n", s.Name, s.Status)
		}
	}

	return nil
}
