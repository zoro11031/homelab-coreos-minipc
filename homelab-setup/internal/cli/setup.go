package cli

import (
	"fmt"

	"github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/config"
	"github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/steps"
	"github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/system"
	"github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/ui"
)

// SetupContext holds all dependencies needed for setup operations
type SetupContext struct {
	Config  *config.Config
	Markers *config.Markers
	UI      *ui.UI
	Steps   *StepManager
}

// NewSetupContext creates a new SetupContext with all dependencies initialized
func NewSetupContext() (*SetupContext, error) {
	return NewSetupContextWithOptions(false)
}

// NewSetupContextWithOptions creates a new SetupContext with custom options
func NewSetupContextWithOptions(nonInteractive bool) (*SetupContext, error) {
	// Initialize configuration
	cfg := config.New("")
	if err := cfg.Load(); err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize markers
	markers := config.NewMarkers("")

	// Initialize UI
	uiInstance := ui.New()
	uiInstance.SetNonInteractive(nonInteractive)

	// Initialize system components
	packages := system.NewPackageManager()
	network := system.NewNetwork()
	userMgr := system.NewUserManager()
	fs := system.NewFileSystem()
	containers := system.NewContainerManager()
	services := system.NewServiceManager()

	// Initialize step manager
	stepMgr := NewStepManager(
		cfg,
		markers,
		uiInstance,
		packages,
		network,
		userMgr,
		fs,
		containers,
		services,
	)

	return &SetupContext{
		Config:  cfg,
		Markers: markers,
		UI:      uiInstance,
		Steps:   stepMgr,
	}, nil
}

// StepManager manages all setup steps
type StepManager struct {
	config     *config.Config
	markers    *config.Markers
	ui         *ui.UI
	packages   *system.PackageManager
	network    *system.Network
	userMgr    *system.UserManager
	fs         *system.FileSystem
	containers *system.ContainerManager
	services   *system.ServiceManager

	// Step instances
	preflight  *steps.PreflightChecker
	user       *steps.UserConfigurator
	directory  *steps.DirectorySetup
	wireguard  *steps.WireGuardSetup
	nfs        *steps.NFSConfigurator
	container  *steps.ContainerSetup
	deployment *steps.Deployment
}

// NewStepManager creates a new StepManager
func NewStepManager(
	cfg *config.Config,
	markers *config.Markers,
	uiInstance *ui.UI,
	packages *system.PackageManager,
	network *system.Network,
	userMgr *system.UserManager,
	fs *system.FileSystem,
	containers *system.ContainerManager,
	services *system.ServiceManager,
) *StepManager {
	return &StepManager{
		config:     cfg,
		markers:    markers,
		ui:         uiInstance,
		packages:   packages,
		network:    network,
		userMgr:    userMgr,
		fs:         fs,
		containers: containers,
		services:   services,
		preflight:  steps.NewPreflightChecker(packages, network, uiInstance, markers, cfg),
		user:       steps.NewUserConfigurator(userMgr, cfg, uiInstance, markers),
		directory:  steps.NewDirectorySetup(fs, cfg, uiInstance, markers),
		wireguard:  steps.NewWireGuardSetup(packages, services, fs, network, cfg, uiInstance, markers),
		nfs:        steps.NewNFSConfigurator(fs, network, cfg, uiInstance, markers),
		container:  steps.NewContainerSetup(containers, fs, cfg, uiInstance, markers),
		deployment: steps.NewDeployment(containers, fs, services, cfg, uiInstance, markers),
	}
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
func (sm *StepManager) GetAllSteps() []StepInfo {
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
func (sm *StepManager) IsStepComplete(markerName string) bool {
	exists, err := sm.markers.Exists(markerName)
	if err != nil {
		return false
	}
	return exists
}

// RunStep executes a specific step by short name
func (sm *StepManager) RunStep(shortName string) error {
	sm.ui.Header(fmt.Sprintf("Running: %s", shortName))

	var err error
	var markerName string

	switch shortName {
	case "preflight":
		markerName = "preflight-complete"
		err = sm.runPreflight()
	case "user":
		markerName = "user-setup-complete"
		err = sm.runUser()
	case "directory":
		markerName = "directory-setup-complete"
		err = sm.runDirectory()
	case "wireguard":
		markerName = "wireguard-setup-complete"
		err = sm.runWireGuard()
	case "nfs":
		markerName = "nfs-setup-complete"
		err = sm.runNFS()
	case "container":
		markerName = "container-setup-complete"
		err = sm.runContainer()
	case "deployment":
		markerName = "service-deployment-complete"
		err = sm.runDeployment()
	default:
		return fmt.Errorf("unknown step: %s", shortName)
	}

	if err != nil {
		return err
	}

	// Mark step as complete
	if err := sm.markers.Create(markerName); err != nil {
		sm.ui.Warning(fmt.Sprintf("Failed to create completion marker: %v", err))
	}

	sm.ui.Success(fmt.Sprintf("Step '%s' completed successfully!", shortName))
	return nil
}

// Individual step runners
func (sm *StepManager) runPreflight() error {
	// Check if already completed
	if sm.IsStepComplete("preflight-complete") {
		sm.ui.Info("Pre-flight check already completed")
		rerun, err := sm.ui.PromptYesNo("Run again?", false)
		if err != nil || !rerun {
			return nil
		}
	}

	// Use the RunAll method that exists in PreflightChecker
	return sm.preflight.RunAll()
}

func (sm *StepManager) runUser() error {
	// Check if already completed
	if sm.IsStepComplete("user-setup-complete") {
		sm.ui.Info("User setup already completed")
		rerun, err := sm.ui.PromptYesNo("Run again?", false)
		if err != nil || !rerun {
			return nil
		}
	}

	// Use the Run method that exists in UserConfigurator
	return sm.user.Run()
}

func (sm *StepManager) runDirectory() error {
	// Check if already completed
	if sm.IsStepComplete("directory-setup-complete") {
		sm.ui.Info("Directory setup already completed")
		rerun, err := sm.ui.PromptYesNo("Run again?", false)
		if err != nil || !rerun {
			return nil
		}
	}

	// Use the Run method that exists in DirectorySetup
	return sm.directory.Run()
}

func (sm *StepManager) runWireGuard() error {
	// Check if already completed
	if sm.IsStepComplete("wireguard-setup-complete") {
		sm.ui.Info("WireGuard setup already completed")
		rerun, err := sm.ui.PromptYesNo("Run again?", false)
		if err != nil || !rerun {
			return nil
		}
	}

	// Use the Run method that exists in WireGuardSetup
	// This handles all the logic including prompting, key generation, config writing, etc.
	return sm.wireguard.Run()
}

func (sm *StepManager) runNFS() error {
	// Check if already completed
	if sm.IsStepComplete("nfs-setup-complete") {
		sm.ui.Info("NFS setup already completed")
		rerun, err := sm.ui.PromptYesNo("Run again?", false)
		if err != nil || !rerun {
			return nil
		}
	}

	// Use the Run method that exists in NFSConfigurator
	return sm.nfs.Run()
}

func (sm *StepManager) runContainer() error {
	// Check if already completed
	if sm.IsStepComplete("container-setup-complete") {
		sm.ui.Info("Container setup already completed")
		rerun, err := sm.ui.PromptYesNo("Run again?", false)
		if err != nil || !rerun {
			return nil
		}
	}

	// Use the Run method that exists in ContainerSetup
	return sm.container.Run()
}

func (sm *StepManager) runDeployment() error {
	// Check if already completed
	if sm.IsStepComplete("service-deployment-complete") {
		sm.ui.Info("Service deployment already completed")
		rerun, err := sm.ui.PromptYesNo("Run again?", false)
		if err != nil || !rerun {
			return nil
		}
	}

	// Use the Run method that exists in Deployment
	return sm.deployment.Run()
}

// RunAll runs all setup steps in order
func (sm *StepManager) RunAll(skipWireGuard bool) error {
	steps := []string{"preflight", "user", "directory"}

	if !skipWireGuard {
		steps = append(steps, "wireguard")
	}

	steps = append(steps, "nfs", "container", "deployment")

	for _, step := range steps {
		if err := sm.RunStep(step); err != nil {
			return fmt.Errorf("step %s failed: %w", step, err)
		}
	}

	sm.ui.Success("All steps completed successfully!")
	return nil
}
