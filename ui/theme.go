package ui

import "github.com/charmbracelet/lipgloss"

var theme = struct {
	ContentBackground lipgloss.Color
	ContentForeground lipgloss.Color
	StepBackground    lipgloss.Color
	StepForeground    lipgloss.Color
	FooterBackground  lipgloss.Color
	FooterForeground  lipgloss.Color
	BrickBorder       lipgloss.Color
	ArrowColor        lipgloss.Color
}{
	ContentBackground: lipgloss.Color("0"),
	ContentForeground: lipgloss.Color("15"),
	StepBackground:    lipgloss.Color("#000000"),
	StepForeground:    lipgloss.Color("#ffffff"),
	FooterBackground:  lipgloss.Color("232"),
	FooterForeground:  lipgloss.Color("15"),
	BrickBorder:       lipgloss.Color("240"),
	ArrowColor:        lipgloss.Color("245"),
}
