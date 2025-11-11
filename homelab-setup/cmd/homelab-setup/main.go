package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
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
- Troubleshooting and diagnostics`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(version.Info())
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
