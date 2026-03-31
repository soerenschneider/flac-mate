package tui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

// Icons (you can tweak or remove)
const (
	iconSuccess = "✓"
	iconError   = "✗"
	iconWarn    = "⚠"
	iconInfo    = "ℹ"
)

// Styles
var (
	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("10")). // green
			Bold(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("9")). // red
			Bold(true)

	warnStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("11")). // yellow
			Bold(true)

	infoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("12")) // blue

	mutedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8")) // gray
)

func printMessage(style lipgloss.Style, icon, msg string) {
	fmt.Println(style.Render(fmt.Sprintf("%s %s", icon, msg)))
}

func Success(msg string) {
	printMessage(successStyle, iconSuccess, msg)
}

func Error(msg string) {
	printMessage(errorStyle, iconError, msg)
}

func Warn(msg string) {
	printMessage(warnStyle, iconWarn, msg)
}

func Info(msg string) {
	printMessage(infoStyle, iconInfo, msg)
}

func Muted(msg string) {
	fmt.Println(mutedStyle.Render(msg))
}