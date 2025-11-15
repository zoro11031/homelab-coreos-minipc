package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/cli"
)

var (
	resetForce  bool
	resetConfig bool
)

var resetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Reset setup progress",
	Long: `Clear all completion markers to allow re-running setup steps.

By default, this command will clear all completion markers but will NOT delete
your configuration file.

Use --config to also delete the configuration file and start completely fresh.`,
	RunE: resetSetup,
}

func init() {
	resetCmd.Flags().BoolVarP(&resetForce, "force", "f", false, "Skip confirmation prompt")
	resetCmd.Flags().BoolVarP(&resetConfig, "config", "c", false, "Also delete configuration file")
	rootCmd.AddCommand(resetCmd)
}

func resetSetup(cmd *cobra.Command, args []string) error {
	ctx, err := cli.NewSetupContext()
	if err != nil {
		return fmt.Errorf("failed to initialize setup context: %w", err)
	}

	// Confirmation prompt
	if !resetForce {
		ctx.UI.Header("Reset Setup State")
		ctx.UI.Warning("This will clear all completion markers")
		if resetConfig {
			ctx.UI.Warning("Configuration file will also be DELETED")
			ctx.UI.Warningf("  %s", ctx.Config.FilePath())
		} else {
			ctx.UI.Info("Configuration file will NOT be deleted")
			ctx.UI.Info("Use --config flag to also delete configuration")
		}
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

	// Remove all markers
	ctx.UI.Info("Removing completion markers...")
	if err := ctx.Markers.RemoveAll(); err != nil {
		return fmt.Errorf("failed to remove markers: %w", err)
	}
	ctx.UI.Success("✓ Completion markers cleared")

	// Remove configuration if requested
	if resetConfig {
		ctx.UI.Info("Removing configuration file...")
		configPath := ctx.Config.FilePath()
		if err := os.Remove(configPath); err != nil {
			if !os.IsNotExist(err) {
				return fmt.Errorf("failed to remove config file: %w", err)
			}
			ctx.UI.Info("  (Config file did not exist)")
		} else {
			ctx.UI.Successf("✓ Configuration file deleted: %s", configPath)
		}
	}

	fmt.Println()
	ctx.UI.Separator()
	ctx.UI.Success("Reset complete!")
	ctx.UI.Info("You can now run the setup steps again from scratch")

	return nil
}
