package core

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

type StepID string

type StepRunStatus string

type StepVisualStatus string

const (
	StepRunIdle   StepRunStatus = "idle"
	StepRunDoing  StepRunStatus = "doing"
	StepRunFailed StepRunStatus = "failed"
	StepRunDone   StepRunStatus = "done"
)

const (
	StatusBlack  StepVisualStatus = "StatusBlack"
	StatusGray   StepVisualStatus = "StatusGray"
	StatusGreen  StepVisualStatus = "StatusGreen"
	StatusRed    StepVisualStatus = "StatusRed"
	StatusYellow StepVisualStatus = "StatusYellow"
	StatusBlue   StepVisualStatus = "StatusBlue"
	StatusPurple StepVisualStatus = "StatusPurple"
	StatusSelected StepVisualStatus = "StatusSelected"
)

type PipelineRunStatus string

const (
	PipelineRunStatusIdle      PipelineRunStatus = "idle"
	PipelineRunStatusRunning   PipelineRunStatus = "running"
	PipelineRunStatusSucceeded PipelineRunStatus = "succeeded"
	PipelineRunStatusFailed    PipelineRunStatus = "failed"
)

type StepSpec struct {
	ID        StepID
	Status    StepVisualStatus
	JobName   string
	DependsOn []StepID
}

type PipelineSpec struct {
	ID    string
	Steps []StepSpec
}

type StepPosition struct {
	Column int
	Row    int
}

type StepRun struct {
	Status     StepRunStatus
	StartedAt  *time.Time
	FinishedAt *time.Time
	ExitCode   *int
	LogRef     string
	RetryCount int
	Duration   time.Duration
}

type PipelineRun struct {
	ID         string
	SpecID     string
	Status     PipelineRunStatus
	StartedAt  time.Time
	FinishedAt *time.Time
	StepRuns   map[StepID]*StepRun
}

func NewPipelineSpec(id string, steps []StepSpec) PipelineSpec {
	return PipelineSpec{ID: id, Steps: steps}
}

func (p PipelineSpec) StepByID(id StepID) (StepSpec, bool) {
	for _, step := range p.Steps {
		if step.ID == id {
			return step, true
		}
	}
	return StepSpec{}, false
}

func (p PipelineSpec) Validate() error {
	if p.ID == "" {
		return errors.New("pipeline id is required")
	}
	if len(p.Steps) == 0 {
		return errors.New("pipeline requires at least one step")
	}

	seen := map[StepID]bool{}
	for _, step := range p.Steps {
		if step.ID == "" {
			return errors.New("step id is required")
		}
		if step.JobName == "" {
			return fmt.Errorf("step %q job name is required", step.ID)
		}
		if seen[step.ID] {
			return fmt.Errorf("duplicate step id %q", step.ID)
		}
		seen[step.ID] = true
	}

	for _, step := range p.Steps {
		for _, dep := range step.DependsOn {
			if dep == step.ID {
				return fmt.Errorf("step %q cannot depend on itself", step.ID)
			}
			if !seen[dep] {
				return fmt.Errorf("step %q depends on unknown step %q", step.ID, dep)
			}
		}
	}

	visiting := map[StepID]bool{}
	visited := map[StepID]bool{}
	var dfs func(StepID) error
	dfs = func(id StepID) error {
		if visited[id] {
			return nil
		}
		if visiting[id] {
			return fmt.Errorf("cycle detected at step %q", id)
		}
		visiting[id] = true
		step, _ := p.StepByID(id)
		for _, dep := range step.DependsOn {
			if err := dfs(dep); err != nil {
				return err
			}
		}
		visiting[id] = false
		visited[id] = true
		return nil
	}

	for _, step := range p.Steps {
		if err := dfs(step.ID); err != nil {
			return err
		}
	}

	return nil
}

func NewPipelineRun(spec PipelineSpec, runID string, startedAt time.Time) (PipelineRun, error) {
	if err := spec.Validate(); err != nil {
		return PipelineRun{}, err
	}
	if runID == "" {
		return PipelineRun{}, errors.New("run id is required")
	}

	stepRuns := make(map[StepID]*StepRun, len(spec.Steps))
	for _, step := range spec.Steps {
		stepRuns[step.ID] = &StepRun{Status: StepRunIdle}
	}

	return PipelineRun{ID: runID, SpecID: spec.ID, Status: PipelineRunStatusRunning, StartedAt: startedAt, StepRuns: stepRuns}, nil
}

func (r PipelineRun) IsTerminal() bool {
	return r.Status == PipelineRunStatusSucceeded || r.Status == PipelineRunStatusFailed
}

func (r PipelineRun) RunningStepID() (StepID, bool) {
	for id, stepRun := range r.StepRuns {
		if stepRun != nil && stepRun.Status == StepRunDoing {
			return id, true
		}
	}
	return "", false
}

func (r PipelineRun) ReadySteps(spec PipelineSpec) []StepID {
	ready := make([]StepID, 0, len(spec.Steps))
	for _, step := range spec.Steps {
		stepRun := r.StepRuns[step.ID]
		if stepRun == nil || stepRun.Status != StepRunIdle {
			continue
		}
		if r.dependenciesDone(step.DependsOn) {
			ready = append(ready, step.ID)
		}
	}
	return ready
}

func (r *PipelineRun) StartStep(stepID StepID, at time.Time) error {
	stepRun, ok := r.StepRuns[stepID]
	if !ok {
		return fmt.Errorf("unknown step %q", stepID)
	}
	if stepRun.Status != StepRunIdle {
		return fmt.Errorf("step %q is not idle", stepID)
	}
	stepRun.Status = StepRunDoing
	stepRun.StartedAt = &at
	r.Status = PipelineRunStatusRunning
	return nil
}

func (r *PipelineRun) CompleteStep(stepID StepID, at time.Time, success bool, exitCode int, logRef string) error {
	stepRun, ok := r.StepRuns[stepID]
	if !ok {
		return fmt.Errorf("unknown step %q", stepID)
	}
	if stepRun.Status != StepRunDoing {
		return fmt.Errorf("step %q is not doing", stepID)
	}

	stepRun.FinishedAt = &at
	stepRun.LogRef = logRef
	stepRun.ExitCode = &exitCode
	if stepRun.StartedAt != nil {
		stepRun.Duration = at.Sub(*stepRun.StartedAt)
	}
	if success {
		stepRun.Status = StepRunDone
	} else {
		stepRun.Status = StepRunFailed
	}
	return nil
}

func (r *PipelineRun) RefreshStatus(spec PipelineSpec, at time.Time) {
	for _, step := range spec.Steps {
		stepRun := r.StepRuns[step.ID]
		if stepRun != nil && stepRun.Status == StepRunFailed {
			r.Status = PipelineRunStatusFailed
			r.FinishedAt = &at
			return
		}
	}

	for _, step := range spec.Steps {
		stepRun := r.StepRuns[step.ID]
		if stepRun == nil || stepRun.Status != StepRunDone {
			r.Status = PipelineRunStatusRunning
			return
		}
	}

	r.Status = PipelineRunStatusSucceeded
	r.FinishedAt = &at
}

func (r PipelineRun) dependenciesDone(dependencies []StepID) bool {
	for _, depID := range dependencies {
		depRun, ok := r.StepRuns[depID]
		if !ok || depRun == nil || depRun.Status != StepRunDone {
			return false
		}
	}
	return true
}

func (p PipelineSpec) StepLevels() (map[StepID]int, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}

	levels := map[StepID]int{}
	var levelOf func(StepID) (int, error)
	levelOf = func(id StepID) (int, error) {
		if level, ok := levels[id]; ok {
			return level, nil
		}

		step, ok := p.StepByID(id)
		if !ok {
			return 0, fmt.Errorf("unknown step %q", id)
		}
		if len(step.DependsOn) == 0 {
			levels[id] = 0
			return 0, nil
		}

		maxDepLevel := 0
		for _, dep := range step.DependsOn {
			depLevel, err := levelOf(dep)
			if err != nil {
				return 0, err
			}
			if depLevel > maxDepLevel {
				maxDepLevel = depLevel
			}
		}

		levels[id] = maxDepLevel + 1
		return levels[id], nil
	}

	for _, step := range p.Steps {
		if _, err := levelOf(step.ID); err != nil {
			return nil, err
		}
	}

	return levels, nil
}

func (p PipelineSpec) Columns() ([][]StepSpec, map[StepID]int, error) {
	levels, err := p.StepLevels()
	if err != nil {
		return nil, nil, err
	}

	maxLevel := 0
	for _, level := range levels {
		if level > maxLevel {
			maxLevel = level
		}
	}

	columns := make([][]StepSpec, maxLevel+1)
	for _, step := range p.Steps {
		level := levels[step.ID]
		columns[level] = append(columns[level], step)
	}

	for i := range columns {
		sort.SliceStable(columns[i], func(a, b int) bool {
			return columns[i][a].ID < columns[i][b].ID
		})
	}

	return columns, levels, nil
}

func (p PipelineSpec) Layout() ([][]StepSpec, map[StepID]StepPosition, int, error) {
	columns, levels, err := p.Columns()
	if err != nil {
		return nil, nil, 0, err
	}

	positions, maxRow := computePositions(columns, levels, p.Steps)
	positions, maxRow = stackDisconnectedComponentsBelowPrimary(p.Steps, positions)
	return columns, positions, maxRow + 1, nil
}

func computePositions(columns [][]StepSpec, levels map[StepID]int, allSteps []StepSpec) (map[StepID]StepPosition, int) {
	positions := make(map[StepID]StepPosition, len(allSteps))
	occupied := map[int]map[int]bool{}
	maxRow := -1
	nextColumnDependents := buildNextColumnDependents(allSteps, levels)

	for colIdx, colSteps := range columns {
		if occupied[colIdx] == nil {
			occupied[colIdx] = map[int]bool{}
		}

		grouped := groupStepsByDependencySignature(colSteps)
		sort.SliceStable(grouped, func(i, j int) bool {
			aPreferred := int(math.Round(dependencyRowScore(grouped[i][0], positions)))
			bPreferred := int(math.Round(dependencyRowScore(grouped[j][0], positions)))
			if aPreferred != bPreferred {
				return aPreferred < bPreferred
			}
			return grouped[i][0].ID < grouped[j][0].ID
		})

		columnCursor := 0
		for _, group := range grouped {
			preferred := int(math.Round(dependencyRowScore(group[0], positions)))
			if preferred < 0 {
				preferred = 0
			}

			sort.SliceStable(group, func(i, j int) bool {
				aSpan := branchFootprint(group[i], nextColumnDependents)
				bSpan := branchFootprint(group[j], nextColumnDependents)
				if aSpan != bSpan {
					return aSpan > bSpan
				}
				return group[i].ID < group[j].ID
			})

			cursor := max(columnCursor, preferred)

			for _, step := range group {
				row := findNearestFreeRowAtOrBelow(occupied[colIdx], cursor)
				occupied[colIdx][row] = true
				positions[step.ID] = StepPosition{Column: colIdx, Row: row}

				cursor = row + branchFootprint(step, nextColumnDependents)
				if row > maxRow {
					maxRow = row
				}
			}
			columnCursor = cursor
		}
	}

	return positions, maxRow
}

func stackDisconnectedComponentsBelowPrimary(steps []StepSpec, positions map[StepID]StepPosition) (map[StepID]StepPosition, int) {
	if len(steps) == 0 || len(positions) == 0 {
		return positions, -1
	}

	comps := connectedComponents(steps)
	if len(comps) <= 1 {
		maxRow := -1
		for _, pos := range positions {
			if pos.Row > maxRow {
				maxRow = pos.Row
			}
		}
		return positions, maxRow
	}

	stepOrder := make(map[StepID]int, len(steps))
	for i, s := range steps {
		stepOrder[s.ID] = i
	}

	componentByStep := make(map[StepID]int, len(steps))
	for i, comp := range comps {
		for _, id := range comp {
			componentByStep[id] = i
		}
	}

	primaryComp := componentByStep[steps[0].ID]
	type bounds struct {
		minRow int
		maxRow int
	}
	compBounds := make(map[int]bounds, len(comps))
	for compIdx, comp := range comps {
		minRow := math.MaxInt
		maxRow := -1
		for _, id := range comp {
			pos, ok := positions[id]
			if !ok {
				continue
			}
			if pos.Row < minRow {
				minRow = pos.Row
			}
			if pos.Row > maxRow {
				maxRow = pos.Row
			}
		}
		if minRow == math.MaxInt {
			minRow = 0
		}
		compBounds[compIdx] = bounds{minRow: minRow, maxRow: maxRow}
	}

	primaryMax := compBounds[primaryComp].maxRow
	if primaryMax < 0 {
		primaryMax = 0
	}

	otherComps := make([]int, 0, len(comps)-1)
	for i := range comps {
		if i == primaryComp {
			continue
		}
		otherComps = append(otherComps, i)
	}
	sort.SliceStable(otherComps, func(i, j int) bool {
		a := comps[otherComps[i]]
		b := comps[otherComps[j]]
		if len(a) == 0 || len(b) == 0 {
			return len(a) < len(b)
		}
		return stepOrder[a[0]] < stepOrder[b[0]]
	})

	nextBase := primaryMax + 1
	for _, compIdx := range otherComps {
		compSet := make(map[StepID]bool, len(comps[compIdx]))
		for _, id := range comps[compIdx] {
			compSet[id] = true
		}

		subSteps := make([]StepSpec, 0, len(compSet))
		for _, step := range steps {
			if compSet[step.ID] {
				subSteps = append(subSteps, step)
			}
		}

		subSpec := PipelineSpec{ID: "component", Steps: subSteps}
		subColumns, subLevels, err := subSpec.Columns()
		if err != nil {
			// Fall back to existing positions when a sub-layout fails unexpectedly.
			b := compBounds[compIdx]
			shift := nextBase - b.minRow
			for _, id := range comps[compIdx] {
				pos := positions[id]
				pos.Row += shift
				positions[id] = pos
			}
			nextBase = b.maxRow + shift + 1
			continue
		}

		subPos, subMax := computePositions(subColumns, subLevels, subSteps)
		for _, id := range comps[compIdx] {
			local := subPos[id]
			local.Row += nextBase
			positions[id] = StepPosition{Column: local.Column, Row: local.Row}
		}
		nextBase += subMax + 1
	}

	maxRow := -1
	for _, pos := range positions {
		if pos.Row > maxRow {
			maxRow = pos.Row
		}
	}
	return positions, maxRow
}

func connectedComponents(steps []StepSpec) [][]StepID {
	adj := make(map[StepID][]StepID, len(steps))
	for _, s := range steps {
		adj[s.ID] = adj[s.ID]
	}
	for _, s := range steps {
		for _, dep := range s.DependsOn {
			adj[s.ID] = append(adj[s.ID], dep)
			adj[dep] = append(adj[dep], s.ID)
		}
	}

	seen := make(map[StepID]bool, len(steps))
	comps := make([][]StepID, 0)
	for _, s := range steps {
		if seen[s.ID] {
			continue
		}
		queue := []StepID{s.ID}
		seen[s.ID] = true
		comp := make([]StepID, 0)
		for len(queue) > 0 {
			id := queue[0]
			queue = queue[1:]
			comp = append(comp, id)
			for _, n := range adj[id] {
				if seen[n] {
					continue
				}
				seen[n] = true
				queue = append(queue, n)
			}
		}
		sort.SliceStable(comp, func(i, j int) bool { return comp[i] < comp[j] })
		comps = append(comps, comp)
	}
	return comps
}

func buildNextColumnDependents(steps []StepSpec, levels map[StepID]int) map[StepID][]StepID {
	dependents := make(map[StepID][]StepID, len(steps))
	for _, step := range steps {
		dependents[step.ID] = []StepID{}
	}

	for _, target := range steps {
		targetLevel, ok := levels[target.ID]
		if !ok {
			continue
		}
		for _, dep := range target.DependsOn {
			depLevel, ok := levels[dep]
			if !ok {
				continue
			}
			if targetLevel == depLevel+1 {
				dependents[dep] = append(dependents[dep], target.ID)
			}
		}
	}

	for dep := range dependents {
		sort.SliceStable(dependents[dep], func(i, j int) bool {
			return dependents[dep][i] < dependents[dep][j]
		})
	}

	return dependents
}

func groupStepsByDependencySignature(colSteps []StepSpec) [][]StepSpec {
	groups := make(map[string][]StepSpec)
	keys := make([]string, 0, len(colSteps))
	for _, step := range colSteps {
		key := dependencySignature(step.DependsOn)
		if _, ok := groups[key]; !ok {
			keys = append(keys, key)
		}
		groups[key] = append(groups[key], step)
	}

	sort.Strings(keys)
	result := make([][]StepSpec, 0, len(keys))
	for _, key := range keys {
		result = append(result, groups[key])
	}
	return result
}

func dependencySignature(dependsOn []StepID) string {
	if len(dependsOn) == 0 {
		return "__root__"
	}

	parts := make([]string, 0, len(dependsOn))
	for _, dep := range dependsOn {
		parts = append(parts, string(dep))
	}
	sort.Strings(parts)
	return strings.Join(parts, "|")
}

func branchFootprint(step StepSpec, nextColumnDependents map[StepID][]StepID) int {
	children := nextColumnDependents[step.ID]
	if len(children) == 0 {
		return 1
	}
	return len(children)
}

func dependencyRowScore(step StepSpec, positions map[StepID]StepPosition) float64 {
	if len(step.DependsOn) == 0 {
		return 0
	}

	total := 0
	count := 0
	for _, dep := range step.DependsOn {
		pos, ok := positions[dep]
		if !ok {
			continue
		}
		total += pos.Row
		count++
	}
	if count == 0 {
		return 0
	}
	return float64(total) / float64(count)
}

func findNearestFreeRow(used map[int]bool, preferred int) int {
	if preferred < 0 {
		preferred = 0
	}
	if !used[preferred] {
		return preferred
	}

	for offset := 1; ; offset++ {
		up := preferred - offset
		if up >= 0 && !used[up] {
			return up
		}
		down := preferred + offset
		if !used[down] {
			return down
		}
	}
}

func findNearestFreeRowAtOrBelow(used map[int]bool, preferred int) int {
	if preferred < 0 {
		preferred = 0
	}
	for row := preferred; ; row++ {
		if !used[row] {
			return row
		}
	}
}
