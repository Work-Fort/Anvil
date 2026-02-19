// SPDX-License-Identifier: Apache-2.0
package ui

import (
	"fmt"

	"github.com/charmbracelet/huh"
)

// Confirm shows a yes/no confirmation dialog using huh
func Confirm(prompt string) (bool, error) {
	var confirmed bool

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title(prompt).
				Value(&confirmed),
		),
	)

	err := form.Run()
	if err != nil {
		return false, err
	}

	return confirmed, nil
}

// TypedConfirm shows a confirmation that requires typing a specific phrase
func TypedConfirm(prompt, expectedInput string) (bool, error) {
	var input string

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title(prompt).
				Placeholder(expectedInput).
				Value(&input).
				Validate(func(s string) error {
					if s != expectedInput {
						return fmt.Errorf("must type exactly: %s", expectedInput)
					}
					return nil
				}),
		),
	)

	err := form.Run()
	if err != nil {
		return false, err
	}

	return input == expectedInput, nil
}
