# Final Code Review Prompt for Codex

Please perform a comprehensive final review of the UBlue uCore Homelab Setup Go implementation. This is a complete rewrite of bash scripts into Go for better maintainability and reliability.

## Project Context

**Purpose**: Setup tool for configuring homelab services on Fedora CoreOS/UBlue uCore
**Language**: Go 1.24
**Target System**: rpm-ostree based immutable OS with systemd, podman/docker

## Implementation Status

✅ **Phase 1 Complete**: Foundation (config, UI, validators, version)
✅ **Phase 2 Complete**: System operations (packages, services, users, filesystem, network, containers)
🔄 **Phase 3 Pending**: Setup steps (preflight, user, directory, NFS, container, deployment, WireGuard)
🔄 **Phase 4 Pending**: CLI & features (interactive menu, commands, troubleshooting)

## Recent Fixes Applied

The following issues have been addressed in recent commits:

**Security Fixes**:
- ✅ config.go: Atomic write pattern (temp file + sync + rename)
- ✅ markers.go: Path traversal prevention (validates marker names)
- ✅ filesystem.go: RemoveDirectory safety checks (blocks critical paths)
- ✅ filesystem.go: Type assertion panic prevention (GetOwner, IsMount, BackupFile)
- ✅ filesystem.go: WriteFile ownership security (chown to root after sudo mv)

**Reliability Fixes**:
- ✅ services.go: Added -n flag to sudo operations (fail fast without password)
- ✅ containers.go: Fixed CheckRootless Docker detection (checks SecurityOptions)
- ✅ validation.go: Empty path validation
- ✅ prompts.go: Password validation before confirmation

## Review Scope

Please review the following files in order of priority:

### High Priority (System Operations - Phase 2)

1. **homelab-setup/internal/system/filesystem.go** (~356 lines)
   - File operations with sudo (WriteFile, RemoveDirectory, CopyFile, BackupFile)
   - Mount point detection (IsMount)
   - Ownership and permissions (GetOwner, Chown, Chmod)
   - Disk usage tracking

2. **homelab-setup/internal/system/services.go** (~209 lines)
   - Systemd service management (Enable, Disable, Start, Stop, Restart)
   - Service status checking
   - Journal log retrieval
   - All use `sudo -n` for non-interactive operation

3. **homelab-setup/internal/system/users.go** (~230 lines)
   - User/group creation and management
   - UID/GID lookups
   - Subuid/subgid mappings for rootless containers
   - Shell configuration

4. **homelab-setup/internal/system/packages.go** (~115 lines)
   - rpm-ostree package detection
   - Package version queries
   - Batch package checking (CheckMultiple)

5. **homelab-setup/internal/system/network.go** (~165 lines)
   - Network connectivity testing
   - Port scanning (TCP/UDP)
   - NFS server validation
   - DNS resolution

6. **homelab-setup/internal/system/containers.go** (~315 lines)
   - Container runtime detection (podman/docker)
   - Rootless mode validation
   - Container/image listing
   - Compose command detection

### Medium Priority (Configuration & Validation - Phase 1)

7. **homelab-setup/internal/config/config.go** (~170 lines)
   - Key-value config file management
   - Atomic write pattern (recently fixed)
   - Auto-loading to prevent data loss

8. **homelab-setup/internal/config/markers.go** (~105 lines)
   - Completion marker tracking
   - Path traversal prevention (recently fixed)

9. **homelab-setup/internal/common/validation.go** (~132 lines)
   - Input validators (IP, port, path, username, domain, timezone)

10. **homelab-setup/internal/ui/prompts.go** (~144 lines)
    - Interactive prompts using survey library
    - Password confirmation with validation
    - Multi-select optimization (O(n+m) using hash map)

11. **homelab-setup/internal/ui/output.go** (~100 lines)
    - Colored output formatting

### Low Priority (Infrastructure)

12. **homelab-setup/cmd/homelab-setup/main.go** (~46 lines)
    - Cobra CLI setup
    - Error handling

13. **homelab-setup/Makefile** (~106 lines)
    - Build system

14. **homelab-setup/pkg/version/version.go** (~30 lines)
    - Version information

## What to Look For

### 🔴 Critical Issues (Must Fix)

1. **Security Vulnerabilities**:
   - Command injection in exec.Command calls
   - Path traversal vulnerabilities
   - Privilege escalation risks
   - Unsafe file operations
   - Information disclosure
   - Race conditions in file operations

2. **Data Loss Risks**:
   - Non-atomic writes that could corrupt data
   - Missing error checks on file operations
   - Truncation without proper backups
   - Resource leaks (unclosed files, connections)

3. **Panic/Crash Risks**:
   - Unchecked type assertions
   - Out-of-bounds array/slice access
   - Nil pointer dereferences
   - Division by zero
   - Unhandled errors that should be fatal

### 🟡 High Priority Issues (Should Fix)

4. **Logic Errors**:
   - Incorrect error handling
   - Wrong exit conditions in loops
   - Off-by-one errors
   - Incorrect operator usage (&&/|| confusion)
   - Type conversion issues

5. **Concurrency Issues**:
   - Data races (if goroutines are used)
   - Deadlocks
   - Missing synchronization

6. **Resource Management**:
   - File descriptor leaks
   - Missing defers for cleanup
   - Goroutine leaks
   - Memory leaks

### 🟢 Medium Priority Issues (Good to Fix)

7. **Error Messages**:
   - Unclear or unhelpful error messages
   - Missing context in errors
   - Errors that don't suggest solutions

8. **Performance Issues**:
   - Inefficient algorithms (O(n²) where O(n) possible)
   - Unnecessary allocations
   - Repeated expensive operations
   - Missing caching where appropriate

9. **Code Quality**:
   - Code duplication
   - Overly complex functions
   - Inconsistent error handling patterns
   - Magic numbers without explanation
   - Missing validation

### ℹ️ Low Priority Issues (Nice to Have)

10. **Style & Conventions**:
    - Non-idiomatic Go code
    - Inconsistent naming
    - Missing documentation
    - Unused variables/imports

## Special Focus Areas

### Sudo Operations
All sudo operations use `-n` flag for non-interactive execution. Verify:
- ✅ Error messages are helpful when sudo fails
- ✅ Operations are properly validated before sudo execution
- ✅ No command injection vulnerabilities in sudo arguments

### File Operations
Recent fixes implemented atomic writes and safety checks. Verify:
- ✅ All file writes are atomic or have explicit safety measures
- ✅ Permissions and ownership are set correctly
- ✅ Critical paths are protected from deletion
- ✅ Path traversal is prevented in all user input

### Container Operations
System supports both Podman and Docker. Verify:
- ✅ Runtime detection is accurate
- ✅ Rootless mode detection works for both runtimes
- ✅ Commands are constructed safely for exec.Command

### Error Handling
Verify consistent patterns:
- ✅ All errors are wrapped with context using fmt.Errorf with %w
- ✅ Errors from external commands include output for debugging
- ✅ User-facing errors are actionable

## Output Format

Please provide findings in this format:

```
## Critical Issues Found: X

### 1. [File:Line] Brief Description
**Severity**: Critical
**Category**: Security / Data Loss / Panic Risk
**Issue**: Detailed explanation of the problem
**Impact**: What could go wrong
**Recommendation**: Specific fix suggestion
**Code**:
```go
// Current code (if applicable)
```

---

## High Priority Issues Found: X

[Same format as above]

---

## Medium Priority Issues Found: X

[Same format as above]

---

## Summary

- Critical issues: X
- High priority: X
- Medium priority: X
- Low priority: X

**Overall Assessment**: [Ready for Phase 3 / Needs fixes before proceeding / Major refactoring needed]
```

## What NOT to Report

Please **do not** report these already-documented items:

1. ❌ users.go error messages could be more helpful (documented in future-refactoring.md)
2. ❌ network.go TestTCPConnection duplicates IsPortOpen (documented in future-refactoring.md)
3. ❌ packages.go CheckMultiple could batch rpm queries (documented in future-refactoring.md)
4. ❌ containers.go GetComposeCommand returns string instead of []string (documented in future-refactoring.md)
5. ❌ Missing tests for UI package (acceptable - UI testing is interactive)
6. ❌ No tests for main.go (acceptable - integration point only)

## Additional Context

**Test Coverage**: All 49 tests pass
- internal/common: 6 validation tests
- internal/config: 7 config/marker tests
- internal/system: 7 system operation tests

**Target Deployment**:
- Single static binary
- No runtime dependencies
- Runs on Fedora CoreOS with passwordless sudo
- May call: systemctl, rpm, journalctl, useradd, groupadd, chown, chmod, mkdir, rm, cp, mount, etc.

## Review Goals

The goal is to ensure **Phase 2 is production-ready** before proceeding to Phase 3. We want to:

1. ✅ Catch any remaining security vulnerabilities
2. ✅ Prevent data loss or corruption scenarios
3. ✅ Eliminate panic/crash risks
4. ✅ Ensure error handling is robust
5. ✅ Validate all user inputs properly

Please be thorough and treat this as a security-critical system administration tool that will run with elevated privileges.

---

**Thank you for your review!** Your findings will help ensure a robust, secure implementation before we proceed to Phase 3 (setup steps implementation).
