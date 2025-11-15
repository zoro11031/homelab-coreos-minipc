# Future Refactoring Tasks

This document tracks code improvements identified during code review that require more extensive refactoring. These are not critical bugs, but improvements for code quality, performance, and maintainability.

## Priority: Medium

### 1. users.go - Improve Error Messages

**File**: `internal/system/users.go`
**Functions**: `CreateUser`, `DeleteUser`, `AddUserToGroup`, `SetUserShell`

**Issue**: Error messages don't provide guidance on common failure causes.

**Suggested Improvement**:
```go
output, err := cmd.CombinedOutput()
if err != nil {
    outputStr := string(output)
    if strings.Contains(outputStr, "already exists") {
        return fmt.Errorf("user %s already exists", username)
    }
    if strings.Contains(outputStr, "permission denied") || strings.Contains(outputStr, "not in sudoers") {
        return fmt.Errorf("insufficient permissions to create user %s: ensure you have sudo access", username)
    }
    return fmt.Errorf("failed to create user %s: %w\nOutput: %s", username, err, outputStr)
}
```

**Impact**: Better user experience with more helpful error messages

---

### 2. network.go - Remove Code Duplication

**File**: `internal/system/network.go`
**Function**: `TestTCPConnection`

**Issue**: Duplicates logic from `IsPortOpen` with only a hardcoded timeout difference.

**Suggested Improvement**:
```go
// Remove TestTCPConnection and update callers to use IsPortOpen directly
func (n *Network) TestTCPConnection(host string, port int) (bool, error) {
    return n.IsPortOpen(host, port, 5)
}
```

**Impact**: Reduced code duplication, easier maintenance

---

### 3. packages.go - Optimize CheckMultiple Performance

**File**: `internal/system/packages.go`
**Function**: `CheckMultiple`

**Issue**: Sequential rpm queries spawn multiple processes (slow for many packages).

**Suggested Improvement**:
```go
func (pm *PackageManager) CheckMultiple(packages []string) (map[string]bool, error) {
    result := make(map[string]bool)

    // Query all packages at once
    args := append([]string{"-q"}, packages...)
    cmd := exec.Command("rpm", args...)
    output, _ := cmd.CombinedOutput()

    // Parse output to determine which packages are installed
    lines := strings.Split(string(output), "\n")
    installedSet := make(map[string]bool)
    for _, line := range lines {
        if line != "" && !strings.HasPrefix(line, "package") {
            // Extract package name from "package-version-release" format
            parts := strings.Split(line, "-")
            if len(parts) > 0 {
                installedSet[parts[0]] = true
            }
        }
    }

    // Map results
    for _, pkg := range packages {
        result[pkg] = installedSet[pkg]
    }

    return result, nil
}
```

**Impact**: Significant performance improvement when checking multiple packages

---

### 4. containers.go - Fix GetComposeCommand Return Type

**File**: `internal/system/containers.go`
**Function**: `GetComposeCommand`

**Issue**: Returns command as single string ("podman compose") but `exec.Command()` expects separate arguments.

**Suggested Improvement - Option 1** (Return slice):
```go
func (cm *ContainerManager) GetComposeCommand(runtime ContainerRuntime) ([]string, error) {
    switch runtime {
    case RuntimePodman:
        if CommandExists("podman-compose") {
            return []string{"podman-compose"}, nil
        }
        cmd := exec.Command("podman", "compose", "version")
        if err := cmd.Run(); err == nil {
            return []string{"podman", "compose"}, nil
        }
        return nil, fmt.Errorf("neither podman-compose nor podman compose plugin found")
    case RuntimeDocker:
        if CommandExists("docker-compose") {
            return []string{"docker-compose"}, nil
        }
        cmd := exec.Command("docker", "compose", "version")
        if err := cmd.Run(); err == nil {
            return []string{"docker", "compose"}, nil
        }
        return nil, fmt.Errorf("neither docker-compose nor docker compose plugin found")
    default:
        return nil, fmt.Errorf("unsupported runtime: %s", runtime)
    }
}

// Usage would change to:
// composeCmd, err := cm.GetComposeCommand(runtime)
// cmd := exec.Command(composeCmd[0], append(composeCmd[1:], "up", "-d")...)
```

**Suggested Improvement - Option 2** (Document behavior):
```go
// GetComposeCommand returns the compose command as a space-separated string.
// Callers must split on space when using with exec.Command:
//   composeStr, _ := cm.GetComposeCommand(runtime)
//   parts := strings.Fields(composeStr)
//   cmd := exec.Command(parts[0], append(parts[1:], args...)...)
func (cm *ContainerManager) GetComposeCommand(runtime ContainerRuntime) (string, error) {
    // existing implementation
}
```

**Impact**: Prevents incorrect usage of exec.Command, improves API clarity

---

## Implementation Notes

These refactorings can be tackled incrementally:

1. **Phase 1**: Fix error messages in users.go (low risk, high value)
2. **Phase 2**: Remove network.go duplication (simple refactor)
3. **Phase 3**: Optimize packages.go (needs testing with various package states)
4. **Phase 4**: Refactor containers.go API (breaking change, needs careful migration)

## Testing Checklist

When implementing these changes:

- [ ] Run full test suite: `make test-coverage`
- [ ] Test with real system operations (not just unit tests)
- [ ] Verify error messages are helpful and actionable
- [ ] Check performance impact of changes
- [ ] Update documentation if API changes

## Related Files

- Tests: `internal/system/system_test.go`
- Documentation: `README.md`, `docs/go-rewrite-plan.md`
