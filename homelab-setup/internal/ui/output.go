package ui

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/fatih/color"
)

// UI provides user interface methods
type UI struct {
	output         io.Writer
	nonInteractive bool // If true, don't prompt user for input
	// Color functions
	colorInfo    *color.Color
	colorSuccess *color.Color
	colorWarning *color.Color
	colorError   *color.Color
	colorBold    *color.Color
	colorCyan    *color.Color
}

// New creates a new UI instance
func New() *UI {
	return &UI{
		output:         os.Stderr,
		nonInteractive: false,
		colorInfo:      color.New(color.FgBlue),
		colorSuccess:   color.New(color.FgGreen),
		colorWarning:   color.New(color.FgYellow),
		colorError:     color.New(color.FgRed),
		colorBold:      color.New(color.Bold),
		colorCyan:      color.New(color.FgCyan, color.Bold),
	}
}

// SetNonInteractive enables or disables non-interactive mode
func (u *UI) SetNonInteractive(enabled bool) {
	u.nonInteractive = enabled
}

// IsNonInteractive returns true if non-interactive mode is enabled
func (u *UI) IsNonInteractive() bool {
	return u.nonInteractive
}

// NewWithWriter creates a UI with custom output writer (useful for testing)
func NewWithWriter(w io.Writer) *UI {
	ui := New()
	ui.output = w
	return ui
}

// Info prints an info message
func (u *UI) Info(msg string) {
	u.colorInfo.Fprintf(u.output, "[INFO] %s\n", msg)
}

// Infof prints a formatted info message
func (u *UI) Infof(format string, args ...interface{}) {
	u.Info(fmt.Sprintf(format, args...))
}

// Success prints a success message
func (u *UI) Success(msg string) {
	u.colorSuccess.Fprintf(u.output, "[✓] %s\n", msg)
}

// Successf prints a formatted success message
func (u *UI) Successf(format string, args ...interface{}) {
	u.Success(fmt.Sprintf(format, args...))
}

// Warning prints a warning message
func (u *UI) Warning(msg string) {
	u.colorWarning.Fprintf(u.output, "[WARNING] %s\n", msg)
}

// Warningf prints a formatted warning message
func (u *UI) Warningf(format string, args ...interface{}) {
	u.Warning(fmt.Sprintf(format, args...))
}

// Error prints an error message
func (u *UI) Error(msg string) {
	u.colorError.Fprintf(u.output, "[ERROR] %s\n", msg)
}

// Errorf prints a formatted error message
func (u *UI) Errorf(format string, args ...interface{}) {
	u.Error(fmt.Sprintf(format, args...))
}

// Step prints a step header
func (u *UI) Step(msg string) {
	fmt.Fprintln(u.output)
	u.colorCyan.Fprintf(u.output, "==> %s\n", msg)
	fmt.Fprintln(u.output)
}

// Header prints a header with a box
func (u *UI) Header(title string) {
	width := 70
	border := strings.Repeat("=", width)

	fmt.Fprintln(u.output)
	u.colorCyan.Fprintln(u.output, border)
	u.colorCyan.Fprintf(u.output, "  %s\n", title)
	u.colorCyan.Fprintln(u.output, border)
	fmt.Fprintln(u.output)
}

// Separator prints a separator line
func (u *UI) Separator() {
	u.colorCyan.Fprintln(u.output, strings.Repeat("-", 70))
}

// Print prints a plain message without formatting
func (u *UI) Print(msg string) {
	fmt.Fprintln(u.output, msg)
}

// Printf prints a formatted plain message
func (u *UI) Printf(format string, args ...interface{}) {
	fmt.Fprintf(u.output, format+"\n", args...)
}

// Bold prints bold text
func (u *UI) Bold(msg string) {
	u.colorBold.Fprintln(u.output, msg)
}
