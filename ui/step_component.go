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

	bg, fg := stepStatusColors(c.Step.Status)
	baseTextStyle := lipgloss.NewStyle().
		Background(bg).
		Foreground(fg)

	iconPart := ""
	if c.Step.Icon != "" {
		iconPart = c.Step.Icon + " "
	}
	label := "  " + iconPart + c.Step.JobName + "  "
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

func (c StepComponent) PreferredWidth() int {
	iconPart := ""
	if c.Step.Icon != "" {
		iconPart = c.Step.Icon + " "
	}
	// Text + 4 chars total padding (2 on each side).
	return utf8.RuneCountInString(iconPart+c.Step.JobName) + 4
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
