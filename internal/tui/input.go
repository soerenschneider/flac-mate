package tui

import (
	"errors"
	"strconv"

	"github.com/charmbracelet/huh"
	"github.com/rs/zerolog/log"
)

const prompt = "> "

func SelectInput(title string, choices []string) string {
	var input string
	err := huh.NewSelect[string]().
		Title(title).
		Options(huh.NewOptions(choices...)...).
		Value(&input).
		Run()

	if err != nil {
		log.Fatal().Err(err).Msg("could not get choice")
	}

	return input
}

func ReadInput(title string, suggestions []string) string {
	return ReadInputWithValidation(title, suggestions, huh.ValidateNotEmpty())
}

func ReadInputWithValidation(title string, suggestions []string, validation func(string) error) string {
	var input string
	err := huh.NewInput().
		Title(title).
		Suggestions(suggestions).
		Prompt(prompt).
		Value(&input).
		Validate(validation).
		Run()

	if err != nil {
		log.Fatal().Err(err).Msg("could not read input")
	}

	return input
}

func ReadOtp(title string) string {
	var input string
	err := huh.NewInput().
		Title(title).
		Prompt(prompt).
		Value(&input).
		Validate(func(val string) error {
			_, err := strconv.Atoi(val)
			if err != nil {
				return errors.New("only digits allowed")
			}

			if len(val) < 6 {
				return errors.New("needs at least 6 digits")
			}
			return nil
		}).
		Run()

	if err != nil {
		log.Fatal().Err(err).Msg("could not read input")
	}

	return input
}

func ReadSensitiveInput(title string) string {
	var input string
	err := huh.NewInput().
		Title(title).
		Prompt("> ").
		EchoMode(huh.EchoModePassword).
		Value(&input).
		Validate(huh.ValidateNotEmpty()).
		Run()

	if err != nil {
		log.Fatal().Err(err).Msg("could not read input")
	}

	return input
}
