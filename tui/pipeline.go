package tui

var Spinner = []rune("⣾⣽⣻⢿⡿⣟⣯⣷")

const statusSelected StepVisualStatus = "StatusSelected"

type stepView struct {
	ID        string
	JobName   string
	DependsOn []string
	Status    StepVisualStatus
	Spinner   bool
	SpinChar  string
}

type stepPositionView struct {
	Column int
	Row    int
	Width  int
}

func (v stepPositionView) PortIn() stepPositionView {
	return stepPositionView{v.Column - 1, v.Row, v.Width}
}

type pipelineView struct {
	Columns         [][]stepView
	Positions       map[string]stepPositionView
	RowCount        int
	HighlightedEdge map[string]bool
}

func buildPipelineView(spec PipelineSpec, stepStates map[StepID]StepRuntimeState, spinnerFrame int, selectedStepID string) (pipelineView, error) {
	columns, positions, rowCount, err := spec.Layout()
	if err != nil {
		return pipelineView{}, err
	}
	highlightedEdges := highlightedEdgesForSelection(spec, selectedStepID)

	viewCols := make([][]stepView, len(columns))
	stepsByID := make(map[string]stepView, len(spec.Steps))
	for i, col := range columns {
		viewCol := make([]stepView, 0, len(col))
		for _, step := range col {
			deps := make([]string, 0, len(step.DependsOn))
			for _, dep := range step.DependsOn {
				deps = append(deps, string(dep))
			}

			status := step.Status
			spinner := false
			if state, ok := stepStates[step.ID]; ok {
				if state.Status != "" {
					status = state.Status
				}
				spinner = state.Spinner
			}
			if selectedStepID != "" && selectedStepID == string(step.ID) {
				status = statusSelected
			}

			viewStep := stepView{
				ID:        string(step.ID),
				JobName:   step.JobName,
				DependsOn: deps,
				Status:    status,
				Spinner:   spinner,
				SpinChar:  spinnerGlyph(spinnerFrame),
			}
			viewCol = append(viewCol, viewStep)
			stepsByID[viewStep.ID] = viewStep
		}
		viewCols[i] = viewCol
	}

	viewPos := make(map[string]stepPositionView, len(positions))
	for stepID, pos := range positions {
		step, ok := stepsByID[string(stepID)]
		width := 0
		if ok {
			width = NewStepComponent(step, 0).PreferredWidth()
		}
		viewPos[string(stepID)] = stepPositionView{Column: pos.Column, Row: pos.Row, Width: width}
	}

	return pipelineView{
		Columns:         viewCols,
		Positions:       viewPos,
		RowCount:        rowCount,
		HighlightedEdge: highlightedEdges,
	}, nil
}

func edgeKey(sourceID, targetID string) string {
	return sourceID + "->" + targetID
}

func highlightedEdgesForSelection(spec PipelineSpec, selectedStepID string) map[string]bool {
	highlighted := map[string]bool{}
	if selectedStepID == "" {
		return highlighted
	}

	stepsByID := make(map[string]StepSpec, len(spec.Steps))
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
