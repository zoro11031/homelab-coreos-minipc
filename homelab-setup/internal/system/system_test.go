package system

import (
	"testing"
)

// Test CommandExists
func TestCommandExists(t *testing.T) {
	tests := []struct {
		name    string
		command string
		want    bool
	}{
		{"ls exists", "ls", true},
		{"bash exists", "bash", true},
		{"nonexistent command", "this-command-does-not-exist-xyz", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CommandExists(tt.command)
			if got != tt.want {
				t.Errorf("CommandExists(%s) = %v, want %v", tt.command, got, tt.want)
			}
		})
	}
}

// Test IsRpmOstreeSystem
func TestIsRpmOstreeSystem(t *testing.T) {
	// This just tests the function runs without error
	_ = IsRpmOstreeSystem()
}

// Test ContainerRuntime DetectRuntime
func TestDetectRuntime(t *testing.T) {
	cm := NewContainerManager()
	runtime, err := cm.DetectRuntime()

	// Should find either podman, docker, or return error
	if err != nil {
		if runtime != RuntimeNone {
			t.Errorf("DetectRuntime() returned error but runtime is %v (expected none)", runtime)
		}
	} else {
		if runtime != RuntimePodman && runtime != RuntimeDocker {
			t.Errorf("DetectRuntime() = %v, want podman or docker", runtime)
		}
	}
}

// Test FileSystem basic functions
func TestFileSystemBasics(t *testing.T) {
	fs := NewFileSystem()

	// Test FileExists with /etc/os-release (should exist on all Linux systems)
	exists, err := fs.FileExists("/etc/os-release")
	if err != nil {
		t.Errorf("FileExists(/etc/os-release) error: %v", err)
	}
	if !exists {
		t.Error("FileExists(/etc/os-release) = false, want true")
	}

	// Test DirectoryExists with /tmp (should exist)
	exists, err = fs.DirectoryExists("/tmp")
	if err != nil {
		t.Errorf("DirectoryExists(/tmp) error: %v", err)
	}
	if !exists {
		t.Error("DirectoryExists(/tmp) = false, want true")
	}

	// Test non-existent path
	exists, err = fs.FileExists("/this/path/does/not/exist/xyz")
	if err != nil {
		t.Errorf("FileExists(non-existent) error: %v", err)
	}
	if exists {
		t.Error("FileExists(non-existent) = true, want false")
	}
}

// Test Network basic functions
func TestNetworkBasics(t *testing.T) {
	net := NewNetwork()

	// Test GetHostname
	hostname, err := net.GetHostname()
	if err != nil {
		t.Errorf("GetHostname() error: %v", err)
	}
	if hostname == "" {
		t.Error("GetHostname() returned empty string")
	}

	// Test GetAllInterfaces (should have at least loopback)
	interfaces, err := net.GetAllInterfaces()
	if err != nil {
		t.Errorf("GetAllInterfaces() error: %v", err)
	}
	if len(interfaces) == 0 {
		t.Error("GetAllInterfaces() returned no interfaces")
	}

	// Test GetLocalIP (may fail in some environments, so just check it doesn't panic)
	_, _ = net.GetLocalIP()
}

// Test UserManager basic functions
func TestUserManagerBasics(t *testing.T) {
	um := NewUserManager()

	// Test GetCurrentUser
	user, err := um.GetCurrentUser()
	if err != nil {
		t.Errorf("GetCurrentUser() error: %v", err)
	}
	if user == nil {
		t.Error("GetCurrentUser() returned nil")
	}

	// Test UserExists with root (should always exist)
	exists, err := um.UserExists("root")
	if err != nil {
		t.Errorf("UserExists(root) error: %v", err)
	}
	if !exists {
		t.Error("UserExists(root) = false, want true")
	}

	// Test GroupExists with root (should always exist)
	exists, err = um.GroupExists("root")
	if err != nil {
		t.Errorf("GroupExists(root) error: %v", err)
	}
	if !exists {
		t.Error("GroupExists(root) = false, want true")
	}

	// Test GetUID for root (should be 0)
	uid, err := um.GetUID("root")
	if err != nil {
		t.Errorf("GetUID(root) error: %v", err)
	}
	if uid != 0 {
		t.Errorf("GetUID(root) = %d, want 0", uid)
	}
}

// Test ServiceManager basic functions
func TestServiceManagerBasics(t *testing.T) {
	sm := NewServiceManager()

	// Just test that the struct can be created
	if sm == nil {
		t.Error("NewServiceManager() returned nil")
	}

	// Test ServiceExists with a system service that typically exists
	// Using a simple check that doesn't require the service to be present
	// Just verify the function doesn't panic
	_, _ = sm.ServiceExists("systemd-journald.service")
}

// Test PackageManager basic functions
func TestPackageManagerBasics(t *testing.T) {
	pm := NewPackageManager()

	// Just test that the struct can be created
	if pm == nil {
		t.Error("NewPackageManager() returned nil")
	}

	// Test CommandExists with bash (should exist on all Linux systems)
	if !CommandExists("bash") {
		t.Error("bash command not found, expected to exist")
	}
}
