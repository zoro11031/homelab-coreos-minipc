# Phase 1 Implementation Plan: Core Simplifications

**Status:** Ready for Implementation
**Estimated Lines to Remove:** ~2,500-3,000
**Risk Level:** Low (high-impact, low-risk changes)
**Estimated Time:** 2-3 hours

---

## Overview

Phase 1 focuses on removing infrastructure overhead that provides no value for a single-user TUI tool:
- Test files and mocking infrastructure
- Interface abstractions and dependency injection
- Manager structs with unnecessary constructors
- Cobra CLI framework

These changes have the **highest impact** with **lowest risk** because they primarily remove code rather than changing behavior.

---

## Pre-Requisites

### 1. Backup Current State
```bash
cd /home/user/homelab-coreos-minipc
git checkout -b simplification-backup
git push -u origin simplification-backup
git checkout claude/simplify-homelab-setup-01V8Ce7STywikQzQfaKEqe2h
```

### 2. Create Test Build
```bash
cd homelab-setup
make build
./bin/homelab-setup version  # Verify current build works
```

### 3. Document Current Entry Points
```bash
# Current command structure:
homelab-setup                    # Interactive menu
homelab-setup menu              # Interactive menu (explicit)
homelab-setup version           # Version info
homelab-setup run all           # Run all steps
homelab-setup run quick         # Skip WireGuard
homelab-setup run <step>        # Individual step
homelab-setup status            # Show status
homelab-setup reset             # Reset markers
homelab-setup troubleshoot      # Troubleshooting
homelab-setup wireguard peer    # Add WireGuard peer
```

---

## Implementation Steps

### Step 1: Remove Test Files and Mocks (15 minutes)

**Files to Delete:**
```bash
rm homelab-setup/internal/system/system_test.go
rm homelab-setup/internal/system/filesystem_mock.go
rm homelab-setup/internal/system/filesystem_dryrun.go
rm homelab-setup/internal/config/config_test.go
rm homelab-setup/internal/common/validation_test.go
rm homelab-setup/internal/steps/steps_test.go
rm homelab-setup/internal/steps/container_test.go
rm homelab-setup/internal/steps/deployment_test.go
rm homelab-setup/internal/steps/nfs_config_test.go
rm homelab-setup/internal/steps/wireguard_config_test.go
rm homelab-setup/internal/steps/wireguard_peer_test.go
```

**Update Makefile:**
Remove test-related targets:
- `test`
- `test-verbose`
- `test-coverage`
- `coverage.html` in clean target

**Verification:**
```bash
find homelab-setup -name "*_test.go" -o -name "*_mock.go"  # Should return nothing
make build  # Should still compile
```

**Impact:** ~1,000+ lines removed, no functional change

---

### Step 2: Simplify filesystem implementation (20 minutes)

**Current Structure:**
```
internal/system/filesystem.go          - Implementation
internal/system/filesystem_mock.go     - Mock (already deleted)
internal/system/filesystem_dryrun.go   - Dry-run mode (already deleted)
```

**Actions:**

1. Keep filesystem helpers consolidated in `filesystem.go` with direct function calls.
2. Remove any lingering abstractions that no longer provide value after consolidation.

**Verification:**
```bash
rg "filesystem" homelab-setup/internal/system
make build
```

**Lines Saved:** ~150-200

---

### Step 3: Remove Manager Structs (45 minutes)

Apply the same pattern to all manager structs:

#### 3a. PackageManager
**File:** `internal/system/packages.go`

- Remove `PackageManager` struct
- Remove `NewPackageManager()` constructor
- Convert to functions: `IsPackageInstalled()`, `InstallPackages()`, etc.
- Update callers in: `internal/steps/preflight.go`, `internal/steps/wireguard.go`, `internal/steps/nfs.go`

#### 3b. NetworkManager
**File:** `internal/system/network.go`

- Remove `Network` struct
- Remove `NewNetwork()` constructor
- Convert to functions: `CheckConnectivity()`, `GetDefaultInterface()`, etc.
- Update callers in: `internal/steps/preflight.go`, `internal/steps/wireguard.go`, `internal/steps/nfs.go`

#### 3c. UserManager
**File:** `internal/system/users.go`

- Remove `UserManager` struct
- Remove `NewUserManager()` constructor
- Convert to functions: `GetCurrentUser()`, `UserExists()`, `CreateUser()`, etc.
- Update callers in: `internal/steps/user.go`

#### 3d. ContainerManager
**File:** `internal/system/containers.go`

- Remove `ContainerManager` struct
- Remove `NewContainerManager()` constructor
- Convert to functions: `GetContainerRuntime()`, `PullImages()`, etc.
- Update callers in: `internal/steps/container.go`, `internal/steps/deployment.go`

#### 3e. ServiceManager
**File:** `internal/system/services.go`

- Remove `ServiceManager` struct
- Remove `NewServiceManager()` constructor
- Convert to functions: `EnableService()`, `StartService()`, `ServiceIsActive()`, etc.
- Update callers in: `internal/steps/wireguard.go`, `internal/steps/deployment.go`

**Pattern for Each Manager:**

1. Open the manager file (e.g., `packages.go`)
2. Remove the struct definition and constructor
3. Convert all methods to package-level functions:
   ```go
   // Before:
   type PackageManager struct {}
   func NewPackageManager() *PackageManager { return &PackageManager{} }
   func (pm *PackageManager) IsInstalled(pkg string) bool { ... }

   // After:
   func IsPackageInstalled(pkg string) bool { ... }
   ```
4. Search for all uses of that manager in steps files
5. Update step structs to remove manager field
6. Update step constructors to remove manager parameter
7. Update method calls to use direct function calls

**Verification After Each:**
```bash
make build
./bin/homelab-setup version
```

**Lines Saved:** ~500-800

---

### Step 4: Simplify SetupContext and StepManager (30 minutes)

**Current:** `internal/cli/setup.go` (352 lines)

**Changes:**

1. **Simplify SetupContext:**
   ```go
   // Before:
   type SetupContext struct {
       Config  *config.Config
       Markers *config.Markers
       UI      *ui.UI
       Steps   *StepManager
       SkipWireGuard bool
   }

   // After:
   type SetupContext struct {
       Config  *config.Config
       Markers *config.Markers
       UI      *ui.UI
   }
   ```

2. **Remove StepManager struct entirely:**
   - Delete the entire `StepManager` type
   - Delete `NewStepManager()` function (lines 95-124)
   - Move `GetAllSteps()` to a simple function that returns step metadata
   - Move step runner functions to individual step files

3. **Update NewSetupContext:**
   - Remove all manager initialization (lines 43-48)
   - Remove StepManager initialization (lines 51-61)
   - Just create Config, Markers, UI

   ```go
   func NewSetupContext() (*SetupContext, error) {
       cfg := config.New("")
       if err := cfg.Load(); err != nil {
           return nil, fmt.Errorf("failed to load config: %w", err)
       }

       markers := config.NewMarkers("")
       uiInstance := ui.New()

       return &SetupContext{
           Config:  cfg,
           Markers: markers,
           UI:      uiInstance,
       }, nil
   }
   ```

4. **Move step orchestration to menu.go:**
   - Move `RunAll()` logic to `menu.go`
   - Move `RunStep()` logic to `menu.go`
   - Each step file has its own `Run()` function that takes config, markers, ui directly

**Verification:**
```bash
make build
./bin/homelab-setup  # Test menu navigation
```

**Lines Saved:** ~200-300

---

### Step 5: Remove Cobra Framework (40 minutes)

**Current Structure:**
```
cmd/homelab-setup/main.go          - Root command + entry point
cmd/homelab-setup/cmd_run.go       - run subcommand
cmd/homelab-setup/cmd_status.go    - status subcommand
cmd/homelab-setup/cmd_reset.go     - reset subcommand
cmd/homelab-setup/cmd_troubleshoot.go - troubleshoot subcommand
cmd/homelab-setup/cmd_wireguard.go - wireguard peer subcommand
```

**New Structure:**
```
cmd/homelab-setup/main.go          - Simple main with menu launch
```

**Implementation:**

1. **Rewrite main.go:**
   ```go
   package main

   import (
       "fmt"
       "os"

       "github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/cli"
       "github.com/zoro11031/homelab-coreos-minipc/homelab-setup/pkg/version"
   )

   func main() {
       // Handle version flag
       if len(os.Args) > 1 && os.Args[1] == "version" {
           fmt.Println(version.Info())
           return
       }

       // Launch interactive menu
       ctx, err := cli.NewSetupContext()
       if err != nil {
           fmt.Fprintf(os.Stderr, "Error: failed to initialize: %v\n", err)
           os.Exit(1)
       }

       menu := cli.NewMenu(ctx)
       if err := menu.Show(); err != nil {
           fmt.Fprintf(os.Stderr, "Error: %v\n", err)
           os.Exit(1)
       }
   }
   ```

2. **Move subcommand functionality to menu:**
   - Status → Already in menu as option [S]
   - Reset → Already in menu as option [R]
   - Troubleshoot → Already in menu as option [T]
   - WireGuard peer → Already in menu as option [P]
   - Run all/quick/steps → Already in menu as options [A], [Q], [0-6]

3. **Delete command files:**
   ```bash
   rm homelab-setup/cmd/homelab-setup/cmd_run.go
   rm homelab-setup/cmd/homelab-setup/cmd_status.go
   rm homelab-setup/cmd/homelab-setup/cmd_reset.go
   rm homelab-setup/cmd/homelab-setup/cmd_troubleshoot.go
   rm homelab-setup/cmd/homelab-setup/cmd_wireguard.go
   ```

4. **Update go.mod:**
   ```bash
   cd homelab-setup
   go mod tidy  # Will remove cobra dependency
   ```

**Verification:**
```bash
make build
./bin/homelab-setup          # Should show menu
./bin/homelab-setup version  # Should show version
./bin/homelab-setup invalid  # Should show menu (ignore unknown args)
```

**Lines Saved:** ~350-400

---

## Order of Operations

**Critical:** Follow this exact order to maintain buildability:

1. ✅ Remove test files (safe, no dependencies)
2. ✅ Simplify filesystem helpers
3. ✅ Remove individual manager structs (one at a time, verify build after each)
4. ✅ Simplify SetupContext/StepManager
5. ✅ Remove Cobra framework (last, touches entry point)

**Between each step:** Run `make build` to ensure no breakage

---

## Risk Mitigation

### Low Risk Areas
- Deleting test files ✅ (no runtime impact)
- Removing unused interfaces ✅ (just indirection)
- Converting Manager structs to functions ✅ (equivalent logic)

### Medium Risk Areas
- Removing Cobra framework ⚠️ (changes entry point)
  - Mitigation: Keep version flag, ignore unknown args gracefully
  - All functionality already exists in menu

### Testing Approach

**After each major step:**
```bash
make build
./bin/homelab-setup version
./bin/homelab-setup  # Navigate menu, don't run steps
```

**After Phase 1 complete:**
```bash
# Test on a VM or safe environment:
./bin/homelab-setup
# Walk through menu options:
# - [S] Status → Should show current state
# - [R] Reset → Should prompt for confirmation
# - [T] Troubleshoot → Should show message
# - [P] Add WireGuard Peer → Should prompt
# - [X] Exit → Should exit cleanly
```

---

## Rollback Plan

If anything breaks during implementation:

```bash
git stash  # Save work in progress
git checkout simplification-backup
cd homelab-setup && make build
```

Or revert individual commits:
```bash
git log --oneline  # Find commit to revert
git revert <commit-hash>
```

---

## Success Criteria

- ✅ Binary compiles successfully
- ✅ `homelab-setup version` shows version info
- ✅ `homelab-setup` launches interactive menu
- ✅ All menu options display correctly
- ✅ No test files remain in codebase
- ✅ No "Manager" structs or `New*Manager()` constructors remain
- ✅ No Cobra imports in `go.mod`
- ✅ Codebase reduced by ~2,500-3,000 lines

---

## After Phase 1

**Next Steps:**
- Build and copy binary to CoreOS image: `make build && cp bin/homelab-setup ../files/system/usr/local/bin/`
- Test on actual CoreOS system (if available)
- Commit changes with clear message
- Proceed to Phase 2 (Configuration & State simplification)

**Expected State:**
- ~9,000 lines of Go code (from 11,468)
- Simpler architecture, easier to maintain
- Same functionality, less overhead
- Ready for Phase 2 simplifications

---

## Notes

- **Don't rush:** Verify build after each manager removal
- **Keep commits small:** One manager per commit for easy revert
- **Test incrementally:** Don't wait until the end
- **Document issues:** Note any unexpected dependencies discovered

---

## Questions Before Starting?

- Are there any commands besides `homelab-setup` and `homelab-setup version` that you actually use?
- Do you want to keep any test files for specific components?
- Should we preserve the ability to run individual steps from command line, or is menu-only fine?
