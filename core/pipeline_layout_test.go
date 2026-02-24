package core

import "testing"

func TestLayout_PushesSiblingBelowFanoutFootprint(t *testing.T) {
	spec := NewPipelineSpec("layout-footprint", []StepSpec{
		{ID: "checkout", JobName: "checkout"},
		{ID: "build", JobName: "build", DependsOn: []StepID{"checkout"}},
		{ID: "build-ui", JobName: "build ui", DependsOn: []StepID{"checkout"}},
		{ID: "test-postgresql", JobName: "test postgresql", DependsOn: []StepID{"build"}},
		{ID: "test-sqlite", JobName: "test sqlite", DependsOn: []StepID{"build"}},
		{ID: "test-duckdb", JobName: "test duckdb", DependsOn: []StepID{"build"}},
		{ID: "deploy", JobName: "deploy", DependsOn: []StepID{"test-postgresql", "test-sqlite", "test-duckdb"}},
		{ID: "deploy-ui", JobName: "deploy ui", DependsOn: []StepID{"build-ui"}},
		{ID: "notify", JobName: "notify", DependsOn: []StepID{"deploy", "deploy-ui"}},
	})

	_, positions, _, err := spec.Layout()
	if err != nil {
		t.Fatalf("layout failed: %v", err)
	}

	build := positions["build"]
	buildUI := positions["build-ui"]
	if build.Column != buildUI.Column {
		t.Fatalf("expected build and build-ui in same column, got %d and %d", build.Column, buildUI.Column)
	}
	if buildUI.Row <= build.Row {
		t.Fatalf("expected build-ui below build, got build row=%d build-ui row=%d", build.Row, buildUI.Row)
	}

	maxTestRow := positions["test-postgresql"].Row
	for _, id := range []StepID{"test-sqlite", "test-duckdb"} {
		if positions[id].Row > maxTestRow {
			maxTestRow = positions[id].Row
		}
	}
	if buildUI.Row <= maxTestRow {
		t.Fatalf("expected build-ui row=%d to be below fanout rows up to %d", buildUI.Row, maxTestRow)
	}
}
