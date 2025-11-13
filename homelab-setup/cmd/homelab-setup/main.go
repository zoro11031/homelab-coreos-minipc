package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/cli"
	"github.com/zoro11031/homelab-coreos-minipc/homelab-setup/pkg/version"
)

var rootCmd = &cobra.Command{
	Use:   "homelab-setup",
	Short: "UBlue uCore Homelab Setup Tool",
	Long: `A comprehensive setup tool for configuring homelab services on UBlue uCore.

This tool provides an interactive menu and command-line interface for:
- System validation and pre-flight checks
- User and group configuration
- Directory structure creation
- WireGuard VPN setup (optional)
- NFS mount configuration
- Container service deployment
- Troubleshooting and diagnostics

Run without arguments to launch the interactive menu.`,
	SilenceUsage:  true, // We handle errors manually, but silence usage on error
	SilenceErrors: true, // We format errors ourselves for consistent output
	RunE:          runInteractiveMenu,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(version.Info())
	},
}

var menuCmd = &cobra.Command{
	Use:   "menu",
	Short: "Launch interactive menu",
	Long:  `Launch the interactive menu interface for setup.`,
	RunE:  runInteractiveMenu,
}

func init() {
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(menuCmd)
}

func runInteractiveMenu(cmd *cobra.Command, args []string) error {
	ctx, err := cli.NewSetupContext()
	if err != nil {
		return fmt.Errorf("failed to initialize setup context: %w", err)
	}

	menu := cli.NewMenu(ctx)
	return menu.Show()
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
