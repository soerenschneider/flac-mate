package tui

import "github.com/charmbracelet/huh"

func Confirm(prompt string) (bool, error) {
	var answer bool
	err := huh.NewConfirm().
		Title(prompt).
		Value(&answer).
		Run()

	return answer, err
}
