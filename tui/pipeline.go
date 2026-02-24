package tui

import "tui/core"

var Spinner = []rune("⣾⣽⣻⢿⡿⣟⣯⣷")

type StepView struct {
	ID        string
	JobName   string
	DependsOn []string
	Status    core.StepVisualStatus
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
	Columns         [][]StepView
	Positions       map[string]StepPositionView
	RowCount        int
	SelectedStepID  string
	HighlightedEdge map[string]bool
}

func BuildPipelineView(spec core.PipelineSpec, run core.PipelineRun, spinnerFrame int, selectedStepID string) (PipelineView, error) {
	columns, positions, rowCount, err := spec.Layout()
	if err != nil {
		return PipelineView{}, err
	}
	runningStepID, hasRunning := run.RunningStepID()
	highlightedEdges := highlightedEdgesForSelection(spec, selectedStepID)

	viewCols := make([][]StepView, len(columns))
	stepsByID := make(map[string]StepView, len(spec.Steps))
	for i, col := range columns {
		viewCol := make([]StepView, 0, len(col))
		for _, step := range col {
			deps := make([]string, 0, len(step.DependsOn))
			for _, dep := range step.DependsOn {
				deps = append(deps, string(dep))
			}

			status := step.Status
			if selectedStepID != "" && selectedStepID == string(step.ID) {
				status = core.StatusSelected
			}

			viewStep := StepView{
				ID:        string(step.ID),
				JobName:   step.JobName,
				DependsOn: deps,
				Status:    status,
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

	return PipelineView{
		Columns:         viewCols,
		Positions:       viewPos,
		RowCount:        rowCount,
		SelectedStepID:  selectedStepID,
		HighlightedEdge: highlightedEdges,
	}, nil
}

func edgeKey(sourceID, targetID string) string {
	return sourceID + "->" + targetID
}

func highlightedEdgesForSelection(spec core.PipelineSpec, selectedStepID string) map[string]bool {
	highlighted := map[string]bool{}
	if selectedStepID == "" {
		return highlighted
	}

	stepsByID := make(map[string]core.StepSpec, len(spec.Steps))
	dependents := make(map[string][]string, len(spec.Steps))
	for _, step := range spec.Steps {
		id := string(step.ID)
		stepsByID[id] = step
		dependents[id] = []string{}
	}
	if _, ok := stepsByID[selectedStepID]; !ok {
		return highlighted
	}
	for _, step := range spec.Steps {
		targetID := string(step.ID)
		for _, dep := range step.DependsOn {
			sourceID := string(dep)
			dependents[sourceID] = append(dependents[sourceID], targetID)
		}
	}

	// Upstream: selected -> dependencies.
	seenUp := map[string]bool{selectedStepID: true}
	queueUp := []string{selectedStepID}
	for len(queueUp) > 0 {
		curr := queueUp[0]
		queueUp = queueUp[1:]
		step := stepsByID[curr]
		for _, dep := range step.DependsOn {
			sourceID := string(dep)
			highlighted[edgeKey(sourceID, curr)] = true
			if seenUp[sourceID] {
				continue
			}
			seenUp[sourceID] = true
			queueUp = append(queueUp, sourceID)
		}
	}

	// Downstream: selected -> dependents.
	seenDown := map[string]bool{selectedStepID: true}
	queueDown := []string{selectedStepID}
	for len(queueDown) > 0 {
		curr := queueDown[0]
		queueDown = queueDown[1:]
		for _, child := range dependents[curr] {
			highlighted[edgeKey(curr, child)] = true
			if seenDown[child] {
				continue
			}
			seenDown[child] = true
			queueDown = append(queueDown, child)
		}
	}

	return highlighted
}

func spinnerGlyph(frame int) string {
	if len(Spinner) == 0 {
		return ""
	}
	if frame < 0 {
		frame = 0
	}
	return string(Spinner[frame%len(Spinner)])
}

func spinnerFrameCount() int {
	return len(Spinner)
}
