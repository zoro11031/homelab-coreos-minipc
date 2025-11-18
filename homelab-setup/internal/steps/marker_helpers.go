package steps

import "github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/config"

// ensureCanonicalMarker checks for the canonical completion marker and migrates any legacy markers
// to the canonical name to maintain backward compatibility.
//
// Race Safety: This function is safe to call concurrently from multiple processes:
//   - Uses MarkCompleteIfNotExists() with os.O_EXCL for atomic marker creation
//   - Multiple processes can safely check/migrate markers simultaneously
//   - Only the process that successfully creates the canonical marker cleans up legacy markers
//   - If another process creates the canonical marker first, this returns (true, nil) without error
//   - Legacy marker cleanup is best-effort and failures are silently ignored
//
// This ensures that even if two setup processes run concurrently, marker migration
// happens exactly once without conflicts or duplicate work.
func ensureCanonicalMarker(cfg *config.Config, canonical string, legacy ...string) (bool, error) {
	// First check if canonical marker exists (fast path for completed steps)
	if cfg.IsComplete(canonical) {
		return true, nil
	}

	// Check for legacy markers and migrate them
	for _, legacyName := range legacy {
		if legacyName == "" || legacyName == canonical {
			continue
		}

		if !cfg.IsComplete(legacyName) {
			continue
		}

		// Atomically create canonical marker (race-safe)
		// If another process already created it between our check and now, that's fine
		wasCreated, err := cfg.MarkCompleteIfNotExists(canonical)
		if err != nil {
			return false, err
		}

		// Best-effort cleanup of the legacy marker. Ignore errors since it's non-critical.
		// Only remove if we were the ones who created the canonical marker
		if wasCreated {
			_ = cfg.ClearMarker(legacyName)
		}
		return true, nil
	}

	return false, nil
}
