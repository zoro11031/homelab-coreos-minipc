package steps

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/config"
)

// TestFstabReplacement tests the fstab entry replacement logic
func TestFstabReplacement(t *testing.T) {
	cfg := config.New(filepath.Join(t.TempDir(), "config"))
	options := getNFSMountOptions(cfg)

	tests := []struct {
		name          string
		existingFstab string
		newEntry      string
		mountPoint    string
		wantCommented bool
		wantNewEntry  bool
		wantLineCount int
	}{
		{
			name: "replace existing entry",
			existingFstab: `# /etc/fstab
/dev/sda1 / ext4 defaults 0 1
192.168.1.10:/volume1/media /mnt/nas nfs defaults 0 0
`,
			newEntry:      "192.168.1.10:/volume1/media /mnt/nas nfs " + options + " 0 0",
			mountPoint:    "/mnt/nas",
			wantCommented: true,
			wantNewEntry:  true,
			wantLineCount: 5, // header + sda1 + commented old + comment + new entry
		},
		{
			name: "no conflict",
			existingFstab: `# /etc/fstab
/dev/sda1 / ext4 defaults 0 1
`,
			newEntry:      "192.168.1.10:/volume1/media /mnt/nas nfs " + options + " 0 0",
			mountPoint:    "/mnt/nas",
			wantCommented: false,
			wantNewEntry:  true,
			wantLineCount: 4, // header + sda1 + comment + new entry
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the logic from createFstabEntry
			fstabLines := strings.Split(tt.existingFstab, "\n")
			var updatedLines []string

			for _, line := range fstabLines {
				trimmed := strings.TrimSpace(line)

				// Check for exact duplicate
				if trimmed == tt.newEntry {
					// Would skip in real implementation
					return
				}

				// Check if mount point is already used (even with different options)
				if strings.Contains(trimmed, " "+tt.mountPoint+" ") && !strings.HasPrefix(trimmed, "#") {
					// Comment out the old entry
					updatedLines = append(updatedLines, "# "+trimmed+" # Replaced by homelab-setup")
					continue
				}

				// Keep all other lines as-is
				updatedLines = append(updatedLines, line)
			}

			// Build new content from updated lines
			newContent := strings.Join(updatedLines, "\n")
			if !strings.HasSuffix(newContent, "\n") {
				newContent += "\n"
			}

			// Append new entry
			newContent += "# NFS mount added by homelab-setup\n"
			newContent += tt.newEntry + "\n"

			// Verify results
			lines := strings.Split(newContent, "\n")
			actualLineCount := 0
			hasCommented := false
			hasNewEntry := false

			for _, line := range lines {
				if strings.TrimSpace(line) != "" {
					actualLineCount++
				}
				if strings.Contains(line, "# Replaced by homelab-setup") {
					hasCommented = true
				}
				if strings.TrimSpace(line) == tt.newEntry {
					hasNewEntry = true
				}
			}

			if hasCommented != tt.wantCommented {
				t.Errorf("commented old entry = %v, want %v", hasCommented, tt.wantCommented)
			}
			if hasNewEntry != tt.wantNewEntry {
				t.Errorf("has new entry = %v, want %v", hasNewEntry, tt.wantNewEntry)
			}
			if actualLineCount != tt.wantLineCount {
				t.Errorf("line count = %d, want %d\nContent:\n%s", actualLineCount, tt.wantLineCount, newContent)
			}

			// Most importantly: verify no duplicate active entries for the mount point
			activeEntries := 0
			for _, line := range lines {
				trimmed := strings.TrimSpace(line)
				if strings.Contains(trimmed, " "+tt.mountPoint+" ") && !strings.HasPrefix(trimmed, "#") {
					activeEntries++
				}
			}
			if activeEntries != 1 {
				t.Errorf("active entries for %s = %d, want 1 (no duplicates)", tt.mountPoint, activeEntries)
			}
		})
	}
}
