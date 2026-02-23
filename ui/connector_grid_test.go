package ui

import (
	"strings"
	"testing"

	core "hestia/hestia"
)

func TestConnectorGrid_FanoutFromSingleSource(t *testing.T) {
	spec := core.NewPipelineSpec("fanout", []core.StepSpec{
		{ID: "a", JobName: "a"},
		{ID: "b", JobName: "b", DependsOn: []core.StepID{"a"}},
		{ID: "c", JobName: "c", DependsOn: []core.StepID{"a"}},
		{ID: "d", JobName: "d", DependsOn: []core.StepID{"a"}},
	})

	view, err := BuildPipelineView(spec, core.PipelineRun{})
	if err != nil {
		t.Fatalf("build pipeline view: %v", err)
	}

	grid := buildConnectorGrid(view)
	sourcePos := view.Positions["a"]
	j, ok := grid.rowJunction(sourcePos.Column, sourcePos.Row)
	if !ok {
		t.Fatalf("missing source junction")
	}
	if !j.Left || !j.Right || !j.Down {
		t.Fatalf("expected source fanout junction left+right+down, got %+v", j)
	}

	if !grid.hasBoundaryVertical(sourcePos.Column, sourcePos.Row) {
		t.Fatalf("expected vertical boundary continuation below source row")
	}
}

func TestConnectorGrid_FaninToSingleTarget(t *testing.T) {
	spec := core.NewPipelineSpec("fanin", []core.StepSpec{
		{ID: "a", JobName: "a"},
		{ID: "b", JobName: "b"},
		{ID: "c", JobName: "c"},
		{ID: "t", JobName: "t", DependsOn: []core.StepID{"a", "b", "c"}},
	})

	view, err := BuildPipelineView(spec, core.PipelineRun{})
	if err != nil {
		t.Fatalf("build pipeline view: %v", err)
	}

	grid := buildConnectorGrid(view)
	targetPos := view.Positions["t"]
	j, ok := grid.rowJunction(targetPos.Column-1, targetPos.Row)
	if !ok {
		t.Fatalf("missing fanin junction at target lane")
	}
	if !j.Right || !j.Down {
		t.Fatalf("expected merge-compatible target lane junction (right+down), got %+v", j)
	}
}

func TestConnectorGrid_LongEdgeAcrossColumns(t *testing.T) {
	spec := core.NewPipelineSpec("long-edge", []core.StepSpec{
		{ID: "checkout", JobName: "checkout"},
		{ID: "build", JobName: "build", DependsOn: []core.StepID{"checkout"}},
		{ID: "build-ui", JobName: "build ui", DependsOn: []core.StepID{"checkout"}},
		{ID: "test-postresql", JobName: "test postresql", DependsOn: []core.StepID{"build"}},
		{ID: "test-sqlite", JobName: "test sqlite", DependsOn: []core.StepID{"build"}},
		{ID: "test-duckdb", JobName: "test duckdb", DependsOn: []core.StepID{"build"}},
		{ID: "deploy", JobName: "deploy", DependsOn: []core.StepID{"test-postresql", "test-sqlite", "test-duckdb"}},
		{ID: "deploy-ui", JobName: "deploy ui", DependsOn: []core.StepID{"build-ui"}},
		{ID: "notify", JobName: "notify", DependsOn: []core.StepID{"deploy", "deploy-ui"}},
	})

	view, err := BuildPipelineView(spec, core.PipelineRun{})
	if err != nil {
		t.Fatalf("build pipeline view: %v", err)
	}

	grid := buildConnectorGrid(view)
	sourcePos := view.Positions["deploy-ui"]
	targetPos := view.Positions["notify"]
	if targetPos.Column-sourcePos.Column <= 1 {
		t.Fatalf("expected long edge across columns, source=%+v target=%+v", sourcePos, targetPos)
	}

	targetLane := targetPos.Column - 1
	if targetLane <= sourcePos.Column {
		t.Fatalf("expected distinct target lane for long edge")
	}
	for lane := sourcePos.Column + 1; lane < targetLane; lane++ {
		j, ok := grid.rowJunction(lane, sourcePos.Row)
		if !ok {
			t.Fatalf("missing pass-through lane junction at lane=%d row=%d", lane, sourcePos.Row)
		}
		if !j.Left || !j.Right {
			t.Fatalf("expected pass-through in intermediate lane, got %+v", j)
		}
	}
	if !grid.hasBoundaryVertical(targetLane, min(sourcePos.Row, targetPos.Row)) {
		t.Fatalf("expected vertical bend at target lane")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func TestConnectorGrid_EqualMergeProducesCross(t *testing.T) {
	view := PipelineView{
		Columns: [][]StepView{
			{
				{ID: "a", JobName: "a"},
				{ID: "m", JobName: "m"},
				{ID: "z", JobName: "z"},
			},
			{
				{ID: "t", JobName: "t", DependsOn: []string{"a", "m", "z"}},
			},
		},
		Positions: map[string]StepPositionView{
			"a": {Column: 0, Row: 0},
			"m": {Column: 0, Row: 1},
			"z": {Column: 0, Row: 2},
			"t": {Column: 1, Row: 1},
		},
		RowCount: 3,
	}

	grid := buildConnectorGrid(view)
	j, ok := grid.rowJunction(0, 1)
	if !ok {
		t.Fatalf("missing merged junction")
	}
	if !j.Left || !j.Right || !j.Up || !j.Down {
		t.Fatalf("expected cross junction (all directions), got %+v", j)
	}

	arrow := NewArrowComponent(5, ArrowTypeSolid, "", "")
	rendered := arrow.RenderJunction(j.Left, j.Right, j.Up, j.Down, true)
	if !strings.Contains(rendered, "╋") {
		t.Fatalf("expected cross glyph, got %q", rendered)
	}
}

func TestRenderPipelineGraph_SampleContainsOutPortMarkersOnly(t *testing.T) {
	spec := core.NewPipelineSpec("sample-cicd", []core.StepSpec{
		{ID: "checkout", JobName: "checkout"},
		{ID: "build", JobName: "build", DependsOn: []core.StepID{"checkout"}},
		{ID: "build-ui", JobName: "build ui", DependsOn: []core.StepID{"checkout"}},
		{ID: "test-postresql", JobName: "test postresql", DependsOn: []core.StepID{"build"}},
		{ID: "test-sqlite", JobName: "test sqlite", DependsOn: []core.StepID{"build"}},
		{ID: "test-duckdb", JobName: "test duckdb", DependsOn: []core.StepID{"build"}},
		{ID: "deploy", JobName: "deploy", DependsOn: []core.StepID{"test-postresql", "test-sqlite", "test-duckdb"}},
		{ID: "deploy-ui", JobName: "deploy ui", DependsOn: []core.StepID{"build-ui"}},
		{ID: "notify", JobName: "notify", DependsOn: []core.StepID{"deploy", "deploy-ui"}},
	})

	view, err := BuildPipelineView(spec, core.PipelineRun{})
	if err != nil {
		t.Fatalf("build pipeline view: %v", err)
	}

	lines := renderPipelineGraph(view)
	raw := strings.Join(lines, "\n")
	if !strings.Contains(raw, ">") {
		t.Fatalf("expected out-port markers in rendered graph, got %q", raw)
	}
	if !strings.Contains(raw, "*") {
		t.Fatalf("expected in-port markers in rendered graph, got %q", raw)
	}
	if strings.ContainsAny(raw, "┃━┣┫┳┻┗┏╋#.") {
		t.Fatalf("expected no connector/old debug glyphs except '*' in-ports, got %q", raw)
	}
}
