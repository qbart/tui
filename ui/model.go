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
	spinnerFrame  int
}

func NewModel() Model {
	spec := core.NewPipelineSpec("sample-cicd", []core.StepSpec{
		{ID: "checkout", JobName: "checkout"},
		{ID: "build", JobName: "build", DependsOn: []core.StepID{"checkout"}},
		{ID: "build-ui", JobName: "build super ui", DependsOn: []core.StepID{"checkout"}},
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
		frameCount := spinnerFrameCount()
		if frameCount > 0 {
			m.spinnerFrame = (m.spinnerFrame + 1) % frameCount
		}
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
	content := renderContent(m.width, contentHeight, m.spec, m.run, m.spinnerFrame)
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

func renderContent(width, height int, spec core.PipelineSpec, run core.PipelineRun, spinnerFrame int) string {
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

	view, err := BuildPipelineView(spec, run, spinnerFrame)
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

	if view.RowCount == 0 {
		return []string{"(no steps)"}
	}

	columnMetrics := buildColumnRenderMetrics(view.Columns)
	const gapWidth = 5

	columnStarts := make([]int, len(view.Columns))
	totalWidth := 0
	for col := 0; col < len(view.Columns); col++ {
		columnStarts[col] = totalWidth
		totalWidth += columnMetrics[col].MaxStepWidth
		if col < len(view.Columns)-1 {
			totalWidth += gapWidth
		}
	}
	totalRows := view.RowCount*2 - 1
	canvas := make([][]rune, totalRows)
	for y := 0; y < totalRows; y++ {
		canvas[y] = []rune(strings.Repeat(" ", totalWidth))
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

	// Pass 1: draw lines/ports by absolute (x,y) positions.
	connPoints := map[[2]int]linePointConn{}
	addConn := func(x, y int, left, right, up, down bool) {
		if x < 0 || x >= totalWidth || y < 0 || y >= totalRows {
			return
		}
		p := [2]int{x, y}
		c := connPoints[p]
		c.Left = c.Left || left
		c.Right = c.Right || right
		c.Up = c.Up || up
		c.Down = c.Down || down
		connPoints[p] = c
	}
	for _, col := range view.Columns {
		for _, target := range col {
			targetPos, ok := view.Positions[target.ID]
			if !ok {
				continue
			}
			targetPort := targetPos.PortIn()
			if targetPort.Column < 0 || targetPort.Column >= len(view.Columns)-1 {
				continue
			}
			xIn := columnStarts[targetPort.Column] + columnMetrics[targetPort.Column].MaxStepWidth + 2
			for _, depID := range target.DependsOn {
				sourcePos, ok := view.Positions[depID]
				if !ok {
					continue
				}
				sourceStep, ok := stepsByCell[sourcePos.Column][sourcePos.Row]
				stepWidth := columnMetrics[sourcePos.Column].MaxStepWidth
				if ok {
					stepWidth = NewStepComponent(sourceStep, 0).PreferredWidth()
				}
				xOut := columnStarts[sourcePos.Column] + stepWidth
				y := sourcePos.Row * 2
				if y < 0 || y >= totalRows {
					continue
				}
				from := xOut
				to := xIn
				if from > to {
					from, to = to, from
				}
				for x := from; x <= to; x++ {
					addConn(x, y, x > from, x < to, false, false)
				}

				targetY := targetPos.Row * 2
				if targetY != y {
					fromY := y
					toY := targetY
					if fromY > toY {
						fromY, toY = toY, fromY
					}
					for yy := fromY; yy <= toY; yy++ {
						addConn(xIn, yy, false, false, yy > fromY, yy < toY)
					}
				}

				// Final leg: from in-port to the beginning of the target step block.
				targetStepStartX := columnStarts[targetPos.Column]
				if targetStepStartX > xIn {
					for x := xIn; x < targetStepStartX; x++ {
						addConn(x, targetY, x > xIn, x < targetStepStartX-1, false, false)
					}
				}
			}
		}
	}
	for p, c := range connPoints {
		x, y := p[0], p[1]
		canvas[y][x] = connectorRune(c)
	}

	// Pass 2: place styled step nodes on top of arrows.
	overlaysByRow := map[int][]stepOverlay{}
	for row := 0; row < view.RowCount; row++ {
		y := row * 2
		for col := 0; col < len(view.Columns); col++ {
			x := columnStarts[col]
			if step, ok := stepsByCell[col][row]; ok {
				component := NewStepComponent(step, 0)
				overlaysByRow[y] = append(overlaysByRow[y], stepOverlay{
					start:  x,
					width:  component.PreferredWidth(),
					styled: component.RenderBrick(),
				})
			}
		}
	}

	rows := make([]string, totalRows)
	for y := 0; y < totalRows; y++ {
		rows[y] = composeRowWithOverlays(canvas[y], overlaysByRow[y])
	}

	return rows
}

func connectorRune(c linePointConn) rune {
	switch {
	case c.Left && c.Right && c.Up && c.Down:
		return '╋'
	case c.Left && c.Right && c.Down:
		return '┳'
	case c.Left && c.Right && c.Up:
		return '┻'
	case c.Up && c.Down && c.Right:
		return '┣'
	case c.Up && c.Down && c.Left:
		return '┫'
	case c.Down && c.Right:
		return '┏'
	case c.Up && c.Right:
		return '┗'
	case c.Down && c.Left:
		return '┓'
	case c.Up && c.Left:
		return '┛'
	case c.Left || c.Right:
		return '━'
	case c.Up || c.Down:
		return '┃'
	default:
		return ' '
	}
}

func composeRowWithOverlays(base []rune, overlays []stepOverlay) string {
	baseStyle := lipgloss.NewStyle().
		Background(theme.ContentBackground).
		Foreground(theme.ArrowColor)

	if len(overlays) == 0 {
		return baseStyle.Render(string(base))
	}

	sort.SliceStable(overlays, func(i, j int) bool {
		return overlays[i].start < overlays[j].start
	})

	var b strings.Builder
	cursor := 0
	for _, ov := range overlays {
		if ov.width <= 0 || ov.start >= len(base) {
			continue
		}
		start := ov.start
		if start < cursor {
			start = cursor
		}
		if start > len(base) {
			start = len(base)
		}
		if start > cursor {
			b.WriteString(baseStyle.Render(string(base[cursor:start])))
		}
		b.WriteString(ov.styled)
		end := start + ov.width
		if end > len(base) {
			end = len(base)
		}
		cursor = end
	}
	if cursor < len(base) {
		b.WriteString(baseStyle.Render(string(base[cursor:])))
	}
	return b.String()
}

type horizontalConnectorGrid struct {
	lines map[int]map[int]bool
}

func (g horizontalConnectorGrid) has(lane, row int) bool {
	laneRows, ok := g.lines[lane]
	if !ok {
		return false
	}
	return laneRows[row]
}

func buildHorizontalConnectorGrid(view PipelineView) horizontalConnectorGrid {
	grid := horizontalConnectorGrid{lines: map[int]map[int]bool{}}

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
		targetPort := targetPos.PortIn()
		for _, depID := range target.DependsOn {
			sourcePos, ok := view.Positions[depID]
			if !ok {
				continue
			}
			sourcePort := sourcePos.PortOut()
			row := sourcePort.Row
			from := sourcePort.Column
			to := targetPort.Column
			if to < from {
				continue
			}
			for lane := from; lane <= to; lane++ {
				if grid.lines[lane] == nil {
					grid.lines[lane] = map[int]bool{}
				}
				grid.lines[lane][row] = true
			}
		}
	}

	return grid
}

type columnRenderMetrics struct {
	StepCount    int
	MaxStepWidth int
}

type linePointConn struct {
	Left  bool
	Right bool
	Up    bool
	Down  bool
}

type stepOverlay struct {
	start  int
	width  int
	styled string
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

func renderPipelineSpacerRow(boundaryRow int, columnMetrics []columnRenderMetrics, arrow ArrowComponent, connectors connectorGrid) string {
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

type connectorGrid struct {
	rowJunctions map[int]map[int]connectorJunction
	boundaries   map[int]map[int]bool
}

func (c connectorGrid) rowJunction(lane, row int) (connectorJunction, bool) {
	laneMap, ok := c.rowJunctions[lane]
	if !ok {
		return connectorJunction{}, false
	}
	j, ok := laneMap[row]
	return j, ok
}

func (c connectorGrid) hasBoundaryVertical(lane, boundaryRow int) bool {
	laneMap, ok := c.boundaries[lane]
	if !ok {
		return false
	}
	return laneMap[boundaryRow]
}

func buildConnectorGrid(view PipelineView) connectorGrid {
	grid := connectorGrid{
		rowJunctions: map[int]map[int]connectorJunction{},
		boundaries:   map[int]map[int]bool{},
	}

	for _, col := range view.Columns {
		for _, target := range col {
			targetPos, ok := view.Positions[target.ID]
			if !ok || targetPos.Column == 0 {
				continue
			}
			for _, depID := range target.DependsOn {
				sourcePos, ok := view.Positions[depID]
				if !ok || sourcePos.Column >= targetPos.Column {
					continue
				}
				drawEdgeByPoints(&grid, sourcePos, targetPos)
			}
		}
	}

	return grid
}

func drawEdgeByPoints(grid *connectorGrid, source StepPositionView, target StepPositionView) {
	sourceLane := source.Column
	targetLane := target.Column - 1
	if targetLane < sourceLane {
		return
	}

	switch {
	case source.Row == target.Row:
		drawHorizontalToMarker(grid, sourceLane, targetLane, source.Row, true)
	case source.Row < target.Row:
		// # -> * (transit) at source row, then * -> * vertically at target lane.
		drawHorizontalToMarker(grid, sourceLane, targetLane, source.Row, false)
		drawVerticalBetweenMarkers(grid, targetLane, source.Row, target.Row)
		addJunction(grid, targetLane, target.Row, false, true, true, false) // * -> step
	default:
		// # -> . at source row, then . -> * vertically at target lane.
		drawHorizontalToMarker(grid, sourceLane, targetLane, source.Row, false)
		drawVerticalBetweenMarkers(grid, targetLane, source.Row, target.Row)
		addJunction(grid, targetLane, target.Row, false, true, false, true) // * -> step
	}
}

func drawHorizontalToMarker(grid *connectorGrid, fromLane, toLane, row int, endToStep bool) {
	for lane := fromLane; lane <= toLane; lane++ {
		right := lane < toLane || endToStep
		addJunction(grid, lane, row, true, right, false, false)
	}
}

func drawVerticalBetweenMarkers(grid *connectorGrid, lane, fromRow, toRow int) {
	if fromRow == toRow {
		return
	}
	if fromRow < toRow {
		addJunction(grid, lane, fromRow, false, false, false, true)
		for boundary := fromRow; boundary < toRow; boundary++ {
			addBoundary(grid, lane, boundary)
		}
		addJunction(grid, lane, toRow, false, false, true, false)
		return
	}
	addJunction(grid, lane, fromRow, false, false, true, false)
	for boundary := toRow; boundary < fromRow; boundary++ {
		addBoundary(grid, lane, boundary)
	}
	addJunction(grid, lane, toRow, false, false, false, true)
}

func addJunction(grid *connectorGrid, lane, row int, left, right, up, down bool) {
	if grid.rowJunctions[lane] == nil {
		grid.rowJunctions[lane] = map[int]connectorJunction{}
	}
	current := grid.rowJunctions[lane][row]
	current.Left = current.Left || left
	current.Right = current.Right || right
	current.Up = current.Up || up
	current.Down = current.Down || down
	grid.rowJunctions[lane][row] = current
}

func addBoundary(grid *connectorGrid, lane, boundaryRow int) {
	if grid.boundaries[lane] == nil {
		grid.boundaries[lane] = map[int]bool{}
	}
	grid.boundaries[lane][boundaryRow] = true
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
