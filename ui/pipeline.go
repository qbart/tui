package ui

import core "hestia/hestia"

type StepVisualStatus string

const (
	StatusBlack  StepVisualStatus = "StatusBlack"
	StatusGray   StepVisualStatus = "StatusGray"
	StatusGreen  StepVisualStatus = "StatusGreen"
	StatusRed    StepVisualStatus = "StatusRed"
	StatusYellow StepVisualStatus = "StatusYellow"
	StatusBlue   StepVisualStatus = "StatusBlue"
)

type StepView struct {
	ID        string
	Icon      string
	JobName   string
	DependsOn []string
	Status    StepVisualStatus
	Spinner   bool
	SpinChar  string
}

type StepPositionView struct {
	Column int
	Row    int
	Width  int
}

func (v StepPositionView) PortIn() StepPositionView {
	return StepPositionView{v.Column - 1, v.Row, v.Width}
}
func (v StepPositionView) PortOut() StepPositionView {
	return StepPositionView{v.Column, v.Row, v.Width}
}

type PipelineView struct {
	Columns   [][]StepView
	Positions map[string]StepPositionView
	RowCount  int
}

func BuildPipelineView(spec core.PipelineSpec, run core.PipelineRun, spinnerFrame int) (PipelineView, error) {
	columns, positions, rowCount, err := spec.Layout()
	if err != nil {
		return PipelineView{}, err
	}
	runningStepID, hasRunning := run.RunningStepID()

	viewCols := make([][]StepView, len(columns))
	stepsByID := make(map[string]StepView, len(spec.Steps))
	for i, col := range columns {
		viewCol := make([]StepView, 0, len(col))
		for _, step := range col {
			deps := make([]string, 0, len(step.DependsOn))
			for _, dep := range step.DependsOn {
				deps = append(deps, string(dep))
			}

			viewStep := StepView{
				ID:        string(step.ID),
				Icon:      "",
				JobName:   step.JobName,
				DependsOn: deps,
				Status:    visualStatusForStepID(string(step.ID)),
				Spinner:   hasRunning && string(runningStepID) == string(step.ID),
				SpinChar:  spinnerGlyph(spinnerFrame),
			}
			viewCol = append(viewCol, viewStep)
			stepsByID[viewStep.ID] = viewStep
		}
		viewCols[i] = viewCol
	}

	viewPos := make(map[string]StepPositionView, len(positions))
	for stepID, pos := range positions {
		step, ok := stepsByID[string(stepID)]
		width := 0
		if ok {
			width = NewStepComponent(step, 0).PreferredWidth()
		}
		viewPos[string(stepID)] = StepPositionView{Column: pos.Column, Row: pos.Row, Width: width}
	}

	return PipelineView{Columns: viewCols, Positions: viewPos, RowCount: rowCount}, nil
}

func visualStatusForStepID(stepID string) StepVisualStatus {
	switch stepID {
	case "checkout":
		return StatusGray
	case "build":
		return StatusBlue
	case "test-postresql":
		return StatusYellow
	case "test-sqlite":
		return StatusGreen
	case "test-duckdb":
		return StatusRed
	default:
		return StatusBlack
	}
}

func spinnerGlyph(frame int) string {
	frames := []rune("⣾⣽⣻⢿⡿⣟⣯⣷")
	if len(frames) == 0 {
		return ""
	}
	if frame < 0 {
		frame = 0
	}
	return string(frames[frame%len(frames)])
}

func spinnerFrameCount() int {
	return len([]rune("⣾⣽⣻⢿⡿⣟⣯⣷"))
}
