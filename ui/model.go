package ui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	core "hestia/hestia"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type tickMsg time.Time

type Model struct {
	width         int
	height        int
	spec          core.PipelineSpec
	run           core.PipelineRun
	stepDurations map[core.StepID]time.Duration
}

func NewModel() Model {
	spec := core.NewPipelineSpec("sample-cicd", []core.StepSpec{
		{ID: "checkout", JobName: "checkout"},
		{ID: "build", JobName: "build", DependsOn: []core.StepID{"checkout"}},
		{ID: "test-postresql", JobName: "test postresql", DependsOn: []core.StepID{"build"}},
		{ID: "test-sqlite", JobName: "test sqlite", DependsOn: []core.StepID{"build"}},
		{ID: "test-duckdb", JobName: "test duckdb", DependsOn: []core.StepID{"build"}},
		{ID: "deploy", JobName: "deploy", DependsOn: []core.StepID{"test-postresql", "test-sqlite", "test-duckdb"}},
	})

	now := time.Now()
	run, err := core.NewPipelineRun(spec, "run-1", now)
	if err != nil {
		return Model{
			spec: spec,
			run: core.PipelineRun{
				ID:        "run-1",
				SpecID:    spec.ID,
				Status:    core.PipelineRunStatusFailed,
				StartedAt: now,
				StepRuns:  map[core.StepID]*core.StepRun{},
			},
			stepDurations: map[core.StepID]time.Duration{},
		}
	}

	return Model{
		spec: spec,
		run:  run,
		stepDurations: map[core.StepID]time.Duration{
			"checkout":       1 * time.Second,
			"build":          2 * time.Second,
			"test-postresql": 2 * time.Second,
			"test-sqlite":    2 * time.Second,
			"test-duckdb":    2 * time.Second,
			"deploy":         1 * time.Second,
		},
	}
}

func (m Model) Init() tea.Cmd {
	return tickCmd()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tickMsg:
		m.advance(time.Time(msg))
		return m, tickCmd()
	}

	return m, nil
}

func (m Model) View() string {
	if m.width <= 0 || m.height <= 0 {
		return "Loading..."
	}

	contentHeight := max(m.height-1, 0)
	content := renderContent(m.width, contentHeight, m.spec, m.run)
	footer := renderFooter(m.width, fmt.Sprintf("run:%s | q to quit", m.run.Status))

	if content == "" {
		return footer
	}

	return content + "\n" + footer
}

func (m *Model) advance(at time.Time) {
	if m.run.IsTerminal() {
		return
	}

	if stepID, ok := m.run.RunningStepID(); ok {
		stepRun := m.run.StepRuns[stepID]
		if stepRun != nil && stepRun.StartedAt != nil {
			required := m.stepDurations[stepID]
			if required == 0 {
				required = time.Second
			}
			if at.Sub(*stepRun.StartedAt) >= required {
				_ = m.run.CompleteStep(stepID, at, true, 0, "")
				m.run.RefreshStatus(m.spec, at)
			}
		}
		return
	}

	ready := m.run.ReadySteps(m.spec)
	if len(ready) == 0 {
		m.run.RefreshStatus(m.spec, at)
		return
	}

	_ = m.run.StartStep(ready[0], at)
	m.run.RefreshStatus(m.spec, at)
}

func tickCmd() tea.Cmd {
	return tea.Tick(250*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func renderContent(width, height int, spec core.PipelineSpec, run core.PipelineRun) string {
	if height <= 0 {
		return ""
	}

	const topPadding = 1
	innerWidth := max(width, 0)
	hPadding := 0
	if width >= 2 {
		innerWidth = width - 2
		hPadding = 1
	}
	innerHeight := max(height-topPadding, 0)

	view, err := BuildPipelineView(spec, run)
	if err != nil {
		return lipgloss.NewStyle().
			Background(theme.ContentBackground).
			Foreground(theme.ContentForeground).
			Width(innerWidth).
			Height(innerHeight).
			Padding(topPadding, hPadding, 0, hPadding).
			Render(fmt.Sprintf("invalid pipeline: %v", err))
	}

	lines := make([]string, 0, innerHeight)
	lines = append(lines, renderPipelineGraph(view)...)

	for len(lines) > innerHeight {
		lines = lines[:innerHeight]
	}

	for len(lines) < innerHeight {
		lines = append(lines, "")
	}

	return lipgloss.NewStyle().
		Background(theme.ContentBackground).
		Foreground(theme.ContentForeground).
		Width(innerWidth).
		Height(innerHeight).
		Padding(topPadding, hPadding, 0, hPadding).
		Render(strings.Join(lines, "\n"))
}

func renderPipelineGraph(view PipelineView) []string {
	if len(view.Columns) == 0 {
		return []string{"(no steps)"}
	}

	const gap = ""
	arrow := NewArrowComponent(5, ArrowTypeSolid, theme.ArrowColor, theme.ContentBackground)
	if view.RowCount == 0 {
		return []string{"(no steps)"}
	}

	columnWidths := make([]int, len(view.Columns))
	for col := range view.Columns {
		maxWidth := 0
		for _, step := range view.Columns[col] {
			w := NewStepComponent(step, 0).PreferredWidth()
			if w > maxWidth {
				maxWidth = w
			}
		}
		columnWidths[col] = maxWidth
	}

	stepsByCell := map[int]map[int]StepView{}
	for _, colSteps := range view.Columns {
		for _, step := range colSteps {
			pos := view.Positions[step.ID]
			if stepsByCell[pos.Column] == nil {
				stepsByCell[pos.Column] = map[int]StepView{}
			}
			stepsByCell[pos.Column][pos.Row] = step
		}
	}

	rows := make([]string, 0, view.RowCount)
	for row := 0; row < view.RowCount; row++ {
		var b strings.Builder
		for col := 0; col < len(view.Columns); col++ {
			cell := blankBrick(columnWidths[col])
			var source StepView
			var hasSource bool
			if step, ok := stepsByCell[col][row]; ok {
				source = step
				hasSource = true
				cell = NewStepComponent(source, 0).RenderBrick()
			}
			b.WriteString(cell)

			if col == len(view.Columns)-1 {
				continue
			}

			connector := arrow.RenderHorizontal(false)
			if hasSource {
				if c, ok := sourceRowConnector(source, view.Columns[col+1], view.Positions, arrow); ok {
					connector = c
				}
			} else {
				if c, ok := targetBranchRowConnector(row, view.Columns[col], view.Columns[col+1], view.Positions, arrow); ok {
					connector = c
				}
			}
			b.WriteString(gap)
			b.WriteString(connector)
			b.WriteString(gap)
		}
		rows = append(rows, b.String())
		if row < view.RowCount-1 {
			rows = append(rows, renderPipelineSpacerRow(row, view, columnWidths, arrow))
		}
	}

	return rows
}

func renderPipelineSpacerRow(boundaryRow int, view PipelineView, columnWidths []int, arrow ArrowComponent) string {
	const gap = ""
	var b strings.Builder

	for col := 0; col < len(view.Columns); col++ {
		b.WriteString(blankBrick(columnWidths[col]))
		if col == len(view.Columns)-1 {
			continue
		}
		b.WriteString(gap)
		if c, ok := spacerBoundaryConnector(boundaryRow, view.Columns[col], view.Columns[col+1], view.Positions, arrow); ok {
			b.WriteString(c)
		} else {
			b.WriteString(arrow.RenderVertical(false))
		}
		b.WriteString(gap)
	}

	return b.String()
}

func sourceRowConnector(source StepView, nextCol []StepView, positions map[string]StepPositionView, arrow ArrowComponent) (string, bool) {
	targetRows := dependentRowsForSource(source, nextCol, positions)
	if len(targetRows) == 0 {
		return "", false
	}
	if len(targetRows) == 1 {
		return arrow.RenderHorizontal(true), true
	}
	return arrow.RenderSplit(true), true
}

func targetBranchRowConnector(row int, prevCol []StepView, nextCol []StepView, positions map[string]StepPositionView, arrow ArrowComponent) (string, bool) {
	for _, source := range prevCol {
		targetRows := dependentRowsForSource(source, nextCol, positions)
		if len(targetRows) <= 1 {
			continue
		}
		for i, tr := range targetRows {
			if tr != row {
				continue
			}
			if i == len(targetRows)-1 {
				return arrow.RenderCornerRight(true), true
			}
			return arrow.RenderTeeRight(true), true
		}
	}
	return "", false
}

func spacerBoundaryConnector(boundaryRow int, prevCol []StepView, nextCol []StepView, positions map[string]StepPositionView, arrow ArrowComponent) (string, bool) {
	for _, source := range prevCol {
		sourcePos, ok := positions[source.ID]
		if !ok {
			continue
		}
		targetRows := dependentRowsForSource(source, nextCol, positions)
		if len(targetRows) <= 1 {
			continue
		}
		last := targetRows[len(targetRows)-1]
		if boundaryRow >= sourcePos.Row && boundaryRow < last {
			return arrow.RenderVertical(true), true
		}
	}
	return "", false
}

func dependentRowsForSource(source StepView, nextCol []StepView, positions map[string]StepPositionView) []int {
	rows := make([]int, 0)
	for _, target := range nextCol {
		if !dependsOn(target, source.ID) {
			continue
		}
		pos, ok := positions[target.ID]
		if !ok {
			continue
		}
		rows = append(rows, pos.Row)
	}
	sort.Ints(rows)
	return rows
}

func renderFooter(width int, text string) string {
	footer := text
	if width > 0 {
		footer = fitToWidth(text, width)
	}

	return lipgloss.NewStyle().
		Background(theme.FooterBackground).
		Foreground(theme.FooterForeground).
		Render(footer)
}

func fitToWidth(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if len(s) > width {
		return s[:width]
	}
	if len(s) < width {
		return s + strings.Repeat(" ", width-len(s))
	}
	return s
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
