package ui

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
	FooterBackground   lipgloss.Color
	FooterForeground   lipgloss.Color
	BrickBorder        lipgloss.Color
	ArrowColor         lipgloss.Color
	ArrowSelectedColor lipgloss.Color
}{
	ContentBackground:  lipgloss.Color("0"),
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
	FooterBackground:   lipgloss.Color("232"),
	FooterForeground:   lipgloss.Color("15"),
	BrickBorder:        lipgloss.Color("240"),
	ArrowColor:         lipgloss.Color("#9ca3af"),
	ArrowSelectedColor: lipgloss.Color("#60a5fa"),
}
