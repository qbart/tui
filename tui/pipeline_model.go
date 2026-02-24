package tui

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"tui/core"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

type tickMsg time.Time

type SetStepStatusMsg struct {
	StepID core.StepID
	Status core.StepVisualStatus
}

type SetStepSpinnerMsg struct {
	StepID  core.StepID
	Spinner bool
}

type SetStepSelectedMsg struct {
	StepID core.StepID
}

type PipelineModel struct {
	width          int
	height         int
	spec           core.PipelineSpec
	stepStates     map[core.StepID]StepRuntimeState
	spinnerFrame   int
	scrollX        int
	scrollY        int
	selectedStepID string
}

type StepRuntimeState struct {
	Status  core.StepVisualStatus
	Spinner bool
}

func NewPipelineModel(spec core.PipelineSpec) PipelineModel {
	stepStates := make(map[core.StepID]StepRuntimeState, len(spec.Steps))
	for _, step := range spec.Steps {
		stepStates[step.ID] = StepRuntimeState{
			Status:  step.Status,
			Spinner: false,
		}
	}

	return PipelineModel{
		spec:           spec,
		stepStates:     stepStates,
		selectedStepID: "",
	}
}

func (m PipelineModel) Init() tea.Cmd {
	return tickCmd()
}

func (m PipelineModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
		case "tab":
			m.cycleSelectedStep()
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
		m.clampScroll()
		return m, tickCmd()
	case SetStepStatusMsg:
		_ = m.SetStepStatus(msg.StepID, msg.Status)
		return m, nil
	case SetStepSpinnerMsg:
		_ = m.SetStepSpinner(msg.StepID, msg.Spinner)
		return m, nil
	case SetStepSelectedMsg:
		_ = m.SetStepSelected(msg.StepID)
		return m, nil
	}

	return m, nil
}

func (m PipelineModel) View() string {
	if m.width <= 0 || m.height <= 0 {
		return "Loading..."
	}

	renderWidth := max(m.width-1, 1)
	contentHeight := max(m.height, 0)
	content := renderContent(renderWidth, contentHeight, m.spec, m.stepStates, m.spinnerFrame, m.scrollX, m.scrollY, m.selectedStepID)

	return content
}

func tickCmd() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func renderContent(width, height int, spec core.PipelineSpec, stepStates map[core.StepID]StepRuntimeState, spinnerFrame, scrollX, scrollY int, selectedStepID string) string {
	if height <= 0 {
		return ""
	}

	const topPadding = 1
	const bottomPadding = 1
	sidePadding := 0
	if width >= 2 {
		sidePadding = 1
	}
	contentWidth := max(width-(sidePadding*2), 0)
	innerHeight := max(height-topPadding-bottomPadding, 0)

	view, err := buildPipelineView(spec, stepStates, spinnerFrame, selectedStepID)
	if err != nil {
		msg := clampVisibleLine(fmt.Sprintf("invalid pipeline: %v", err), contentWidth)
		rows := make([]string, 0, height)
		for i := 0; i < topPadding; i++ {
			rows = append(rows, strings.Repeat(" ", contentWidth))
		}
		rows = append(rows, msg)
		for i := 0; i < bottomPadding; i++ {
			rows = append(rows, strings.Repeat(" ", contentWidth))
		}
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
	for i := 0; i < bottomPadding; i++ {
		rows = append(rows, strings.Repeat(" ", contentWidth))
	}
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

func renderPipelineGraph(view pipelineView, scrollX, scrollY, viewportWidth, viewportHeight int) []string {
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
	highlightMask := make([][]bool, totalRows)
	for y := 0; y < totalRows; y++ {
		canvas[y] = []rune(strings.Repeat(" ", totalWidth))
		highlightMask[y] = make([]bool, totalWidth)
	}

	stepsByCell := map[int]map[int]stepView{}
	for _, colSteps := range view.Columns {
		for _, step := range colSteps {
			pos := view.Positions[step.ID]
			if stepsByCell[pos.Column] == nil {
				stepsByCell[pos.Column] = map[int]stepView{}
			}
			stepsByCell[pos.Column][pos.Row] = step
		}
	}

	// Pass 1: draw lines/ports by absolute (x,y) positions.
	connPoints := map[[2]int]linePointConn{}
	addConn := func(x, y int, left, right, up, down, highlighted bool) {
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
		if highlighted {
			highlightMask[y][x] = true
		}
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
				highlighted := view.HighlightedEdge[edgeKey(depID, target.ID)]
				from := xOut
				to := xIn
				if from > to {
					from, to = to, from
				}
				for x := from; x <= to; x++ {
					addConn(x, y, x > from, x < to, false, false, highlighted)
				}

				targetY := targetPos.Row * 2
				if targetY != y {
					fromY := y
					toY := targetY
					if fromY > toY {
						fromY, toY = toY, fromY
					}
					for yy := fromY; yy <= toY; yy++ {
						addConn(xIn, yy, false, false, yy > fromY, yy < toY, highlighted)
					}
				}

				// Final leg: from in-port to the beginning of the target step block.
				targetStepStartX := columnStarts[targetPos.Column]
				if targetStepStartX > xIn {
					for x := xIn; x < targetStepStartX; x++ {
						addConn(x, targetY, x > xIn, x < targetStepStartX-1, false, false, highlighted)
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
					label: component.DisplayLabel(),
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
		rows = append(rows, composeRowWithOverlaysViewport(canvas[y], highlightMask[y], overlaysByRow[y], scrollX, viewportWidth))
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

func composeRowWithOverlaysViewport(base []rune, highlighted []bool, overlays []stepOverlay, scrollX, viewportWidth int) string {
	if scrollX < 0 {
		scrollX = 0
	}
	if scrollX > len(base) {
		scrollX = len(base)
	}
	right := min(scrollX+viewportWidth, len(base))

	row := make([]styledCell, len(base))
	for i, ch := range base {
		fg := theme.ArrowColor
		if i < len(highlighted) && highlighted[i] {
			fg = theme.ArrowSelectedColor
		}
		row[i] = styledCell{
			ch: ch,
			bg: theme.ContentBackground,
			fg: fg,
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

func buildColumnRenderMetrics(columns [][]stepView) []columnRenderMetrics {
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

func (m *PipelineModel) clampScroll() {
	if m.width <= 0 || m.height <= 0 {
		m.scrollX = 0
		m.scrollY = 0
		return
	}
	view, err := buildPipelineView(m.spec, m.stepStates, m.spinnerFrame, m.selectedStepID)
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

func (m *PipelineModel) SetStepStatus(stepID core.StepID, status core.StepVisualStatus) error {
	if _, ok := m.spec.StepByID(stepID); !ok {
		return fmt.Errorf("unknown step %q", stepID)
	}
	state := m.stepStates[stepID]
	state.Status = status
	m.stepStates[stepID] = state
	m.clampScroll()
	return nil
}

func (m *PipelineModel) SetStepSpinner(stepID core.StepID, spinning bool) error {
	if _, ok := m.spec.StepByID(stepID); !ok {
		return fmt.Errorf("unknown step %q", stepID)
	}
	state := m.stepStates[stepID]
	state.Spinner = spinning
	m.stepStates[stepID] = state
	m.clampScroll()
	return nil
}

func (m *PipelineModel) SetStepSelected(stepID core.StepID) error {
	if stepID == "" {
		m.selectedStepID = ""
		m.clampScroll()
		return nil
	}
	if _, ok := m.spec.StepByID(stepID); !ok {
		return fmt.Errorf("unknown step %q", stepID)
	}
	m.selectedStepID = string(stepID)
	m.clampScroll()
	return nil
}

func (m *PipelineModel) cycleSelectedStep() {
	if len(m.spec.Steps) == 0 {
		m.selectedStepID = ""
		return
	}
	if m.selectedStepID == "" {
		m.selectedStepID = string(m.spec.Steps[0].ID)
		return
	}

	currentIdx := -1
	for i, step := range m.spec.Steps {
		if string(step.ID) == m.selectedStepID {
			currentIdx = i
			break
		}
	}
	if currentIdx < 0 {
		m.selectedStepID = ""
		return
	}
	if currentIdx >= len(m.spec.Steps)-1 {
		m.selectedStepID = ""
		return
	}
	m.selectedStepID = string(m.spec.Steps[currentIdx+1].ID)
}

func graphDimensions(view pipelineView) (int, int) {
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
