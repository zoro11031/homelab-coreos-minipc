package ui

import (
	"fmt"

	"github.com/AlecAivazis/survey/v2"
)

// PromptYesNo prompts the user for a yes/no answer
func (u *UI) PromptYesNo(prompt string, defaultYes bool) (bool, error) {
	var result bool
	p := &survey.Confirm{
		Message: prompt,
		Default: defaultYes,
	}

	err := survey.AskOne(p, &result)
	return result, err
}

// PromptInput prompts the user for text input
func (u *UI) PromptInput(prompt, defaultValue string) (string, error) {
	var result string
	p := &survey.Input{
		Message: prompt,
		Default: defaultValue,
	}

	err := survey.AskOne(p, &result)
	return result, err
}

// PromptPassword prompts the user for password input (hidden)
func (u *UI) PromptPassword(prompt string) (string, error) {
	var result string
	p := &survey.Password{
		Message: prompt,
	}

	err := survey.AskOne(p, &result)
	return result, err
}

// PromptPasswordConfirm prompts for password with confirmation
func (u *UI) PromptPasswordConfirm(prompt string) (string, error) {
	for {
		password1, err := u.PromptPassword(prompt)
		if err != nil {
			return "", err
		}

		password2, err := u.PromptPassword("Confirm password")
		if err != nil {
			return "", err
		}

		if password1 == password2 {
			if password1 == "" {
				u.Error("Password cannot be empty")
				continue
			}
			return password1, nil
		}

		u.Error("Passwords do not match. Please try again.")
	}
}

// PromptSelect prompts the user to select from a list
func (u *UI) PromptSelect(prompt string, options []string) (int, error) {
	var selected string
	p := &survey.Select{
		Message: prompt,
		Options: options,
	}

	if err := survey.AskOne(p, &selected); err != nil {
		return -1, err
	}

	// Find the index of the selected option
	for i, opt := range options {
		if opt == selected {
			return i, nil
		}
	}

	return -1, fmt.Errorf("selected option not found")
}

// PromptMultiSelect prompts the user to select multiple items from a list
func (u *UI) PromptMultiSelect(prompt string, options []string) ([]int, error) {
	var selected []string
	p := &survey.MultiSelect{
		Message: prompt,
		Options: options,
	}

	if err := survey.AskOne(p, &selected); err != nil {
		return nil, err
	}

	// Find indices of selected options
	var indices []int
	for i, opt := range options {
		for _, sel := range selected {
			if opt == sel {
				indices = append(indices, i)
				break
			}
		}
	}

	return indices, nil
}

// PromptInputRequired prompts for required input (cannot be empty)
func (u *UI) PromptInputRequired(prompt string) (string, error) {
	var result string
	p := &survey.Input{
		Message: prompt,
	}

	validator := survey.Required
	err := survey.AskOne(p, &result, survey.WithValidator(validator))
	return result, err
}

// PromptInputWithValidation prompts with custom validation
func (u *UI) PromptInputWithValidation(prompt, defaultValue string, validator survey.Validator) (string, error) {
	var result string
	p := &survey.Input{
		Message: prompt,
		Default: defaultValue,
	}

	err := survey.AskOne(p, &result, survey.WithValidator(validator))
	return result, err
}
