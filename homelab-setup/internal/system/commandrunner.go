package system

import "os/exec"

// CommandRunner defines an interface for running system commands.
type CommandRunner interface {
	Run(name string, args ...string) (string, error)
}

// ExecCommandRunner executes commands using the local shell.
type ExecCommandRunner struct{}

// NewCommandRunner returns a default command runner implementation.
func NewCommandRunner() CommandRunner {
	return &ExecCommandRunner{}
}

// Run executes a command and returns its combined output.
func (r *ExecCommandRunner) Run(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}
