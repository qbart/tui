package tui

import (
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
)

type StepComponent struct {
	Step  stepView
	Width int
}

func NewStepComponent(step stepView, width int) StepComponent {
	return StepComponent{
		Step:  step,
		Width: width,
	}
}

func (c StepComponent) RenderBrick() string {
	width := c.Width
	if width <= 0 {
		width = c.PreferredWidth()
	}

	bg, fg := c.Colors()
	baseTextStyle := lipgloss.NewStyle().
		Background(bg).
		Foreground(fg)

	label := c.DisplayLabel()

	return baseTextStyle.
		Width(width).
		Render(label)
}

func (c StepComponent) PlainLabel() string {
	return "  " + c.Step.JobName + "  "
}

func (c StepComponent) DisplayLabel() string {
	label := c.PlainLabel()
	if c.Step.Spinner && c.Step.SpinChar != "" {
		r := []rune(label)
		if len(r) > 0 {
			spin := []rune(c.Step.SpinChar)[0]
			r[len(r)-1] = spin
			label = string(r)
		}
	}
	return label
}

func (c StepComponent) PreferredWidth() int {
	// Text + 4 chars total padding (2 on each side).
	return utf8.RuneCountInString(c.Step.JobName) + 4
}

func (c StepComponent) Colors() (lipgloss.Color, lipgloss.Color) {
	return stepStatusColors(c.Step.Status)
}

func stepStatusColors(status StepVisualStatus) (lipgloss.Color, lipgloss.Color) {
	switch status {
	case StatusGray:
		return theme.StatusGrayBg, theme.StatusGrayFg
	case StatusGreen:
		return theme.StatusGreenBg, theme.StatusGreenFg
	case StatusRed:
		return theme.StatusRedBg, theme.StatusRedFg
	case StatusYellow:
		return theme.StatusYellowBg, theme.StatusYellowFg
	case StatusBlue:
		return theme.StatusBlueBg, theme.StatusBlueFg
	case StatusOrange:
		return theme.StatusOrangeBg, theme.StatusOrangeFg
	case StatusPurple:
		return theme.StatusPurpleBg, theme.StatusPurpleFg
	case statusSelected:
		return theme.SelectedBg, theme.SelectedFg
	default:
		return theme.StatusBlackBg, theme.StatusBlackFg
	}
}
