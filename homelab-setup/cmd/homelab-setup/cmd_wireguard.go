package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/cli"
	"github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/steps"
)

var (
	addPeerInterface      string
	addPeerName           string
	addPeerEndpoint       string
	addPeerDNS            string
	addPeerAllowedIPs     string
	addPeerRouteAll       bool
	addPeerOutputDir      string
	addPeerKeepalive      int
	addPeerNoPSK          bool
	addPeerPresharedKey   string
	addPeerNonInteractive bool
	addPeerSkipQR         bool
)

var wireguardCmd = &cobra.Command{
	Use:   "wireguard",
	Short: "WireGuard helper commands",
}

var wireguardAddPeerCmd = &cobra.Command{
	Use:   "add-peer",
	Short: "Add a WireGuard peer and export its config",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, err := cli.NewSetupContextWithOptions(addPeerNonInteractive)
		if err != nil {
			return fmt.Errorf("failed to initialize setup context: %w", err)
		}

		opts := &steps.WireGuardPeerWorkflowOptions{
			InterfaceName:              addPeerInterface,
			PeerName:                   addPeerName,
			Endpoint:                   addPeerEndpoint,
			DNS:                        addPeerDNS,
			ClientAllowedIPs:           addPeerAllowedIPs,
			OutputDir:                  addPeerOutputDir,
			PersistentKeepaliveSeconds: addPeerKeepalive,
			ProvidedPresharedKey:       addPeerPresharedKey,
			NonInteractive:             addPeerNonInteractive,
			SkipQRCode:                 addPeerSkipQR,
		}

		if addPeerOutputDir == "" {
			opts.OutputDir = ""
		}

		if flag := cmd.Flags().Lookup("route-all"); flag != nil && flag.Changed {
			opts.RouteAll = &addPeerRouteAll
		}

		if addPeerNoPSK {
			generate := false
			opts.GeneratePresharedKey = &generate
		} else if flag := cmd.Flags().Lookup("no-psk"); flag != nil && !flag.Changed {
			generate := true
			opts.GeneratePresharedKey = &generate
		}

		return ctx.Steps.AddWireGuardPeer(opts)
	},
}

func init() {
	wireguardAddPeerCmd.Flags().StringVar(&addPeerInterface, "interface", "", "WireGuard interface to update (default: stored value)")
	wireguardAddPeerCmd.Flags().StringVar(&addPeerName, "name", "", "Peer name")
	wireguardAddPeerCmd.Flags().StringVar(&addPeerEndpoint, "endpoint", "", "Server endpoint host:port")
	wireguardAddPeerCmd.Flags().StringVar(&addPeerDNS, "dns", "", "Client DNS resolver")
	wireguardAddPeerCmd.Flags().StringVar(&addPeerAllowedIPs, "client-allowed-ips", "", "Override client AllowedIPs")
	wireguardAddPeerCmd.Flags().BoolVar(&addPeerRouteAll, "route-all", true, "Route all client traffic through the VPN")
	wireguardAddPeerCmd.Flags().StringVar(&addPeerOutputDir, "export-dir", "", "Directory for exported peer configs")
	wireguardAddPeerCmd.Flags().IntVar(&addPeerKeepalive, "keepalive", 25, "PersistentKeepalive in seconds")
	wireguardAddPeerCmd.Flags().BoolVar(&addPeerNoPSK, "no-psk", false, "Skip preshared key generation")
	wireguardAddPeerCmd.Flags().StringVar(&addPeerPresharedKey, "preshared-key", "", "Provide an explicit preshared key")
	wireguardAddPeerCmd.Flags().BoolVar(&addPeerNonInteractive, "non-interactive", false, "Run without prompts (requires values)")
wireguardAddPeerCmd.Flags().BoolVar(&addPeerSkipQR, "no-qr", false, "Skip QR output (for CI/testing)")
if err := wireguardAddPeerCmd.Flags().MarkHidden("no-qr"); err != nil {
	fmt.Printf("Warning: could not hide 'no-qr' flag: %v\n", err)
}

wireguardCmd.AddCommand(wireguardAddPeerCmd)
rootCmd.AddCommand(wireguardCmd)
}
