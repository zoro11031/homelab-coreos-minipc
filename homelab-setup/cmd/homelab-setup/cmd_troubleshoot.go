package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/cli"
)

var troubleshootCmd = &cobra.Command{
	Use:   "troubleshoot",
	Short: "Run troubleshooting diagnostics",
	Long:  `Run diagnostic checks to troubleshoot common issues.`,
	RunE:  runTroubleshoot,
}

func init() {
	rootCmd.AddCommand(troubleshootCmd)
}

func runTroubleshoot(cmd *cobra.Command, args []string) error {
	ctx, err := cli.NewSetupContext()
	if err != nil {
		return fmt.Errorf("failed to initialize setup context: %w", err)
	}

	ctx.UI.Header("Troubleshooting Tool")

	ctx.UI.Warning("Troubleshooting tool not yet fully implemented in Go version")
	ctx.UI.Info("For now, you can use: /usr/share/home-lab-setup-scripts/scripts/troubleshoot.sh")
	fmt.Println()

	// Basic diagnostics we can do
	ctx.UI.Info("Running basic diagnostics...")
	fmt.Println()

	// Check configuration
	ctx.UI.Step("Configuration Check")
	if _, err := ctx.Config.Get("SETUP_USER"); err != nil {
		ctx.UI.Warning("SETUP_USER not configured")
	} else {
		ctx.UI.Success("Configuration file exists and is readable")
	}

	// Check markers
	ctx.UI.Step("Completion Status")
	markers, err := ctx.Markers.List()
	if err != nil {
		ctx.UI.Error(fmt.Sprintf("Failed to list markers: %v", err))
	} else {
		if len(markers) == 0 {
			ctx.UI.Info("No steps completed yet")
		} else {
			ctx.UI.Infof("Completed steps: %d", len(markers))
			for _, marker := range markers {
				ctx.UI.Infof("  - %s", marker)
			}
		}
	}

	return nil
}
