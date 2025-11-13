package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/cli"
)

var (
	resetForce bool
)

var resetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Reset setup progress",
	Long: `Clear all completion markers to allow re-running setup steps.

This command will clear all completion markers but will NOT delete
your configuration file.`,
	RunE: resetSetup,
}

func init() {
	resetCmd.Flags().BoolVarP(&resetForce, "force", "f", false, "Skip confirmation prompt")
	rootCmd.AddCommand(resetCmd)
}

func resetSetup(cmd *cobra.Command, args []string) error {
	ctx, err := cli.NewSetupContext()
	if err != nil {
		return fmt.Errorf("failed to initialize setup context: %w", err)
	}

	if !resetForce {
		ctx.UI.Warning("This will clear all completion markers")
		ctx.UI.Warning("Configuration file will NOT be deleted")
		fmt.Println()

		confirm, err := ctx.UI.PromptYesNo("Are you sure you want to reset?", false)
		if err != nil {
			return err
		}

		if !confirm {
			ctx.UI.Info("Reset cancelled")
			return nil
		}
	}

	if err := ctx.Markers.RemoveAll(); err != nil {
		return fmt.Errorf("failed to remove markers: %w", err)
	}

	ctx.UI.Success("All completion markers have been cleared")
	ctx.UI.Info("You can now run the setup steps again")

	return nil
}
