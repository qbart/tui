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

	baseTextStyle := lipgloss.NewStyle().
		Background(theme.StepBackground).
		Foreground(theme.StepForeground)

	iconPart := ""
	if c.Step.Icon != "" {
		iconPart = c.Step.Icon + " "
	}
	label := " " + iconPart + c.Step.JobName + " "

	return baseTextStyle.
		Width(width).
		Render(label)
}

func (c StepComponent) PreferredWidth() int {
	iconPart := ""
	if c.Step.Icon != "" {
		iconPart = c.Step.Icon + " "
	}
	// Text + 2 chars total padding.
	return utf8.RuneCountInString(iconPart+c.Step.JobName) + 2
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
