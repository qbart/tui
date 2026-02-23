package hestia

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"time"
)

type StepID string

type StepStatus string

const (
	StepStatusIdle   StepStatus = "idle"
	StepStatusDoing  StepStatus = "doing"
	StepStatusDone   StepStatus = "done"
	StepStatusFailed StepStatus = "failed"
)

type PipelineRunStatus string

const (
	PipelineRunStatusRunning   PipelineRunStatus = "running"
	PipelineRunStatusSucceeded PipelineRunStatus = "succeeded"
	PipelineRunStatusFailed    PipelineRunStatus = "failed"
)

type StepSpec struct {
	ID        StepID
	JobName   string
	DependsOn []StepID
	Command   string
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
	Status     StepStatus
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
		stepRuns[step.ID] = &StepRun{Status: StepStatusIdle}
	}

	return PipelineRun{ID: runID, SpecID: spec.ID, Status: PipelineRunStatusRunning, StartedAt: startedAt, StepRuns: stepRuns}, nil
}

func (r PipelineRun) IsTerminal() bool {
	return r.Status == PipelineRunStatusSucceeded || r.Status == PipelineRunStatusFailed
}

func (r PipelineRun) RunningStepID() (StepID, bool) {
	for id, stepRun := range r.StepRuns {
		if stepRun != nil && stepRun.Status == StepStatusDoing {
			return id, true
		}
	}
	return "", false
}

func (r PipelineRun) ReadySteps(spec PipelineSpec) []StepID {
	ready := make([]StepID, 0, len(spec.Steps))
	for _, step := range spec.Steps {
		stepRun := r.StepRuns[step.ID]
		if stepRun == nil || stepRun.Status != StepStatusIdle {
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
	if stepRun.Status != StepStatusIdle {
		return fmt.Errorf("step %q is not idle", stepID)
	}
	stepRun.Status = StepStatusDoing
	stepRun.StartedAt = &at
	r.Status = PipelineRunStatusRunning
	return nil
}

func (r *PipelineRun) CompleteStep(stepID StepID, at time.Time, success bool, exitCode int, logRef string) error {
	stepRun, ok := r.StepRuns[stepID]
	if !ok {
		return fmt.Errorf("unknown step %q", stepID)
	}
	if stepRun.Status != StepStatusDoing {
		return fmt.Errorf("step %q is not doing", stepID)
	}

	stepRun.FinishedAt = &at
	stepRun.LogRef = logRef
	stepRun.ExitCode = &exitCode
	if stepRun.StartedAt != nil {
		stepRun.Duration = at.Sub(*stepRun.StartedAt)
	}
	if success {
		stepRun.Status = StepStatusDone
	} else {
		stepRun.Status = StepStatusFailed
	}
	return nil
}

func (r *PipelineRun) RefreshStatus(spec PipelineSpec, at time.Time) {
	for _, step := range spec.Steps {
		stepRun := r.StepRuns[step.ID]
		if stepRun != nil && stepRun.Status == StepStatusFailed {
			r.Status = PipelineRunStatusFailed
			r.FinishedAt = &at
			return
		}
	}

	for _, step := range spec.Steps {
		stepRun := r.StepRuns[step.ID]
		if stepRun == nil || stepRun.Status != StepStatusDone {
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
		if !ok || depRun == nil || depRun.Status != StepStatusDone {
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
	columns, _, err := p.Columns()
	if err != nil {
		return nil, nil, 0, err
	}

	positions := make(map[StepID]StepPosition, len(p.Steps))
	occupied := map[int]map[int]bool{}
	nextBaseRow := 0
	maxRow := -1

	for colIdx, colSteps := range columns {
		if occupied[colIdx] == nil {
			occupied[colIdx] = map[int]bool{}
		}

		sorted := append([]StepSpec(nil), colSteps...)
		sort.SliceStable(sorted, func(i, j int) bool {
			a := sorted[i]
			b := sorted[j]
			aScore := dependencyRowScore(a, positions)
			bScore := dependencyRowScore(b, positions)
			if aScore != bScore {
				return aScore < bScore
			}
			return a.ID < b.ID
		})

		for _, step := range sorted {
			preferred := 0
			if len(step.DependsOn) == 0 {
				preferred = nextBaseRow
			} else {
				preferred = int(math.Round(dependencyRowScore(step, positions)))
			}

			row := findNearestFreeRow(occupied[colIdx], preferred)
			occupied[colIdx][row] = true
			positions[step.ID] = StepPosition{Column: colIdx, Row: row}
			if len(step.DependsOn) == 0 && row >= nextBaseRow {
				nextBaseRow = row + 1
			}
			if row > maxRow {
				maxRow = row
			}
		}
	}

	return columns, positions, maxRow + 1, nil
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
