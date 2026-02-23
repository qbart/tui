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
	for i, col := range columns {
		viewCol := make([]StepView, 0, len(col))
		for _, step := range col {
			deps := make([]string, 0, len(step.DependsOn))
			for _, dep := range step.DependsOn {
				deps = append(deps, string(dep))
			}

			viewCol = append(viewCol, StepView{
				ID:        string(step.ID),
				Icon:      "",
				JobName:   step.JobName,
				DependsOn: deps,
			})
		}
		viewCols[i] = viewCol
	}

	viewPos := make(map[string]StepPositionView, len(positions))
	for stepID, pos := range positions {
		viewPos[string(stepID)] = StepPositionView{Column: pos.Column, Row: pos.Row}
	}

	return PipelineView{Columns: viewCols, Positions: viewPos, RowCount: rowCount}, nil
}
