package tui

import (
	"tui/core"
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

func stepStatusColors(status core.StepVisualStatus) (lipgloss.Color, lipgloss.Color) {
	switch status {
	case core.StatusGray:
		return theme.StatusGrayBg, theme.StatusGrayFg
	case core.StatusGreen:
		return theme.StatusGreenBg, theme.StatusGreenFg
	case core.StatusRed:
		return theme.StatusRedBg, theme.StatusRedFg
	case core.StatusYellow:
		return theme.StatusYellowBg, theme.StatusYellowFg
	case core.StatusBlue:
		return theme.StatusBlueBg, theme.StatusBlueFg
	case core.StatusOrange:
		return theme.StatusOrangeBg, theme.StatusOrangeFg
	case core.StatusPurple:
		return theme.StatusPurpleBg, theme.StatusPurpleFg
	case core.StatusSelected:
		return theme.SelectedBg, theme.SelectedFg
	default:
		return theme.StatusBlackBg, theme.StatusBlackFg
	}
}
