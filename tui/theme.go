package tui

import "github.com/charmbracelet/lipgloss"

var theme = struct {
	ContentBackground  lipgloss.Color
	ContentForeground  lipgloss.Color
	StatusBlackBg      lipgloss.Color
	StatusBlackFg      lipgloss.Color
	StatusGrayBg       lipgloss.Color
	StatusGrayFg       lipgloss.Color
	StatusGreenBg      lipgloss.Color
	StatusGreenFg      lipgloss.Color
	StatusRedBg        lipgloss.Color
	StatusRedFg        lipgloss.Color
	StatusYellowBg     lipgloss.Color
	StatusYellowFg     lipgloss.Color
	StatusBlueBg       lipgloss.Color
	StatusBlueFg       lipgloss.Color
	StatusPurpleBg     lipgloss.Color
	StatusPurpleFg     lipgloss.Color
	SelectedBg         lipgloss.Color
	SelectedFg         lipgloss.Color
	ArrowColor         lipgloss.Color
	ArrowSelectedColor lipgloss.Color
}{
	ContentBackground:  lipgloss.Color("#151515"),
	ContentForeground:  lipgloss.Color("15"),
	StatusBlackBg:      lipgloss.Color("#000000"),
	StatusBlackFg:      lipgloss.Color("#ffffff"),
	StatusGrayBg:       lipgloss.Color("#4b5563"),
	StatusGrayFg:       lipgloss.Color("#ffffff"),
	StatusGreenBg:      lipgloss.Color("#16a34a"),
	StatusGreenFg:      lipgloss.Color("#ffffff"),
	StatusRedBg:        lipgloss.Color("#dc2626"),
	StatusRedFg:        lipgloss.Color("#ffffff"),
	StatusYellowBg:     lipgloss.Color("#facc15"),
	StatusYellowFg:     lipgloss.Color("#000000"),
	StatusBlueBg:       lipgloss.Color("#2563eb"),
	StatusBlueFg:       lipgloss.Color("#ffffff"),
	StatusPurpleBg:     lipgloss.Color("#7c3aed"),
	StatusPurpleFg:     lipgloss.Color("#ffffff"),
	SelectedBg:         lipgloss.Color("#ffffff"),
	SelectedFg:         lipgloss.Color("#000000"),
	ArrowColor:         lipgloss.Color("#333333"),
	ArrowSelectedColor: lipgloss.Color("#ffffff"),
}
