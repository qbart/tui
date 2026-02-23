package ui

import core "hestia/hestia"

type StepView struct {
	ID        string
	Icon      string
	JobName   string
	DependsOn []string
}

type StepPositionView struct {
	Column int
	Row    int
	Width  int
}

func (v StepPositionView) PortIn() StepPositionView {
	return StepPositionView{v.Column - 2, v.Row, v.Width}
}
func (v StepPositionView) PortOut() StepPositionView {
	return StepPositionView{v.Column + 1, v.Row, v.Width}
}

type PipelineView struct {
	Columns   [][]StepView
	Positions map[string]StepPositionView
	RowCount  int
}

func BuildPipelineView(spec core.PipelineSpec, run core.PipelineRun) (PipelineView, error) {
	columns, positions, rowCount, err := spec.Layout()
	if err != nil {
		return PipelineView{}, err
	}

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
