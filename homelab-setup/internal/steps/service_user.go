package steps

import (
	"fmt"
	"strings"

	"github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/config"
)

// getServiceUser returns the configured user account that should own compose files and run services.
// It prefers the HOMELAB_USER key and falls back to legacy SETUP_USER for compatibility.
func getServiceUser(cfg *config.Config) (string, error) {
	serviceUser := strings.TrimSpace(cfg.GetOrDefault(config.KeyHomelabUser, ""))
	if serviceUser == "" {
		serviceUser = strings.TrimSpace(cfg.GetOrDefault("SETUP_USER", ""))
	}

	if serviceUser == "" {
		return "", fmt.Errorf("service user not configured; run user setup to select a homelab account")
	}

	return serviceUser, nil
}
