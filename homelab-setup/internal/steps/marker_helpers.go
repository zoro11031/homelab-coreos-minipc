package steps

import "github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/config"

// ensureCanonicalMarker checks for the canonical completion marker and migrates any legacy markers
// to the canonical name to maintain backward compatibility.
func ensureCanonicalMarker(markers *config.Markers, canonical string, legacy ...string) (bool, error) {
	exists, err := markers.Exists(canonical)
	if err != nil {
		return false, err
	}
	if exists {
		return true, nil
	}

	for _, legacyName := range legacy {
		if legacyName == "" || legacyName == canonical {
			continue
		}

		legacyExists, err := markers.Exists(legacyName)
		if err != nil {
			return false, err
		}
		if !legacyExists {
			continue
		}

		if err := markers.Create(canonical); err != nil {
			return false, err
		}
		// Best-effort cleanup of the legacy marker. Ignore errors since it's non-critical.
		_ = markers.Remove(legacyName)
		return true, nil
	}

	return false, nil
}
