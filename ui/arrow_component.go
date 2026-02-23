package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type ArrowType string

const (
	ArrowTypeSolid  ArrowType = "Solid"
	ArrowTypeDashed ArrowType = "Dashed"
)

type ArrowComponent struct {
	Width      int
	Type       ArrowType
	Color      lipgloss.Color
	Background lipgloss.Color
}

func NewArrowComponent(width int, arrowType ArrowType, color lipgloss.Color, background lipgloss.Color) ArrowComponent {
	if width <= 0 {
		width = 5
	}
	if arrowType == "" {
		arrowType = ArrowTypeSolid
	}
	return ArrowComponent{
		Width:      width,
		Type:       arrowType,
		Color:      color,
		Background: background,
	}
}

func (a ArrowComponent) Render(active bool) string {
	return a.RenderHorizontal(active)
}

func (a ArrowComponent) RenderHorizontal(active bool) string {
	if !active {
		return strings.Repeat(" ", a.Width)
	}
	return lipgloss.NewStyle().
		Background(a.Background).
		Foreground(a.Color).
		Render(strings.Repeat(a.symbol(), a.Width))
}

func (a ArrowComponent) RenderTeeRight(active bool) string {
	if !active {
		return strings.Repeat(" ", a.Width)
	}
	return a.renderCenteredRight(a.teeRightSymbol())
}

func (a ArrowComponent) RenderVertical(active bool) string {
	if !active {
		return strings.Repeat(" ", a.Width)
	}
	return a.renderCentered(a.verticalSymbol())
}

func (a ArrowComponent) RenderSplit(active bool) string {
	if !active {
		return strings.Repeat(" ", a.Width)
	}
	pattern := []rune(strings.Repeat(a.symbol(), a.Width))
	pattern[a.centerIndex()] = []rune(a.splitSymbol())[0]
	return lipgloss.NewStyle().
		Background(a.Background).
		Foreground(a.Color).
		Render(string(pattern))
}

func (a ArrowComponent) RenderCornerRight(active bool) string {
	if !active {
		return strings.Repeat(" ", a.Width)
	}
	return a.renderCenteredRight(a.cornerRightSymbol())
}

func (a ArrowComponent) symbol() string {
	switch a.Type {
	case ArrowTypeDashed:
		return "┅"
	default:
		return "━"
	}
}

func (a ArrowComponent) verticalSymbol() string {
	switch a.Type {
	case ArrowTypeDashed:
		return "┇"
	default:
		return "┃"
	}
}

func (a ArrowComponent) teeRightSymbol() string {
	switch a.Type {
	case ArrowTypeDashed:
		return "┠"
	default:
		return "┣"
	}
}

func (a ArrowComponent) splitSymbol() string {
	switch a.Type {
	case ArrowTypeDashed:
		return "┯"
	default:
		return "┳"
	}
}

func (a ArrowComponent) cornerRightSymbol() string {
	switch a.Type {
	case ArrowTypeDashed:
		return "┖"
	default:
		return "┗"
	}
}

func (a ArrowComponent) centerIndex() int {
	if a.Width <= 0 {
		return 0
	}
	return a.Width / 2
}

func (a ArrowComponent) renderCentered(symbol string) string {
	pattern := []rune(strings.Repeat(" ", a.Width))
	pattern[a.centerIndex()] = []rune(symbol)[0]
	return lipgloss.NewStyle().
		Background(a.Background).
		Foreground(a.Color).
		Render(string(pattern))
}

func (a ArrowComponent) renderCenteredRight(symbol string) string {
	pattern := []rune(strings.Repeat(" ", a.Width))
	center := a.centerIndex()
	pattern[center] = []rune(symbol)[0]
	for i := center + 1; i < len(pattern); i++ {
		pattern[i] = []rune(a.symbol())[0]
	}
	return lipgloss.NewStyle().
		Background(a.Background).
		Foreground(a.Color).
		Render(string(pattern))
}
