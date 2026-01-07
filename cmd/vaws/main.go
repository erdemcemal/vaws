// vaws is a terminal UI for browsing AWS CloudFormation stacks and ECS services.
package main

import (
	"flag"
	"fmt"
	"os"

	"vaws/internal/app"
)

func main() {
	// Define flags
	profile := flag.String("profile", "", "AWS profile to use (default: use default credentials)")
	region := flag.String("region", "", "AWS region (default: use profile/environment default)")
	debug := flag.Bool("debug", false, "Enable debug logging")
	version := flag.Bool("version", false, "Print version information")
	listProfiles := flag.Bool("list-profiles", false, "List available AWS profiles")
	testConn := flag.Bool("test", false, "Test AWS connection without starting TUI")
	noAltScreen := flag.Bool("no-alt-screen", false, "Disable alternate screen (allows text selection/copy)")
	themeFlag := flag.String("theme", "auto", "Color theme: auto, dark, or light")

	// Custom usage
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "vaws - AWS CloudFormation & ECS Explorer\n\n")
		fmt.Fprintf(os.Stderr, "Usage: vaws [options]\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nNavigation:\n")
		fmt.Fprintf(os.Stderr, "  ↑/k, ↓/j    Navigate list\n")
		fmt.Fprintf(os.Stderr, "  Enter       Select item / drill down\n")
		fmt.Fprintf(os.Stderr, "  Esc         Go back\n")
		fmt.Fprintf(os.Stderr, "  /           Filter list\n")
		fmt.Fprintf(os.Stderr, "  r           Refresh\n")
		fmt.Fprintf(os.Stderr, "  q           Quit\n")
	}

	flag.Parse()

	// Handle special flags
	if *version {
		app.PrintVersion()
		return
	}

	if *listProfiles {
		if err := app.PrintProfiles(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Build config
	cfg := app.Config{
		Profile:     *profile,
		Region:      *region,
		Debug:       *debug,
		NoAltScreen: *noAltScreen,
		Theme:       *themeFlag,
	}

	// Test connection mode
	if *testConn {
		if err := app.TestConnection(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Run the application
	app.MustRun(cfg)
}
