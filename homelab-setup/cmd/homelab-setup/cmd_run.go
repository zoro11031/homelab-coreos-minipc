package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/cli"
)

var (
	// Flags for non-interactive mode
	nonInteractive bool
	setupUser      string
	nfsServer      string
	homelabBaseDir string
	skipWireguard  bool
)

var runCmd = &cobra.Command{
	Use:   "run [step|all|quick]",
	Short: "Run setup steps",
	Long: `Run one or more setup steps.

Steps:
  all         - Run all setup steps
  quick       - Run all steps except WireGuard
  preflight   - Pre-flight system checks
  user        - User and group configuration
  directory   - Directory structure creation
  wireguard   - WireGuard VPN setup
  nfs         - NFS mount configuration
  container   - Container service setup
  deployment  - Service deployment`,
	Args: cobra.ExactArgs(1),
	RunE: runSetup,
}

func init() {
	runCmd.Flags().BoolVar(&nonInteractive, "non-interactive", false, "Run in non-interactive mode")
	runCmd.Flags().StringVar(&setupUser, "setup-user", "", "Username for homelab setup")
	runCmd.Flags().StringVar(&nfsServer, "nfs-server", "", "NFS server address")
	runCmd.Flags().StringVar(&homelabBaseDir, "homelab-base-dir", "", "Base directory for homelab")
	runCmd.Flags().BoolVar(&skipWireguard, "skip-wireguard", false, "Skip WireGuard setup")

	rootCmd.AddCommand(runCmd)
}

func runSetup(cmd *cobra.Command, args []string) error {
	// Create setup context with non-interactive mode if requested
	ctx, err := cli.NewSetupContextWithOptions(nonInteractive)
	if err != nil {
		return fmt.Errorf("failed to initialize setup context: %w", err)
	}

	// Apply non-interactive config if provided
	if nonInteractive {
		if err := applyNonInteractiveConfig(ctx); err != nil {
			return err
		}
		ctx.UI.Info("Running in non-interactive mode")
	}

	step := args[0]

	switch step {
	case "all":
		return ctx.Steps.RunAll(skipWireguard)
	case "quick":
		return ctx.Steps.RunAll(true)
	case "preflight", "user", "directory", "wireguard", "nfs", "container", "deployment":
		return ctx.Steps.RunStep(step)
	default:
		return fmt.Errorf("unknown step: %s", step)
	}
}

func applyNonInteractiveConfig(ctx *cli.SetupContext) error {
	if setupUser != "" {
		if err := ctx.Config.Set("SETUP_USER", setupUser); err != nil {
			return fmt.Errorf("failed to set SETUP_USER: %w", err)
		}
	}

	if nfsServer != "" {
		if err := ctx.Config.Set("NFS_SERVER", nfsServer); err != nil {
			return fmt.Errorf("failed to set NFS_SERVER: %w", err)
		}
	}

	if homelabBaseDir != "" {
		if err := ctx.Config.Set("HOMELAB_BASE_DIR", homelabBaseDir); err != nil {
			return fmt.Errorf("failed to set HOMELAB_BASE_DIR: %w", err)
		}
	}

	return nil
}
