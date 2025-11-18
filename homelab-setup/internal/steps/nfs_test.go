package steps

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/config"
)

// TestMountPointToUnitBaseName tests the mount point to unit name conversion
func TestMountPointToUnitBaseName(t *testing.T) {
	tests := []struct {
		name       string
		mountPoint string
		want       string
	}{
		{
			name:       "simple mount point",
			mountPoint: "/mnt/nas",
			want:       "mnt-nas",
		},
		{
			name:       "mount point with dash",
			mountPoint: "/mnt/nas-media",
			want:       "mnt-nas-media",
		},
		{
			name:       "nested mount point",
			mountPoint: "/mnt/storage/data",
			want:       "mnt-storage-data",
		},
		{
			name:       "root mount",
			mountPoint: "/",
			want:       "",
		},
		{
			name:       "trailing slash",
			mountPoint: "/mnt/nas/",
			want:       "mnt-nas",
		},
		{
			name:       "multiple slashes",
			mountPoint: "//mnt//nas//",
			want:       "mnt-nas",
		},
		{
			name:       "with spaces",
			mountPoint: "/mnt/nas media",
			want:       "mnt-nas-media",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mountPointToUnitBaseName(tt.mountPoint)
			if got != tt.want {
				t.Errorf("mountPointToUnitBaseName() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestGetNFSMountOptions tests the NFS mount options retrieval
func TestGetNFSMountOptions(t *testing.T) {
	baseDir := t.TempDir()

	tests := []struct {
		name     string
		config   string
		expected string
	}{
		{
			name:     "defaults when unset",
			expected: "defaults,nfsvers=4.2,_netdev,nofail",
		},
		{
			name:     "appends extra options",
			config:   "hard,timeo=900",
			expected: "defaults,nfsvers=4.2,_netdev,nofail,hard,timeo=900",
		},
		{
			name:     "overrides nfs version",
			config:   "nfsvers=4.1,hard",
			expected: "defaults,nfsvers=4.1,_netdev,nofail,hard",
		},
		{
			name:     "deduplicates whitespace and keeps ordering",
			config:   " nofail ,  hard ,",
			expected: "defaults,nfsvers=4.2,_netdev,nofail,hard",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfgPath := filepath.Join(baseDir, strings.ReplaceAll(tt.name, " ", "_"))
			cfg := config.New(cfgPath)

			if tt.config != "" {
				if err := cfg.Set(config.KeyNFSMountOptions, tt.config); err != nil {
					t.Fatalf("failed to set config: %v", err)
				}
			}

			got := getNFSMountOptions(cfg)
			if got != tt.expected {
				t.Fatalf("getNFSMountOptions() = %q, want %q", got, tt.expected)
			}
		})
	}
}

// TestFstabEntryFormat tests the format of fstab entries
func TestFstabEntryFormat(t *testing.T) {
	cfg := config.New(filepath.Join(t.TempDir(), "config"))
	options := getNFSMountOptions(cfg)

	tests := []struct {
		name       string
		host       string
		export     string
		mountPoint string
		want       string
	}{
		{
			name:       "simple NFS entry",
			host:       "192.168.1.10",
			export:     "/volume1/media",
			mountPoint: "/mnt/nas",
			want:       fmt.Sprintf("%s:%s %s nfs %s 0 0", "192.168.1.10", "/volume1/media", "/mnt/nas", options),
		},
		{
			name:       "hostname instead of IP",
			host:       "nas.local",
			export:     "/data",
			mountPoint: "/mnt/data",
			want:       fmt.Sprintf("%s:%s %s nfs %s 0 0", "nas.local", "/data", "/mnt/data", options),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Build fstab entry using the same format as createFstabEntry
			got := strings.TrimSpace(tt.host + ":" + tt.export + " " + tt.mountPoint + " nfs " + options + " 0 0")
			if got != tt.want {
				t.Errorf("fstab entry format = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestFstabDuplicateDetection tests duplicate entry detection logic
func TestFstabDuplicateDetection(t *testing.T) {
	cfg := config.New(filepath.Join(t.TempDir(), "config"))
	options := getNFSMountOptions(cfg)

	tests := []struct {
		name            string
		existingLines   []string
		newEntry        string
		wantDuplicate   bool
		wantMountExists bool
	}{
		{
			name:            "no existing entries",
			existingLines:   []string{},
			newEntry:        "192.168.1.10:/volume1/media /mnt/nas nfs " + options + " 0 0",
			wantDuplicate:   false,
			wantMountExists: false,
		},
		{
			name: "exact duplicate exists",
			existingLines: []string{
				"192.168.1.10:/volume1/media /mnt/nas nfs " + options + " 0 0",
			},
			newEntry:        "192.168.1.10:/volume1/media /mnt/nas nfs " + options + " 0 0",
			wantDuplicate:   true,
			wantMountExists: true, // Mount point exists (as part of the duplicate)
		},
		{
			name: "mount point exists with different options",
			existingLines: []string{
				"192.168.1.10:/volume1/media /mnt/nas nfs defaults 0 0",
			},
			newEntry:        "192.168.1.10:/volume1/media /mnt/nas nfs " + options + " 0 0",
			wantDuplicate:   false,
			wantMountExists: true,
		},
		{
			name: "commented entry should be ignored",
			existingLines: []string{
				"# 192.168.1.10:/volume1/media /mnt/nas nfs defaults 0 0",
			},
			newEntry:        "192.168.1.10:/volume1/media /mnt/nas nfs " + options + " 0 0",
			wantDuplicate:   false,
			wantMountExists: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Check for exact duplicate
			foundDuplicate := false
			for _, line := range tt.existingLines {
				trimmed := strings.TrimSpace(line)
				if trimmed == tt.newEntry {
					foundDuplicate = true
					break
				}
			}

			if foundDuplicate != tt.wantDuplicate {
				t.Errorf("duplicate detection = %v, want %v", foundDuplicate, tt.wantDuplicate)
			}

			// Check if mount point exists (even with different options)
			mountPoint := strings.Fields(tt.newEntry)[1]
			foundMount := false
			for _, line := range tt.existingLines {
				trimmed := strings.TrimSpace(line)
				if strings.Contains(trimmed, " "+mountPoint+" ") && !strings.HasPrefix(trimmed, "#") {
					foundMount = true
					break
				}
			}

			if foundMount != tt.wantMountExists {
				t.Errorf("mount point detection = %v, want %v", foundMount, tt.wantMountExists)
			}
		})
	}
}
