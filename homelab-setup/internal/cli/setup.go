// Package cli provides the command-line interface layer for the homelab setup
// tool, including step orchestration, menu-driven interaction, and command
// dispatch. It bridges user commands to the underlying setup step functions.
package cli

import (
	"fmt"

	"github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/config"
	"github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/steps"
	"github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/ui"
)

// SetupContext holds all dependencies needed for setup operations
type SetupContext struct {
	Config *config.Config
	UI     *ui.UI
	// SkipWireGuard indicates whether WireGuard should be skipped when running all steps
	SkipWireGuard bool
}

// NewSetupContext creates a new SetupContext with all dependencies initialized
func NewSetupContext() (*SetupContext, error) {
	return NewSetupContextWithOptions(false, false)
}

// NewSetupContextWithOptions creates a new SetupContext with custom options
func NewSetupContextWithOptions(nonInteractive bool, skipWireGuard bool) (*SetupContext, error) {
	// Initialize configuration
	cfg := config.New("")
	if err := cfg.Load(); err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize UI
	uiInstance := ui.New()
	uiInstance.SetNonInteractive(nonInteractive)

	return &SetupContext{
		Config:        cfg,
		UI:            uiInstance,
		SkipWireGuard: skipWireGuard,
	}, nil
}

// StepInfo contains metadata about a setup step
type StepInfo struct {
	Name        string
	ShortName   string
	Description string
	MarkerName  string
	Optional    bool
}

// GetAllSteps returns information about all steps in order
func GetAllSteps() []StepInfo {
	return []StepInfo{
		{Name: "Pre-flight Check", ShortName: "preflight", Description: "Verify system requirements", MarkerName: "preflight-complete", Optional: false},
		{Name: "User Setup", ShortName: "user", Description: "Configure user account and permissions", MarkerName: "user-setup-complete", Optional: false},
		{Name: "Directory Setup", ShortName: "directory", Description: "Create directory structure", MarkerName: "directory-setup-complete", Optional: false},
		{Name: "WireGuard Setup", ShortName: "wireguard", Description: "Configure VPN (optional)", MarkerName: "wireguard-setup-complete", Optional: true},
		{Name: "NFS Setup", ShortName: "nfs", Description: "Configure network storage", MarkerName: "nfs-setup-complete", Optional: false},
		{Name: "Container Setup", ShortName: "container", Description: "Configure container services", MarkerName: "container-setup-complete", Optional: false},
		{Name: "Service Deployment", ShortName: "deployment", Description: "Deploy and start services", MarkerName: "service-deployment-complete", Optional: false},
	}
}

// IsStepComplete checks if a step is complete
func IsStepComplete(cfg *config.Config, markerName string) bool {
	return cfg.IsComplete(markerName)
}

// removeMarkerIfRerun removes a marker if the user chooses to rerun the step
func removeMarkerIfRerun(ui *ui.UI, cfg *config.Config, markerName string, rerun bool) {
	if rerun {
		if err := cfg.ClearMarker(markerName); err != nil {
			ui.Warning(fmt.Sprintf("Failed to remove marker: %v", err))
		}
	}
}

// RunStep executes a specific step by short name
func RunStep(ctx *SetupContext, shortName string) error {
	ctx.UI.Header(fmt.Sprintf("Running: %s", shortName))

	var err error

	switch shortName {
	case "preflight":
		err = runPreflight(ctx)
	case "user":
		err = runUser(ctx)
	case "directory":
		err = runDirectory(ctx)
	case "wireguard":
		err = runWireGuard(ctx)
	case "nfs":
		err = runNFS(ctx)
	case "container":
		err = runContainer(ctx)
	case "deployment":
		err = runDeployment(ctx)
	default:
		return fmt.Errorf("unknown step: %s", shortName)
	}

	if err != nil {
		return err
	}

	ctx.UI.Success(fmt.Sprintf("Step '%s' completed successfully!", shortName))
	return nil
}

// AddWireGuardPeer invokes the WireGuard peer workflow helper.
func AddWireGuardPeer(ctx *SetupContext, opts *steps.WireGuardPeerWorkflowOptions) error {
	return steps.RunWireGuardPeerWorkflow(ctx.Config, ctx.UI, opts)
}

// Individual step runners
func runPreflight(ctx *SetupContext) error {
	// Check if already completed
	if IsStepComplete(ctx.Config, "preflight-complete") {
		ctx.UI.Info("Pre-flight check already completed")
		rerun, err := ctx.UI.PromptYesNo("Run again?", false)
		if err != nil || !rerun {
			return nil
		}
		removeMarkerIfRerun(ctx.UI, ctx.Config, "preflight-complete", rerun)
	}

	return steps.RunPreflightChecks(ctx.Config, ctx.UI)
}

func runUser(ctx *SetupContext) error {
	// Check if already completed
	if IsStepComplete(ctx.Config, "user-setup-complete") {
		ctx.UI.Info("User setup already completed")
		rerun, err := ctx.UI.PromptYesNo("Run again?", false)
		if err != nil || !rerun {
			return nil
		}
		removeMarkerIfRerun(ctx.UI, ctx.Config, "user-setup-complete", rerun)
	}

	return steps.RunUserSetup(ctx.Config, ctx.UI)
}

func runDirectory(ctx *SetupContext) error {
	// Check if already completed
	if IsStepComplete(ctx.Config, "directory-setup-complete") {
		ctx.UI.Info("Directory setup already completed")
		rerun, err := ctx.UI.PromptYesNo("Run again?", false)
		if err != nil || !rerun {
			return nil
		}
		removeMarkerIfRerun(ctx.UI, ctx.Config, "directory-setup-complete", rerun)
	}

	return steps.RunDirectorySetup(ctx.Config, ctx.UI)
}

func runWireGuard(ctx *SetupContext) error {
	// Check if already completed
	if IsStepComplete(ctx.Config, "wireguard-setup-complete") {
		ctx.UI.Info("WireGuard setup already completed")
		rerun, err := ctx.UI.PromptYesNo("Run again?", false)
		if err != nil || !rerun {
			return nil
		}
		removeMarkerIfRerun(ctx.UI, ctx.Config, "wireguard-setup-complete", rerun)
	}

	// Use RunWireGuardSetup function
	// This handles all the logic including prompting, key generation, config writing, etc.
	return steps.RunWireGuardSetup(ctx.Config, ctx.UI)
}

func runNFS(ctx *SetupContext) error {
	// Check if already completed
	if IsStepComplete(ctx.Config, "nfs-setup-complete") {
		ctx.UI.Info("NFS setup already completed")
		rerun, err := ctx.UI.PromptYesNo("Run again?", false)
		if err != nil || !rerun {
			return nil
		}
		removeMarkerIfRerun(ctx.UI, ctx.Config, "nfs-setup-complete", rerun)
	}

	// Use RunNFSSetup function
	return steps.RunNFSSetup(ctx.Config, ctx.UI)
}

func runContainer(ctx *SetupContext) error {
	// Check if already completed
	if IsStepComplete(ctx.Config, "container-setup-complete") {
		ctx.UI.Info("Container setup already completed")
		rerun, err := ctx.UI.PromptYesNo("Run again?", false)
		if err != nil || !rerun {
			return nil
		}
		removeMarkerIfRerun(ctx.UI, ctx.Config, "container-setup-complete", rerun)
	}

	// Use RunContainerSetup function
	return steps.RunContainerSetup(ctx.Config, ctx.UI)
}

func runDeployment(ctx *SetupContext) error {
	// Check if already completed
	if IsStepComplete(ctx.Config, "service-deployment-complete") {
		ctx.UI.Info("Service deployment already completed")
		rerun, err := ctx.UI.PromptYesNo("Run again?", false)
		if err != nil || !rerun {
			return nil
		}
		removeMarkerIfRerun(ctx.UI, ctx.Config, "service-deployment-complete", rerun)
	}

	// Use RunDeployment function
	return steps.RunDeployment(ctx.Config, ctx.UI)
}

// RunAll runs all setup steps in order

func RunAll(ctx *SetupContext, skipWireGuard bool) error {
	steps := []string{"preflight", "user", "directory"}

	if !skipWireGuard {
		steps = append(steps, "wireguard")
	}

	steps = append(steps, "nfs", "container", "deployment")

	for _, step := range steps {
		if err := RunStep(ctx, step); err != nil {
			return fmt.Errorf("step %s failed: %w", step, err)
		}
	}

	ctx.UI.Success("All steps completed successfully!")
	return nil
}
