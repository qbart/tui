package ui

import (
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
)

type StepComponent struct {
	Step  StepView
	Width int
}

func NewStepComponent(step StepView, width int) StepComponent {
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

	label := c.PlainLabel()
	if c.Step.Spinner && c.Step.SpinChar != "" {
		r := []rune(label)
		if len(r) > 0 {
			spin := []rune(c.Step.SpinChar)[0]
			r[len(r)-1] = spin
			label = string(r)
		}
	}

	return baseTextStyle.
		Width(width).
		Render(label)
}

func (c StepComponent) PlainLabel() string {
	return "  " + c.Step.JobName + "  "
}

func (c StepComponent) PreferredWidth() int {
	// Text + 4 chars total padding (2 on each side).
	return utf8.RuneCountInString(c.Step.JobName) + 4
}

func (c StepComponent) Colors() (lipgloss.Color, lipgloss.Color) {
	return stepStatusColors(c.Step.Status)
}

func (c StepComponent) RenderConnectorTo(target StepView, arrow ArrowComponent) string {
	return arrow.RenderTeeRight(dependsOn(target, c.Step.ID))
}

func dependsOn(step StepView, depID string) bool {
	for _, dep := range step.DependsOn {
		if dep == depID {
			return true
		}
	}
	return false
}

func blankBrick(width int) string {
	return strings.Repeat(" ", max(width, 0))
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
	default:
		return theme.StatusBlackBg, theme.StatusBlackFg
	}
}
