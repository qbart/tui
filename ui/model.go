package ui

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	core "hestia/hestia"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

type tickMsg time.Time

type Model struct {
	width         int
	height        int
	spec          core.PipelineSpec
	run           core.PipelineRun
	stepDurations map[core.StepID]time.Duration
	spinnerFrame  int
	scrollX       int
	scrollY       int
}

func NewModel() Model {
	spec := core.NewPipelineSpec("sample-cicd", []core.StepSpec{
		{ID: "checkout", JobName: "checkout"},
		{ID: "lint", JobName: "lint", DependsOn: []core.StepID{"checkout"}},
		{ID: "unit-core", JobName: "unit core", DependsOn: []core.StepID{"checkout"}},
		{ID: "unit-api", JobName: "unit api", DependsOn: []core.StepID{"checkout"}},
		{ID: "build-ui-assets", JobName: "build ui assets", DependsOn: []core.StepID{"checkout"}},
		{ID: "build-api-image", JobName: "build api image", DependsOn: []core.StepID{"checkout"}},
		{ID: "build-worker-image", JobName: "build worker image", DependsOn: []core.StepID{"checkout"}},
		{ID: "policy-scan", JobName: "policy scan", DependsOn: []core.StepID{"lint"}},
		{ID: "int-postgres", JobName: "int postgres", DependsOn: []core.StepID{"unit-core", "unit-api"}},
		{ID: "int-sqlite", JobName: "int sqlite", DependsOn: []core.StepID{"unit-core"}},
		{ID: "int-duckdb", JobName: "int duckdb", DependsOn: []core.StepID{"unit-api"}},
		{ID: "e2e-web", JobName: "e2e web", DependsOn: []core.StepID{"build-ui-assets", "build-api-image"}},
		{ID: "e2e-mobile", JobName: "e2e mobile", DependsOn: []core.StepID{"build-ui-assets", "build-api-image"}},
		{ID: "worker-smoke", JobName: "worker smoke", DependsOn: []core.StepID{"build-worker-image"}},
		{ID: "quality-gate", JobName: "quality gate", DependsOn: []core.StepID{"policy-scan", "int-postgres", "int-sqlite", "int-duckdb", "e2e-web", "e2e-mobile", "worker-smoke"}},
		{ID: "package-api", JobName: "package api", DependsOn: []core.StepID{"quality-gate"}},
		{ID: "package-worker", JobName: "package worker", DependsOn: []core.StepID{"quality-gate"}},
		{ID: "package-ui", JobName: "package ui", DependsOn: []core.StepID{"quality-gate"}},
		{ID: "deploy-staging", JobName: "deploy staging", DependsOn: []core.StepID{"package-api", "package-worker", "package-ui"}},
		{ID: "smoke-staging", JobName: "smoke staging", DependsOn: []core.StepID{"deploy-staging"}},
		{ID: "perf-staging", JobName: "perf staging", DependsOn: []core.StepID{"deploy-staging"}},
		{ID: "approve-prod", JobName: "approve prod", DependsOn: []core.StepID{"smoke-staging", "perf-staging"}},
		{ID: "deploy-prod", JobName: "deploy prod", DependsOn: []core.StepID{"approve-prod"}},
		{ID: "verify-prod", JobName: "verify prod", DependsOn: []core.StepID{"deploy-prod"}},
		{ID: "notify-success", JobName: "notify success", DependsOn: []core.StepID{"verify-prod"}},
		{ID: "sbom-generate", JobName: "sbom generate", DependsOn: []core.StepID{"package-api", "package-worker", "package-ui"}},
		{ID: "security-sign", JobName: "security sign", DependsOn: []core.StepID{"sbom-generate"}},
		{ID: "release-notes", JobName: "release notes", DependsOn: []core.StepID{"quality-gate"}},
		{ID: "publish-release", JobName: "publish release", DependsOn: []core.StepID{"security-sign", "release-notes"}},
		{ID: "chaos-staging", JobName: "chaos staging", DependsOn: []core.StepID{"deploy-staging"}},
		{ID: "rollback-drill", JobName: "rollback drill", DependsOn: []core.StepID{"chaos-staging"}},
		{ID: "audit-prod", JobName: "audit prod", DependsOn: []core.StepID{"deploy-prod"}},
		{ID: "notify-security", JobName: "notify security", DependsOn: []core.StepID{"audit-prod"}},
		// Disconnected test data (not connected to existing pipeline nodes).
		{ID: "sandbox-a-prepare", JobName: "sandbox a prepare"},
		{ID: "sandbox-a-run", JobName: "sandbox a run", DependsOn: []core.StepID{"sandbox-a-prepare"}},
		{ID: "sandbox-a-report", JobName: "sandbox a report", DependsOn: []core.StepID{"sandbox-a-run"}},
		{ID: "sandbox-b-prepare", JobName: "sandbox b prepare"},
		{ID: "sandbox-b-run", JobName: "sandbox b run", DependsOn: []core.StepID{"sandbox-b-prepare"}},
		{ID: "sandbox-b-cleanup", JobName: "sandbox b cleanup", DependsOn: []core.StepID{"sandbox-b-run"}},
		{ID: "orphan-healthcheck", JobName: "orphan healthcheck"},
		{ID: "orphan-metrics", JobName: "orphan metrics"},
		{ID: "perf-a-setup", JobName: "perf a setup"},
		{ID: "perf-a-run", JobName: "perf a run", DependsOn: []core.StepID{"perf-a-setup"}},
		{ID: "perf-a-report", JobName: "perf a report", DependsOn: []core.StepID{"perf-a-run"}},
		{ID: "perf-b-setup", JobName: "perf b setup"},
		{ID: "perf-b-run", JobName: "perf b run", DependsOn: []core.StepID{"perf-b-setup"}},
		{ID: "perf-b-compare", JobName: "perf b compare", DependsOn: []core.StepID{"perf-b-run"}},
		{ID: "lab-seed", JobName: "lab seed"},
		{ID: "lab-simulate", JobName: "lab simulate", DependsOn: []core.StepID{"lab-seed"}},
		{ID: "lab-analyze", JobName: "lab analyze", DependsOn: []core.StepID{"lab-simulate"}},
		{ID: "lab-archive", JobName: "lab archive", DependsOn: []core.StepID{"lab-analyze"}},
		{ID: "isolated-alpha", JobName: "isolated alpha"},
		{ID: "isolated-beta", JobName: "isolated beta"},
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
			"checkout":           1 * time.Second,
			"lint":               1 * time.Second,
			"unit-core":          2 * time.Second,
			"unit-api":           2 * time.Second,
			"build-ui-assets":    2 * time.Second,
			"build-api-image":    2 * time.Second,
			"build-worker-image": 2 * time.Second,
			"policy-scan":        1 * time.Second,
			"int-postgres":       2 * time.Second,
			"int-sqlite":         2 * time.Second,
			"int-duckdb":         2 * time.Second,
			"e2e-web":            2 * time.Second,
			"e2e-mobile":         2 * time.Second,
			"worker-smoke":       1 * time.Second,
			"quality-gate":       1 * time.Second,
			"package-api":        1 * time.Second,
			"package-worker":     1 * time.Second,
			"package-ui":         1 * time.Second,
			"deploy-staging":     2 * time.Second,
			"smoke-staging":      1 * time.Second,
			"perf-staging":       2 * time.Second,
			"approve-prod":       1 * time.Second,
			"deploy-prod":        2 * time.Second,
			"verify-prod":        1 * time.Second,
			"notify-success":     1 * time.Second,
			"sbom-generate":      1 * time.Second,
			"security-sign":      1 * time.Second,
			"release-notes":      1 * time.Second,
			"publish-release":    1 * time.Second,
			"chaos-staging":      2 * time.Second,
			"rollback-drill":     1 * time.Second,
			"audit-prod":         1 * time.Second,
			"notify-security":    1 * time.Second,
			"sandbox-a-prepare":  1 * time.Second,
			"sandbox-a-run":      2 * time.Second,
			"sandbox-a-report":   1 * time.Second,
			"sandbox-b-prepare":  1 * time.Second,
			"sandbox-b-run":      2 * time.Second,
			"sandbox-b-cleanup":  1 * time.Second,
			"orphan-healthcheck": 1 * time.Second,
			"orphan-metrics":     1 * time.Second,
			"perf-a-setup":       1 * time.Second,
			"perf-a-run":         2 * time.Second,
			"perf-a-report":      1 * time.Second,
			"perf-b-setup":       1 * time.Second,
			"perf-b-run":         2 * time.Second,
			"perf-b-compare":     1 * time.Second,
			"lab-seed":           1 * time.Second,
			"lab-simulate":       2 * time.Second,
			"lab-analyze":        1 * time.Second,
			"lab-archive":        1 * time.Second,
			"isolated-alpha":     1 * time.Second,
			"isolated-beta":      1 * time.Second,
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
		case "up", "k":
			m.scrollY--
		case "down", "j":
			m.scrollY++
		case "left", "h":
			m.scrollX--
		case "right", "l":
			m.scrollX++
		}
		m.clampScroll()
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.clampScroll()
	case tickMsg:
		frameCount := spinnerFrameCount()
		if frameCount > 0 {
			m.spinnerFrame = (m.spinnerFrame + 1) % frameCount
		}
		m.advance(time.Time(msg))
		m.clampScroll()
		return m, tickCmd()
	}

	return m, nil
}

func (m Model) View() string {
	if m.width <= 0 || m.height <= 0 {
		return "Loading..."
	}

	renderWidth := max(m.width-1, 1)
	contentHeight := max(m.height-1, 0)
	content := renderContent(renderWidth, contentHeight, m.spec, m.run, m.spinnerFrame, m.scrollX, m.scrollY)
	footer := renderFooter(renderWidth, fmt.Sprintf("run:%s | q to quit", m.run.Status))

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

func renderContent(width, height int, spec core.PipelineSpec, run core.PipelineRun, spinnerFrame, scrollX, scrollY int) string {
	if height <= 0 {
		return ""
	}

	const topPadding = 1
	sidePadding := 0
	if width >= 2 {
		sidePadding = 1
	}
	contentWidth := max(width-(sidePadding*2), 0)
	innerHeight := max(height-topPadding, 0)

	view, err := BuildPipelineView(spec, run, spinnerFrame)
	if err != nil {
		msg := clampVisibleLine(fmt.Sprintf("invalid pipeline: %v", err), contentWidth)
		rows := make([]string, 0, height)
		for i := 0; i < topPadding; i++ {
			rows = append(rows, strings.Repeat(" ", contentWidth))
		}
		rows = append(rows, msg)
		for len(rows) < height {
			rows = append(rows, strings.Repeat(" ", contentWidth))
		}
		return renderContentRows(rows, width, sidePadding)
	}

	lines := make([]string, 0, innerHeight)
	lines = append(lines, renderPipelineGraph(view, scrollX, scrollY, contentWidth, innerHeight)...)

	for len(lines) > innerHeight {
		lines = lines[:innerHeight]
	}

	for len(lines) < innerHeight {
		lines = append(lines, "")
	}
	for i := range lines {
		lines[i] = clampVisibleLine(lines[i], contentWidth)
	}

	rows := make([]string, 0, height)
	for i := 0; i < topPadding; i++ {
		rows = append(rows, strings.Repeat(" ", contentWidth))
	}
	rows = append(rows, lines...)
	for len(rows) < height {
		rows = append(rows, strings.Repeat(" ", contentWidth))
	}

	return renderContentRows(rows, width, sidePadding)
}

func renderContentRows(rows []string, width, sidePadding int) string {
	innerWidth := max(width-(sidePadding*2), 0)
	side := strings.Repeat(" ", sidePadding)
	sideStyle := lipgloss.NewStyle().
		Background(theme.ContentBackground).
		Foreground(theme.ContentForeground)
	sideBlock := sideStyle.Render(side)
	lineStyle := sideStyle
	for i := range rows {
		body := clampVisibleLine(rows[i], innerWidth)
		if !strings.Contains(body, "\x1b[") {
			body = lineStyle.Render(body)
		}
		rows[i] = sideBlock + body + sideBlock + paintToEOL(theme.ContentBackground)
	}
	return strings.Join(rows, "\n")
}

func paintToEOL(bg lipgloss.Color) string {
	return backgroundSeq(bg) + "\x1b[K\x1b[0m"
}

func backgroundSeq(bg lipgloss.Color) string {
	s := strings.TrimSpace(string(bg))
	if s == "" {
		return ""
	}
	if strings.HasPrefix(s, "#") && len(s) == 7 {
		r, errR := strconv.ParseInt(s[1:3], 16, 64)
		g, errG := strconv.ParseInt(s[3:5], 16, 64)
		b, errB := strconv.ParseInt(s[5:7], 16, 64)
		if errR == nil && errG == nil && errB == nil {
			return fmt.Sprintf("\x1b[48;2;%d;%d;%dm", r, g, b)
		}
	}
	if n, err := strconv.Atoi(s); err == nil {
		return fmt.Sprintf("\x1b[48;5;%dm", n)
	}
	return ""
}

func renderPipelineGraph(view PipelineView, scrollX, scrollY, viewportWidth, viewportHeight int) []string {
	if len(view.Columns) == 0 {
		return []string{"(no steps)"}
	}

	if view.RowCount == 0 || viewportWidth <= 0 || viewportHeight <= 0 {
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
				bg, fg := component.Colors()
				overlaysByRow[y] = append(overlaysByRow[y], stepOverlay{
					start: x,
					width: component.PreferredWidth(),
					label: component.PlainLabel(),
					bg:    bg,
					fg:    fg,
				})
			}
		}
	}

	if scrollX < 0 {
		scrollX = 0
	}
	if scrollY < 0 {
		scrollY = 0
	}
	maxScrollX := max(totalWidth-viewportWidth, 0)
	maxScrollY := max(totalRows-viewportHeight, 0)
	if scrollX > maxScrollX {
		scrollX = maxScrollX
	}
	if scrollY > maxScrollY {
		scrollY = maxScrollY
	}

	rows := make([]string, 0, viewportHeight)
	for y := scrollY; y < min(scrollY+viewportHeight, totalRows); y++ {
		rows = append(rows, composeRowWithOverlaysViewport(canvas[y], overlaysByRow[y], scrollX, viewportWidth))
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

func composeRowWithOverlaysViewport(base []rune, overlays []stepOverlay, scrollX, viewportWidth int) string {
	if scrollX < 0 {
		scrollX = 0
	}
	if scrollX > len(base) {
		scrollX = len(base)
	}
	right := min(scrollX+viewportWidth, len(base))

	row := make([]styledCell, len(base))
	for i, ch := range base {
		row[i] = styledCell{
			ch: ch,
			bg: theme.ContentBackground,
			fg: theme.ArrowColor,
		}
	}

	sort.SliceStable(overlays, func(i, j int) bool {
		return overlays[i].start < overlays[j].start
	})

	for _, ov := range overlays {
		if ov.width <= 0 || ov.start >= len(row) {
			continue
		}
		labelRunes := []rune(ov.label)
		start := max(ov.start, 0)
		end := min(ov.start+ov.width, len(row))
		for x := start; x < end; x++ {
			labelIdx := x - ov.start
			ch := ' '
			if labelIdx >= 0 && labelIdx < len(labelRunes) {
				ch = labelRunes[labelIdx]
			}
			row[x] = styledCell{ch: ch, bg: ov.bg, fg: ov.fg}
		}
	}

	visible := row[scrollX:right]
	var b strings.Builder
	if len(visible) > 0 {
		curBG := visible[0].bg
		curFG := visible[0].fg
		segment := make([]rune, 0, len(visible))
		flush := func() {
			if len(segment) == 0 {
				return
			}
			b.WriteString(lipgloss.NewStyle().
				Background(curBG).
				Foreground(curFG).
				Render(string(segment)))
			segment = segment[:0]
		}
		for _, cell := range visible {
			if cell.bg != curBG || cell.fg != curFG {
				flush()
				curBG = cell.bg
				curFG = cell.fg
			}
			segment = append(segment, cell.ch)
		}
		flush()
	}
	currentWidth := lipgloss.Width(b.String())
	for currentWidth < viewportWidth {
		b.WriteString(lipgloss.NewStyle().
			Background(theme.ContentBackground).
			Foreground(theme.ArrowColor).
			Render(" "))
		currentWidth++
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
	start int
	width int
	label string
	bg    lipgloss.Color
	fg    lipgloss.Color
}

type styledCell struct {
	ch rune
	bg lipgloss.Color
	fg lipgloss.Color
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
	fitted := ansi.Truncate(s, width, "")
	w := lipgloss.Width(fitted)
	if w < width {
		return fitted + strings.Repeat(" ", width-w)
	}
	return fitted
}

func (m *Model) clampScroll() {
	if m.width <= 0 || m.height <= 0 {
		m.scrollX = 0
		m.scrollY = 0
		return
	}
	view, err := BuildPipelineView(m.spec, m.run, m.spinnerFrame)
	if err != nil {
		m.scrollX = 0
		m.scrollY = 0
		return
	}
	totalWidth, totalRows := graphDimensions(view)

	contentHeight := max(m.height-1, 0)
	innerHeight := max(contentHeight-1, 0)
	renderWidth := max(m.width-1, 1)
	innerWidth := max(renderWidth, 0)
	if renderWidth >= 2 {
		innerWidth = renderWidth - 2
	}

	maxX := max(totalWidth-innerWidth, 0)
	maxY := max(totalRows-innerHeight, 0)
	if m.scrollX < 0 {
		m.scrollX = 0
	}
	if m.scrollY < 0 {
		m.scrollY = 0
	}
	if m.scrollX > maxX {
		m.scrollX = maxX
	}
	if m.scrollY > maxY {
		m.scrollY = maxY
	}
}

func graphDimensions(view PipelineView) (int, int) {
	if len(view.Columns) == 0 || view.RowCount == 0 {
		return 0, 0
	}
	columnMetrics := buildColumnRenderMetrics(view.Columns)
	const gapWidth = 5
	totalWidth := 0
	for col := 0; col < len(view.Columns); col++ {
		totalWidth += columnMetrics[col].MaxStepWidth
		if col < len(view.Columns)-1 {
			totalWidth += gapWidth
		}
	}
	totalRows := view.RowCount*2 - 1
	return totalWidth, totalRows
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func clampVisibleLine(line string, width int) string {
	if width <= 0 {
		return ""
	}
	clamped := ansi.Truncate(line, width, "")
	w := lipgloss.Width(clamped)
	if w < width {
		clamped += strings.Repeat(" ", width-w)
	}
	return clamped
}
