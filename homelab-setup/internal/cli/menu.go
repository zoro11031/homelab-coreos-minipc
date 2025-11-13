package cli

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
)

// ErrExit is returned when the user chooses to exit the menu
var ErrExit = errors.New("exit")

// Menu provides an interactive menu interface
type Menu struct {
	ctx *SetupContext
}

// NewMenu creates a new Menu instance
func NewMenu(ctx *SetupContext) *Menu {
	return &Menu{ctx: ctx}
}

// clearScreen clears the terminal screen using ANSI escape codes
// This is more portable than calling the 'clear' command
func clearScreen() {
	// ANSI escape codes: \033[2J clears screen, \033[H moves cursor to home
	fmt.Print("\033[2J\033[H")
}

// Show displays the main menu and handles user input
func (m *Menu) Show() error {
	for {
		clearScreen()
		m.displayMenu()

		choice, err := m.ctx.UI.PromptInput("Enter your choice", "")
		if err != nil {
			return err
		}

		choice = strings.ToUpper(strings.TrimSpace(choice))

		if err := m.handleChoice(choice); err != nil {
			if errors.Is(err, ErrExit) {
				return nil
			}
			m.ctx.UI.Error(fmt.Sprintf("%v", err))
			m.ctx.UI.Print("")
			m.ctx.UI.Info("Press Enter to continue...")
			fmt.Scanln()
		}
	}
}

// displayMenu displays the main menu
func (m *Menu) displayMenu() {
	bold := color.New(color.Bold)
	cyan := color.New(color.FgCyan, color.Bold)
	green := color.New(color.FgGreen)

	// Header
	border := strings.Repeat("=", 70)
	cyan.Println(border)
	cyan.Println("  UBlue uCore Homelab Setup")
	cyan.Println(border)
	fmt.Println()

	m.ctx.UI.Info("Welcome to the homelab setup wizard!")
	fmt.Println()
	m.ctx.UI.Info("This tool will guide you through setting up your homelab environment")
	m.ctx.UI.Info("on UBlue uCore (immutable Fedora with rpm-ostree).")
	fmt.Println()

	// Setup Options
	cyan.Println(strings.Repeat("-", 70))
	m.ctx.UI.Info("Setup Options:")
	cyan.Println(strings.Repeat("-", 70))
	fmt.Println()

	bold.Print("  [A] ")
	fmt.Println("Run All Steps (Complete Setup)")

	bold.Print("  [Q] ")
	fmt.Println("Quick Setup (Skip WireGuard)")
	fmt.Println()

	// Individual Steps
	cyan.Println(strings.Repeat("-", 70))
	m.ctx.UI.Info("Individual Steps:")
	cyan.Println(strings.Repeat("-", 70))
	fmt.Println()

	steps := m.ctx.Steps.GetAllSteps()
	for i, step := range steps {
		// Check completion status
		status := "  "
		if m.ctx.Steps.IsStepComplete(step.MarkerName) {
			status = green.Sprint("✓")
		}

		bold.Printf("  [%d] ", i)
		fmt.Printf("%s %s\n", status, step.Name)
		fmt.Printf("      → %s\n", step.Description)
		fmt.Println()
	}

	// Other Options
	cyan.Println(strings.Repeat("-", 70))
	m.ctx.UI.Info("Other Options:")
	cyan.Println(strings.Repeat("-", 70))
	fmt.Println()

	bold.Print("  [T] ")
	fmt.Println("Troubleshooting Tool")

	bold.Print("  [S] ")
	fmt.Println("Show Setup Status")

	bold.Print("  [R] ")
	fmt.Println("Reset Setup (Clear markers)")

	bold.Print("  [H] ")
	fmt.Println("Help")

	bold.Print("  [X] ")
	fmt.Println("Exit")
	fmt.Println()
}

// handleChoice processes the user's menu choice
func (m *Menu) handleChoice(choice string) error {
	switch choice {
	case "A":
		return m.runAllSteps(false)
	case "Q":
		return m.runAllSteps(true)
	case "0", "1", "2", "3", "4", "5", "6":
		return m.runIndividualStep(choice)
	case "T":
		return m.runTroubleshoot()
	case "S":
		return m.showStatus()
	case "R":
		return m.resetSetup()
	case "H":
		return m.showHelp()
	case "X":
		return ErrExit
	default:
		return fmt.Errorf("invalid choice: %s", choice)
	}
}

// runAllSteps runs all setup steps
func (m *Menu) runAllSteps(skipWireGuard bool) error {
	clearScreen()
	m.ctx.UI.Header("Running Complete Setup")

	if skipWireGuard {
		m.ctx.UI.Info("WireGuard will be skipped")
	}

	err := m.ctx.Steps.RunAll(skipWireGuard)

	fmt.Println()
	m.ctx.UI.Info("Press Enter to return to menu...")
	fmt.Scanln()

	return err
}

// runIndividualStep runs a single setup step
func (m *Menu) runIndividualStep(choice string) error {
	steps := m.ctx.Steps.GetAllSteps()
	stepIndex := int(choice[0] - '0')

	if stepIndex < 0 || stepIndex >= len(steps) {
		return fmt.Errorf("invalid step number: %s", choice)
	}

	step := steps[stepIndex]

	clearScreen()
	m.ctx.UI.Header(fmt.Sprintf("Step %d: %s", stepIndex, step.Name))

	err := m.ctx.Steps.RunStep(step.ShortName)

	fmt.Println()
	m.ctx.UI.Info("Press Enter to return to menu...")
	fmt.Scanln()

	return err
}

// runTroubleshoot runs the troubleshooting tool
func (m *Menu) runTroubleshoot() error {
	clearScreen()
	m.ctx.UI.Header("Troubleshooting Tool")

	m.ctx.UI.Warning("Troubleshooting tool not yet implemented in Go version")
	m.ctx.UI.Info("For now, please use: /usr/share/home-lab-setup-scripts/scripts/troubleshoot.sh")

	fmt.Println()
	m.ctx.UI.Info("Press Enter to return to menu...")
	fmt.Scanln()

	return nil
}

// showStatus shows the current setup status
func (m *Menu) showStatus() error {
	clearScreen()
	m.ctx.UI.Header("Setup Status")

	fmt.Println()
	m.ctx.UI.Info("Completed Steps:")
	fmt.Println()

	steps := m.ctx.Steps.GetAllSteps()
	completedCount := 0

	for i, step := range steps {
		if m.ctx.Steps.IsStepComplete(step.MarkerName) {
			m.ctx.UI.Successf("[%d] ✓ %s", i, step.Name)
			completedCount++
		} else {
			m.ctx.UI.Infof("[%d] - %s (not completed)", i, step.Name)
		}
	}

	fmt.Println()
	cyan := color.New(color.FgCyan, color.Bold)
	cyan.Println(strings.Repeat("-", 70))
	m.ctx.UI.Infof("Progress: %d/%d steps completed", completedCount, len(steps))
	cyan.Println(strings.Repeat("-", 70))
	fmt.Println()

	// Show configuration file location
	if _, err := os.Stat(m.ctx.Config.FilePath()); err == nil {
		m.ctx.UI.Infof("Configuration file: %s", m.ctx.Config.FilePath())
	}

	// Show marker directory
	if _, err := os.Stat(m.ctx.Markers.Dir()); err == nil {
		m.ctx.UI.Infof("Marker directory: %s", m.ctx.Markers.Dir())
	}

	fmt.Println()
	m.ctx.UI.Info("Press Enter to return to menu...")
	fmt.Scanln()

	return nil
}

// resetSetup resets all completion markers
func (m *Menu) resetSetup() error {
	clearScreen()
	m.ctx.UI.Header("Reset Setup")

	m.ctx.UI.Warning("This will clear all completion markers")
	m.ctx.UI.Warning("Configuration file will NOT be deleted")
	fmt.Println()

	confirm, err := m.ctx.UI.PromptYesNo("Are you sure you want to reset?", false)
	if err != nil {
		return err
	}

	if !confirm {
		m.ctx.UI.Info("Reset cancelled")
		fmt.Println()
		m.ctx.UI.Info("Press Enter to return to menu...")
		fmt.Scanln()
		return nil
	}

	if err := m.ctx.Markers.RemoveAll(); err != nil {
		return fmt.Errorf("failed to remove markers: %w", err)
	}

	m.ctx.UI.Success("All completion markers have been cleared")
	m.ctx.UI.Info("You can now run the setup steps again")

	fmt.Println()
	m.ctx.UI.Info("Press Enter to return to menu...")
	fmt.Scanln()

	return nil
}

// showHelp displays help information
func (m *Menu) showHelp() error {
	clearScreen()
	m.ctx.UI.Header("Help")

	help := `
UBlue uCore Homelab Setup - Help

This setup tool configures your homelab environment on UBlue uCore, an
immutable Fedora-based operating system that uses rpm-ostree for updates.

SETUP WORKFLOW:

  The setup process consists of these steps:

  0. Pre-flight Check
     - Verifies system requirements
     - Checks for required packages
     - Detects pre-existing configurations

  1. User Setup
     - Configures user account for containers
     - Sets up groups and permissions
     - Detects UID/GID and timezone

  2. Directory Setup
     - Creates base homelab directory structure
     - Sets up configuration and data directories
     - Configures proper ownership and permissions

  3. WireGuard Setup (Optional)
     - Configures VPN server
     - Generates keys
     - Sets up peer connections

  4. NFS Setup
     - Configures network storage mounts
     - Sets up systemd mount units
     - Verifies connectivity

  5. Container Setup
     - Copies compose templates
     - Configures environment variables
     - Creates .env files

  6. Service Deployment
     - Enables systemd services
     - Pulls container images
     - Starts all services

RECOMMENDED APPROACH:

  For first-time setup, use option [A] "Run All Steps" or [Q] for
  "Quick Setup" (skips WireGuard).

  If a step fails, you can re-run just that step using the individual
  step options (0-6).

CONFIGURATION FILES:

  Configuration: ~/.homelab-setup.conf
  Markers: ~/.local/homelab-setup/

COMMAND-LINE MODE:

  You can also run this tool in non-interactive mode:

    homelab-setup run all              # Run all steps
    homelab-setup run quick            # Skip WireGuard
    homelab-setup run <step>           # Run specific step
    homelab-setup status               # Show status
    homelab-setup reset                # Reset markers
    homelab-setup troubleshoot         # Run troubleshooter

For more information, see the project README.
`

	fmt.Println(help)
	m.ctx.UI.Info("Press Enter to return to menu...")
	fmt.Scanln()

	return nil
}
