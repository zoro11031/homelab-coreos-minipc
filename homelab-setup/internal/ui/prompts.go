//go:build !lint
// +build !lint

// Package ui provides interactive terminal UI components for the homelab setup
// tool, including prompts (input, yes/no, select, multi-select, password),
// formatted output (headers, steps, success/error messages), and progress indicators.
// Supports both interactive and non-interactive modes for automation.
package ui

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"golang.org/x/term"
)

// InputValidator represents a simple string validation function used by prompts.
type InputValidator func(string) error

func (u *UI) stdinReader() *bufio.Reader {
	return bufio.NewReader(os.Stdin)
}

func (u *UI) promptLine(message string) (string, error) {
	fmt.Printf("%s ", message)
	line, err := u.stdinReader().ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

// PromptYesNo prompts the user for a yes/no answer.
func (u *UI) PromptYesNo(prompt string, defaultYes bool) (bool, error) {
	if u.nonInteractive {
		u.Infof("[Non-interactive] %s -> %v (default)", prompt, defaultYes)
		return defaultYes, nil
	}

	defaultHint := "Y/n"
	if !defaultYes {
		defaultHint = "y/N"
	}

	for {
		answer, err := u.promptLine(fmt.Sprintf("%s [%s]", prompt, defaultHint))
		if err != nil {
			return false, err
		}
		answer = strings.ToLower(strings.TrimSpace(answer))
		if answer == "" {
			return defaultYes, nil
		}
		if answer == "y" || answer == "yes" {
			return true, nil
		}
		if answer == "n" || answer == "no" {
			return false, nil
		}
		u.Warning("Please enter y or n")
	}
}

// PromptInput prompts the user for text input.
func (u *UI) PromptInput(prompt, defaultValue string) (string, error) {
	if u.nonInteractive {
		if defaultValue == "" {
			return "", fmt.Errorf("non-interactive mode requires a default value for: %s", prompt)
		}
		u.Infof("[Non-interactive] %s -> %s (default)", prompt, defaultValue)
		return defaultValue, nil
	}

	message := prompt
	if defaultValue != "" {
		message = fmt.Sprintf("%s (default: %s)", prompt, defaultValue)
	}
	answer, err := u.promptLine(message)
	if err != nil {
		return "", err
	}
	if answer == "" {
		return defaultValue, nil
	}
	return answer, nil
}

// PromptPassword prompts the user for password input (hidden).
func (u *UI) PromptPassword(prompt string) (string, error) {
	if u.nonInteractive {
		return "", fmt.Errorf("non-interactive mode does not support password prompts: %s", prompt)
	}

	fmt.Printf("%s ", prompt)
	bytes, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
	return string(bytes), err
}

// PromptPasswordConfirm prompts for password with confirmation.
func (u *UI) PromptPasswordConfirm(prompt string) (string, error) {
	for {
		first, err := u.PromptPassword(prompt)
		if err != nil {
			return "", err
		}
		if strings.TrimSpace(first) == "" {
			u.Error("Password cannot be empty")
			continue
		}
		second, err := u.PromptPassword("Confirm password")
		if err != nil {
			return "", err
		}
		if first == second {
			return first, nil
		}
		u.Error("Passwords do not match. Please try again.")
	}
}

// PromptSelect prompts the user to select from a list.
func (u *UI) PromptSelect(prompt string, options []string) (int, error) {
	if len(options) == 0 {
		return -1, fmt.Errorf("no options provided for prompt: %s", prompt)
	}
	if u.nonInteractive {
		u.Infof("[Non-interactive] %s -> %s (first option)", prompt, options[0])
		return 0, nil
	}

	u.Print(prompt)
	for i, opt := range options {
		u.Printf("  %d) %s\n", i+1, opt)
	}

	for {
		answer, err := u.promptLine("Enter the number of your choice")
		if err != nil {
			return -1, err
		}
		idx, err := strconv.Atoi(strings.TrimSpace(answer))
		if err != nil || idx < 1 || idx > len(options) {
			u.Warning("Please enter a number from the list")
			continue
		}
		return idx - 1, nil
	}
}

// PromptMultiSelect prompts the user to select multiple items from a list.
func (u *UI) PromptMultiSelect(prompt string, options []string) ([]int, error) {
	if len(options) == 0 {
		return []int{}, nil
	}
	if u.nonInteractive {
		indices := make([]int, len(options))
		for i := range options {
			indices[i] = i
		}
		u.Infof("[Non-interactive] %s -> all options (%d)", prompt, len(options))
		return indices, nil
	}

	u.Print(prompt)
	for i, opt := range options {
		u.Printf("  %d) %s\n", i+1, opt)
	}
	line, err := u.promptLine("Enter comma-separated numbers (leave blank for none, * for all)")
	if err != nil {
		return nil, err
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return []int{}, nil
	}
	if line == "*" {
		indices := make([]int, len(options))
		for i := range options {
			indices[i] = i
		}
		return indices, nil
	}

	parts := strings.Split(line, ",")
	indices := make([]int, 0, len(parts))
	seen := make(map[int]struct{})
	for _, part := range parts {
		part = strings.TrimSpace(part)
		idx, err := strconv.Atoi(part)
		if err != nil || idx < 1 || idx > len(options) {
			return nil, fmt.Errorf("invalid selection: %s", part)
		}
		idx--
		if _, exists := seen[idx]; exists {
			continue
		}
		seen[idx] = struct{}{}
		indices = append(indices, idx)
	}
	return indices, nil
}

// PromptInputRequired prompts for input that cannot be empty.
func (u *UI) PromptInputRequired(prompt string) (string, error) {
	if u.nonInteractive {
		return "", fmt.Errorf("non-interactive mode requires a default value for required input: %s", prompt)
	}
	for {
		value, err := u.PromptInput(prompt, "")
		if err != nil {
			return "", err
		}
		if strings.TrimSpace(value) == "" {
			u.Warning("Value cannot be empty")
			continue
		}
		return value, nil
	}
}

// PromptInputWithValidation prompts with custom validation logic.
func (u *UI) PromptInputWithValidation(prompt, defaultValue string, validator InputValidator) (string, error) {
	if validator == nil {
		return u.PromptInput(prompt, defaultValue)
	}
	for {
		value, err := u.PromptInput(prompt, defaultValue)
		if err != nil {
			return "", err
		}
		if err := validator(value); err != nil {
			u.Warningf("Invalid input: %v", err)
			continue
		}
		return value, nil
	}
}
