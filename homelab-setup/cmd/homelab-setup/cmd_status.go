package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/cli"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show setup status",
	Long:  `Display the current status of all setup steps.`,
	RunE:  showStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func showStatus(cmd *cobra.Command, args []string) error {
	ctx, err := cli.NewSetupContext()
	if err != nil {
		return fmt.Errorf("failed to initialize setup context: %w", err)
	}

	cyan := color.New(color.FgCyan, color.Bold)

	cyan.Println(strings.Repeat("=", 70))
	cyan.Println("  Setup Status")
	cyan.Println(strings.Repeat("=", 70))
	fmt.Println()

	ctx.UI.Info("Completed Steps:")
	fmt.Println()

	steps := ctx.Steps.GetAllSteps()
	completedCount := 0

	for i, step := range steps {
		if ctx.Steps.IsStepComplete(step.MarkerName) {
			ctx.UI.Successf("[%d] ✓ %s", i, step.Name)
			completedCount++
		} else {
			ctx.UI.Infof("[%d] - %s (not completed)", i, step.Name)
		}
	}

	fmt.Println()
	cyan.Println(strings.Repeat("-", 70))
	ctx.UI.Infof("Progress: %d/%d steps completed", completedCount, len(steps))
	cyan.Println(strings.Repeat("-", 70))
	fmt.Println()

	// Show configuration file location
	if _, err := os.Stat(ctx.Config.FilePath()); err == nil {
		ctx.UI.Infof("Configuration file: %s", ctx.Config.FilePath())
	}

	// Show marker directory
	if _, err := os.Stat(ctx.Markers.Dir()); err == nil {
		ctx.UI.Infof("Marker directory: %s", ctx.Markers.Dir())
	}

	fmt.Println()

	return nil
}
