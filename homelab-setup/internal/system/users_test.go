package system

import (
	"testing"
)

// TestValidateUserIDsMatch tests UID/GID validation logic
func TestValidateUserIDsMatch(t *testing.T) {
	// This test requires actual users to exist
	// It's primarily for documenting expected behavior
	t.Skip("Requires real user accounts on system")

	tests := []struct {
		name         string
		username     string
		expectedUID  int
		expectedGID  int
		wantUIDMatch bool
		wantGIDMatch bool
		wantErr      bool
	}{
		{
			name:         "matching IDs",
			username:     "testuser",
			expectedUID:  1000,
			expectedGID:  1000,
			wantUIDMatch: true,
			wantGIDMatch: true,
			wantErr:      false,
		},
		{
			name:         "mismatched UID",
			username:     "testuser",
			expectedUID:  1001,
			expectedGID:  1000,
			wantUIDMatch: false,
			wantGIDMatch: true,
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uidMatch, gidMatch, _, _, err := ValidateUserIDsMatch(tt.username, tt.expectedUID, tt.expectedGID)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateUserIDsMatch() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if uidMatch != tt.wantUIDMatch {
				t.Errorf("ValidateUserIDsMatch() uidMatch = %v, want %v", uidMatch, tt.wantUIDMatch)
			}
			if gidMatch != tt.wantGIDMatch {
				t.Errorf("ValidateUserIDsMatch() gidMatch = %v, want %v", gidMatch, tt.wantGIDMatch)
			}
		})
	}
}

// TestGetUserShell tests shell retrieval
func TestGetUserShell(t *testing.T) {
	// This test requires a real user
	// For documentation purposes
	t.Skip("Requires real user account on system")
}

// TestCreateSystemUser tests system user creation
func TestCreateSystemUser(t *testing.T) {
	// This test would require sudo privileges and system modification
	// Should only be run in a test environment
	t.Skip("Requires sudo and modifies system - manual test only")
}

// Test user existence check
func TestUserExists(t *testing.T) {
	// Test with root user which should always exist on Unix systems
	exists, err := UserExists("root")
	if err != nil {
		t.Fatalf("UserExists(root) returned error: %v", err)
	}
	if !exists {
		t.Error("UserExists(root) = false, want true")
	}

	// Test with a user that definitely doesn't exist
	exists, err = UserExists("nonexistent_user_12345_test")
	if err != nil {
		t.Fatalf("UserExists(nonexistent) returned error: %v", err)
	}
	if exists {
		t.Error("UserExists(nonexistent_user_12345_test) = true, want false")
	}
}

// Test UID/GID retrieval for root
func TestGetUIDGIDForRoot(t *testing.T) {
	// Root should always have UID 0
	uid, err := GetUID("root")
	if err != nil {
		t.Fatalf("GetUID(root) returned error: %v", err)
	}
	if uid != 0 {
		t.Errorf("GetUID(root) = %d, want 0", uid)
	}

	// Root should always have GID 0
	gid, err := GetGID("root")
	if err != nil {
		t.Fatalf("GetGID(root) returned error: %v", err)
	}
	if gid != 0 {
		t.Errorf("GetGID(root) = %d, want 0", gid)
	}
}
