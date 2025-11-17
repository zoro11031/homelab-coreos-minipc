# Phase 2 Simplification - Audit Prompt

## Context

This repository contains a Go-based CLI tool for setting up a homelab on Fedora CoreOS. Phase 2 simplification has been completed, removing 373 lines of code (4.6% reduction from 8,066 to 7,693 lines) through architectural improvements.

## Your Task

Audit the Phase 2 changes for correctness, maintainability, and potential issues. Review the git commits on branch `claude/simplify-homelab-phase2-01W8uFbBYmigdeEotYPFvmfx`.

## Changes Made in Phase 2

### Phase 2A: Foundation Simplification (276 lines removed)

1. **Removed CommandRunner abstraction** (commit: d68fd9f)
   - Deleted `internal/system/commandrunner.go`
   - Replaced interface with direct `exec.Command()` calls
   - Updated `internal/steps/nfs.go` to use direct execution

2. **Inlined common validators** (commit: 016664a)
   - Reduced `internal/common/validation.go` from 230 → 88 lines
   - Inlined: ValidateIP, ValidatePort, ValidateDomain, ValidateWireGuardKey, ValidateCIDR
   - Kept: ValidateSafePath (security), ValidatePath, ValidateUsername
   - Updated wireguard.go and nfs.go with inline validation

3. **Consolidated config and markers** (commit: 9ee7105)
   - Deleted `internal/config/markers.go` (211 lines)
   - Merged marker operations into `Config` type
   - New API: `config.MarkComplete()`, `config.IsComplete()`, etc.
   - Updated all step constructors: 3 params → 2 params (removed markers param)
   - Updated 8 step files + 2 CLI files

### Phase 2B: Step Architecture Transformation (97 lines removed)

4. **Converted simple steps to functions** (commit: af5aab8)
   - Converted: directory.go, preflight.go, user.go
   - Removed structs and constructors
   - Pattern: `type X struct + NewX() + (x *X) Run()` → `func RunX(cfg, ui)`

5. **Converted remaining steps to functions** (commit: 33a3033)
   - Converted: container.go, nfs.go, wireguard.go, deployment.go, wireguard_peer.go
   - All helper methods → unexported functions
   - Preserved WireGuardKeyGenerator interface for testing

## Audit Checklist

### 1. Correctness Review

**Build and Compilation**:
- [ ] Run `cd homelab-setup && make build` - does it compile?
- [ ] Check for unused imports
- [ ] Check for unreachable code
- [ ] Verify all exported functions are properly capitalized

**Functionality Preservation**:
- [ ] Compare before/after behavior - are all features still accessible?
- [ ] Check marker file operations - do completion markers work correctly?
- [ ] Verify config operations - does Get/Set still work?
- [ ] Check step execution flow - do steps run in correct order?

**Error Handling**:
- [ ] Review all `fmt.Errorf` calls - proper error wrapping?
- [ ] Check for lost error context in conversions
- [ ] Verify error messages are still user-friendly
- [ ] Look for swallowed errors

### 2. Security Review

**Input Validation**:
- [ ] Check that path validation is still present (ValidateSafePath)
- [ ] Verify CIDR validation in wireguard.go is correct
- [ ] Confirm IP address validation in nfs.go works
- [ ] Review WireGuard key validation logic

**Command Injection**:
- [ ] Verify all `exec.Command` calls avoid shell interpretation
- [ ] Check that paths passed to system commands are validated
- [ ] Confirm no user input is passed to shell via `sh -c`
- [ ] Review sudo command construction

**File Operations**:
- [ ] Check marker file creation for path traversal vulnerabilities
- [ ] Verify config file operations use atomic writes
- [ ] Review file permissions (0600, 0644, 0755)

### 3. Architecture Review

**Function-Based Pattern**:
- [ ] Are all `Run*()` functions consistent in signature?
- [ ] Do all take `(cfg *config.Config, ui *ui.UI)` as first two params?
- [ ] Are helper functions properly unexported (lowercase)?
- [ ] Is the pattern applied consistently across all 8 step files?

**Dependency Injection**:
- [ ] Are dependencies passed explicitly as parameters?
- [ ] No hidden dependencies in closures?
- [ ] Config and UI passed to all functions that need them?

**API Design**:
- [ ] Is the config.MarkComplete() API intuitive?
- [ ] Are marker method names clear (MarkComplete vs Create)?
- [ ] Is the removal of Markers parameter justified?

### 4. Maintainability Review

**Code Duplication**:
- [ ] Was inline validation worth it? (validation.go: 230→88 lines)
- [ ] Check for duplicated validation logic across files
- [ ] Look for repeated patterns that could be extracted

**Function Size**:
- [ ] Are converted functions too long? (look for 100+ line functions)
- [ ] Could any helpers be further decomposed?
- [ ] Are function responsibilities clear?

**Documentation**:
- [ ] Do exported functions have clear docstrings?
- [ ] Are complex algorithms explained?
- [ ] Are security-critical sections documented?

### 5. Potential Issues

**Concurrency**:
- [ ] Check `ensureCanonicalMarker()` - is it still race-safe?
- [ ] Review `config.MarkCompleteIfNotExists()` implementation
- [ ] Verify atomic file operations

**State Management**:
- [ ] Does config.IsComplete() work correctly?
- [ ] Are markers properly created/checked?
- [ ] Can steps be re-run after marker deletion?

**Edge Cases**:
- [ ] What happens if config file doesn't exist?
- [ ] What if marker directory can't be created?
- [ ] Handle concurrent execution properly?

### 6. Testing Concerns

**Testability**:
- [ ] Can the new function-based code be tested?
- [ ] Is WireGuardKeyGenerator interface sufficient for mocking?
- [ ] Are pure functions (no side effects) identified?

**Missing Test Coverage**:
- [ ] Identify functions that should have tests
- [ ] Note any complex logic without coverage
- [ ] Check if inline validators need tests

## Specific Files to Review

### Critical Files (security/correctness):
1. `internal/config/config.go` - marker and config operations
2. `internal/steps/marker_helpers.go` - race-safe marker migration
3. `internal/common/validation.go` - remaining validators
4. `internal/steps/wireguard.go` - inline crypto validation
5. `internal/steps/nfs.go` - inline IP/domain validation

### Architecture Files (maintainability):
1. All files in `internal/steps/` - function-based pattern
2. `internal/cli/setup.go` - updated call sites
3. `internal/cli/menu.go` - updated call sites

## Output Format

Please provide your audit report in this format:

```markdown
# Phase 2 Audit Report

## Summary
- Overall assessment: [PASS/FAIL/NEEDS_WORK]
- Critical issues: [number]
- Warnings: [number]
- Suggestions: [number]

## Critical Issues
[List any blocking issues that must be fixed]

## Warnings
[List potential problems that should be addressed]

## Suggestions
[List improvements for future consideration]

## Detailed Findings

### 1. Correctness
[Your findings]

### 2. Security
[Your findings]

### 3. Architecture
[Your findings]

### 4. Maintainability
[Your findings]

### 5. Edge Cases
[Your findings]

### 6. Testability
[Your findings]

## Specific File Reviews

### internal/config/config.go
[Review findings]

### internal/steps/marker_helpers.go
[Review findings]

[... continue for other critical files]

## Recommendations

### Must Fix (Blocking)
[List critical items]

### Should Fix (Important)
[List important items]

### Could Fix (Nice to have)
[List optional improvements]

## Conclusion
[Your overall assessment of the Phase 2 changes]
```

## How to Perform the Audit

1. **Clone the repository**:
   ```bash
   git clone <repo-url>
   cd homelab-coreos-minipc
   git checkout claude/simplify-homelab-phase2-01W8uFbBYmigdeEotYPFvmfx
   ```

2. **Review the commits**:
   ```bash
   git log --oneline main..HEAD
   git show d68fd9f  # CommandRunner removal
   git show 016664a  # Validator inlining
   git show 9ee7105  # Config consolidation
   git show af5aab8  # Simple steps conversion
   git show 33a3033  # Remaining steps conversion
   ```

3. **Build and test**:
   ```bash
   cd homelab-setup
   make build
   make test  # if tests exist
   ```

4. **Review each file** listed in the checklist above

5. **Look for patterns** across the changes

6. **Generate your audit report** using the format above

## Questions to Consider

- Did the simplification actually improve maintainability?
- Are there security regressions from removing abstractions?
- Is the function-based pattern consistently applied?
- Are there better ways to achieve the same simplification?
- Did we lose any important testing hooks?
- Are the inline validators worth the code duplication?

## Success Criteria

The audit should assess whether Phase 2:
- ✅ Maintains all existing functionality
- ✅ Introduces no security vulnerabilities
- ✅ Improves code maintainability
- ✅ Follows Go best practices
- ✅ Has acceptable test coverage
- ✅ Handles edge cases properly

---

**Note**: This is a production system for managing homelab infrastructure. Be thorough in your security and correctness reviews.
