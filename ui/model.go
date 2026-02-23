package ui

import (
	"fmt"
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
		{ID: "build-ui", JobName: "build ui", DependsOn: []core.StepID{"checkout"}},
		{ID: "test-postresql", JobName: "test postresql", DependsOn: []core.StepID{"build"}},
		{ID: "test-sqlite", JobName: "test sqlite", DependsOn: []core.StepID{"build"}},
		{ID: "test-duckdb", JobName: "test duckdb", DependsOn: []core.StepID{"build"}},
		{ID: "deploy", JobName: "deploy", DependsOn: []core.StepID{"test-postresql", "test-sqlite", "test-duckdb"}},
		{ID: "deploy-ui", JobName: "deploy ui", DependsOn: []core.StepID{"build-ui"}},
		{ID: "notify", JobName: "notify", DependsOn: []core.StepID{"deploy", "deploy-ui"}},
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
			"build-ui":       2 * time.Second,
			"test-postresql": 2 * time.Second,
			"test-sqlite":    2 * time.Second,
			"test-duckdb":    2 * time.Second,
			"deploy":         1 * time.Second,
			"deploy-ui":      1 * time.Second,
			"notify":         1 * time.Second,
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

	columnMetrics := buildColumnRenderMetrics(view.Columns)
	connectors := buildConnectorLayout(view)

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
			cellWidth := columnMetrics[col].MaxStepWidth
			cell := blankBrick(cellWidth)
			if step, ok := stepsByCell[col][row]; ok {
				cellWidth = NewStepComponent(step, 0).PreferredWidth()
				cell = NewStepComponent(step, 0).RenderBrick()
			}
			b.WriteString(cell)

			if col == len(view.Columns)-1 {
				continue
			}

			connectorArrow := arrow
			connectorArrow.Width = arrow.Width + (columnMetrics[col].MaxStepWidth - cellWidth)
			connector := connectorArrow.RenderHorizontal(false)
			if junction, ok := connectors.rowJunction(col, row); ok {
				connector = connectorArrow.RenderJunction(junction.Left, junction.Right, junction.Up, junction.Down, junction.active())
			}
			b.WriteString(gap)
			b.WriteString(connector)
			b.WriteString(gap)
		}
		rows = append(rows, b.String())
		if row < view.RowCount-1 {
			rows = append(rows, renderPipelineSpacerRow(row, columnMetrics, arrow, connectors))
		}
	}

	return rows
}

type columnRenderMetrics struct {
	StepCount    int
	MaxStepWidth int
}

func buildColumnRenderMetrics(columns [][]StepView) []columnRenderMetrics {
	metrics := make([]columnRenderMetrics, len(columns))
	for col := range columns {
		maxWidth := 0
		for _, step := range columns[col] {
			w := NewStepComponent(step, 0).PreferredWidth()
			if w > maxWidth {
				maxWidth = w
			}
		}
		metrics[col] = columnRenderMetrics{
			StepCount:    len(columns[col]),
			MaxStepWidth: maxWidth,
		}
	}
	return metrics
}

func renderPipelineSpacerRow(boundaryRow int, columnMetrics []columnRenderMetrics, arrow ArrowComponent, connectors connectorLayout) string {
	const gap = ""
	var b strings.Builder

	for col := 0; col < len(columnMetrics); col++ {
		b.WriteString(blankBrick(columnMetrics[col].MaxStepWidth))
		if col == len(columnMetrics)-1 {
			continue
		}
		b.WriteString(gap)
		if connectors.hasBoundaryVertical(col, boundaryRow) {
			b.WriteString(arrow.RenderVertical(true))
		} else {
			b.WriteString(arrow.RenderVertical(false))
		}
		b.WriteString(gap)
	}

	return b.String()
}

type connectorJunction struct {
	Left  bool
	Right bool
	Up    bool
	Down  bool
}

func (j connectorJunction) active() bool {
	return j.Left || j.Right || j.Up || j.Down
}

type connectorLayout struct {
	rowJunctions map[int]map[int]connectorJunction
	boundaries   map[int]map[int]bool
}

func (c connectorLayout) rowJunction(lane, row int) (connectorJunction, bool) {
	laneMap, ok := c.rowJunctions[lane]
	if !ok {
		return connectorJunction{}, false
	}
	j, ok := laneMap[row]
	return j, ok
}

func (c connectorLayout) hasBoundaryVertical(lane, boundaryRow int) bool {
	laneMap, ok := c.boundaries[lane]
	if !ok {
		return false
	}
	return laneMap[boundaryRow]
}

func buildConnectorLayout(view PipelineView) connectorLayout {
	layout := connectorLayout{
		rowJunctions: map[int]map[int]connectorJunction{},
		boundaries:   map[int]map[int]bool{},
	}

	stepsByID := map[string]StepView{}
	for _, col := range view.Columns {
		for _, step := range col {
			stepsByID[step.ID] = step
		}
	}

	for _, target := range stepsByID {
		targetPos, ok := view.Positions[target.ID]
		if !ok {
			continue
		}
		for _, depID := range target.DependsOn {
			sourcePos, ok := view.Positions[depID]
			if !ok {
				continue
			}
			if targetPos.Column <= sourcePos.Column {
				continue
			}
			addRoutedDependency(&layout, sourcePos, targetPos)
		}
	}

	return layout
}

func addRoutedDependency(layout *connectorLayout, source StepPositionView, target StepPositionView) {
	for lane := source.Column; lane < target.Column; lane++ {
		if lane == source.Column {
			addSourceLaneRoute(layout, lane, source.Row, target.Row)
			continue
		}
		addJunction(layout, lane, target.Row, true, true, false, false)
	}
}

func addSourceLaneRoute(layout *connectorLayout, lane, sourceRow, targetRow int) {
	switch {
	case sourceRow == targetRow:
		addJunction(layout, lane, sourceRow, true, true, false, false)
	case targetRow > sourceRow:
		addJunction(layout, lane, sourceRow, true, true, false, true)
		for boundary := sourceRow; boundary < targetRow; boundary++ {
			addBoundary(layout, lane, boundary)
		}
		addJunction(layout, lane, targetRow, false, true, true, false)
	default:
		addJunction(layout, lane, sourceRow, true, true, true, false)
		for boundary := targetRow; boundary < sourceRow; boundary++ {
			addBoundary(layout, lane, boundary)
		}
		addJunction(layout, lane, targetRow, false, true, false, true)
	}
}

func addJunction(layout *connectorLayout, lane, row int, left, right, up, down bool) {
	if layout.rowJunctions[lane] == nil {
		layout.rowJunctions[lane] = map[int]connectorJunction{}
	}
	current := layout.rowJunctions[lane][row]
	current.Left = current.Left || left
	current.Right = current.Right || right
	current.Up = current.Up || up
	current.Down = current.Down || down
	layout.rowJunctions[lane][row] = current
}

func addBoundary(layout *connectorLayout, lane, boundaryRow int) {
	if layout.boundaries[lane] == nil {
		layout.boundaries[lane] = map[int]bool{}
	}
	layout.boundaries[lane][boundaryRow] = true
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
