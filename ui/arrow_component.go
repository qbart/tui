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

func (a ArrowComponent) RenderJunction(left, right, up, down bool, active bool) string {
	return a.renderJunctionInternal(left, right, up, down, active, 0, 0)
}

func (a ArrowComponent) RenderJunctionMarked(left, right, up, down bool, active bool) string {
	return a.RenderJunctionWithMarker(left, right, up, down, active, '*')
}

func (a ArrowComponent) RenderJunctionWithMarker(left, right, up, down bool, active bool, marker rune) string {
	return a.RenderJunctionWithMarkers(left, right, up, down, active, 0, marker)
}

func (a ArrowComponent) RenderJunctionWithMarkers(left, right, up, down bool, active bool, leftMarker, rightMarker rune) string {
	return a.renderJunctionInternal(left, right, up, down, active, leftMarker, rightMarker)
}

func (a ArrowComponent) renderJunctionInternal(left, right, up, down bool, active bool, leftMarker, rightMarker rune) string {
	if !active {
		return strings.Repeat(" ", a.Width)
	}
	pattern := []rune(strings.Repeat(" ", a.Width))
	center := a.centerIndex()
	h := []rune(a.symbol())[0]

	if left {
		for i := 0; i < center; i++ {
			pattern[i] = h
		}
	}
	if right {
		for i := center + 1; i < len(pattern); i++ {
			pattern[i] = h
		}
	}
	pattern[center] = []rune(a.junctionSymbol(left, right, up, down))[0]
	a.applyMarkers(pattern, leftMarker, rightMarker)

	return lipgloss.NewStyle().
		Background(a.Background).
		Foreground(a.Color).
		Render(string(pattern))
}

func (a ArrowComponent) RenderHorizontal(active bool) string {
	return a.renderHorizontalInternal(active, 0, 0)
}

func (a ArrowComponent) RenderHorizontalMarked(active bool) string {
	return a.RenderHorizontalWithMarker(active, '*')
}

func (a ArrowComponent) RenderHorizontalWithMarker(active bool, marker rune) string {
	return a.RenderHorizontalWithMarkers(active, 0, marker)
}

func (a ArrowComponent) RenderHorizontalWithMarkers(active bool, leftMarker, rightMarker rune) string {
	return a.renderHorizontalInternal(active, leftMarker, rightMarker)
}

func (a ArrowComponent) RenderMarkerOnly(active bool, marker rune) string {
	if !active {
		return strings.Repeat(" ", a.Width)
	}
	pattern := []rune(strings.Repeat(" ", a.Width))
	if a.Width >= 3 {
		pattern[a.Width-3] = marker
	}
	return lipgloss.NewStyle().
		Background(a.Background).
		Foreground(a.Color).
		Render(string(pattern))
}

func (a ArrowComponent) renderHorizontalInternal(active bool, leftMarker, rightMarker rune) string {
	if !active {
		return strings.Repeat(" ", a.Width)
	}
	pattern := []rune(strings.Repeat(a.symbol(), a.Width))
	a.applyMarkers(pattern, leftMarker, rightMarker)
	return lipgloss.NewStyle().
		Background(a.Background).
		Foreground(a.Color).
		Render(string(pattern))
}

func (a ArrowComponent) applyMarkers(pattern []rune, leftMarker, rightMarker rune) {
	if leftMarker != 0 && a.Width >= 1 {
		// Immediately after the source step.
		pattern[0] = leftMarker
	}
	if rightMarker != 0 && a.Width >= 3 {
		// 3 chars before target node.
		pattern[a.Width-3] = rightMarker
	}
}

func (a ArrowComponent) RenderTeeRight(active bool) string {
	if !active {
		return strings.Repeat(" ", a.Width)
	}
	return a.RenderJunction(false, true, true, true, true)
}

func (a ArrowComponent) RenderVertical(active bool) string {
	if !active {
		return strings.Repeat(" ", a.Width)
	}
	return a.RenderJunction(false, false, true, true, true)
}

func (a ArrowComponent) RenderSplit(active bool) string {
	if !active {
		return strings.Repeat(" ", a.Width)
	}
	return a.RenderJunction(true, true, false, true, true)
}

func (a ArrowComponent) RenderCornerRight(active bool) string {
	if !active {
		return strings.Repeat(" ", a.Width)
	}
	return a.RenderJunction(false, true, true, false, true)
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

func (a ArrowComponent) splitUpSymbol() string {
	switch a.Type {
	case ArrowTypeDashed:
		return "┷"
	default:
		return "┻"
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

func (a ArrowComponent) junctionSymbol(left, right, up, down bool) string {
	switch {
	case left && right && up && down:
		if a.Type == ArrowTypeDashed {
			return "╂"
		}
		return "╋"
	case left && right && down:
		return a.splitSymbol()
	case left && right && up:
		return a.splitUpSymbol()
	case up && down && right:
		return a.teeRightSymbol()
	case up && down && left:
		if a.Type == ArrowTypeDashed {
			return "┨"
		}
		return "┫"
	case up && right:
		return a.cornerRightSymbol()
	case down && right:
		if a.Type == ArrowTypeDashed {
			return "┎"
		}
		return "┏"
	case up && left:
		if a.Type == ArrowTypeDashed {
			return "┚"
		}
		return "┛"
	case down && left:
		if a.Type == ArrowTypeDashed {
			return "┒"
		}
		return "┓"
	case up || down:
		return a.verticalSymbol()
	case left || right:
		return a.symbol()
	default:
		return " "
	}
}

func (a ArrowComponent) centerIndex() int {
	if a.Width <= 0 {
		return 0
	}
	return a.Width / 2
}
